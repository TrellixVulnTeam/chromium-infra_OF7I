// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gs

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	gcgs "go.chromium.org/luci/common/gcloud/gs"
)

func TestUploadSkipsNamedPipes(t *testing.T) {
	f, closer := newTestFixture(t)
	defer closer()

	l, err := net.Listen("unix", filepath.Join(f.src, "some.socket"))
	if err != nil {
		t.Fatalf("Failed to create unix domain socket: %s", err)
	}
	defer l.Close()

	if _, err := os.Stat(filepath.Join(f.src, "some.socket")); os.IsNotExist(err) {
		t.Errorf("Unix domain socket file not created. os.Stat() returned: %s", err)
	}
	if err := f.w.WriteDir(context.Background(), f.src, gcgs.Path(f.dst)); err != nil {
		t.Fatalf("Error writing directory: %s", err)
	}
	if _, err := os.Stat(filepath.Join(f.dst, "some.socket")); !os.IsNotExist(err) {
		t.Errorf("Unix domain socket copied, want skipped")
	}
}
