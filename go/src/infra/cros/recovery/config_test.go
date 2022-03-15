// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"testing"

	"infra/cros/recovery/config"
	"infra/cros/recovery/internal/execs"
)

// Testing plans here as that required access to execs and we try to avoid extra linking.

// verifyConfig verifies that configuration can be parsed and contains all execs present in library.
// Fail if:
// 1) Cannot parse by loader,
// 2) Missing dependency, condition or recovery action used in the actions,
// 3) Used unknown exec function.
func verifyConfig(name string, t *testing.T, c *config.Configuration) {
	ctx := context.Background()
	p, err := config.Load(ctx, mustCreateConfigJSON(c), execs.Exist)
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
	verifyConfig("labstation-repair", t, config.LabstationRepairConfig())
}

// TestLabstationDeployConfig verifies the labstation deploy configuration.
func TestLabstationDeployConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("labstation-deploy", t, config.LabstationDeployConfig())
}

// TestCrosRepairConfig verifies the cros repair configuration.
func TestCrosRepairConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("dut-repair", t, config.CrosRepairConfig())
}

// TestCrosDeployConfig verifies the cros deploy configuration.
func TestCrosDeployConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("dut-deploy", t, config.CrosDeployConfig())
}

func mustCreateConfigJSON(c *config.Configuration) io.Reader {
	b, err := json.Marshal(c)
	if err != nil {
		log.Fatalf("Failed to create JSON config: %v", err)
	}
	return bytes.NewBuffer(b)
}
