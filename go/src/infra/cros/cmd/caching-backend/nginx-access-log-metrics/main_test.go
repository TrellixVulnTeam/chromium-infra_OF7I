// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package main

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseLine(t *testing.T) {
	t.Parallel()
	line := `127.0.0.1 - - [2021-06-09T20:24:39+00:00] "GET /download/abc HTTP/1.1" 200 369 "-" 0.123 "-" "curl/7.66.0" "-" HIT`

	got := parseLine(line)
	want := &record{
		timestamp:     time.Date(2021, 06, 9, 20, 24, 39, 0, time.UTC),
		clientIP:      "127.0.0.1",
		httpMethod:    "GET",
		path:          "/download/abc",
		status:        200,
		bodyBytesSent: 369,
		expectedSize:  -1,
		requestTime:   0.123,
		cacheStatus:   "HIT",
	}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(record{})); diff != "" {
		t.Errorf("emitMetric returned unexpected diff (-want +got):\n%s", diff)
	}
}
