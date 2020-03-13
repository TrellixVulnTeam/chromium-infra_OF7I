// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"

	"google.golang.org/api/googleapi"
)

func TestSwarmingCallWithRetries_TransientFailure(t *testing.T) {
	ctx, testClock := testclock.UseTime(context.Background(), time.Now())
	testClock.SetTimerCallback(func(time.Duration, clock.Timer) {
		// Longer than any reasonable retry parameters so that the test always
		// makes instantaneous progress.
		testClock.Add(10 * time.Minute)
	})
	count := 0
	f := func() error {
		defer func() { count++ }()
		if count == 0 {
			return &googleapi.Error{
				Code: http.StatusInternalServerError, // 500
			}
		}
		return nil
	}
	err := callWithRetries(ctx, "test transient failure", f)
	if err != nil {
		t.Fatalf("call error actual != expected, %v != %v", err, nil)
	}
	if count != 2 {
		t.Fatalf("try count actual != expected, %d != %d", count, 1)
	}
}

func TestSwarmingCallWithRetries_ConnectionReset(t *testing.T) {
	ctx, testClock := testclock.UseTime(context.Background(), time.Now())
	testClock.SetTimerCallback(func(time.Duration, clock.Timer) {
		// Longer than any reasonable retry parameters so that the test always
		// makes instantaneous progress.
		testClock.Add(10 * time.Minute)
	})
	count := 0
	f := func() error {
		defer func() { count++ }()
		if count == 0 {
			return errors.New("read tcp: read: connection reset by peer")
		}
		return nil
	}
	err := callWithRetries(ctx, "test connection reset", f)
	if err != nil {
		t.Fatalf("call error actual != expected, %v != %v", err, nil)
	}
	if count != 2 {
		t.Fatalf("try count actual != expected, %d != %d", count, 1)
	}
}

func TestSwarmingCallWithRetries_NontransientFailure(t *testing.T) {
	count := 0
	f := func() error {
		count++
		return errors.New("foo")
	}
	err := callWithRetries(context.Background(), "test non-transient failure", f)
	if err == nil {
		t.Fatalf("call error unexpectedly nil")
	}
	if count != 1 {
		t.Fatalf("try count actual != expected, %d != %d", count, 1)
	}
}

type testTaskListURLForTagsData struct {
	tags        []string
	tasklistURL string
}

func TestTaskListURL(t *testing.T) {
	swarmingService := "swarming.appspot.com"
	testCases := []testTaskListURLForTagsData{
		{
			tags:        []string{"attemptID:123"},
			tasklistURL: "https://swarming.appspot.com/tasklist?f=attemptID%3A123",
		},
		{
			tags:        []string{"attemptID:123", "multipleTags:True"},
			tasklistURL: "https://swarming.appspot.com/tasklist?f=attemptID%3A123&f=multipleTags%3ATrue",
		},
	}

	for _, c := range testCases {
		got := TaskListURLForTags(swarmingService, c.tags)
		if c.tasklistURL != got {
			t.Fatalf("Non-matched tasklist URL:\nExpected: %s\nActual %s", c.tasklistURL, got)
		}
	}
}

func TestParseSwarmingHost(t *testing.T) {
	swarmingService := "chromeos-swarming.appspot.com"
	toParse := []string{
		"https://chromeos-swarming.appspot.com/",
		"https://chromeos-swarming.appspot.com",
		"chromeos-swarming.appspot.com",
	}
	for _, p := range toParse {
		if s := parseSwarmingHost(p); s != swarmingService {
			t.Fatalf("Failed to parse %s to %s", p, s)
		}
	}
}
