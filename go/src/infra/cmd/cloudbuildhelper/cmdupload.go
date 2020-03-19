// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"

	"infra/cmd/cloudbuildhelper/fileset"
	"infra/cmd/cloudbuildhelper/storage"
)

var cmdUpload = &subcommands.Command{
	UsageLine: "upload <target-manifest-path> [...]",
	ShortDesc: "uploads the tarball with the context directory to the storage",
	LongDesc: `Uploads the tarball with the context directory to the storage.

Evaluates input YAML manifest specified via the positional argument, executes
all local build steps there, and rewrites Dockerfile to use pinned digests
instead of tags. Writes the resulting context dir to a temp *.tar.gz file and
then uploads this file to Google Storage, if not already there, using its SHA256
digest as a file name (the bucket and prefix are picked from the "infra" section
in the manifest).

Finally, attaches the given -canonical-tag to Google Storage object's metadata.
The first such attached tag becomes an immutable alias for this object. "upload"
subcommand returns it in its JSON output.

The canonical tag should identify the exact version of inputs (e.g. it usually
includes git revision or other unique version identifier).

If -canonical-tag is set to a literal constant ":inputs-hash", it is calculated
from SHA256 of the tarball. This is mostly needed for compatibility with
"build" subcommand.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdUploadRun{}
		c.init()
		return c
	},
}

type cmdUploadRun struct {
	commandBase

	targetManifest string
}

func (c *cmdUploadRun) init() {
	c.commandBase.init(c.exec,
		extraFlags{
			auth:         true,
			infra:        true,
			canonicalTag: true,
			buildID:      true,
			jsonOutput:   true,
		},
		[]*string{
			&c.targetManifest,
		},
	)
}

func (c *cmdUploadRun) exec(ctx context.Context) error {
	m, infra, err := c.loadManifest(c.targetManifest, true, false)
	if err != nil {
		return err
	}
	if err := validateCanonicalTag(c.canonicalTag); err != nil {
		return err
	}

	// Initialize Storage instance based on what's in the manifest.
	ts, err := c.tokenSource(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to setup auth").Err()
	}
	store, err := storage.New(ctx, ts, infra.Storage)
	if err != nil {
		return errors.Annotate(err, "failed to initialize Storage").Err()
	}

	_ = m
	_ = store

	// TODO:
	//
	// Build the tarball as temp file, grab its hash.
	// Upload to GS if not already there.
	// Transactionally update metadata:
	//   * If it is the first tag ever, declare it "canonical", store.
	//   * Otherwise just grab the existing canonical tag.
	//   * Add reference to this build to the metadata.
	// Return (GS path, SHA256 digest, canonical tag) as JSON output.

	return errors.Reason("not implemented").Err()
}

////////////////////////////////////////////////////////////////////////////////

// writeToTemp saves the fileset.Set as a temporary *.tar.gz file, returning it
// and its SHA256 hex digest.
//
// The file is opened in read/write mode. The caller is responsible for closing
// and deleting it when done.
func writeToTemp(out *fileset.Set) (*os.File, string, error) {
	f, err := ioutil.TempFile("", "cloudbuildhelper_*.tar.gz")
	if err != nil {
		return nil, "", err
	}
	h := sha256.New()
	if err := out.ToTarGz(io.MultiWriter(f, h)); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, "", err
	}
	return f, hex.EncodeToString(h.Sum(nil)), nil
}

// uploadToStorage uploads the given file to the storage if it's not there yet.
func uploadToStorage(ctx context.Context, s storageImpl, obj, digest string, f *os.File) (*storage.Object, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	switch uploaded, err := s.Check(ctx, obj); {
	case err != nil:
		return nil, errors.Annotate(err, "failed to query the storage for presence of uploaded tarball").Err()
	case uploaded != nil:
		return uploaded, nil
	}

	// Rewind the temp file we have open in read/write mode.
	if _, err := f.Seek(0, 0); err != nil {
		return nil, errors.Annotate(err, "failed to seek inside the temp file").Err()
	}

	uploaded, err := s.Upload(ctx, obj, digest, f)
	return uploaded, errors.Annotate(err, "failed to upload the tarball").Err()
}
