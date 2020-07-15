// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package updater

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
)

// Updater computed metadata from a Chromium checkout and uploads it to GCS.
type Updater struct {
	// ChromiumCheckout is a path to chromium/src.git checkout.
	ChromiumCheckout string

	// Prod indicates whether to make production side effects.
	// If false, does not upload to GCS.
	Prod bool

	// OutDir is a path to the directory where to write output files
	// in addition to writing to uploading to the cloud.
	OutDir string
}

// Run updates the metadata stored in GCS.
func (u *Updater) Run(ctx context.Context) error {
	mapping, err := dirmd.ReadMapping(u.ChromiumCheckout, dirmdpb.MappingForm_FULL)
	if err != nil {
		return err
	}

	legacyData := toLegacyFormat(mapping)
	if u.OutDir != "" {
		if err := ioutil.WriteFile(filepath.Join(u.OutDir, "component_map_subdirs.json"), legacyData, 0666); err != nil {
			return err
		}
	}

	if u.Prod {
		// TODO(crbug.com/1104246): upload to GCS.
	}

	return nil
}
