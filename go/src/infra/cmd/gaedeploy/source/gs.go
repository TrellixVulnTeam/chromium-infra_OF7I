// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package source

import (
	"context"
	"io"
	"os"
	"os/exec"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// gsSource implements Source using a `gsutil cp` invocation.
//
// Note that gaedeploy tool depends significantly on `gcloud` in PATH, which
// means `gsutil` is also in PATH (by being a part of Google Cloud SDK). By
// using `gsutil cp` (instead of Go level APIs) we ensure users will have to
// login only once (with `gcloud auth login`), instead of twice (with `gcloud`
// for `gcloud app deploy` and luci-auth for Go APIs).
type gsSource struct {
	path   string // has form "gs://..."
	sha256 []byte
}

func (gs *gsSource) SHA256() []byte {
	return gs.sha256
}

func (gs *gsSource) Open(ctx context.Context, tmp string) (io.ReadCloser, error) {
	logging.Infof(ctx, "Running gsutil cp %q %q...", gs.path, tmp)

	cmd := exec.CommandContext(ctx, "gsutil", "cp", gs.path, tmp)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, errors.Annotate(err, "gsutil call failed").Err()
	}

	return os.Open(tmp)
}
