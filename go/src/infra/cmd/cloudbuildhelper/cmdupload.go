// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/cloudbuildhelper/fileset"
	"infra/cmd/cloudbuildhelper/manifest"
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

	// Execute the main logic.
	obj, digest, oldestRef, err := buildAndUpload(ctx, uploadParams{
		Manifest:     m,
		CanonicalTag: c.canonicalTag,
		BuildID:      c.buildID,
		Store:        store,
	})

	// Return (GS path, SHA256 digest, canonical tag) or error as JSON output.
	res := uploadResult{Name: m.Name}
	if err == nil {
		res.GS.Bucket = obj.Bucket
		res.GS.Name = obj.Name
		res.SHA256 = digest
		res.CanonicalTag = oldestRef.CanonicalTag
	} else {
		res.Error = err.Error()
	}
	if jerr := c.writeJSONOutput(&res); jerr != nil {
		return errors.Annotate(jerr, "failed to write JSON output").Err()
	}
	return err
}

////////////////////////////////////////////////////////////////////////////////

// uploadResult is returned as JSON output by `upload` command.
type uploadResult struct {
	Name  string `json:"name"`            // artifacts name from the manifest YAML
	Error string `json:"error,omitempty"` // non-empty if the upload failed
	GS    struct {
		Bucket string `json:"bucket"` // GCS bucket with the file
		Name   string `json:"name"`   // filename within the bucket
	} `json:"gs,omitempty"`
	SHA256       string `json:"sha256,omitempty"`        // hex digest of the file
	CanonicalTag string `json:"canonical_tag,omitempty"` // its oldest tag
}

// uploadParams are parameters for buildAndUpload.
type uploadParams struct {
	// Inputs.
	Manifest     *manifest.Manifest
	CanonicalTag string
	BuildID      string

	// Infra.
	Store storageImpl
}

// buildAndUpload builds the tarball, uploads it to the storage and updates
// its metadata.
//
// Returns the object in the storage (either new or an existing one) and
// the *oldest* buildRef associated with it.
func buildAndUpload(ctx context.Context, p uploadParams) (obj *storage.Object, digest string, oldestRef *buildRef, err error) {
	err = stage(ctx, p.Manifest, func(out *fileset.Set) error {
		var f *os.File
		if f, digest, err = writeToTemp(ctx, out); err != nil {
			return errors.Annotate(err, "failed to write the tarball with context dir").Err()
		}

		// Cleanup no matter what. Note that we don't care about IO flush errors in
		// f.Close() as long as uploadToStorage sent everything successfully (as
		// verified by checking the hash there).
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()

		// Now that we know the inputs, we can resolve ":inputs-hash".
		if p.CanonicalTag == inputsHashCanonicalTag {
			p.CanonicalTag = calcInputsHashCanonicalTag(digest)
		}
		logging.Infof(ctx, "Canonical tag:  %s", p.CanonicalTag)

		// Upload the tarball (or grab metadata of the existing object).
		obj, err = uploadToStorage(ctx, p.Store,
			fmt.Sprintf("%s/%s.tar.gz", p.Manifest.Name, digest), digest, f)
		if err != nil {
			return err // err is annotated already
		}

		// Dump metadata into the log, just FYI. In particular this logs all
		// previous buildRef's that reused this tarball.
		obj.Log(ctx)

		// Add metadata about *this* build to associate the tarball with it, grab
		// oldest such association (perhaps the one we are adding) to use its tag
		// as the immutable alias for the tarball.
		oldestRef, err = updateMetadata(ctx, obj, p.Store, nil, &buildRef{
			BuildID:      p.BuildID,
			CanonicalTag: p.CanonicalTag,
		})
		return err // already annotated
	})

	// Log if we are reusing an existing tarball with an existing canonical tag.
	if err == nil {
		if p.CanonicalTag == oldestRef.CanonicalTag {
			logging.Infof(ctx, "New canonical tag %q", p.CanonicalTag)
		} else {
			logging.Infof(ctx,
				"Reusing the existing canonical tag %q (built %s)",
				oldestRef.CanonicalTag, humanize.Time(oldestRef.Timestamp))
		}
	}
	return
}

// writeToTemp saves the fileset.Set as a temporary *.tar.gz file, returning it
// and its SHA256 hex digest.
//
// Logs the file size and its digest inside.
//
// The file is opened in read/write mode. The caller is responsible for closing
// and deleting it when done.
func writeToTemp(ctx context.Context, out *fileset.Set) (f *os.File, digest string, err error) {
	logging.Infof(ctx, "Writing tarball with %d files to a temp file...", out.Len())

	f, err = ioutil.TempFile("", "cloudbuildhelper_*.tar.gz")
	if err != nil {
		return nil, "", err
	}

	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()

	h := sha256.New()
	if err := out.ToTarGz(io.MultiWriter(f, h)); err != nil {
		return nil, "", err
	}
	digest = hex.EncodeToString(h.Sum(nil))

	size, err := f.Seek(0, 1)
	if err != nil {
		return nil, "", errors.Annotate(err, "failed to query the size of the temp file").Err()
	}

	logging.Infof(ctx, "Tarball digest: %s", digest)
	logging.Infof(ctx, "Tarball length: %s", humanize.Bytes(uint64(size)))

	return f, digest, nil
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
