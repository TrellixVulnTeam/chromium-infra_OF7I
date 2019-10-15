// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestValidateCrOSVersion(t *testing.T) {
	good := func(s string) {
		if err := ValidateCrOSVersion(s); err != nil {
			t.Errorf("expected `%s' to be good (%s)", s, err)
		}
	}
	bad := func(s string) {
		if ValidateCrOSVersion(s) == nil {
			t.Errorf("expected `%s' to be bad", s)
		}
	}
	bad("")
	good("R1-2.3.4")
	bad("a-firmware/R1-2.3.4")
	bad("octopus-firmware/R72-11297.75.0")
	bad("Google_Rammus.11275.41.0")
}

func TestSerializeCrOSVersion(t *testing.T) {
	out := SerializeCrOSVersion(1, 2, 3, 4)
	if out != "R1-2.3.4" {
		t.Errorf("expected: R1-2.3.4 got:%s", out)
	}
}

func TestParseCrOSVersion(t *testing.T) {
	Convey("Test parsing CrOS Version", t, func() {
		release, tip, branch, branchBranch, err := ParseCrOSVersion("R1-2.3.4")
		if err != nil {
			t.Errorf("expected R1-2.3.4 to parse: %s", err)
		} else {
			So(release, ShouldEqual, 1)
			So(tip, ShouldEqual, 2)
			So(branch, ShouldEqual, 3)
			So(branchBranch, ShouldEqual, 4)
		}
	})
}

func TestValidateFirmwareVersion(t *testing.T) {
	good := func(s string) {
		if err := ValidateFirmwareVersion(s); err != nil {
			t.Errorf("expected `%s' to be good (%s)", s, err)
		}
	}
	bad := func(s string) {
		if ValidateFirmwareVersion(s) == nil {
			t.Errorf("expected `%s' to be bad", s)
		}
	}
	bad("")
	bad("R1-2.3.4")
	good("a-firmware/R1-2.3.4")
	good("octopus-firmware/R72-11297.75.0")
	bad("Google_Rammus.11275.41.0")
}

func TestSerializeFirmwareVersion(t *testing.T) {
	out := SerializeFirmwareVersion("a", 1, 2, 3, 4)
	if out != "a-firmware/R1-2.3.4" {
		t.Errorf("expected: R1-2.3.4 got:%s", out)
	}
}

func TestParseFirmwareVersion(t *testing.T) {
	Convey("Test Parsing Firwmare Version", t, func() {
		platform, release, tip, branch, branchBranch, err := ParseFirmwareVersion("a-firmware/R1-2.3.4")
		if err != nil {
			t.Errorf("expected a-firmware/R1-2.3.4 to parse: %s", err)
		} else {
			So(platform, ShouldEqual, "a")
			So(release, ShouldEqual, 1)
			So(tip, ShouldEqual, 2)
			So(branch, ShouldEqual, 3)
			So(branchBranch, ShouldEqual, 4)
		}
	})
}

func TestValidateFaftVersion(t *testing.T) {
	good := func(s string) {
		if err := ValidateFaftVersion(s); err != nil {
			t.Errorf("expected `%s' to be good (%s)", s, err)
		}
	}
	bad := func(s string) {
		if ValidateFaftVersion(s) == nil {
			t.Errorf("expected `%s' to be bad", s)
		}
	}
	bad("")
	bad("R1-2.3.4")
	bad("a-firmware/R1-2.3.4")
	bad("octopus-firmware/R72-11297.75.0")
	good("Google_Rammus.11275.41.0")
}

func TestSerializeFaftVersion(t *testing.T) {
	out := SerializeFaftVersion("Google", "Rammus", 11275, 41, 0)
	if out != "Google_Rammus.11275.41.0" {
		t.Errorf("expected: R1-2.3.4 got:%s", out)
	}
}

func TestParseFaftVersion(t *testing.T) {
	Convey("Test Parsing RW Firmware Version", t, func() {
		company, platform, tip, branch, branchBranch, err := ParseFaftVersion("Google_Rammus.11275.41.0")
		if err != nil {
			t.Errorf("expected Google_Rammus.11275.41.0 to parse: %s", err)
		} else {
			So(company, ShouldEqual, "Google")
			So(platform, ShouldEqual, "Rammus")
			So(tip, ShouldEqual, 11275)
			So(branch, ShouldEqual, 41)
			So(branchBranch, ShouldEqual, 0)
		}
	})
}
