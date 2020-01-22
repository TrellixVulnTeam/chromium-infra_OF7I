// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package validateconfig

import (
	"fmt"
	"strings"
	"testing"

	labPlatform "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
)

// errorStartsWithDWIM checks if the error's message starts with a given prefix.
// If the prefix is empty, it checks whether the error is nil.
func errorStartsWithDWIM(err error, prefix string) bool {
	if prefix == "" {
		return err == nil
	}
	return strings.HasPrefix(err.Error(), prefix)
}

// errorWithDefault is a helper function for testing against error strings.
func errorWithDefault(e error, def string) string {
	if e == nil {
		return def
	}
	return e.Error()
}

// parseStableVersionsOrPanic is a helper function that's used in tests to feed
// a stable version file contained in a string literal to a test.
func parseStableVersionsOrPanic(contents []byte) *labPlatform.StableVersions {
	out, err := ParseStableVersions(contents)
	if err != nil {
		panic(err.Error())
	}
	return out
}

// wellformedStableVersion is based on the production stable_versions.cfg file.
// It does not contain any errors.
const wellformedStableVersion = `
{
	"cros": [{
		"key": {
			"modelId": {},
			"buildTarget": {"name": "arkham"}
		},
		"version": "R77-12371.52.22"
	}],
	"firmware": [{
		"key": {
			"modelId": {"value": "arkham"},
			"buildTarget": {"name": "arkham"}
		},
		"version": "Google_Storm.6315.91.0"
	}],
	"faft": [{
		"key": {
			"modelId": {"value": "asuka"},
			"buildTarget": {"name": "asuka"}
		},
		"version": "asuka-firmware/R49-7820.314.0"
	}]
}
`

// stableVersionsWithDuplicates is similar to wellformedStableVersion, but every
// entry under cros, firmware, and faft is duplicated.
const stableVersionsWithDuplicates = `
{
	"cros": [
		{
			"key": {
				"modelId": {},
				"buildTarget": {"name": "arkham"}
			},
			"version": "R77-12371.52.22"
		},
		{
			"key": {
				"modelId": {},
				"buildTarget": {"name": "arkham"}
			},
			"version": "R77-12371.52.22"
		}
	],
	"firmware": [
		{
			"key": {
				"modelId": {"value": "arkham"},
				"buildTarget": {"name": "arkham"}
			},
			"version": "Google_Storm.6315.91.0"
		},
		{
			"key": {
				"modelId": {"value": "arkham"},
				"buildTarget": {"name": "arkham"}
			},
			"version": "Google_Storm.6315.91.0"
		}
	],
	"faft": [
		{
			"key": {
				"modelId": {"value": "asuka"},
				"buildTarget": {"name": "asuka"}
			},
			"version": "asuka-firmware/R49-7820.314.0"
		},
		{
			"key": {
				"modelId": {"value": "asuka"},
				"buildTarget": {"name": "asuka"}
			},
			"version": "asuka-firmware/R49-7820.314.0"
		}
	]
}
`

var testInspectBufferData = []struct {
	uuid string
	name string
	in   string
	out  string
}{
	{
		"4f3d1ae0-9e01-46c7-8b66-c48a226c4cb7",
		"len zero string",
		"",
		FileLenZero,
	},
	{
		"9178c0b9-20c7-4274-8fa3-839562f4305d",
		"not UTF-8",
		"\xee\xee\xee\xff",
		FileNotUTF8,
	},
	{
		"7ab6971c-7285-4e8a-9655-70ea95108da6",
		"invalid JSON",
		"aaaa",
		FileNotJSON,
	},
	{
		"4af11fce-bcc3-41cf-b4de-1544eb195ee7",
		"doesn't fit schema",
		"[2, 3, 4]",
		"JSON does not conform to schema",
	},
	{
		"27a855e7-639c-4de6-a32d-050c8a7574fe",
		"well-formed but empty",
		"{}",
		FileMissingCrosKey,
	},
}

func TestInspectBuffer(t *testing.T) {
	t.Parallel()
	for _, tt := range testInspectBufferData {
		t.Run(tt.uuid, func(t *testing.T) {
			e := InspectBuffer([]byte(tt.in))
			if !errorStartsWithDWIM(e, tt.out) {
				msg := fmt.Sprintf("uuid (%s): name (%s): got: (%q), want: (%q)", tt.uuid, tt.name, e.Error(), tt.out)
				t.Errorf(msg)
			}
		})
	}
}

var testIsValidJSONData = []struct {
	uuid string
	in   string
	out  bool
}{
	{
		"03564d2e-4000-43f0-8e28-a42be4233faa",
		"{}",
		true,
	},
	{
		"f97f6629-599f-4650-a5da-c3b32704323a",
		"{",
		false,
	},
}

func TestIsValidJSON(t *testing.T) {
	t.Parallel()
	for _, tt := range testIsValidJSONData {
		t.Run(tt.in, func(t *testing.T) {
			if res := isValidJSON([]byte(tt.in)); res != tt.out {
				msg := fmt.Sprintf("item (%s): got: (%v), want: (%v)", tt.in, res, tt.out)
				t.Errorf(msg)
			}
		})
	}
}

var testShallowValidateCrosVersionsData = []struct {
	uuid string
	in   string
	out  string
}{
	{
		"54fea7f2-06dc-4240-b829-adc523666682",
		wellformedStableVersion,
		"",
	},
	{
		"c6ebed7d-670a-4d8a-be22-ad8d37419247",
		stableVersionsWithDuplicates,
		makeShallowlyMalformedCrosEntry("duplicate entry for buildTarget", 1, "arkham", "", "R77-12371.52.22").Error(),
	},
}

func TestShallowValidateCrosVersions(t *testing.T) {
	for _, tt := range testShallowValidateCrosVersionsData {
		t.Run(tt.uuid, func(t *testing.T) {
			sv := parseStableVersionsOrPanic([]byte(tt.in))
			res := errorWithDefault(shallowValidateCrosVersions(sv), "")
			if res != tt.out {
				msg := fmt.Sprintf("uuid (%s): got: (%v), want: (%v)", tt.uuid, res, tt.out)
				t.Errorf(msg)
			}
		})
	}
}

var testShallowValidateFirmwareVersionsData = []struct {
	uuid string
	in   string
	out  string
}{
	{
		"d79ad50c-d280-409a-ac61-852388fac5d7",
		wellformedStableVersion,
		"",
	},
	{
		"2a83eb17-62cd-46ec-bbd6-ac254ae9c747",
		stableVersionsWithDuplicates,
		makeShallowlyMalformedFirmwareEntry("duplicate entry", 1, "arkham", "arkham", "Google_Storm.6315.91.0").Error(),
	},
}

func TestShallowValidateFirmwareVersions(t *testing.T) {
	for _, tt := range testShallowValidateFirmwareVersionsData {
		t.Run(tt.uuid, func(t *testing.T) {
			sv := parseStableVersionsOrPanic([]byte(tt.in))
			res := errorWithDefault(shallowValidateFirmwareVersions(sv), "")
			if res != tt.out {
				msg := fmt.Sprintf("uuid (%s): got: (%v), want: (%v)", tt.uuid, res, tt.out)
				t.Errorf(msg)
			}
		})
	}
}

var testShallowValidateFaftVersionsData = []struct {
	uuid string
	in   string
	out  string
}{
	{
		"f4acbcd2-5da7-419a-8697-bcd197ede200",
		wellformedStableVersion,
		"",
	},
	{
		"ce6af6c8-d67a-42b5-87f7-8b11e66d4377",
		stableVersionsWithDuplicates,
		makeShallowlyMalformedFaftEntry("duplicate entry", 1, "asuka", "asuka", "asuka-firmware/R49-7820.314.0").Error(),
	},
}

func TestShallowValidateFaftVersions(t *testing.T) {
	for _, tt := range testShallowValidateFaftVersionsData {
		t.Run(tt.uuid, func(t *testing.T) {
			sv := parseStableVersionsOrPanic([]byte(tt.in))
			res := errorWithDefault(shallowValidateFaftVersions(sv), "")
			if res != tt.out {
				msg := fmt.Sprintf("uuid (%s): got: (%v), want: (%v)", tt.uuid, res, tt.out)
				t.Errorf(msg)
			}
		})
	}
}
