// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildbucket

import (
	"fmt"
	"infra/cmd/crosfleet/internal/common"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
)

func TestAddServiceVersion(t *testing.T) {
	t.Parallel()
	startingProps := map[string]interface{}{"foo": "bar"}
	wantProps := map[string]interface{}{
		"foo": "bar",
		"$chromeos/service_version": map[string]interface{}{
			// Convert to protoreflect.ProtoMessage for easier type comparison.
			"version": (&test_platform.ServiceVersion{
				CrosfleetTool: 1,
				SkylabTool:    1,
			}).ProtoReflect().Interface(),
		},
	}
	gotProps := addServiceVersion(startingProps)
	if diff := cmp.Diff(wantProps, gotProps, common.CmpOpts); diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}

func TestBuildURL(t *testing.T) {
	client := Client{
		builderID: &buildbucketpb.BuilderID{
			Project: "chromeos",
			Bucket:  "test_runner",
			Builder: "dut_leaser",
		},
	}
	wantURL := "https://ci.chromium.org/ui/p/chromeos/builders/test_runner/dut_leaser/b8855075479708816960"
	gotURL := client.BuildURL(8855075479708816960)
	if wantURL != gotURL {
		t.Errorf("unexpected build URL: wanted %s, got %s", wantURL, gotURL)
	}
}

func TestBBDims(t *testing.T) {
	dims := map[string]string{
		"foo ": " bar",
	}
	wantBBDims := []*buildbucketpb.RequestedDimension{
		{Key: "foo", Value: "bar"},
	}
	gotBBDims := bbDims(dims)
	diff := cmp.Diff(wantBBDims, gotBBDims, common.CmpOpts)
	if diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}

func TestBBTags(t *testing.T) {
	tags := map[string]string{
		" foo": "bar ",
	}
	wantBBTags := []*buildbucketpb.StringPair{
		{Key: "foo", Value: "bar"},
	}
	gotBBTags := bbTags(tags)
	diff := cmp.Diff(wantBBTags, gotBBTags, common.CmpOpts)
	if diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}

var testUnfinishedBuildWithBotIDData = []struct {
	build      *buildbucketpb.Build
	wantAnswer bool
}{
	{ // Scheduled with the matching bot ID requested
		&buildbucketpb.Build{
			Status: buildbucketpb.Status_SCHEDULED,
			Infra: &buildbucketpb.BuildInfra{
				Buildbucket: &buildbucketpb.BuildInfra_Buildbucket{
					RequestedDimensions: []*buildbucketpb.RequestedDimension{
						{Key: "foo", Value: "bar"},
						{Key: "id", Value: "matching-id"},
					},
				},
			},
		},
		true,
	},
	{ // Scheduled with a different bot ID requested
		&buildbucketpb.Build{
			Status: buildbucketpb.Status_SCHEDULED,
			Infra: &buildbucketpb.BuildInfra{
				Buildbucket: &buildbucketpb.BuildInfra_Buildbucket{
					RequestedDimensions: []*buildbucketpb.RequestedDimension{
						{Key: "foo", Value: "bar"},
						{Key: "id", Value: "wrong-id"},
					},
				},
			},
		},
		false,
	},
	{ // Started with the matching bot ID provisioned
		&buildbucketpb.Build{
			Status: buildbucketpb.Status_STARTED,
			Infra: &buildbucketpb.BuildInfra{
				Buildbucket: &buildbucketpb.BuildInfra_Buildbucket{
					RequestedDimensions: []*buildbucketpb.RequestedDimension{
						{Key: "foo", Value: "bar"},
						{Key: "id", Value: "wrong-id"},
					},
				},
				Swarming: &buildbucketpb.BuildInfra_Swarming{
					BotDimensions: []*buildbucketpb.StringPair{
						{Key: "foo", Value: "bar"},
						{Key: "id", Value: "matching-id"},
					},
				},
			},
		},
		true,
	},
	{ // Started with a different bot ID provisioned
		&buildbucketpb.Build{
			Status: buildbucketpb.Status_STARTED,
			Infra: &buildbucketpb.BuildInfra{
				Buildbucket: &buildbucketpb.BuildInfra_Buildbucket{
					RequestedDimensions: []*buildbucketpb.RequestedDimension{
						{Key: "foo", Value: "bar"},
						{Key: "id", Value: "matching-id"},
					},
				},
				Swarming: &buildbucketpb.BuildInfra_Swarming{
					BotDimensions: []*buildbucketpb.StringPair{
						{Key: "foo", Value: "bar"},
						{Key: "id", Value: "different-id"},
					},
				},
			},
		},
		false,
	},
	{ // Neither scheduled nor started
		&buildbucketpb.Build{
			Status: buildbucketpb.Status_FAILURE,
			Infra: &buildbucketpb.BuildInfra{
				Buildbucket: &buildbucketpb.BuildInfra_Buildbucket{
					RequestedDimensions: []*buildbucketpb.RequestedDimension{
						{Key: "foo", Value: "bar"},
						{Key: "id", Value: "matching-id"},
					},
				},
				Swarming: &buildbucketpb.BuildInfra_Swarming{
					BotDimensions: []*buildbucketpb.StringPair{
						{Key: "foo", Value: "bar"},
						{Key: "id", Value: "matching-id"},
					},
				},
			},
		},
		false,
	},
}

func TestUnfinishedBuildWithBotID(t *testing.T) {
	t.Parallel()
	for _, tt := range testUnfinishedBuildWithBotIDData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.build), func(t *testing.T) {
			gotAnswer := isUnfinishedBuildWithBotID(tt.build, "matching-id")
			if tt.wantAnswer != gotAnswer {
				t.Errorf("unexpected error: wanted %v, got %v", tt.wantAnswer, gotAnswer)
			}
		})
	}
}
