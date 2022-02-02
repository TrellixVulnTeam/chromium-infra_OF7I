// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jdxcode/netrc"
)

type fakeRepo struct {
	name string
	tags []string
}

func (r *fakeRepo) allTagsOnImage(_ authn.Authenticator, tag string) ([]string, error) {
	return r.tags, nil
}

func (r *fakeRepo) String() string { return r.name }

type fakeSrcServer struct {
	resp string
}

func (s *fakeSrcServer) download(string, *netrc.Netrc) (string, error) {
	return s.resp, nil
}

func TestResolveImageToOfficial(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		image *parsedImage
		want  string
	}{
		{
			name: "resolve latest-official",
			image: &parsedImage{
				repo:  &fakeRepo{"fake.io/image1", []string{"tag1", latestOfficial, "official-100", "tag2"}},
				regex: regexp.MustCompile(`^official-\d+$`),
				tag:   latestOfficial,
			},
			want: "fake.io/image1:official-100",
		},
		{
			name: "resolve canary",
			image: &parsedImage{
				repo:  &fakeRepo{"fake.io/image2", []string{"random-tag", "canary", "TAG-22"}},
				regex: regexp.MustCompile(`^TAG-\d+$`),
				tag:   "canary",
			},
			want: "fake.io/image2:TAG-22",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveImageToOfficial(tc.image, nil)
			if err != nil {
				t.Errorf("resolveImages(%v) failed: %s", tc.image, err)
			}
			if got != tc.want {
				t.Errorf("resolveImageToOfficial (%v) = %q, want %q", tc.image, got, tc.want)
			}
		})
	}
}

func TestResolveImageToOfficialErrors(t *testing.T) {
	t.Parallel()
	image := &parsedImage{
		repo:  &fakeRepo{"fake.io/image1", []string{"tag1", latestOfficial, "bad-official-100", "tag2"}},
		regex: regexp.MustCompile(`^official-\d+$`),
	}
	if _, err := resolveImageToOfficial(image, nil); err == nil {
		t.Errorf("resolveImageToOfficial(%v) succeeded with no official tags, want error", image)
	}
}

func TestResolveImageErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		images []image
	}{
		{
			name: "regex doesn't start with ^",
			images: []image{
				{
					Name:             "image1",
					Repo:             "repo",
					OfficialTagRegex: "regex$",
				},
			},
		},
		{
			name: "regex doesn't end with $",
			images: []image{
				{
					Name:             "image1",
					Repo:             "repo",
					OfficialTagRegex: "^regex",
				},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := resolveImages(tc.images, nil); err == nil {
				t.Errorf("resolveImages(%v) succeeded, want error", tc.images)
			}
		})
	}
}

func TestSplitYAMLDoc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "zero yaml doc",
			content: "",
			want:    nil,
		},
		{
			name:    "one yaml doc",
			content: "a: 1",
			want:    []string{"a: 1\n"},
		},
		{
			name:    "one yaml doc with seperator",
			content: "---\na: 1",
			want:    []string{"a: 1\n"},
		},
		{
			name:    "two yaml docs",
			content: "---\na: 1\n---\nb: 2",
			want:    []string{"a: 1\n", "b: 2\n"},
		},
		{
			name:    "two yaml docs without leading '---'",
			content: "a: 1\n---\nb: 2",
			want:    []string{"a: 1\n", "b: 2\n"},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, _ := splitYAMLDoc(tc.content)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("splitYAMLDoc(%q) mismatch: (-want, +got):\n%s", tc.content, diff)
			}
		})
	}
}
