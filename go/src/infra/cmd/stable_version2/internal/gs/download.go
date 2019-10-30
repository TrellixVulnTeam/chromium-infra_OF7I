// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gs

import (
	"io"
	"os"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
)

// Download objects specified by a path from Google Storage to local.
func (gsc *Client) Download(gsPath gs.Path, localPath string) error {
	r, err := gsc.C.NewReader(gsPath, 0, -1)
	if err != nil {
		return errors.Annotate(err, "download").Err()
	}
	w, err := os.Create(localPath)
	if err != nil {
		return errors.Annotate(err, "download").Err()
	}
	if _, err := io.Copy(w, r); err != nil {
		return errors.Annotate(err, "download %s to %s", gsPath, localPath).Err()
	}
	return nil
}
