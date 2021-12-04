// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/uuid"
)

func TestAppConfigLatestTagOnly(t *testing.T) {
	tests := []struct {
		name     string
		tagsList [][]string // In the order of newest -> oldest.
		want     [][]string
	}{
		{
			name:     "add: no images eligible",
			tagsList: [][]string{{"tag1"}, {"tag2", "tag2-0"}, {"tag3"}},
			want:     [][]string{{"tag1"}, {"tag2", "tag2-0"}, {"tag3"}},
		},
		{
			name:     "add: target is the newest",
			tagsList: [][]string{{"official-1", "tag1"}, {"tag2"}, {"tag3", "official-4"}},
			want:     [][]string{{"official-1", "tag1", latestOfficial}, {"tag2"}, {"tag3", "official-4"}},
		},
		{
			name:     "add: target is not the newest",
			tagsList: [][]string{{"tag1"}, {"tag2", "official-9"}, {"tag3", "official-1"}, {"tag4"}},
			want:     [][]string{{"tag1"}, {"tag2", "official-9", latestOfficial}, {"tag3", "official-1"}, {"tag4"}},
		},
		{
			name:     "move: forward to newest official",
			tagsList: [][]string{{"tag1"}, {"official-3"}, {"tag2"}, {"official-4", latestOfficial}, {"tag4"}},
			want:     [][]string{{"tag1"}, {"official-3", latestOfficial}, {"tag2"}, {"official-4"}, {"tag4"}},
		},
		{
			name:     "move: backward when original image was downgraded",
			tagsList: [][]string{{"official-1-bad", latestOfficial}, {"official-2-bad"}, {"official-9"}},
			want:     [][]string{{"official-1-bad"}, {"official-2-bad"}, {"official-9", latestOfficial}},
		},
		{
			name:     "remove: no images eligible",
			tagsList: [][]string{{"tag1", latestOfficial}, {"tag2"}, {"tag3"}},
			want:     [][]string{{"tag1"}, {"tag2"}, {"tag3"}},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &fakeRepo{tagsList: tc.tagsList}

			c := newAppConfig(`^official-\d{1,2}$`, latestOfficialPolicy)
			err := c.apply(r)
			if err != nil {
				t.Fatalf("apply() failed: %s", err)
			}
			if diff := cmp.Diff(tc.want, r.tagsList); diff != "" {
				t.Errorf("AppConfig(%q)(latest tag only) mismatch: (-want, +got):\n%s", r.Name(), diff)
			}
		})
	}
}

func TestAppConfigCanaryAndProd(t *testing.T) {
	tests := []struct {
		name     string
		tagsList [][]string // In the order of newest -> oldest.
		want     [][]string
	}{
		{
			name:     "add: no eligible images",
			tagsList: [][]string{{"tag1"}, {"tag2"}},
			want:     [][]string{{"tag1"}, {"tag2"}},
		},
		{
			name:     "add: both on the newest official",
			tagsList: [][]string{{"tag1"}, {"official-2"}, {"official-1"}},
			want:     [][]string{{"tag1"}, {"official-2", latestOfficial, canary, prod}, {"official-1"}},
		},
		{
			name: "move: no need to move (1)",
			tagsList: [][]string{
				{"tag1"}, {"tag2"}, {"official-2", prod, canary}, {"tag3"}, {"official-1"},
			},
			want: [][]string{
				{"tag1"}, {"tag2"}, {"official-2", prod, canary, latestOfficial}, {"tag3"}, {"official-1"},
			},
		},
		{
			name: "move: no need to move (2)",
			tagsList: [][]string{
				{"official-3", canary}, {"tag1"}, {"tag2"}, {"official-2", prod}, {"tag3"}, {"official-1"},
			},
			want: [][]string{
				{"official-3", canary, latestOfficial}, {"tag1"}, {"tag2"}, {"official-2", prod}, {"tag3"}, {"official-1"},
			},
		},
		{
			name: "move: roll out a new canary; prod stays",
			tagsList: [][]string{
				{"tag1"}, {"official-3"}, {"tag2"}, {"tag3"}, {"official-2", prod, canary}, {"official-1"},
			},
			want: [][]string{
				{"tag1"}, {"official-3", latestOfficial, canary}, {"tag2"}, {"tag3"}, {"official-2", prod}, {"official-1"},
			},
		},
		{
			name: "move: roll back canary",
			tagsList: [][]string{
				{"official-3-bad", canary}, {"tag1"}, {"tag2"}, {"official-2", prod}, {"tag3"}, {"official-1"},
			},
			want: [][]string{
				{"official-3-bad"}, {"tag1"}, {"tag2"}, {"official-2", prod, latestOfficial, canary}, {"tag3"}, {"official-1"},
			},
		},
		{
			name: "move: roll back prod",
			tagsList: [][]string{
				{"official-3", canary}, {"tag1"}, {"tag2"}, {"official-2-bad", prod}, {"tag3"}, {"official-1"},
			},
			want: [][]string{
				{"official-3", canary, latestOfficial}, {"tag1"}, {"tag2"}, {"official-2-bad"}, {"tag3"}, {"official-1", prod},
			},
		},
		{
			name: "move: roll back both",
			tagsList: [][]string{
				{"official-3-bad", canary}, {"tag1"}, {"tag2"}, {"official-2-bad", prod}, {"tag3"}, {"official-1"},
			},
			want: [][]string{
				{"official-3-bad"}, {"tag1"}, {"tag2"}, {"official-2-bad"}, {"tag3"}, {"official-1", latestOfficial, canary, prod},
			},
		},
		{
			name: "remove tags",
			tagsList: [][]string{
				{"official-3-bad", latestOfficial, canary}, {"tag1"}, {"tag2"}, {"official-2-bad", prod}, {"tag3"},
			},
			want: [][]string{
				{"official-3-bad"}, {"tag1"}, {"tag2"}, {"official-2-bad"}, {"tag3"},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			r := &fakeRepo{tagsList: tc.tagsList}
			c := newAppConfig(`^official-\d{1,2}$`, latestOfficialPolicy, canaryMaxDistancePolicy, prodMaxDistancePolicy)
			err := c.apply(r)
			if err != nil {
				t.Fatalf("apply() failed: %s", err)
			}
			if diff := cmp.Diff(tc.want, r.tagsList); diff != "" {
				t.Errorf("AppConfig(%q)(canary and prod) mismatch: (-want, +got):\n%s", r.Name(), diff)
			}
		})
	}
}

type fakeRepo struct {
	tagsList [][]string
}

func (f *fakeRepo) List(ctx context.Context) (*google.Tags, error) {
	return &google.Tags{Manifests: f.genManifests()}, nil
}

func (f *fakeRepo) genManifests() map[string]google.ManifestInfo {
	baseTime := time.Date(2021, time.May, 1, 0, 0, 0, 0, time.UTC)
	mm := map[string]google.ManifestInfo{}
	for i, ts := range f.tagsList {
		mm[uuid.New().String()] = google.ManifestInfo{
			Created: baseTime.Add(time.Duration(-i) * time.Second),
			Tags:    ts,
		}
	}
	return mm
}

func (f *fakeRepo) Tag(ctx context.Context, newTag, existingTag string) error {
	r := [][]string{}
	for _, ts := range f.tagsList {
		rr := []string{}
		// Remove newTag first (if there is), then add it to the slice including 'tag'.
		addHere := false
		for _, t := range ts {
			if t != newTag {
				rr = append(rr, t)
			}
			if t == existingTag {
				addHere = true
			}
		}
		if addHere {
			rr = append(rr, newTag)
		}
		r = append(r, rr)
	}
	f.tagsList = r
	return nil
}

func (f *fakeRepo) Untag(ctx context.Context, tag string) error {
	r := [][]string{}
	for _, ts := range f.tagsList {
		rr := []string{}
		for _, t := range ts {
			if t != tag {
				rr = append(rr, t)
			}
		}
		r = append(r, rr)
	}
	f.tagsList = r
	return nil
}

func (f *fakeRepo) Name() string { return "fake/repo" }
