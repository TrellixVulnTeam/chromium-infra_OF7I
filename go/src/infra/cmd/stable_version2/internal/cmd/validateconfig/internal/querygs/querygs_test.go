// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package querygs

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	labPlatform "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/luci/common/gcloud/gs"

	"infra/libs/cros/stableversion/validateconfig"
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
	out      map[string]map[string]string
}{
	{
		"f959c762-214e-4293-b655-032cd791a85f",
		exampleMetadataJSON,
		map[string]map[string]string{
			"nami": {
				"sona":     "Google_Nami.42.43.44",
				"akali360": "Google_Nami.52.53.54",
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
			r.dld = makeConstantDownloader(tt.metadata)
			version, e := r.getFirmwareVersion(tt.bt, tt.model, tt.CrOSVersion)
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
						"modelId": {}
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
						"modelId": {"value": "NONEXISTENT-MODEL"},
						"buildTarget": {"name": "NONEXISTENT-BOARD"}
					},
					"version": "Google_Nami.52.53.54"
				}
			]
		}`,
		`{
			"missing_boards": null,
			"failed_to_lookup": [
				{
					"build_target": "NONEXISTENT-BOARD",
					"model": "NONEXISTENT-MODEL"
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
						"buildTarget": {"name": "NONEXISTENT-BUILD-TARGET"},
						"modelId": {}
					},
					"version": "R81-12835.0.0"
				}
			]
		}`,
		`{
			"missing_boards": ["NONEXISTENT-BUILD-TARGET"],
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
			r.dld = makeConstantDownloader(tt.metadata)
			sv := parseStableVersionsOrPanic(tt.in)
			expected := parseResultsOrPanic(tt.out)
			result, e := r.ValidateConfig(sv)
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
