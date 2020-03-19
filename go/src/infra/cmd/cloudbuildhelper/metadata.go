// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/cloudbuildhelper/registry"
	"infra/cmd/cloudbuildhelper/storage"
)

// See cmdBuild and cmdUpload help strings.
const inputsHashCanonicalTag = ":inputs-hash"

const (
	imageRefMetaKey = "cbh-image-ref" // GCS metadata key for imageRef{...} JSON blobs
	buildRefMetaKey = "cbh-build-ref" // GCS metadata key for buildRef{...} JSON blobs
)

// imageRef is stored as metadata of context tarballs in Google Storage.
//
// It refers to some image built from the tarball.
type imageRef struct {
	Image        string    `json:"image"`              // name of the uploaded image "<registry>/<name>"
	Digest       string    `json:"digest"`             // docker digest of the uploaded image "sha256:..."
	CanonicalTag string    `json:"tag"`                // its canonical tag
	BuildID      string    `json:"build_id,omitempty"` // parent CI build that produced this image (FYI)
	Timestamp    time.Time `json:"-"`                  // timestamp of the metadata entry
}

// buildRef is stored as metadata of context tarballs in Google Storage.
//
// If refers to some CI build that reused the tarball or image built from it.
// This information is retained for debugging.
type buildRef struct {
	BuildID      string    `json:"build_id"`      // value of -build-id flag (may be "")
	CanonicalTag string    `json:"tag,omitempty"` // value of -canonical-tag flag
	Timestamp    time.Time `json:"-"`             // timestamp of the metadata entry
}

// Log dumps information about the image to the log.
func (r *imageRef) Log(ctx context.Context, preamble string) {
	logging.Infof(ctx, "%s", preamble)
	if r.CanonicalTag == "" {
		logging.Infof(ctx, "    Name:   %s", r.Image)
	} else {
		logging.Infof(ctx, "    Name:   %s:%s", r.Image, r.CanonicalTag)
	}
	logging.Infof(ctx, "    Digest: %s", r.Digest)
	logging.Infof(ctx, "    View:   %s", r.ViewURL())
}

// ViewURL returns an URL of the image, for humans.
func (r *imageRef) ViewURL() string {
	if r.CanonicalTag != "" {
		return fmt.Sprintf("https://%s:%s", r.Image, r.CanonicalTag)
	}
	return fmt.Sprintf("https://%s@%s", r.Image, r.Digest)
}

// validateCanonicalTag returns a CLI error if the canonical tag is invalid.
func validateCanonicalTag(tag string) error {
	if tag == "" {
		return errBadFlag("-canonical-tag", "a value is required")
	}
	if tag != inputsHashCanonicalTag {
		if err := registry.ValidateTag(tag); err != nil {
			return errBadFlag("-canonical-tag", err.Error())
		}
	}
	return nil
}

// validateTags returns a CLI error if some of the tags are invalid.
func validateTags(tags []string) error {
	for _, t := range tags {
		if err := registry.ValidateTag(t); err != nil {
			return errBadFlag("-tag", err.Error())
		}
	}
	return nil
}

// calcInputsHashCanonicalTag returns the actual value to use for :inputs-hash.
func calcInputsHashCanonicalTag(digest string) string {
	return "cbh-inputs-" + digest[:24]
}

// updateMetadata appends to the metadata of the tarball in the storage.
//
// Adds serialized 'img' and 'b' there (if they are non-nil).
func updateMetadata(ctx context.Context, obj *storage.Object, s storageImpl, img *imageRef, b *buildRef) error {
	ts := storage.TimestampFromTime(clock.Now(ctx))

	var imgRefJSON []byte
	if img != nil {
		var err error
		if imgRefJSON, err = json.Marshal(img); err != nil {
			return errors.Annotate(err, "marshalling imageRef %v", img).Err()
		}
	}

	var buildRefJSON []byte
	if b != nil {
		var err error
		if buildRefJSON, err = json.Marshal(b); err != nil {
			return errors.Annotate(err, "marshalling buildRef %v", b).Err()
		}
	}

	err := s.UpdateMetadata(ctx, obj, func(m *storage.Metadata) error {
		if imgRefJSON != nil {
			m.Add(storage.Metadatum{
				Key:       imageRefMetaKey,
				Timestamp: ts,
				Value:     string(imgRefJSON),
			})
		}
		if buildRefJSON != nil {
			m.Add(storage.Metadatum{
				Key:       buildRefMetaKey,
				Timestamp: ts,
				Value:     string(buildRefJSON),
			})
		}
		m.Trim(50) // to avoid growing metadata size indefinitely
		return nil
	})

	return errors.Annotate(err, "failed to update tarball metadata").Err()
}

// imageRefsFromMetadata deserializes imageRefs stored in the object metadata.
//
// Logs and skips invalid entries.
func imageRefsFromMetadata(ctx context.Context, obj *storage.Object) (out []imageRef) {
	for _, md := range obj.Metadata.Values(imageRefMetaKey) {
		var ref imageRef
		if err := json.Unmarshal([]byte(md.Value), &ref); err != nil {
			logging.Warningf(ctx, "Skipping bad metadata value %q", md.Value)
		} else {
			ref.Timestamp = md.Timestamp.Time()
			out = append(out, ref)
		}
	}
	return
}
