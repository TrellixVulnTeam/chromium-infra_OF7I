// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"path/filepath"
	"testing"

	"infra/cros/internal/chromeosversion"

	"github.com/maruel/subcommands"
)

func TestBumpVersionBadArgs(t *testing.T) {
	s := GetApplication()
	ret := subcommands.Run(s, []string{
		"bump_version",
	})
	// missing --chromiumos_overlay_repo
	if ret != 1 {
		t.Fatalf("expected ret code 1, got %d", ret)
	}

	ret = subcommands.Run(s, []string{
		"bump_version",
		"--chromiumos_overlay_repo",
		"foo",
	})

	// bad --chromiumos_overlay_repo
	if ret != 1 {
		t.Fatalf("expected ret code 1, got %d", ret)
	}

	testDir, err := filepath.Abs("test/")
	if err != nil {
		t.Fatal(err)
	}
	ret = subcommands.Run(s, []string{
		"bump_version",
		"--chromiumos_overlay_repo",
		testDir,
		"--bump_milestone_component",
		"--bump_build_component",
	})

	// multiple components specified
	if ret != 1 {
		t.Fatalf("expected ret code 1, got %d", ret)
	}
}

type bumpVersionTest struct {
	flag            string
	expectedVersion chromeosversion.VersionInfo
}

func resetTestFile() error {
	testVersion := chromeosversion.VersionInfo{
		ChromeBranch:      90,
		BuildNumber:       13781,
		BranchBuildNumber: 0,
		PatchNumber:       0,
	}

	testFile, err := filepath.Abs("test/chromeos/config/chromeos_version.sh")
	if err != nil {
		return err
	}
	testVersion.VersionFile = testFile
	return testVersion.UpdateVersionFile()
}

func TestBumpVersion(t *testing.T) {
	s := GetApplication()

	testDir, err := filepath.Abs("test/")
	if err != nil {
		t.Fatal(err)
	}

	tests := []bumpVersionTest{
		{
			flag: "--bump_milestone_component",
			expectedVersion: chromeosversion.VersionInfo{
				ChromeBranch:      91,
				BuildNumber:       13781,
				BranchBuildNumber: 0,
				PatchNumber:       0,
			},
		},
		{
			flag: "--bump_build_component",
			expectedVersion: chromeosversion.VersionInfo{
				ChromeBranch:      90,
				BuildNumber:       13782,
				BranchBuildNumber: 0,
				PatchNumber:       0,
			},
		},
		{
			flag: "--bump_branch_component",
			expectedVersion: chromeosversion.VersionInfo{
				ChromeBranch:      90,
				BuildNumber:       13781,
				BranchBuildNumber: 1,
				PatchNumber:       0,
			},
		},
		{
			flag: "--bump_patch_component",
			expectedVersion: chromeosversion.VersionInfo{
				ChromeBranch:      90,
				BuildNumber:       13781,
				BranchBuildNumber: 0,
				PatchNumber:       1,
			},
		},
	}

	for _, test := range tests {
		if err := resetTestFile(); err != nil {
			t.Fatal(err)
		}

		ret := subcommands.Run(s, []string{
			"bump_version",
			"--chromiumos_overlay_repo",
			testDir,
			test.flag,
		})
		if ret != 0 {
			t.Fatalf("unexpected non-zero return code: %d", ret)
		}
		vinfo, err := chromeosversion.GetVersionInfoFromRepo(testDir)
		if err != nil {
			t.Fatal(err)
		}
		if !chromeosversion.VersionsEqual(vinfo, test.expectedVersion) {
			t.Fatalf("mismatch on bump_version: got %+v, expected %+v", vinfo, test.expectedVersion)
		}
	}
}
