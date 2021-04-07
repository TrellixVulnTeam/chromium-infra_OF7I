// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package gerrit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
)

func TestGetChangeRev_success(t *testing.T) {
	changeNum := int64(123)
	revision := int32(2)
	project := "chromiumos/for/the/win"

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	gerritMock := gerritpb.NewMockGerritClient(ctl)
	gerritMock.EXPECT().GetChange(gomock.Any(), gomock.Any()).Return(
		&gerritpb.ChangeInfo{
			Number:  changeNum,
			Project: project,
			Revisions: map[string]*gerritpb.RevisionInfo{
				"hash1": {
					Number: revision,
					Files: map[string]*gerritpb.FileInfo{
						"/file/1": {},
					},
				},
			},
		},
		nil)
	host := "limited-review.googlesource.com"
	mockGerrit = gerritMock

	expectedChRev := &ChangeRev{
		ChangeRevKey: ChangeRevKey{
			Host:      host,
			ChangeNum: changeNum,
			Revision:  revision,
		},
		Project: project,
		Files: []string{
			"/file/1",
		},
	}

	actualChRev, err := GetChangeRev(context.Background(), http.DefaultClient, changeNum, revision, host)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(expectedChRev, actualChRev); diff != "" {
		t.Errorf("ChangeRev bad result (-want/+got)\n%s", diff)
	}
}

func TestGetChangeRev_missingRevision(t *testing.T) {
	changeNum := int64(123)
	project := "chromiumos/for/the/win"

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	gerritMock := gerritpb.NewMockGerritClient(ctl)
	gerritMock.EXPECT().GetChange(gomock.Any(), gomock.Any()).Return(
		&gerritpb.ChangeInfo{
			Number:  changeNum,
			Project: project,
			Revisions: map[string]*gerritpb.RevisionInfo{
				"hash1": {
					Number: 1,
					Files: map[string]*gerritpb.FileInfo{
						"/file/1": {},
						"/file/2": {},
					},
				},
			},
		},
		nil)
	host := "limited-review.googlesource.com"
	mockGerrit = gerritMock

	// We're asking for revision 2, but there's only a revision 1.
	_, err := GetChangeRev(context.Background(), http.DefaultClient, changeNum, 2, host)
	if err == nil {
		t.Error("expected an error, got none")
	}
	substr := "found no revision 2"
	if !strings.Contains(fmt.Sprintf("%v", err), substr) {
		t.Errorf("Expected error to contain %s, instead %v", substr, err)
	}
}

func TestParseCLURL(t *testing.T) {
	tests := []struct {
		url      string
		expected *ChangeRevKey
	}{
		{
			url: "https://chromium-review.googlesource.com/c/chromiumos/project/with/slashes/+/1234/5",
			expected: &ChangeRevKey{
				Host:      "chromium-review.googlesource.com",
				ChangeNum: 1234,
				Revision:  5,
			},
		},
		{
			url: "https://chromium-review.googlesource.com/c/chromiumos/project/with/slashes/+/1234/5",
			expected: &ChangeRevKey{
				Host:      "chromium-review.googlesource.com",
				ChangeNum: 1234,
				Revision:  5,
			},
		},
		{
			url: "https://chromium-review.googlesource.com/c/projectwithoutslash/+/1234/5",
			expected: &ChangeRevKey{
				Host:      "chromium-review.googlesource.com",
				ChangeNum: 1234,
				Revision:  5,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			got, err := ParseCLURL(test.url)
			if err != nil {
				t.Fatalf("ParseCLURL(%q) failed: %s\n", test.url, err)
			}
			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("ParseCLURL(%q) returned unexpected diff (-want +got):\n%s", test.url, diff)
			}
		})
	}
}

func TestParseCLURLErrors(t *testing.T) {
	urls := []string{
		// URL must have "-review.googlesource.com"
		"https://chromium.googlesource.com/c/chromiumos/project/+/1234/5",
		// URL must have scheme.
		"chromium-review.googlesource.com/c/chromiumos/project/with/slashes/+/1234/5",
		// Missing "/c/"
		"https://chromium-review.googlesource.com/chromiumos/project/+/1234/5",
		// Missing "/+/"
		"https://chromium-review.googlesource.com/c/chromiumos/project/1234/5",
		// Missing patchset
		"https://chromium-review.googlesource.com/c/chromiumos/project/+/1234",
		// URL is part of another string
		"example.com/?redirect=chromium-review.googlesource.com/c/chromiumos/project/with/slashes/+/1234/5",
	}

	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			if _, err := ParseCLURL(url); err == nil {
				t.Errorf("ParseCLURL(%q) succeeded, expected error", url)
			}
		})
	}
}
