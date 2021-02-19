// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildbucket

import (
	"github.com/google/go-cmp/cmp/cmpopts"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAddServiceVersion(t *testing.T) {
	t.Parallel()
	startingProps := map[string]interface{}{"foo": "bar"}
	wantProps := map[string]interface{}{
		"foo":                       "bar",
		"$chromeos/service_version": `{"version":"{\"crosfleetTool\":\"1\"}"}`,
	}
	gotProps := addServiceVersion(startingProps)
	if diff := cmp.Diff(wantProps, gotProps); diff != "" {
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
	diff := cmp.Diff(wantBBDims, gotBBDims, cmpopts.IgnoreUnexported(buildbucketpb.RequestedDimension{}))
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
	diff := cmp.Diff(wantBBTags, gotBBTags, cmpopts.IgnoreUnexported(buildbucketpb.StringPair{}))
	if diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}
