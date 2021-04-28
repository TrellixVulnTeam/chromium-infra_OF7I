// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"fmt"
	"infra/cmd/crosfleet/internal/common"
	"testing"

	"github.com/google/go-cmp/cmp"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
)

func TestRemoveBackfills(t *testing.T) {
	t.Parallel()
	backfillBuild := &buildbucketpb.Build{
		Tags: []*buildbucketpb.StringPair{
			{Key: "crosfleet-tool", Value: "backfill"}}}
	nonBackfillBuild := &buildbucketpb.Build{
		Tags: []*buildbucketpb.StringPair{
			{Key: "crosfleet-tool", Value: "suite"}}}

	wantFilteredBuilds := []*buildbucketpb.Build{nonBackfillBuild}
	gotFilteredBuilds := removeBackfills([]*buildbucketpb.Build{
		backfillBuild, nonBackfillBuild})
	if diff := cmp.Diff(wantFilteredBuilds, gotFilteredBuilds, common.CmpOpts); diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}

var testBackfillTagsData = []struct {
	build    *buildbucketpb.Build
	wantTags map[string]string
}{
	{
		&buildbucketpb.Build{
			Id: 1,
			Tags: []*buildbucketpb.StringPair{
				{Key: "foo", Value: "bar"},
				{Key: "baz", Value: "lol"}}},
		map[string]string{
			"foo":            "bar",
			"baz":            "lol",
			"crosfleet-tool": "backfill",
			"backfill":       "1",
		},
	},
	{
		&buildbucketpb.Build{
			Id: 2,
			Tags: []*buildbucketpb.StringPair{
				{Key: "bar", Value: "foo"},
				{Key: "lol", Value: "baz"},
				{Key: "backfill", Value: "3"},
				{Key: "crosfleet-tool", Value: "suite"}}},
		map[string]string{
			"bar":            "foo",
			"lol":            "baz",
			"crosfleet-tool": "backfill",
			"backfill":       "2",
		},
	},
}

func TestBackfillTags(t *testing.T) {
	t.Parallel()
	for _, tt := range testBackfillTagsData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantTags), func(t *testing.T) {
			t.Parallel()
			gotTags := backfillTags(tt.build)
			if diff := cmp.Diff(tt.wantTags, gotTags); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}
