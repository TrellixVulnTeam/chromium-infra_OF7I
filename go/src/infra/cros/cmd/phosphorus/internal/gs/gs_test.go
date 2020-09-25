// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gs

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
	gcgs "go.chromium.org/luci/common/gcloud/gs"
)

// Implements InnerClient interface, writing to provided local directory instead
// of Google Storage.
type fakeInnerClient struct{}

// fakeWriter implements gs.Writer interface.
type fakeWriter struct {
	os.File
}

func (w *fakeWriter) Count() int64 {
	// Not _really_ implemented.
	return 0
}

func (c *fakeInnerClient) NewWriter(p gcgs.Path) (gs.Writer, error) {
	// This assumes that the incoming path is a valid local path, as opposed to
	// a GS URL (starting with gs://).
	l := string(p)
	d := filepath.Dir(l)
	if err := os.MkdirAll(d, 0777); err != nil {
		return nil, errors.Reason("failed to create directory %s: %s", d, err).Err()
	}
	f, err := os.Create(l)
	if err != nil {
		return nil, err
	}
	return &fakeWriter{File: *f}, nil
}

type testFixture struct {
	// Temporary source directory to copy files from.
	src string
	// Temporary destination directory to copy files to.
	dst string

	// A DirWriter instance to test.
	w *DirWriter
}

// Creates a new test fixture, taking care of common boilerplate.
//
// Returns a function that must be deferred for cleaning up temporary
// directories.
func newTestFixture(t *testing.T) (*testFixture, func()) {
	t.Helper()

	tmp, err := ioutil.TempDir("", "phosphorus")
	if err != nil {
		t.Fatalf("Failed to create temporary directory")
	}

	closer := func() {
		if err := os.RemoveAll(tmp); err != nil {
			panic(fmt.Sprintf("Failed to delete temporary directory %s: %s", tmp, err))
		}
	}

	src := filepath.Join(tmp, "src")
	if err := os.Mkdir(src, 0777); err != nil {
		closer()
		t.Fatalf("Failed to create source directory: %s", err)
	}
	dst := filepath.Join(tmp, "dst")
	if err := os.Mkdir(dst, 0777); err != nil {
		closer()
		t.Fatalf("Failed to create destination directory: %s", err)
	}

	return &testFixture{
		src: src,
		dst: dst,
		w: &DirWriter{
			client:               &fakeInnerClient{},
			maxConcurrentUploads: 1,
		},
	}, closer
}

func TestUploadSingleFile(t *testing.T) {
	f, closer := newTestFixture(t)
	defer closer()
	s, err := os.Create(filepath.Join(f.src, "regular.txt"))
	if err != nil {
		t.Fatalf("Failed to create source file: %s", err)
	}
	defer s.Close()
	if err := f.w.WriteDir(context.Background(), f.src, gcgs.Path(f.dst)); err != nil {
		t.Fatalf("Error writing directory: %s", err)
	}
	if _, err := os.Stat(filepath.Join(f.dst, "regular.txt")); os.IsNotExist(err) {
		t.Errorf("Regular file not copied. os.Stat() returned: %s", err)
	}
}
