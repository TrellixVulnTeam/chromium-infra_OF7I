// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"regexp"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// Command to extract release builder path from device.
	extactReleaseBuilderPathCommand = "cat /etc/lsb-release | grep CHROMEOS_RELEASE_BUILDER_PATH"
)

// releaseBuildPath reads release build path from lsb-release.
func releaseBuildPath(ctx context.Context, resource string, args *execs.RunArgs) (string, error) {
	// lsb-release is set of key=value so we need extract right part from it.
	//  Example: CHROMEOS_RELEASE_BUILDER_PATH=board-release/R99-9999.99.99
	r := args.Access.Run(ctx, resource, extactReleaseBuilderPathCommand)
	if r.ExitCode != 0 {
		return "", errors.Reason("release build path: fail. exit:%d, %s", r.ExitCode, r.Stderr).Err()
	}
	log.Debug(ctx, "Read value: %q.", r.Stdout)
	p, err := regexp.Compile("CHROMEOS_RELEASE_BUILDER_PATH=([\\w\\W]*)")
	if err != nil {
		return "", errors.Annotate(err, "release build path").Err()
	}
	parts := p.FindStringSubmatch(r.Stdout)
	if len(parts) < 2 {
		return "", errors.Reason("release build path: fail to read value from %s", r.Stdout).Err()
	}
	return parts[1], nil
}
