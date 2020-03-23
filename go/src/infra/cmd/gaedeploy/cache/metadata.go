// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// Name of the per-entry metadata file.
const metadataFile = "cache.json"

// cacheMetadata exists as `cache.json` file in the directory dedicated to some
// unpacked tarball.
type cacheMetadata struct {
	Created time.Time `json:"created"` // when it was created
	Touched time.Time `json:"touched"` // when it was touched the last time
}

// metadataPath is a path to a JSON file with metadata for given entry.
func metadataPath(dir string) string {
	return filepath.Join(dir, metadataFile)
}

// readMetadata reads metadata file if it exists and not corrupted.
//
// Returns (cacheMetadata{}, nil) if the file doesn't exist or it's not a valid
// JSON file.
func readMetadata(ctx context.Context, dir string) (m cacheMetadata, err error) {
	blob, err := ioutil.ReadFile(metadataPath(dir))
	if err != nil {
		if !os.IsNotExist(err) {
			err = errors.Annotate(err, "failed to read the existing metadata file").Err()
			return
		}
		err = nil
	}
	if len(blob) != 0 {
		if err := json.Unmarshal(blob, &m); err != nil {
			logging.Warningf(ctx, "Ignoring bad metadata file at %q (%s)", metadataPath(dir), err)
			m = cacheMetadata{}
		}
	}
	return
}

// modifyMetadata reads `cache.json` (if exists), calls the callback to update
// it, and stores the result.
func modifyMetadata(ctx context.Context, dir string, cb func(m *cacheMetadata)) error {
	m, err := readMetadata(ctx, dir)
	if err != nil {
		return err
	}

	cb(&m)

	blob, err := json.MarshalIndent(&m, "", "  ")
	if err != nil {
		return errors.Annotate(err, "failed to marshal metadata").Err()
	}
	if err := ioutil.WriteFile(metadataPath(dir), blob, 0600); err != nil {
		return errors.Annotate(err, "failed to write the metadata file").Err()
	}
	return nil
}
