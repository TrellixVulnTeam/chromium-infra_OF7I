// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package builder

import (
	"context"
	"os"
	"os/exec"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// runRunBuildStep executes manifest.RunBuildStep.
func runRunBuildStep(ctx context.Context, inv *stepRunnerInv) error {
	logging.Infof(ctx, "Running %q in %q", inv.BuildStep.Run, inv.BuildStep.Cwd)
	cmd := exec.CommandContext(ctx, inv.BuildStep.Run[0], inv.BuildStep.Run[1:]...)
	cmd.Dir = inv.BuildStep.Cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Annotate(err, "`run` step failed").Err()
	}

	// "Pick up" newly generated files in the context directory.
	for _, out := range inv.BuildStep.Outputs {
		if err := inv.addFilesToOutput(ctx, out, out, nil); err != nil {
			return err
		}
	}

	return nil
}
