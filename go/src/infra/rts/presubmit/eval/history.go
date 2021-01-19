// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"bufio"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/sync/parallel"

	evalpb "infra/rts/presubmit/eval/proto"
)

// readTestDurations reads test duration records from a directory.
func readTestDurations(ctx context.Context, dir string, dest chan<- *evalpb.TestDurationRecord) error {
	return readHistoryRecords(dir, func(entry []byte) error {
		td := &evalpb.TestDurationRecord{}
		if err := protojson.Unmarshal(entry, td); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
		case dest <- td:
		}
		return ctx.Err()
	})
}

// readHistoryRecords reads JSON values from .jsonl.gz files in the given
// directory.
func readHistoryRecords(dir string, callback func(entry []byte) error) error {
	// Check dir existance first, because filepath.Glob quietly returns an empty
	// slice if the directory doesn't exist.
	switch st, err := os.Stat(dir); {
	case err != nil:
		return err
	case !st.IsDir():
		return errors.Reason("%q is not a directory", dir).Err()
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl.gz"))
	if err != nil {
		return err
	}

	return parallel.WorkPool(100, func(work chan<- func() error) {
		for _, fileName := range files {
			fileName := fileName
			work <- func() error {
				// Open the file.
				f, err := os.Open(fileName)
				if err != nil {
					return err
				}
				defer f.Close()

				// Decompress as GZIP.
				gz, err := gzip.NewReader(f)
				if err != nil {
					return err
				}
				defer gz.Close()

				// Split by line.
				scan := bufio.NewScanner(gz)
				scan.Buffer(nil, 1e8) // 100 MB.
				for scan.Scan() {
					if err := callback(scan.Bytes()); err != nil {
						return err
					}
				}
				return scan.Err()
			}
		}
	})
}
