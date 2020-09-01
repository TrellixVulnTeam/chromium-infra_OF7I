// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package updater

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/protobuf/encoding/protojson"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/sync/parallel"

	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
)

// Updater computed metadata from a Chromium checkout and uploads it to GCS.
type Updater struct {
	// ChromiumCheckout is a path to chromium/src.git checkout.
	ChromiumCheckout string

	// GCSBucket is the destination bucket for metadata.
	GCSBucket *storage.BucketHandle

	// LegacyGSBucket is the destination bucket for metadata in legacy format.
	GCSBucketLegacy *storage.BucketHandle

	// OutDir is a path to the directory where to write output files.
	OutDir string
}

// Run updates the metadata stored in GCS.
func (u *Updater) Run(ctx context.Context) error {
	return parallel.FanOutIn(func(work chan<- func() error) {
		work <- func() error {
			return u.run(ctx)
		}
		work <- func() error {
			return u.runLegacyFull(ctx)
		}
	})
}

func (u *Updater) run(ctx context.Context) error {
	mapping, err := dirmd.ReadMapping(u.ChromiumCheckout, dirmdpb.MappingForm_COMPUTED)
	if err != nil {
		return err
	}

	// Write in computed form.
	err = parallel.FanOutIn(func(work chan<- func() error) {
		// Write in legacy format.
		work <- func() error {
			return u.writeMapping(ctx, "component_map.json", mapping, false)
		}
		// Write in new format.
		work <- func() error {
			return u.writeMapping(ctx, "metadata_computed.json", mapping, true)
		}
	})
	if err != nil {
		return err
	}

	// Write in reduced form.
	mapping.Reduce()
	return u.writeMapping(ctx, "metadata_reduced.json", mapping, true)
}

func (u *Updater) runLegacyFull(ctx context.Context) error {
	mapping, err := dirmd.ReadMapping(u.ChromiumCheckout, dirmdpb.MappingForm_FULL)
	if err != nil {
		return err
	}

	return u.writeMapping(ctx, "component_map_subdirs.json", mapping, false)
}

func (u *Updater) writeMapping(ctx context.Context, name string, mapping *dirmd.Mapping, modernFormat bool) error {
	var contents []byte
	var bucket *storage.BucketHandle
	if modernFormat {
		bucket = u.GCSBucket
		var err error
		if contents, err = protojson.Marshal(mapping.Proto()); err != nil {
			return err
		}
	} else {
		bucket = u.GCSBucketLegacy
		contents = toLegacyFormat(mapping)
	}

	if u.OutDir != "" {
		if err := u.writeOutputFile(ctx, name, contents); err != nil {
			return err
		}
	}

	if bucket != nil {
		if err := upload(ctx, bucket.Object(name), contents); err != nil {
			return errors.Annotate(err, "failed to upload to GCS").Err()
		}
	}

	return nil
}

func (u *Updater) writeOutputFile(ctx context.Context, name string, data []byte) error {
	fullPath := filepath.Join(u.OutDir, name)
	if err := ioutil.WriteFile(fullPath, data, 0666); err != nil {
		return err
	}
	logging.Infof(ctx, "wrote %s", fullPath)
	return nil
}

func upload(ctx context.Context, obj *storage.ObjectHandle, data []byte) error {
	ctx, cancel := context.WithCancel(ctx)
	w := obj.NewWriter(ctx)
	defer func() {
		w.Close()
		cancel()
	}()

	w.PredefinedACL = "publicRead"
	if _, err := w.Write(data); err != nil {
		cancel() // cancel writing before closing
		return err
	}

	logging.Infof(ctx, "wrote gs://%s/%s", obj.BucketName(), obj.ObjectName())
	return nil
}
