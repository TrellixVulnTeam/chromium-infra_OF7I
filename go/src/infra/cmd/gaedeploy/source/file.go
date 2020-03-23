// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package source

import (
	"context"
	"io"
	"os"
)

// fileSource implements Source using a local file.
type fileSource struct {
	path   string
	sha256 []byte
}

func (fs *fileSource) SHA256() []byte {
	return fs.sha256
}

func (fs *fileSource) Open(ctx context.Context, tmp string) (io.ReadCloser, error) {
	return os.Open(fs.path)
}
