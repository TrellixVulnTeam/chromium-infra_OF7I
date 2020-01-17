// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package querygs

import (
	"fmt"

	"go.chromium.org/luci/common/gcloud/gs"

	gslib "infra/cmd/stable_version2/internal/gs"
)

type downloader func(gsPath gs.Path) ([]byte, error)

// Reader reads metadata.json files from google storage and caches the result.
type Reader struct {
	dld   downloader
	cache map[string]map[string]string
}

// maybeDownloadFile fetches a metadata.json corresponding to a buildTarget and version if it doesn't already exist in the cache.
func (r *Reader) maybeDownloadFile(buildTarget string, crosVersion string) error {
	if r.cache == nil {
		r.cache = make(map[string]map[string]string)
	}
	if _, ok := r.cache[buildTarget]; ok {
		return nil
	}
	// TODO(gregorynisbet): extend gslib with function to get path
	remotePath := gs.Path(fmt.Sprintf("gs://chromeos-image-archive/%s-release/%s/metadata.json", buildTarget, crosVersion))
	contents, err := (r.dld)(remotePath)
	if err != nil {
		return fmt.Errorf("Reader::maybeDownloadFile: fetching file: %s", err)
	}
	fws, err := gslib.ParseMetadata(contents)
	if err != nil {
		return fmt.Errorf("Reader::maybeDownloadFile: parsing metadata.json: %s", err)
	}
	// TODO(gregorynisbet): Consider throwing an error or panicking if we encounter
	// a duplicate when populating the cache.
	for _, fw := range fws {
		buildTarget := fw.GetKey().GetBuildTarget().GetName()
		model := fw.GetKey().GetModelId().GetValue()
		version := fw.GetVersion()
		if _, ok := r.cache[buildTarget]; !ok {
			r.cache[buildTarget] = make(map[string]string)
		}
		r.cache[buildTarget][model] = version
	}
	return nil
}
