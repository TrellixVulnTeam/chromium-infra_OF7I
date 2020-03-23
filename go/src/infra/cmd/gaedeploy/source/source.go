// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package source abstracts source of deployable tarballs.
package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// Source indicates how to grab a tarball.
type Source interface {
	// SHA256 returns the expected SHA256 of the tarball.
	SHA256() []byte

	// Open returns a reader with tarballs body.
	//
	// May optionally use the given `tmp` path as a staging file. It's the
	// caller's responsibility to delete it later.
	//
	// The caller should also verify SHA256 of the data it reads matches SHA256().
	Open(ctx context.Context, tmp string) (io.ReadCloser, error)
}

// New initializes a source by validating path format.
//
// `sha256hex` is optional. If given, it indicates the expected digest of the
// tarball. It will be verified when the tarball is fetched by cache.Cache.
func New(path, sha256hex string) (Source, error) {
	// Convert the hex digest to bytes.
	var sha256bin []byte
	if sha256hex != "" {
		var err error
		if sha256bin, err = hex.DecodeString(sha256hex); err != nil {
			return nil, errors.Annotate(err, "bad -tarball-sha256, not hex").Err()
		}
		if len(sha256bin) != sha256.Size {
			return nil, errors.Reason("bad -tarball-sha256, wrong length").Err()
		}
	}

	// Calculating SHA256 of GS source is a lot of work and we don't really have
	// this use case, so require -tarball-sha256 in this case.
	if strings.HasPrefix(path, "gs://") {
		if sha256bin == nil {
			return nil, errors.Reason("-tarball-sha256 is required when using GCS paths").Err()
		}
		return &gsSource{path: path, sha256: sha256bin}, nil
	}

	// Calculating SHA256 of a local file is easy though. Omitting -tarball-sha256
	// is useful when running both cloudbuildhelper (to build a local tarball) and
	// gaedeploy (to deploy it) locally.
	if sha256bin == nil {
		f, err := os.Open(path)
		if err != nil {
			return nil, errors.Annotate(err, "can't open the file to calculate its SHA256").Err()
		}
		defer f.Close()

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return nil, errors.Annotate(err, "failed to read %q to calculate its SHA256", path).Err()
		}
		sha256bin = h.Sum(nil)
	}
	return &fileSource{path: path, sha256: sha256bin}, nil
}
