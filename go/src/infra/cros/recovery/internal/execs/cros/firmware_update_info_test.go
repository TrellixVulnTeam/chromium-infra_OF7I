package cros

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.chromium.org/luci/common/errors"
)

var availableRWFirmwareTests = []struct {
	testName           string
	mf                 *modelFirmware
	expectedRWFirmware string
	expectedErr        error
}{
	{
		"rw firmware found, no error",
		&modelFirmware{
			Host: &host{
				Versions: &versions{
					Rw: "r_v9.9.999-ffffffff",
				},
				Keys: &keys{
					Root:     "root_xxxx",
					Recovery: "recovery_xxxx",
				},
			},
		},
		"r_v9.9.999-ffffffff",
		nil,
	},
	{
		"no rw firmware found, host info error",
		&modelFirmware{},
		"",
		errors.Reason("available rw firmware: host info is not present").Err(),
	},
	{
		"no rw firmware found, versions error",
		&modelFirmware{
			Host: &host{},
		},
		"",
		errors.Reason("available rw firmware: no versions found").Err(),
	},
	{
		"no rw firmware found, fw versions error",
		&modelFirmware{
			Host: &host{
				Versions: &versions{
					Ro: "r_v9.9.999-ffffffff",
				},
			},
		},
		"",
		errors.Reason("available rw firmware: rw version is not provided").Err(),
	},
}

func TestAvailableRWFirmware(t *testing.T) {
	t.Parallel()
	for _, tt := range availableRWFirmwareTests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			actualRWFirmware, actualErr := tt.mf.AvailableRWFirmware()
			if actualErr != nil && tt.expectedErr != nil {
				if actualErr.Error() != tt.expectedErr.Error() {
					t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
				}
			}
			if (actualErr == nil && tt.expectedErr != nil) || (actualErr != nil && tt.expectedErr == nil) {
				t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
			}
			if actualRWFirmware != tt.expectedRWFirmware {
				t.Errorf("Expected rw firmware version: %q, but got: %q", tt.expectedRWFirmware, actualRWFirmware)
			}
		})
	}
}

var ReadFirmwareManifestTest = []struct {
	testName              string
	rawOutput             string
	dutModel              string
	expectedModelFirmware *modelFirmware
	expectedErr           error
}{
	{
		"Good case, no error",
		`
			{
				"model1": {
					"host": {
						"versions": {
							"ro": "model1_host_versions_ro",
							"rw": "model1_host_versions_rw"
						},
						"keys": {
							"root": "model1_host_keys_root",
							"recovery": "model1_host_keys_recovery"
						},
						"image": "model1_host_image",
						"signature_id": "model1"
					},
					"ec": {
						"versions": {
							"ro": "model1_ec_versions_ro",
							"rw": "model1_ec_versions_rw"
						},
						"image": "model1_ec_image"
					},
					"signature_id": "model1"
				},
				"model2": {
					"host": {
						"versions": {
							"ro": "x",
							"rw": "x"
						},
						"keys": {
							"root": "x",
							"recovery": "x"
						}
					},
					"ec": {
						"versions": {
							"ro": "y",
							"rw": "y"
						},
						"image": "y"
					}
				}
			}
		`,
		"model1",
		&modelFirmware{
			Host: &host{
				Versions: &versions{
					Ro: "model1_host_versions_ro",
					Rw: "model1_host_versions_rw",
				},
				Keys: &keys{
					Root:     "model1_host_keys_root",
					Recovery: "model1_host_keys_recovery",
				},
				Image: "model1_host_image",
			},
			Ec: &host{
				Versions: &versions{
					Ro: "model1_ec_versions_ro",
					Rw: "model1_ec_versions_rw",
				},
				Image: "model1_ec_image",
			},
			Signature_ID: "model1",
		},
		nil,
	},
	{
		"Bad case, no model info found",
		`
			{
				"model1": {}
			}
		`,
		"model3",
		nil,
		errors.Reason(`read firmware manifest: model "model3" is not present in manifest`).Err(),
	},
	{
		"Bad case, rawOutput format is wrong",
		`
			xxxxxx
		`,
		"model3",
		nil,
		errors.Reason(`read firmware manifest`).Err(),
	},
}

func TestParseFirmwareManifest(t *testing.T) {
	t.Parallel()
	for _, tt := range ReadFirmwareManifestTest {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			r := func(ctx context.Context, time time.Duration, cmd string, args ...string) (string, error) {
				return tt.rawOutput, nil
			}
			actualModelFirmware, actualErr := ReadFirmwareManifest(ctx, r, tt.dutModel)
			if actualErr != nil && tt.expectedErr != nil {
				if !strings.Contains(actualErr.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
				}
			}
			if (actualErr == nil && tt.expectedErr != nil) || (actualErr != nil && tt.expectedErr == nil) {
				t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
			}
			if !reflect.DeepEqual(actualModelFirmware, tt.expectedModelFirmware) {
				t.Errorf("Expected modelFirmware struct: %+v, but got: %+v", tt.expectedModelFirmware, actualModelFirmware)
			}
		})
	}
}
