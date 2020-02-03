// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build darwin linux

package event

import (
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

func notifyOnAbort(c chan os.Signal) {
	signal.Notify(c, unix.SIGTERM, unix.SIGINT)
}
