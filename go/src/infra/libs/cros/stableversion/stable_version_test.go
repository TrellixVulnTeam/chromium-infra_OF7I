// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
)

func TestCompareCrOSVersions(t *testing.T) {
	Convey("Test v1 > v2", t, func() {
		v1 := "R2-2.3.4"
		v2 := "R1-2.3.4"
		cv, err := CompareCrOSVersions(v1, v2)
		So(err, ShouldBeNil)
		So(cv, ShouldEqual, 1)

		v1 = "R1-2.5.4"
		v2 = "R1-2.3.4"
		cv, err = CompareCrOSVersions(v1, v2)
		So(err, ShouldBeNil)
		So(cv, ShouldEqual, 1)
	})
	Convey("Test v1 < v2", t, func() {
		v1 := "R2-1.3.4"
		v2 := "R2-2.3.4"
		cv, err := CompareCrOSVersions(v1, v2)
		So(err, ShouldBeNil)
		So(cv, ShouldEqual, -1)

		v1 = "R1-2.3.4"
		v2 = "R1-2.3.5"
		cv, err = CompareCrOSVersions(v1, v2)
		So(err, ShouldBeNil)
		So(cv, ShouldEqual, -1)
	})
	Convey("Test v1 == v2", t, func() {
		v1 := "R1-2.3.4"
		v2 := "R1-2.3.4"
		cv, err := CompareCrOSVersions(v1, v2)
		So(err, ShouldBeNil)
		So(cv, ShouldEqual, 0)
	})
}

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

func TestAddUpdatedCros(t *testing.T) {
	old := makeBaseStableVersions(
		[]versions{
			{"b1", "m1", "R1-1.1.1"},
			{"b2", "m2", "R2-2.2.2"},
		},
		nil,
		nil,
	)
	updated := makeBaseStableVersions(
		[]versions{
			{"b1", "m1", "R1-1.1.1111"},
			{"b3", "m3", "R3-3.3.3"},
		},
		nil,
		nil,
	)
	res := AddUpdatedCros(old.Cros, updated.Cros)
	m := make(map[string]string, len(res))
	for _, r := range res {
		m[crosSVKey(r)] = r.GetVersion()
	}

	Convey("Test add", t, func() {
		So(m["b3"], ShouldEqual, "R3-3.3.3")
	})

	Convey("Test update", t, func() {
		So(m["b1"], ShouldEqual, "R1-1.1.1111")
	})

	Convey("Test reserve", t, func() {
		So(m["b2"], ShouldEqual, "R2-2.2.2")
	})
}

func TestAddUpdatedFirmware(t *testing.T) {
	old := makeBaseStableVersions(
		nil,
		nil,
		[]versions{
			{"b1", "m1", "a-firmware/R1-1.1.1"},
			{"b2", "m2", "a-firmware/R2-2.2.2"},
		},
	)
	updated := makeBaseStableVersions(
		nil,
		nil,
		[]versions{
			{"b1", "m1", "a-firmware/R1-1.1.1111"},
			{"b3", "m3", "a-firmware/R3-3.3.3"},
		},
	)
	res := AddUpdatedFirmware(old.Firmware, updated.Firmware)
	m := make(map[string]string, len(res))
	for _, r := range res {
		m[firmwareSVKey(r)] = r.GetVersion()
	}

	Convey("Test add", t, func() {
		So(m["b3:m3"], ShouldEqual, "a-firmware/R3-3.3.3")
	})

	Convey("Test update", t, func() {
		So(m["b1:m1"], ShouldEqual, "a-firmware/R1-1.1.1111")
	})

	Convey("Test reserve", t, func() {
		So(m["b2:m2"], ShouldEqual, "a-firmware/R2-2.2.2")
	})
}

func TestWriteSVToString(t *testing.T) {
	Convey("Test order of stable versions after writing to strings", t, func() {
		all := makeBaseStableVersions(
			[]versions{
				{"b1", "m1", "R1-1.1.1"},
				{"b2", "m2", "R2-2.2.2"},
			},
			[]versions{
				{"b1", "m1", "a-firmware/R1-1.1.1"},
				{"b1", "m2", "a-firmware/R2-2.2.2"},
			},
			[]versions{
				{"b1", "m2", "b-firmware/R1-1.1.1"},
				{"a1", "m1", "a-firmware/R1-1.1.1"},
			},
		)
		source :=
			`{
	"cros": [
		{
			"key": {
				"modelId": {
					"value": "m1"
				},
				"buildTarget": {
					"name": "b1"
				}
			},
			"version": "R1-1.1.1"
		},
		{
			"key": {
				"modelId": {
					"value": "m2"
				},
				"buildTarget": {
					"name": "b2"
				}
			},
			"version": "R2-2.2.2"
		}
	],
	"faft": [
		{
			"key": {
				"modelId": {
					"value": "m1"
				},
				"buildTarget": {
					"name": "b1"
				}
			},
			"version": "a-firmware/R1-1.1.1"
		},
		{
			"key": {
				"modelId": {
					"value": "m2"
				},
				"buildTarget": {
					"name": "b1"
				}
			},
			"version": "a-firmware/R2-2.2.2"
		}
	],
	"firmware": [
		{
			"key": {
				"modelId": {
					"value": "m1"
				},
				"buildTarget": {
					"name": "a1"
				}
			},
			"version": "a-firmware/R1-1.1.1"
		},
		{
			"key": {
				"modelId": {
					"value": "m2"
				},
				"buildTarget": {
					"name": "b1"
				}
			},
			"version": "b-firmware/R1-1.1.1"
		}
	]
}`
		s, err := WriteSVToString(all)
		fmt.Println(s)
		fmt.Println("~~~~~~~")
		fmt.Println(source)
		So(err, ShouldBeNil)
		So(s, ShouldEqual, source)
	})
}

type versions struct {
	bt string
	m  string
	v  string
}

func makeBaseStableVersions(cros, faft, firmware []versions) *sv.StableVersions {
	var cs []*sv.StableCrosVersion
	for _, c := range cros {
		cs = append(cs, &sv.StableCrosVersion{
			Key:     makeStableVersionKey(c.bt, c.m),
			Version: c.v,
		})
	}
	var fis []*sv.StableFirmwareVersion
	for _, c := range firmware {
		fis = append(fis, &sv.StableFirmwareVersion{
			Key:     makeStableVersionKey(c.bt, c.m),
			Version: c.v,
		})
	}
	var fas []*sv.StableFaftVersion
	for _, c := range faft {
		fas = append(fas, &sv.StableFaftVersion{
			Key:     makeStableVersionKey(c.bt, c.m),
			Version: c.v,
		})
	}
	return &sv.StableVersions{
		Cros:     cs,
		Firmware: fis,
		Faft:     fas,
	}
}

func makeStableVersionKey(buildTarget, model string) *sv.StableVersionKey {
	return &sv.StableVersionKey{
		ModelId: &device.ModelId{
			Value: model,
		},
		BuildTarget: &chromiumos.BuildTarget{
			Name: buildTarget,
		},
	}
}
