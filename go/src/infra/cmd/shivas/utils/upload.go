// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"io"
	"os"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
)

const gsBucket = "cros-lab-inventory.appspot.com"
const scanLogPath = "assetScanLogs"

func upload(sc gs.Client, localFilePath string, remoteFilePath string) error {
	wr, err := sc.NewWriter(gs.Path(remoteFilePath))
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
