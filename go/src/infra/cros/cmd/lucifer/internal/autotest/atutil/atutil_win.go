// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build windows

package atutil

import (
	"context"
	"io"

	"infra/cros/cmd/lucifer/internal/autotest"
)

func runTask(ctx context.Context, c autotest.Config, a *autotest.AutoservArgs, w io.Writer) (*Result, error) {
	panic("not implemented on windows")
}
