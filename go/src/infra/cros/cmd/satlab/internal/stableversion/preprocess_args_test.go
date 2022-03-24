// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"infra/cros/cmd/satlab/internal/site"
)

// TestPreprocessHostname checks that preprocessHostname correctly determines the hostname and doesn't call
// some of the functions that inspect the environment (getDHBID and isRemoteAccess) when calling them would
// be unnecessary to preprocess the hostname.
func TestPreprocessHostname(t *testing.T) {
	t.Parallel()

	type inParams struct {
		common         site.CommonFlags
		hostname       string
		getDHBID       getDHBID
		isRemoteAccess isRemoteAccess
	}

	cases := []struct {
		name string
		in   inParams
		out  string
		// Use "" to indicate that the error must be nil.
		errFragment string
	}{
		{
			name: "empty",
			in: inParams{
				getDHBID: func() (string, error) {
					panic("don't call me")
				},
				isRemoteAccess: func() (bool, error) {
					panic("don't call me")
				},
			},
			out:         "",
			errFragment: "hostname cannot be empty",
		},
		{
			name: "explicit satlab ID good",
			in: inParams{
				common: site.CommonFlags{
					SatlabID: "a",
				},
				hostname: "b",
				getDHBID: func() (string, error) {
					panic("don't call me")
				},
				isRemoteAccess: func() (bool, error) {
					panic("don't call me")
				},
			},
			out:         "satlab-a-b",
			errFragment: "",
		},
		{
			name: "explicit satlab ID bad",
			in: inParams{
				common: site.CommonFlags{
					SatlabID: "a",
				},
				hostname: "satlab-b",
				getDHBID: func() (string, error) {
					panic("don't call me")
				},
				isRemoteAccess: func() (bool, error) {
					panic("don't call me")
				},
			},
			out:         "",
			errFragment: "already has satlab prefix",
		},
		{
			name: "remote access good",
			in: inParams{
				hostname: "host1",
				getDHBID: func() (string, error) {
					return "75fc6517-3539-458f-9718-7bbc759eb73a", nil
				},
				isRemoteAccess: func() (bool, error) {
					return true, nil
				},
			},
			out:         "satlab-75fc6517-3539-458f-9718-7bbc759eb73a-host1",
			errFragment: "",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.out
			actual, err := preprocessHostname(
				tt.in.common,
				tt.in.hostname,
				tt.in.getDHBID,
				tt.in.isRemoteAccess,
			)
			if tt.errFragment == "" {
				if err != nil {
					t.Errorf("error should have been nil but was %s", err)
				}
			} else {
				if err == nil {
					t.Errorf("error is unexpectedly nil")
				} else {
					if !strings.Contains(err.Error(), tt.errFragment) {
						t.Errorf("%q does not contain %q so it is probably wrong", err, tt.errFragment)
					}
				}
			}
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff (-want +got): %s", diff)
			}
		})
	}
}
