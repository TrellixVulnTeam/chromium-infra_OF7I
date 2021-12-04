// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package image

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/uuid"
)

// genManifests generates a map of google.ManifestInfo with input data.
func genManifests(tagsList [][]string) map[string]google.ManifestInfo {
	baseTime := time.Date(2021, time.May, 1, 0, 0, 0, 0, time.UTC)
	mm := map[string]google.ManifestInfo{}
	for i, ts := range tagsList {
		mm[uuid.New().String()] = google.ManifestInfo{
			Created: baseTime.Add(time.Duration(-i) * time.Second),
			Tags:    ts,
		}
	}
	return mm
}

func TestNewestTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		list    [][]string
		wantTag string
		wantOk  bool
	}{
		{
			name:    "newest image has only one tag",
			list:    [][]string{{"t1"}, {"t2"}},
			wantTag: "t1",
			wantOk:  true,
		},
		{
			name:    "newest has no tag, will move to next",
			list:    [][]string{{}, {}, {"t1"}, {"t2"}},
			wantTag: "t1",
			wantOk:  true,
		},
		{
			name: "empty list",
			list: [][]string{},
		},
		{
			name: "non-empty list but has no tags",
			list: [][]string{{}, {}},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := NewList("name", genManifests(tc.list))
			tag, ok := l.NewestTag()
			if want, got := tc.wantOk, ok; want != got {
				t.Errorf("NewestTag() = %t, want %t", got, want)
			}
			if want, got := tc.wantTag, tag; want != got {
				t.Errorf("NewestTag() = %q, want %q", got, want)
			}
		})
	}
}

func TestNewerThan(t *testing.T) {
	t.Parallel()
	l := NewList("name", genManifests([][]string{{"t1"}, {"t2.0", "t2.1"}, {"t3"}}))
	t.Run("normal", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name, tag1, tag2 string
			want             bool
		}{
			{
				name: "newer",
				tag1: "t1", tag2: "t3",
				want: true,
			},
			{
				name: "older",
				tag1: "t3", tag2: "t1",
				want: false,
			},
			{
				name: "same age",
				tag1: "t2.0", tag2: "t2.1",
				want: false,
			},
		}
		for _, tc := range tests {
			got, err := l.NewerThan(tc.tag1, tc.tag2)
			if err != nil {
				t.Errorf("NewerThan(%q, %q) failed: %s", tc.tag1, tc.tag2, err)
			}
			if got != tc.want {
				t.Errorf("NewerThan(%q, %q) got %t want %t", tc.tag1, tc.tag2, got, tc.want)
			}
		}
	})
	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		if _, err := l.NewerThan("t1", "non-existing"); err == nil {
			t.Errorf("NewerThan('t1', 'non-existing') succeeded, want error")
		}
		if _, err := l.NewerThan("non-existing", "t1"); err == nil {
			t.Errorf("NewerThan('non-existing', 't1') succeeded, want error")
		}
	})
}

func TestTraverseToOlder(t *testing.T) {
	t.Parallel()

	l := NewList("name", genManifests([][]string{{"t1"}, {"t2"}, {"t3"}, {"t4"}, {"t5"}}))

	tests := []struct {
		name, startingTag string
		stopAt            string
		wantTags          []string
		wantStopped       bool
	}{
		{
			name:        "stop at target",
			startingTag: "t2",
			stopAt:      "t4",
			wantTags:    []string{"t2", "t3", "t4"},
			wantStopped: true,
		},
		{
			name:        "cannot find target (from middle)",
			startingTag: "t2",
			stopAt:      "non-existing",
			wantTags:    []string{"t2", "t3", "t4", "t5"},
			wantStopped: false,
		},
		{
			name:        "cannot find target (from end)",
			startingTag: "t5",
			stopAt:      "non-existing",
			wantTags:    []string{"t5"},
			wantStopped: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotTags []string
			f := func(m *google.ManifestInfo) (bool, error) {
				gotTags = append(gotTags, m.Tags[0])
				return m.Tags[0] == tc.stopAt, nil
			}
			gotStopped, err := l.TraverseToOlder(tc.startingTag, f)
			if err != nil {
				t.Errorf("TraverseToOlder() failed: %s", err)
			}
			if gotStopped != tc.wantStopped {
				t.Errorf("TraverseToOlder() got stopped %t, want %t", gotStopped, tc.wantStopped)
			}
			if diff := cmp.Diff(tc.wantTags, gotTags); diff != "" {
				t.Errorf("TraverseToOlder() mismatch: (-want, +got)\n%s", diff)
			}
		})
	}
}

func TestTraverseToNewer(t *testing.T) {
	t.Parallel()

	l := NewList("name", genManifests([][]string{{"t1"}, {"t2"}, {"t3"}, {"t4"}, {"t5"}}))

	tests := []struct {
		name, startingTag string
		stopAt            string
		wantTags          []string
		wantStopped       bool
	}{
		{
			name:        "stop at target",
			startingTag: "t4",
			stopAt:      "t2",
			wantTags:    []string{"t4", "t3", "t2"},
			wantStopped: true,
		},
		{
			name:        "cannot find target (from middle)",
			startingTag: "t3",
			stopAt:      "non-existing",
			wantTags:    []string{"t3", "t2", "t1"},
			wantStopped: false,
		},
		{
			name:        "cannot find target (from end)",
			startingTag: "t1",
			stopAt:      "non-existing",
			wantTags:    []string{"t1"},
			wantStopped: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotTags []string
			f := func(m *google.ManifestInfo) (bool, error) {
				gotTags = append(gotTags, m.Tags[0])
				return m.Tags[0] == tc.stopAt, nil
			}
			gotStopped, err := l.TraverseToNewer(tc.startingTag, f)
			if err != nil {
				t.Errorf("TraverseToNewer() failed: %s", err)
			}
			if gotStopped != tc.wantStopped {
				t.Errorf("TraverseToNewer() got stopped %t, want %t", gotStopped, tc.wantStopped)
			}
			if diff := cmp.Diff(tc.wantTags, gotTags); diff != "" {
				t.Errorf("TraverseToNewer() mismatch: (-want, +got)\n%s", diff)
			}
		})
	}

}

func TestTraverseErrors(t *testing.T) {
	t.Parallel()

	l := NewList("name", genManifests([][]string{{"t1"}, {"t2"}, {"t3"}}))
	t.Run("non-existing starting tag", func(t *testing.T) {
		if _, err := l.TraverseToNewer("t4", nil); err == nil {
			t.Errorf("Traverse() succeeded, want error")
		}
	})

	t.Run("callback returns error", func(t *testing.T) {
		f := func(*google.ManifestInfo) (bool, error) {
			return false, fmt.Errorf("error")
		}
		if _, err := l.TraverseToOlder("t1", f); err == nil {
			t.Errorf("Traverse() succeeded, want error")
		}
	})
}

func TestPutTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		tagsList       [][]string
		tagToMove      string
		tagWasExisting bool
		target         string
	}{
		{
			name:           "move an existing tag",
			tagsList:       [][]string{{"t1", "move-me"}, {"t2"}, {"t3"}},
			tagToMove:      "move-me",
			tagWasExisting: true,
			target:         "t3",
		},
		{
			name:           "move a non-existing tag",
			tagsList:       [][]string{{"t1"}, {"t2"}, {"t3"}},
			tagToMove:      "move-me",
			tagWasExisting: false,
			target:         "t3",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			l := NewList("name", genManifests(tc.tagsList))
			oldDigest, ok := l.TagToDigest[tc.tagToMove]
			if ok != tc.tagWasExisting {
				t.Fatalf("PutTag(%q, %v) failed on existing check, want %t, got %t", tc.tagToMove, tc.tagsList, tc.tagWasExisting, ok)
			}
			if err := l.PutTag(tc.tagToMove, tc.target); err != nil {
				t.Errorf("PutTag(%q, %v) failed: %s", tc.tagToMove, tc.tagsList, err)
			}
			want, ok := l.Manifest(tc.target)
			if !ok {
				t.Fatalf("PutTag(%q, %v) failed: %q not found", tc.tagToMove, tc.tagsList, tc.target)
			}
			got, ok := l.Manifest(tc.tagToMove)
			if !ok {
				t.Fatalf("PutTag(%q, %v) failed: %q not found", tc.tagToMove, tc.tagsList, tc.tagToMove)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("PutTag(%q, %v) mismatch: (-want, +got):\n%s", tc.tagToMove, tc.tagsList, diff)
			}
			newDigest, ok := l.TagToDigest[tc.tagToMove]
			if !ok {
				t.Errorf("PutTag(%q, %v) digest not found after moving", tc.tagToMove, tc.tagsList)
			}
			if oldDigest == newDigest {
				t.Errorf("PutTag(%q, %v) digest was not changed", tc.tagToMove, tc.tagsList)
			}

		})
	}
}

func TestPutTagErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tagsList  [][]string
		tagToMove string
		target    string
	}{
		{
			name:      "non-exist target",
			tagsList:  [][]string{{"t1"}},
			tagToMove: "move-me",
			target:    "t2",
		},
		{
			name:      "move only tag of a manifest",
			tagsList:  [][]string{{"move-me"}, {"t2"}},
			tagToMove: "move-me",
			target:    "t2",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			l := NewList("name", genManifests(tc.tagsList))
			if err := l.PutTag(tc.tagToMove, tc.target); err == nil {
				t.Errorf("PutTag(%q, %v) succeeded, want error", tc.tagToMove, tc.tagsList)
			}
		})
	}
}

func TestDeleteTag(t *testing.T) {
	t.Parallel()
	tagToDelete := "delete-me"
	l := NewList("name", genManifests([][]string{{"t1"}, {tagToDelete, "t2"}}))
	if _, ok := l.Manifest(tagToDelete); !ok {
		t.Fatalf("Manifest(%q) failed: tag not found", tagToDelete)
	}
	if _, ok := l.TagToDigest[tagToDelete]; !ok {
		t.Fatalf("TagToDigest[%q] failed: tag not found", tagToDelete)
	}
	if err := l.DeleteTag("delete-me"); err != nil {
		t.Fatalf("DeleteTag(%q) fialed: %s", tagToDelete, err)
	}
	if _, ok := l.Manifest(tagToDelete); ok {
		t.Errorf("DeleteTag(%q) failed: the manifest was still there", tagToDelete)
	}
	if _, ok := l.TagToDigest[tagToDelete]; ok {
		t.Errorf("DeleteTag(%q) failed: the digtest was still there", tagToDelete)
	}
}
