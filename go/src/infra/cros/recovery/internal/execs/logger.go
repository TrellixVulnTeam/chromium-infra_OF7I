// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"infra/cros/recovery/logger"
)

// NewLogger returns logger.
func (ei *ExecInfo) NewLogger() logger.Logger {
	return ei.RunArgs.Logger
}
