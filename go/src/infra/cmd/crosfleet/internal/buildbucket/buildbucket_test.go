// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildbucket

import (
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
				CrosfleetTool: 3,
				SkylabTool:    3,
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
