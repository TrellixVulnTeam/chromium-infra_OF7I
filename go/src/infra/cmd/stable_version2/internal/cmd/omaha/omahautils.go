// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package omaha

import (
	"context"
	"io/ioutil"
	"path/filepath"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/common/logging"
	gslib "infra/cmd/stable_version2/internal/gs"

	svlib "infra/libs/cros/stableversion"
)

type gslibClient interface {
	Download(gs.Path, string) error
}

// versionOk checks whether a version string is a valid cros version.
func versionOk(s string) bool {
	return svlib.ValidateCrOSVersion(s) == nil
}

// versionCmp compares two cros versions.
// if either a or b is not a valid cros version, the
// behavior is undefined.
func versionCmp(a string, b string) int {
	cmp, err := svlib.CompareCrOSVersions(a, b)
	if err != nil {
		return 0
	}
	return cmp
}

// getGSFirmwareSV gets a list of firmware versions associated with
// some StableCrosVersions by board.
func getGSFirmwareSV(ctx context.Context, gsc gslibClient, outDir string, updatedCros []*sv.StableCrosVersion) ([]*sv.StableFirmwareVersion, error) {
	var res []*sv.StableFirmwareVersion
	for _, newCros := range updatedCros {
		lf := filepath.Join(outDir, localMetaFilePath(newCros))
		remotePath := gslib.MetaFilePath(newCros)
		if err := gsc.Download(remotePath, lf); err != nil {
			logging.Debugf(ctx, "fail to download %s: %s", remotePath, err)
			continue
		}
		bt, err := ioutil.ReadFile(lf)
		if err != nil {
			logging.Debugf(ctx, "fail to load meta file: %s", lf)
			continue
		}
		firmwareSVs, err := gslib.ParseMetadata(bt)
		if err != nil {
			logging.Debugf(ctx, "fail to parse meta file: %s", err)
			continue
		}
		for _, fsv := range firmwareSVs.FirmwareVersions {
			if fsv.GetVersion() != "" {
				res = append(res, fsv)
			}
		}
	}
	return res, nil
}
