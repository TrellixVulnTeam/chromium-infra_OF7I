// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
)

const gsBucket = "cros-lab-inventory.appspot.com"
const scanLogPath = "assetScanLogs"

func upload(sc gs.Client, localFilePath string) error {
	p := gs.Path(fmt.Sprintf("gs://%s/%s/%s", gsBucket, scanLogPath, filepath.Base(localFilePath)))
	wr, err := sc.NewWriter(p)
	if err != nil {
		return err
	}
	logReader, err := os.Open(localFilePath)
	defer logReader.Close()
	if err != nil {
		return err

	}
	if _, err := io.Copy(wr, logReader); err != nil {
		return errors.Annotate(err, "upload %s to %s", localFilePath, gsBucket).Err()
	}
	if err := wr.Close(); err != nil {
		return errors.Annotate(err, "failed to finalize the upload").Err()
	}
	return nil
}
