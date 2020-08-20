// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package querygs

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	labPlatform "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/luci/common/gcloud/gs"

	"infra/libs/cros/stableversion/validateconfig"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"infra/cmd/stable_version2/internal/utils"
)

const DONTCARE = "f7e8bdf6-f67c-4d63-aea3-46fa5e980403"

// NOERROR is used when inspecting error messages. It only matches a nil error value.
const NOERROR = "NO-ERROR--ca5fc27a-4353-478c-bda2-c20519a2e0ff"

// ANYERROR is used to match any non-nil error.
const ANYERROR = "ANY-ERROR--4430e445-67c1-46a7-90b9-fad144490b5d"

const exampleMetadataJSON = `
{
  "version": {
    "full": "R81-12835.0.0"
  },
  "results": [],
  "unibuild": true,
  "board-metadata": {
    "nami": {
      "models": {
        "sona": {
          "firmware-key-id": "SONA",
          "main-readonly-firmware-version": "Google_Nami.4.7.9",
          "ec-firmware-version": "nami_v1.2.3-4",
          "main-readwrite-firmware-version": "Google_Nami.42.43.44"
        },
        "akali360": {
          "firmware-key-id": "AKALI",
          "main-readonly-firmware-version": "Google_Nami.5.8.13",
          "ec-firmware-version": "nami_v1.2.3-4",
          "main-readwrite-firmware-version": "Google_Nami.52.53.54"
        }
      }
    }
  }
}
`

var testMaybeDownloadFileData = []struct {
	uuid     string
	metadata string
	out      *map[string]map[string]map[string]string
}{
	{
		"f959c762-214e-4293-b655-032cd791a85f",
		exampleMetadataJSON,
		&map[string]map[string]map[string]string{
			"nami": {
				"f7e8bdf6-f67c-4d63-aea3-46fa5e980403": {"akali360": "Google_Nami.52.53.54", "sona": "Google_Nami.42.43.44"},
			},
		},
	},
}

func TestMaybeDownloadFile(t *testing.T) {
	t.Parallel()
	for _, tt := range testMaybeDownloadFileData {
		t.Run(tt.uuid, func(t *testing.T) {
			var r Reader
			r.dld = makeConstantDownloader(tt.metadata)
			e := r.maybeDownloadFile(DONTCARE, DONTCARE)
			if e != nil {
				msg := fmt.Sprintf("uuid (%s): unexpected error (%s)", tt.uuid, e.Error())
				t.Errorf(msg)
			}
			diff := cmp.Diff(tt.out, r.cache)
			if diff != "" {
				msg := fmt.Sprintf("uuid (%s): unexpected diff (%s)", tt.uuid, diff)
				t.Errorf(msg)
			}
		})
	}
}

var testFirmwareVersionData = []struct {
	name        string
	uuid        string
	metadata    string
	bt          string
	model       string
	CrOSVersion string
	out         string
}{
	{
		"extract firmware version \"sona\"",
		"20db725f-9f0e-457b-8da7-3fb6dfb57b7b",
		exampleMetadataJSON,
		"nami",
		"sona",
		"xxx-cros-version",
		"Google_Nami.42.43.44",
	},
	{
		"extract firmware version \"nami\"",
		"c9b5c1e3-a40f-4d5a-8353-6f890d80d9be",
		exampleMetadataJSON,
		"nami",
		"akali360",
		"xxx-cros-version",
		"Google_Nami.52.53.54",
	},
}

func TestGetFirmwareVersion(t *testing.T) {
	t.Parallel()
	for _, tt := range testFirmwareVersionData {
		t.Run(tt.name, func(t *testing.T) {
			var r Reader
			bg := context.Background()
			r.dld = makeConstantDownloader(tt.metadata)
			version, e := r.getFirmwareVersion(bg, tt.bt, tt.model, tt.CrOSVersion)
			if e != nil {
				msg := fmt.Sprintf("name (%s): uuid (%s): unexpected error (%s)", tt.name, tt.uuid, e.Error())
				t.Errorf(msg)
			}
			diff := cmp.Diff(tt.out, version)
			if diff != "" {
				msg := fmt.Sprintf("name (%s): uuid (%s): unexpected diff (%s)", tt.name, tt.uuid, diff)
				t.Errorf(msg)
			}
		})
	}
}

var testValidateConfigData = []struct {
	name          string
	uuid          string
	metadata      string
	in            string
	out           string
	errorFragment string
}{
	{
		"empty",
		"b2b7aa51-3c2b-4c3f-93bf-0b26e6483489",
		`{}`,
		`{}`,
		`{
			"missing_boards": null,
			"failed_to_lookup": null,
			"invalid_versions": null
		}`,
		NOERROR,
	},
	{
		"two present boards",
		"f63476d1-0382-4098-8a15-18fcb1a2e61a",
		exampleMetadataJSON,
		`{
			"cros": [
				{
					"key": {
						"buildTarget": {"name": "nami"},
						"modelId": {"value": "akali360"}
					},
					"version": "R81-12835.0.0"
				},
				{
					"key": {
						"buildTarget": {"name": "nami"},
						"modelId": {"value": "sona"}
					},
					"version": "R81-12835.0.0"
				}
			],
			"firmware": [
				{
					"key": {
						"modelId": {"value": "sona"},
						"buildTarget": {"name": "nami"}
					},
					"version": "Google_Nami.42.43.44"
				},
				{
					"key": {
						"modelId": {"value": "akali360"},
						"buildTarget": {"name": "nami"}
					},
					"version": "Google_Nami.52.53.54"
				}
			]
		}`,
		`{
			"missing_boards": null,
			"failed_to_lookup": null,
			"invalid_versions": null
		}`,
		NOERROR,
	},
	{
		"two present boards with specific CrOS entries",
		"f63476d1-0382-4098-8a15-18fcb1a2e61a",
		exampleMetadataJSON,
		`{
			"cros": [
				{
					"key": {
						"buildTarget": {"name": "nami"},
						"modelId": {"value": "sona"}
					},
					"version": "R81-12835.0.0"
				},
				{
					"key": {
						"buildTarget": {"name": "nami"},
						"modelId": {"value": "akali360"}
					},
					"version": "R81-12835.0.0"
				}
			],
			"firmware": [
				{
					"key": {
						"modelId": {"value": "sona"},
						"buildTarget": {"name": "nami"}
					},
					"version": "Google_Nami.42.43.44"
				},
				{
					"key": {
						"modelId": {"value": "akali360"},
						"buildTarget": {"name": "nami"}
					},
					"version": "Google_Nami.52.53.54"
				}
			]
		}`,
		`{
			"missing_boards": null,
			"failed_to_lookup": null,
			"invalid_versions": null
		}`,
		NOERROR,
	},
	{
		"one nonexistent firmware version",
		"ee6a1776-78b3-4e8f-a4ce-a298bc5428d8",
		exampleMetadataJSON,
		`{
			"firmware": [
				{
					"key": {
						"modelId": {"value": "nonexistent-model"},
						"buildTarget": {"name": "nonexistent-board"}
					},
					"version": "Google_Nami.52.53.54"
				}
			]
		}`,
		`{
			"missing_boards": null,
			"failed_to_lookup": [
				{
					"build_target": "nonexistent-board",
					"model": "nonexistent-model"
				}
			],
			"invalid_versions": null
		}`,
		NOERROR,
	},
	{
		"one nonexistent chrome os version",
		"fab84b16-288d-44f0-b489-3712f8c14ad3",
		exampleMetadataJSON,
		`{
			"cros": [
				{
					"key": {
						"buildTarget": {"name": "nonexistent-build-target"},
						"modelId": {}
					},
					"version": "R81-12835.0.0"
				}
			]
		}`,
		`{
			"missing_boards": ["nonexistent-build-target"],
			"failed_to_lookup": null,
			"invalid_versions": null
		}`,
		NOERROR,
	},
	{
		"invalid Chrome OS version in config file SHOULD PASS",
		"e188b8d4-6c2a-4fc1-b525-70144e0d8148",
		exampleMetadataJSON,
		`{
			"cros": [
				{
					"key": {
						"buildTarget": {"name": "nami"},
						"modelId": {}
					},
					"version": "xxx-fake-version"
				}
			]
		}`,
		`null`,
		NOERROR,
	},
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()
	for _, tt := range testValidateConfigData {
		t.Run(tt.name, func(t *testing.T) {
			var r Reader
			bg := context.Background()
			r.dld = makeConstantDownloader(tt.metadata)
			sv := parseStableVersionsOrPanic(tt.in)
			expected := parseResultsOrPanic(tt.out)
			result, e := r.ValidateConfig(bg, sv)
			if err := validateErrorContainsSubstring(e, tt.errorFragment); err != nil {
				t.Errorf(err.Error())
			}
			diff := cmp.Diff(expected, result)
			if diff != "" {
				msg := fmt.Sprintf("name (%s): uuid (%s): unexpected diff (%s)", tt.name, tt.uuid, diff)
				t.Errorf(msg)
			}
		})
	}
}

var testRemoveAllowedData = []struct {
	name string
	uuid string
	in   string
	out  string
}{
	{
		"nothing to remove from empty result",
		"bb30b384-38a6-4eb4-aa2c-b09815106d0a",
		"{}",
		"{}",
	},
	{
		"remove explicitly allowed board buddy_cfm",
		"ef203c4d-6224-44df-ba80-9253dc47e4f7",
		`{
			"missing_boards": ["buddy_cfm"]
		}`,
		"{}",
	},
	{
		"don't remove non-allowed board",
		"41655030-dd58-481f-bcb4-4be4dfc01f07",
		`{
			"missing_boards": ["NOT AN ALLOWED BOARD"]
		}`,
		`{
			"missing_boards": ["NOT AN ALLOWED BOARD"]
		}`,
	},
	{
		"fizz-labstation is never present in the metadata.json file",
		"ff146760-fa2a-4495-aaea-42708843e6a1",
		`{
			"failed_to_lookup": [{"build_target": "fizz-labstation", "model": "fizz-labstation"}]
		}`,
		"{}",
	},
	{
		"leave non-allowed build target in place",
		"24d51d50-8d34-498f-8c1d-b15341dfc549",
		`{
			"failed_to_lookup": [{"build_target": "NOT ALLOWED", "model": "NOT ALLOWED"}]
		}`,
		`{
			"failed_to_lookup": [{"build_target": "NOT ALLOWED", "model": "NOT ALLOWED"}]
		}`,
	},
	{
		"remove explicit invalid version mismatch",
		"6683d1ca-089e-4401-b189-a6db046631d1",
		`{
			"invalid_versions": [{"build_target": "zork", "model": "dalboz", "wanted": "XXX", "got": "XXX"}]
		}`,
		`{
			"invalid_versions": null
		}`,
	},
	{
		"retain non-allowed invalid version mismatch",
		"fb105639-2eea-438a-809e-f01a9d98492b",
		`{
			"invalid_versions": [{"build_target": "A", "model": "B", "wanted": "C", "got": "D"}]
		}`,
		`{
			"invalid_versions": [{"build_target": "A", "model": "B", "wanted": "C", "got": "D"}]
		}`,
	},
}

func TestRemoveAllowed(t *testing.T) {
	t.Parallel()
	for _, tt := range testRemoveAllowedData {
		t.Run(tt.uuid, func(t *testing.T) {
			var in ValidationResult
			var out ValidationResult
			unmarshalOrPanic(tt.in, &in)
			unmarshalOrPanic(tt.out, &out)
			in.RemoveAllowedDUTs()
			if diff := cmp.Diff(out, in); diff != "" {
				msg := fmt.Sprintf("uuid (%s): unexpected diff (%s)", tt.uuid, diff)
				t.Errorf(msg)
			}
		})
	}
}

func TestNonLowercaseIsMalformed(t *testing.T) {
	cases := []struct {
		name         string
		fileContents string
		in           string
		out          string
	}{
		{
			"uppercase buildTarget in cros version",
			"",
			`{
				"cros": [
					{
						"key": {
							"buildTarget": {"name": "naMi"},
							"modelId": {}
						},
						"version": "xxx-fake-version"
					}
				]
			}`,
			`{
				"non_lowercase_entries": ["naMi"]
			}`,
		},
	}

	t.Parallel()

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bg := context.Background()
			var out ValidationResult
			unmarshalOrPanic(tt.out, &out)

			var r Reader
			r.dld = makeConstantDownloader(tt.fileContents)

			in := parseStableVersionsOrPanic(tt.in)
			res, err := r.ValidateConfig(bg, in)
			if err != nil {
				t.Errorf("unexpected error %s", err)
			}
			if diff := cmp.Diff(&out, res); diff != "" {
				t.Errorf("comparison failure: %s", diff)
			}
		})
	}
}

func TestIsLowercase(t *testing.T) {
	cases := []struct {
		in  string
		out bool
	}{
		{
			"",
			true,
		},
		{
			"a",
			true,
		},
		{
			"A",
			false,
		},
		{
			"aA",
			false,
		},
	}

	t.Parallel()

	for _, tt := range cases {
		tt := tt
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if isLowercase(tt.in) != tt.out {
				t.Errorf("isLowercase(%s) is unexpectedly %v", tt.in, tt.out)
			}
		})
	}
}

func testCombinedKey(t *testing.T) {
	t.Parallel()
	cases := []struct {
		board string
		model string
		out   string
	}{
		{
			"",
			"",
			"",
		},
		{
			"a",
			"",
			"a",
		},
		{
			"",
			"b",
			";b",
		},
		{
			"a",
			"b",
			"a;b",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.out, func(t *testing.T) {
			t.Parallel()
			want := tt.out
			got := combinedKey(tt.board, tt.model)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLookupBestVersion(t *testing.T) {
	cases := []struct {
		cfgCrosVersions map[string]string
		board           string
		model           string
		out             string
		errPat          string
	}{
		{
			nil,
			"",
			"",
			"",
			"^no matching CrOS versions.*$",
		},
		{
			map[string]string{
				"fake-board;fake-model": "107",
			},
			"fake-board",
			"fake-model",
			"107",
			"",
		},
		{
			map[string]string{
				"fake-board":            "107",
				"fake-board;fake-model": "208",
			},
			"fake-board",
			"fake-model",
			"208",
			"",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.out, func(t *testing.T) {
			t.Parallel()
			want := tt.out
			got, e := lookupBestVersion(tt.cfgCrosVersions, tt.board, tt.model)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
			if err := validateMatches(tt.errPat, errorToString(e)); err != nil {
				t.Errorf("error message does not match: %s", err)
				if e != nil {
					t.Errorf("error message: %s", e)
				}
			}
		})
	}
}

func TestMultipleModelsSameBoard(t *testing.T) {
	const namiVayne1metadataJSON = `
	{
	  "version": {
		"full": "R77-12371.52.22"
	  },
	  "results": [],
	  "unibuild": true,
	  "board-metadata": {
		"nami": {
		  "models": {
			"vayne": {
			  "firmware-key-id": "VAYNE",
			  "main-readwrite-firmware-version": "Google_Nami.6315.91.0"
			}
		  }
		}
	  }
	}
	`

	const namiVayne2metadataJSON = `
	{
	  "version": {
		"full": "R85-13310.41.0"
	  },
	  "results": [],
	  "unibuild": true,
	  "board-metadata": {
		"nami": {
		  "models": {
			"vayne2": {
			  "firmware-key-id": "VAYNE2",
			  "main-readwrite-firmware-version": "Google_Nami.7820.314.0"
			}
		  }
		}
	  }
	}
	`

	t.Parallel()
	var r Reader
	bg := context.Background()
	r.dld = func(gsPath gs.Path) ([]byte, error) {
		var s string = string(gsPath)
		if !strings.Contains(s, "nami") {
			panic(fmt.Sprintf("invalid path does not contain nami: %q", s))
		}
		if strings.Contains(s, "77-12371.52.22") {
			return []byte(namiVayne1metadataJSON), nil
		}
		if strings.Contains(s, "85-13310.41.0") {
			return []byte(namiVayne2metadataJSON), nil
		}
		panic(fmt.Sprintf("unhandled test case in downloader: %q", s))
	}

	testSV := &sv.StableVersions{
		Cros: []*sv.StableCrosVersion{
			utils.MakeSpecificCrOSSV(
				"nami",
				"vayne",
				"R77-12371.52.22",
			),
			utils.MakeSpecificCrOSSV(
				"nami",
				"vayne2",
				"R85-13310.41.0",
			),
		},
		Firmware: []*sv.StableFirmwareVersion{
			utils.MakeSpecificFirmwareVersion(
				"nami",
				"vayne",
				"Google_Nami.6315.91.0",
			),
			utils.MakeSpecificFirmwareVersion(
				"nami",
				"vayne2",
				"Google_Nami.7820.314.0",
			),
		},
	}

	res, e := r.ValidateConfig(bg, testSV)
	if e != nil {
		t.Errorf("unexpected error: %q", e.Error())
	}

	if res.AnomalyCount() != 0 {
		msg, err := json.MarshalIndent(res, "", "    ")
		if err != nil {
			panic("internal error: failed to jsonify result")
		}
		t.Errorf("%d errors: %s", res.AnomalyCount(), msg)
	}

	cases := []struct {
		board     string
		version   string
		model     string
		fwversion string
	}{
		{
			"nami",
			"R77-12371.52.22",
			"vayne",
			"Google_Nami.6315.91.0",
		},
		{
			"nami",
			"R77-12371.52.22",
			"vayne2",
			"",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(
			fmt.Sprintf("%q %q %q", tt.board, tt.version, tt.model),
			func(t *testing.T) {
				t.Parallel()
				actual := get(r.cache, tt.board, tt.version, tt.model)
				if diff := cmp.Diff(tt.fwversion, actual); diff != "" {
					t.Errorf("diff (-want +got):\n%s", diff)
				}
			})
	}
}

func makeConstantDownloader(content string) downloader {
	return func(gsPath gs.Path) ([]byte, error) {
		return []byte(content), nil
	}
}

// parseStableVersionsOrPanic is a helper function that's used in tests to feed
// a stable version file contained in a string literal to a test.
func parseStableVersionsOrPanic(content string) *labPlatform.StableVersions {
	out, err := validateconfig.ParseStableVersions([]byte(content))
	if err != nil {
		panic(err.Error())
	}
	return out
}

// parseStableVersionsOrPanic is a helper function that's used in tests to feed
// a result file contained in a string literal to a test.
func parseResultsOrPanic(content string) *ValidationResult {
	var out ValidationResult
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		panic(err.Error())
	}
	return &out
}

// validateErrorContainsSubstring checks whether an error matches a string provided in a table-driven test
func validateErrorContainsSubstring(e error, msg string) error {
	if msg == "" {
		panic("unexpected empty string in validateError function")
	}
	if e == nil {
		switch msg {
		case NOERROR:
			return nil
		case ANYERROR:
			return fmt.Errorf("expected error to be non-nil, but it wasn't")
		default:
			return fmt.Errorf("expected error to contain (%s), but it was nil", msg)
		}
	}
	switch msg {
	case NOERROR:
		return fmt.Errorf("expected error to be nil, but it was (%s)", e.Error())
	case ANYERROR:
		return nil
	default:
		if strings.Contains(e.Error(), msg) {
			return nil
		}
		return fmt.Errorf("expected error (%s) to contain (%s), but it did not", e.Error(), msg)
	}
}

func unmarshalOrPanic(content string, dest interface{}) {
	if err := json.Unmarshal([]byte(content), dest); err != nil {
		panic(err.Error())
	}
}

func validateMatches(pattern string, s string) error {
	b, err := regexp.MatchString(pattern, s)
	if err != nil {
		return err
	}
	if b {
		return nil
	}
	return fmt.Errorf("no part of string %q matches pattern %q", s, pattern)
}

func errorToString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
