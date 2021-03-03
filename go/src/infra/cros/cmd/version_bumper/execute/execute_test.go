// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package execute

import (
	"context"
	"path/filepath"
	"testing"

	"infra/cros/internal/chromeosversion"

	vpb "go.chromium.org/chromiumos/infra/proto/go/chromiumos/version_bumper"
)

func TestBumpVersionBadArgs(t *testing.T) {
	ctx := context.Background()
	err := Run(ctx, &vpb.BumpVersionRequest{})
	// missing --chromiumos_overlay_repo
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	err = Run(ctx, &vpb.BumpVersionRequest{
		ChromiumosOverlayRepo: "foo",
	})

	// bad --chromiumos_overlay_repo
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	testDir, err := filepath.Abs("test/")
	if err != nil {
		t.Fatal(err)
	}

	err = Run(ctx, &vpb.BumpVersionRequest{
		ChromiumosOverlayRepo: testDir,
	})
	// no component specified
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

type bumpVersionTest struct {
	component       vpb.BumpVersionRequest_VersionComponent
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
	ctx := context.Background()
	testDir, err := filepath.Abs("test/")
	if err != nil {
		t.Fatal(err)
	}

	tests := []bumpVersionTest{
		{
			component: vpb.BumpVersionRequest_COMPONENT_TYPE_MILESTONE,
			expectedVersion: chromeosversion.VersionInfo{
				ChromeBranch:      91,
				BuildNumber:       13781,
				BranchBuildNumber: 0,
				PatchNumber:       0,
			},
		},
		{
			component: vpb.BumpVersionRequest_COMPONENT_TYPE_BUILD,
			expectedVersion: chromeosversion.VersionInfo{
				ChromeBranch:      90,
				BuildNumber:       13782,
				BranchBuildNumber: 0,
				PatchNumber:       0,
			},
		},
		{
			component: vpb.BumpVersionRequest_COMPONENT_TYPE_BRANCH,
			expectedVersion: chromeosversion.VersionInfo{
				ChromeBranch:      90,
				BuildNumber:       13781,
				BranchBuildNumber: 1,
				PatchNumber:       0,
			},
		},
		{
			component: vpb.BumpVersionRequest_COMPONENT_TYPE_PATCH,
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

		err := Run(ctx, &vpb.BumpVersionRequest{
			ChromiumosOverlayRepo: testDir,
			ComponentToBump:       test.component,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
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
