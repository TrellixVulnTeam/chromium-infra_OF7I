// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package event

import "os"

func notifyOnAbort(c chan os.Signal) {
	panic("not supported on windows")
}
