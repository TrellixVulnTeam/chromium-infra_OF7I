// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

// These package imports register exec functions for the execs package.
import (
	_ "infra/cros/recovery/internal/execs/btpeer"
	_ "infra/cros/recovery/internal/execs/chameleon"
	_ "infra/cros/recovery/internal/execs/cros"
	_ "infra/cros/recovery/internal/execs/dut"
	_ "infra/cros/recovery/internal/execs/metrics"
	_ "infra/cros/recovery/internal/execs/rpm"
	_ "infra/cros/recovery/internal/execs/servo"
	_ "infra/cros/recovery/internal/execs/stableversion"
	_ "infra/cros/recovery/internal/execs/wifirouter"
)
