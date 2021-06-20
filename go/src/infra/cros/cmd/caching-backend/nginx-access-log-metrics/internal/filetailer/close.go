// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package filetailer

import (
	"golang.org/x/sys/unix"
)

func (t *Tailer) closeTailing() error {
	return t.cmd.Process.Signal(unix.SIGTERM)
}
