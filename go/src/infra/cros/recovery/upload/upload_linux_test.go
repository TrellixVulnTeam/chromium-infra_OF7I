// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package upload

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"go.chromium.org/luci/common/errors"
	lucigs "go.chromium.org/luci/common/gcloud/gs"
)

// Implements InnerClient interface, writing to a temporary directory
// instead of Google Storage.
type fakeInnerClient struct {
	initialized bool
	baseDir     string
}

// FakeWriter implements gs.Writer interface.
type fakeWriter struct {
	os.File
}

func (w *fakeInnerClient) Init(tempdir func() string) error {
	if w.initialized {
		return nil
	}
	path := ""
	if tempdir == nil {
		var err error
		path, err = ioutil.TempDir("", "upload_test-")
		if err != nil {
			return errors.Annotate(err, "initialize fake writer").Err()
		}
	} else {
		path = tempdir()
	}
	w.baseDir = path
	w.initialized = true
	return nil
}

func (w *fakeWriter) Count() int64 {
	// Not _really_ implemented.
	return 0
}

func (c *fakeInnerClient) NewWriter(p lucigs.Path) (lucigs.Writer, error) {
	path := filepath.Join(c.baseDir, string(p))
	d := filepath.Dir(path)
	if err := os.MkdirAll(d, 0b111_111_111); err != nil {
		return nil, errors.Reason("failed to create directory %s: %s", d, err).Err()
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &fakeWriter{File: *f}, nil
}

func TestUpload(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	td := t.TempDir()

	if err := ioutil.WriteFile(filepath.Join(td, "a.txt"), []byte("a"), 0o777); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	client := &fakeInnerClient{}
	if err := client.Init(t.TempDir); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if err := Upload(ctx, client, &Params{
		SourceDir:         td,
		GSURL:             "gs://1/2/3",
		MaxConcurrentJobs: 1,
	}); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if _, err := os.Stat(filepath.Join(client.baseDir, "gs://1/2/3/a.txt")); err != nil {
		t.Errorf("failed to stat %q: %s", "a.txt", err)
	}
}
