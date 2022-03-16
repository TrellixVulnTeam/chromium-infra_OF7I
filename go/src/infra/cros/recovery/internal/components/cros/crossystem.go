// Copyright 2021 The Chromium OS Authors. All rights reserved.  Use
// of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
)

// MatchCrossystemValueToExpectation reads value from crossystem and compared to expected value.
func MatchCrossystemValueToExpectation(ctx context.Context, run execs.Runner, subcommand string, expectedValue string) error {
	out, err := run(ctx, time.Minute, "crossystem", subcommand)
	if err != nil {
		return errors.Annotate(err, "match crossystem value to expectation: fail read %s", subcommand).Err()
	}
	actualValue := strings.TrimSpace(out)
	if actualValue != expectedValue {
		return errors.Reason("match crossystem value to expectation: %q, found: %q", expectedValue, actualValue).Err()
	}
	return nil
}

// UpdateCrossystem sets value of the subcommand to the value passed in.
//
// @params: check: bool value to check whether the crossystem command is being updated successfully.
func UpdateCrossystem(ctx context.Context, run execs.Runner, cmd string, val string, check bool) error {
	if _, err := run(ctx, time.Minute, fmt.Sprintf("crossystem %s=%s", cmd, val)); err != nil {
		return errors.Annotate(err, "update crossystem value").Err()
	}
	if check {
		return errors.Annotate(MatchCrossystemValueToExpectation(ctx, run, cmd, val), "update crossystem value").Err()
	}
	return nil
}
