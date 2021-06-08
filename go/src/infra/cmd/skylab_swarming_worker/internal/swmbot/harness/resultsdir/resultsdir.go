// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package resultsdir implements Autotest results directory creation
// and sealing.
package resultsdir

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"go.chromium.org/luci/common/errors"
)

// Dir represents the results directory of the task. It should be closed
// after all results have been written to seal the directory for upload.
type Dir struct {
	Path string
}

// Open creates the results directory and returns a Dir.
// Dir should be closed after all results have been written to
// seal the directory for upload.
func Open(path string) (*Dir, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, errors.Annotate(err, "open results dir %s", path).Err()
	}
	return &Dir{Path: path}, nil
}

// Close seals the results directory.  This is safe to call multiple
// times. This is safe to call on a nil pointer.
func (d *Dir) Close(ctx context.Context) error {
	if d == nil {
		return nil
	}
	if d.Path == "" {
		return nil
	}
	if err := sealResultsDir(d.Path); err != nil {
		return err
	}
	d.Path = ""
	return nil
}

// OpenSubDir creates a sub directory under the results directory.
// The path is relative to the results directory.
func (d *Dir) OpenSubDir(path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", errors.Reason("Cannot OpenSubDir for an absolute path.").Err()
	}
	subDir := filepath.Join(d.Path, path)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return "", errors.Annotate(err, "open sub dir %s", subDir).Err()
	}
	return subDir, nil
}

const gsOffloaderMarker = ".ready_for_offload"

// sealResultsDir drops a special timestamp file in the results
// directory notifying gs_offloader to offload the directory. The
// results directory should not be touched once sealed.  This should
// not be called on an already sealed results directory.
func sealResultsDir(d string) error {
	ts := []byte(fmt.Sprintf("%d", time.Now().Unix()))
	tsfile := filepath.Join(d, gsOffloaderMarker)
	if err := ioutil.WriteFile(tsfile, ts, 0666); err != nil {
		return errors.Annotate(err, "seal results dir %s", d).Err()
	}
	return nil
}
