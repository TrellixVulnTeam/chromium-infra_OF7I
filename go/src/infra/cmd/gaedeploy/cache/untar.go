// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/gaedeploy/source"
)

// fetchAndUntar fetches the tarball into `tmpName` and then untars it into
// `destDir` (which should already exist).
//
// Verifies its SHA256 along the way. On errors `destDir` may end up having
// incomplete or unverified data, the caller should delete it.
func fetchAndUntar(ctx context.Context, src source.Source, tmpName, destDir string) error {
	logging.Infof(ctx, "Fetching the tarball...")
	rc, err := src.Open(ctx, tmpName)
	if err != nil {
		return errors.Annotate(err, "failed to fetch the tarball").Err()
	}
	defer rc.Close()

	logging.Infof(ctx, "Extracting the tarball...")

	h := sha256.New()
	r := io.TeeReader(rc, h)

	gz, err := gzip.NewReader(r)
	if err != nil {
		return errors.Annotate(err, "failed to read the gzip header").Err()
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Annotate(err, "error when reading the tar file").Err()
		}
		if err := extractOneFromTar(ctx, header, tr, destDir); err != nil {
			return errors.Annotate(err, "when extracting %q", header.Name).Err()
		}
	}

	// Read the rest of the file to update the hash. Theoretically it may have
	// some trailer that the gzip reader didn't read.
	if _, err := io.Copy(ioutil.Discard, r); err != nil {
		return errors.Annotate(err, "failed to read the file trailer").Err()
	}

	// Verify the hash.
	if got, want := h.Sum(nil), src.SHA256(); !bytes.Equal(got, want) {
		return errors.Reason("tarball hash mismatch: got %q, want %q",
			hex.EncodeToString(got), hex.EncodeToString(want)).Err()
	}

	logging.Infof(ctx, "Done.")
	return nil
}

// extractOneFromTar extract one tar archive entry.
//
// Only regular files and directories are extracted. Any other entries (e.g.
// symlinks) trigger an error.
//
// All attributes other than +x owner bit are silently discarded.
func extractOneFromTar(ctx context.Context, h *tar.Header, r io.Reader, destDir string) error {
	if h.Typeflag != tar.TypeDir && h.Typeflag != tar.TypeReg {
		return errors.Reason("unsupported type %d", h.Typeflag).Err()
	}

	name := filepath.Clean(filepath.FromSlash(h.Name))
	if strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return errors.Reason("fishy name").Err()
	}

	if h.Typeflag == tar.TypeDir {
		return os.MkdirAll(filepath.Join(destDir, name), 0700)
	}

	perms := os.FileMode(0600)
	if (h.FileInfo().Mode().Perm() & 0100) != 0 {
		perms |= 0100
	}

	dest, err := os.OpenFile(filepath.Join(destDir, name), os.O_WRONLY|os.O_CREATE, perms)
	if err != nil {
		return errors.Annotate(err, "failed to open the destination file").Err()
	}
	defer dest.Close() // fallback on errors
	if _, err := io.Copy(dest, r); err != nil {
		return errors.Annotate(err, "extraction failed").Err()
	}
	if err := dest.Close(); err != nil {
		return errors.Annotate(err, "failed to flush the file").Err()
	}
	return nil
}
