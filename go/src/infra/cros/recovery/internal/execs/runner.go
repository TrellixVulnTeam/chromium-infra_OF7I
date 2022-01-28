// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

var (
	// 127: linux command line error of command not found.
	SSHErrorCLINotFound = errors.BoolTag{Key: errors.NewTagKey("ssh_error_cli_not_found")}

	// 124: linux command line error of command timeout.
	SSHErrorLinuxTimeout = errors.BoolTag{Key: errors.NewTagKey("linux_timeout")}

	// other linux error tag.
	GeneralError = errors.BoolTag{Key: errors.NewTagKey("general_error")}

	// internal error tag.
	SSHErrorInternal = errors.BoolTag{Key: errors.NewTagKey("ssh_error_internal")}

	// -1: fail to create ssh session.
	FailToCreateSSHErrorInternal = errors.BoolTag{Key: errors.NewTagKey("fail_to_create_ssh_error_internal")}

	// -2: session is down, but the server sends no confirmation of the exit status.
	NoExitStatusErrorInternal = errors.BoolTag{Key: errors.NewTagKey("no_exit_status_error_internal")}

	// other internal error tag.
	OtherErrorInternal = errors.BoolTag{Key: errors.NewTagKey("other_error_internal")}
)

// Runner defines the type for a function that will execute a command
// on a host, and returns the result as a single line.
type Runner func(context.Context, time.Duration, string, ...string) (string, error)

// NewRunner returns a function of type Runner that executes a command
// on a host and returns the results as a single line. This function
// defines the specific host on which the command will be
// executed. Examples of such specific hosts can be the DUT, or the
// servo-host etc.
func (a *RunArgs) NewRunner(host string) Runner {
	runner := func(ctx context.Context, timeout time.Duration, cmd string, args ...string) (string, error) {
		log.Debug(ctx, "Run command %q", cmd)
		r := a.Access.Run(ctx, &tlw.RunRequest{
			Resource: host,
			Timeout:  durationpb.New(timeout),
			Command:  cmd,
			Args:     args,
		})
		exitCode := r.ExitCode
		if exitCode != 0 {
			errAnnotator := errors.Reason("runner: command %q completed with exit code %d", cmd, r.ExitCode)
			// different kinds of internal errors
			if exitCode < 0 {
				errAnnotator.Tag(SSHErrorInternal)
				if exitCode == -1 {
					errAnnotator.Tag(FailToCreateSSHErrorInternal)
				} else if exitCode == -2 {
					errAnnotator.Tag(NoExitStatusErrorInternal)
				} else if exitCode == -3 {
					errAnnotator.Tag(OtherErrorInternal)
				}
				// general linux errors
			} else if exitCode == 124 {
				errAnnotator.Tag(SSHErrorLinuxTimeout)
			} else if exitCode == 127 {
				errAnnotator.Tag(SSHErrorCLINotFound)
			} else {
				errAnnotator.Tag(GeneralError)
			}
			return "", errAnnotator.Err()
		}
		out := strings.TrimSpace(r.Stdout)
		log.Debug(ctx, "Run output:\n%s", out)
		return out, nil
	}
	return runner
}
