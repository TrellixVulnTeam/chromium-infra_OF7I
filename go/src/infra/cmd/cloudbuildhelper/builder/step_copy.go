// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package builder

import (
	"context"
)

// runCopyBuildStep executes manifest.CopyBuildStep.
func runCopyBuildStep(ctx context.Context, inv *stepRunnerInv) error {
	return inv.addToOutput(ctx, inv.BuildStep.CopyBuildStep.Copy, inv.BuildStep.Dest)
}
