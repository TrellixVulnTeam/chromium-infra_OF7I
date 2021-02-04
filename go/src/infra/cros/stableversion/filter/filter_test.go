// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package filter

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"infra/cros/stableversion/validateconfig"

	labPlatform "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
)

var testWithModelData = []struct {
	name  string
	in    *labPlatform.StableVersions
	model string
	out   *labPlatform.StableVersions
}{
	{
		name:  "empty container",
		in:    &labPlatform.StableVersions{},
		model: "",
		out:   &labPlatform.StableVersions{},
	},
	{
		name:  "empty container with nontrivial model",
		in:    &labPlatform.StableVersions{},
		model: "vayne",
		out:   &labPlatform.StableVersions{},
	},
	{
		name: "leave unaffected model",
		in: mustParseStableVersions(`
{
        "cros": [
                {
                        "key": {
                                "modelId": {

                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "R81-12871.41.0"
                }
        ],
        "firmware": [
                {
                        "key": {
                                "modelId": {
                                        "value": "vayne"
                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "Google_Nami.10775.123.0"
                }
        ]
}`),
		model: "xxxxxx",
		out:   &labPlatform.StableVersions{},
	},
	{
		name: "retain explicitly named model",
		in: mustParseStableVersions(`
{
        "cros": [
                {
                        "key": {
                                "modelId": {

                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "R81-12871.41.0"
                }
        ],
        "firmware": [
                {
                        "key": {
                                "modelId": {
                                        "value": "vayne"
                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "Google_Nami.10775.123.0"
                }
        ]
}`),
		model: "vayne",
		out: mustParseStableVersions(`{
        "cros": [
                {
                        "key": {
                                "modelId": {

                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "R81-12871.41.0"
                }
        ],
        "firmware": [
                {
                        "key": {
                                "modelId": {
                                        "value": "vayne"
                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "Google_Nami.10775.123.0"
                }
        ]
}`),
	},
	{
		name: "keep everything with empty model",
		in: mustParseStableVersions(`
{
        "cros": [
                {
                        "key": {
                                "modelId": {

                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "R81-12871.41.0"
                }
        ],
        "firmware": [
                {
                        "key": {
                                "modelId": {
                                        "value": "vayne"
                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "Google_Nami.10775.123.0"
                }
        ]
}`),
		model: "",
		out: mustParseStableVersions(`{
        "cros": [
                {
                        "key": {
                                "modelId": {

                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "R81-12871.41.0"
                }
        ],
        "firmware": [
                {
                        "key": {
                                "modelId": {
                                        "value": "vayne"
                                },
                                "buildTarget": {
                                        "name": "nami"
                                }
                        },
                        "version": "Google_Nami.10775.123.0"
                }
        ]
}`),
	},
}

func TestWithModel(t *testing.T) {
	t.Parallel()
	for _, tt := range testWithModelData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := WithModel(tt.in, tt.model)
			if diff := cmp.Diff(tt.out, out, protocmp.Transform()); diff != "" {
				msg := fmt.Sprintf("name: %s, diff: %s", tt.name, diff)
				t.Errorf("%s", msg)
			}
		})
	}
}

// mustParseStableVersions is a helper function that's used in tests to feed
// a stable version file contained in a string literal to a test.
func mustParseStableVersions(contents string) *labPlatform.StableVersions {
	out, err := validateconfig.ParseStableVersions([]byte(contents))
	if err != nil {
		panic(err.Error())
	}
	return out
}
