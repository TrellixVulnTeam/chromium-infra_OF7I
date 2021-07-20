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
		t.Errorf("default config is not working")
	}
	if p == nil {
		t.Errorf("default config is empty")
	}
	// TODO(otabek@): Add other config verifications.
}
