// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package env

import "os"

// RunningOnBot checks whether or not it is running on a bot, by way of checking
// the USER env var.
func RunningOnBot() bool {
	return os.Getenv("USER") == "chrome-bot"
}
