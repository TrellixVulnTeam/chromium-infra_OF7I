// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service_test

import (
	"context"
	"testing"

	"go.chromium.org/luci/common/data/stringset"

	trservice "infra/cmd/cros_test_platform/internal/execution/testrunner/service"
	"infra/libs/skylab/inventory"
	"infra/libs/skylab/request"
)

func TestBotsAwareFakeClient(t *testing.T) {
	cases := []struct {
		Tag          string
		Client       trservice.BotsAwareFakeClient
		Args         request.Args
		WantValid    bool
		WantRejected []string
	}{
		{
			Tag:    "no bots with free-form dimensions in build",
			Client: trservice.NewBotsAwareFakeClient(),
			Args: request.Args{
				Dimensions: []string{"free-form:value"},
			},
			WantValid:    false,
			WantRejected: []string{"free-form"},
		},
		{
			Tag:    "no bots with schedulable labels in build",
			Client: trservice.NewBotsAwareFakeClient(),
			Args: request.Args{
				SchedulableLabels: &inventory.SchedulableLabels{
					Board: stringPtr("foo"),
				},
			},
			WantValid:    false,
			WantRejected: []string{"label-board"},
		},
		{
			Tag:    "mismatched bot",
			Client: trservice.NewBotsAwareFakeClient(stringset.NewFromSlice("free-form:bot-value")),
			Args: request.Args{
				Dimensions: []string{"free-form:build-value"},
			},
			WantValid:    false,
			WantRejected: []string{"free-form"},
		},
		{
			Tag:    "matched bot",
			Client: trservice.NewBotsAwareFakeClient(stringset.NewFromSlice("free-form:bot-value")),
			Args: request.Args{
				Dimensions: []string{"free-form:bot-value"},
			},
			WantValid:    true,
			WantRejected: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.Tag, func(t *testing.T) {
			b, r, err := c.Client.ValidateArgs(context.Background(), &c.Args)
			if err != nil {
				t.Fatalf("ValidateArgs returned error: %s", err)
			}
			if b != c.WantValid {
				t.Errorf("ValidateArgs returned %t, want %t", b, c.WantValid)
			}
			for _, k := range c.WantRejected {
				found := false
				for _, actualReject := range r {
					if k == actualReject.Key {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Rejected arguments missing key %s", k)
				}
			}
		})
	}
}

func TestClientCallCountingWrapper(t *testing.T) {
	c := trservice.CallCountingClientWrapper{
		Client: trservice.StubClient{},
	}
	c.ValidateArgs(context.Background(), nil)
	c.LaunchTask(context.Background(), nil)
	if c.CallCounts.ValidateArgs != 1 {
		t.Errorf("ValidateArgs counts is %d, want 1", c.CallCounts.ValidateArgs)
	}
	if c.CallCounts.LaunchTask != 1 {
		t.Errorf("LaunchTask counts is %d, want 1", c.CallCounts.LaunchTask)
	}
}

func stringPtr(val string) *string {
	return &val
}
