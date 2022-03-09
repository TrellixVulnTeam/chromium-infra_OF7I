// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"context"
	"infra/cros/recovery/internal/loader"
	"io"
	"testing"
)

// verifyConfig verifies that configuration can be parsed and contains all execs present in library.
// Fail if:
// 1) Cannot parse by loader,
// 2) Missing dependency, condition or recovery action used in the actions,
// 3) Used unknown exec function.
func verifyConfig(name string, t *testing.T, c io.Reader) {
	ctx := context.Background()
	p, err := loader.LoadConfiguration(ctx, c)
	if err != nil {
		t.Errorf("%q expected to pass but failed with error: %s", name, err)
	}
	if p == nil {
		t.Errorf("%q default config is empty", name)
	}
}

// TestLabstationRepairConfig verifies the labstation repair configuration.
func TestLabstationRepairConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("labstation-repair", t, LabstationRepairConfig())
}

// TestLabstationDeployConfig verifies the labstation deploy configuration.
func TestLabstationDeployConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("labstation-deploy", t, LabstationDeployConfig())
}

// TestCrosRepairConfig verifies the cros repair configuration.
func TestCrosRepairConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("dut-repair", t, CrosRepairConfig())
}

// TestCrosDeployConfig verifies the cros deploy configuration.
func TestCrosDeployConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("dut-deploy", t, CrosDeployConfig())
}
