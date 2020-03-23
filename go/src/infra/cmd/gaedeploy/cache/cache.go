// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/gaedeploy/source"
)

// Cache represents an on-disk cache of unpacked tarballs.
//
// It also knows how to populate and trim it.
//
// Directory layout:
//  <Root>/
//    <artifact's sha256 hex digest>/
//      lock           # lock file to manage concurrent access
//      cache.json     # bookkeeping info about this cache entry
//      tmp_*.tar.gz   # exists temporarily when fetching the tarball
//      tmp_data_*/    # exists temporarily when unpacking the tarball
//      data/          # the unpacked tarball goes here
type Cache struct {
	Root string // the root cache directory
}

// WithTarball calls `cb` with a path to the unpacked tarball.
//
// If the cache has such tarball already (as identified by its SHA256 digest),
// calls `cb` right away. Otherwise fetches and unpacks the tarball first.
//
// `cb` may modify files in the directory if necessary. Modification will be
// preserved in the cache, so `cb` should be careful with them.
//
// Access to an unpacked tarball directory is protected by a global file system
// lock. Only one `WithTarball` invocation can touch it concurrently.
func (c *Cache) WithTarball(ctx context.Context, src source.Source, cb func(path string) error) error {
	entryDir := filepath.Join(c.Root, hex.EncodeToString(src.SHA256()))
	if err := os.MkdirAll(entryDir, 0700); err != nil {
		return errors.Annotate(err, "failed to create a directory for the tarball").Err()
	}

	// Enter the global critical section to avoid weird cache states due to
	// concurrent execution of multiple processes.
	unlock, err := lockFS(ctx, filepath.Join(entryDir, "lock"), 15*time.Minute)
	if err != nil {
		return errors.Annotate(err, "failed to grab the FS lock").Err()
	}
	defer func() {
		if err := unlock(); err != nil {
			logging.Errorf(ctx, "Failed to remove the FS lock: %s", err)
		}
	}()

	// Drop a JSON file with info about the cache entry. Used by the GC.
	err = modifyMetadata(ctx, entryDir, func(m *cacheMetadata) {
		now := clock.Now(ctx)
		if m.Created.IsZero() {
			m.Created = now
		}
		m.Touched = now
	})
	if err != nil {
		return errors.Annotate(err, "failed to update the cache metadata file").Err()
	}

	// Fetch and unpack the tarball if haven't done it yet.
	tarballDir := filepath.Join(entryDir, "data")
	if _, err := os.Stat(tarballDir); err != nil {
		if !os.IsNotExist(err) {
			return errors.Annotate(err, "failed to check presence of the unpacked tarball").Err()
		}

		// Prepare a temp file to download the tarball into.
		tmp, err := ioutil.TempFile(entryDir, "tmp_*.tar.gz")
		if err != nil {
			return errors.Annotate(err, "failed to create a temp file to fetch the tarball into").Err()
		}
		tmpName := tmp.Name()
		tmp.Close() // we are only after the file name
		nukeTmpFile := func() {
			if err := os.Remove(tmpName); err != nil && os.IsNotExist(err) {
				logging.Warningf(ctx, "Failed to delete the temp file: %s", err)
			}
		}

		// Note: note using defer for nukeTmpFile and (later) nukeStagingDir because
		// we want them called before cb(...). Defers will be called after.

		// Prepare a staging directory to unzip the tarball into. We'll rename it
		// into `tarballDir` on success.
		stagingDir, err := ioutil.TempDir(entryDir, "tmp_data_*")
		if err != nil {
			return errors.Annotate(err, "failed to create a temp directory to unpack the tarball into").Err()
		}
		nukeStagingDir := func() {
			if err := os.RemoveAll(stagingDir); err != nil {
				logging.Warningf(ctx, "Failed to delete the staging directory: %s", err)
			}
		}

		// Download and untar the file into the staging directory.
		err = fetchAndUntar(ctx, src, tmpName, stagingDir)
		nukeTmpFile() // served its purpose
		if err != nil {
			nukeStagingDir() // contains incomplete garbage, kill it
			return err       // annotated already
		}

		if err := os.Rename(stagingDir, tarballDir); err != nil {
			nukeStagingDir()
			return errors.Annotate(err, "failed to move the staging directory into its final place").Err()
		}
	} else {
		logging.Infof(ctx, "Found the unpackaged tarball in the cache.")
	}

	// Let the callback do the rest.
	return cb(tarballDir)
}

// Trim removes old cache entries, keeping only most recently touched ones.
func (c *Cache) Trim(ctx context.Context, keep int) error {
	// TODO
	return nil
}
