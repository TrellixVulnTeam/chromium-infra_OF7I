// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"context"
	"testing"

	"infra/cros/recovery/internal/loader"
)

func TestDefaultConfig(t *testing.T) {
	ctx := context.Background()
	p, err := loader.LoadConfiguration(ctx, DefaultConfig())
	if err != nil {
		t.Errorf("expected to pass by failed with error: %s", err)
	}
	if p == nil {
		t.Errorf("default config is empty")
	}
}

func TestLabstationRepairConfig(t *testing.T) {
	ctx := context.Background()
	p, err := loader.LoadConfiguration(ctx, LabstationRepairConfig())
	if err != nil {
		t.Errorf("expected to pass by failed with error: %s", err)
	}
	if p == nil {
		t.Errorf("default config is empty")
	}
}

func TestLabstationDeployConfig(t *testing.T) {
	ctx := context.Background()
	p, err := loader.LoadConfiguration(ctx, LabstationDeployConfig())
	if err != nil {
		t.Errorf("expected to pass by failed with error: %s", err)
	}
	if p == nil {
		t.Errorf("default config is empty")
	}
}
