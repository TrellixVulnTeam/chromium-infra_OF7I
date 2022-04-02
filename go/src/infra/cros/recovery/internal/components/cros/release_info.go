// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/logger"
)

const (
	releaseExtactValueGlob = "cat /etc/lsb-release | grep %s"
	releaseValueRegexpGlob = `%s=(\S+)`
	releaseBoardKey        = "CHROMEOS_RELEASE_BOARD"
	releaseTrackKey        = "CHROMEOS_RELEASE_TRACK"
)

// ExtractValueFromleaseInfo reads release info and extract value by provided key.
func ExtractValueFromleaseInfo(ctx context.Context, run components.Runner, log logger.Logger, key string) (string, error) {
	extactValueCommand := fmt.Sprintf(releaseExtactValueGlob, key)
	output, err := run(ctx, time.Minute, extactValueCommand)
	if err != nil {
		return "", errors.Annotate(err, "extract value from release info").Err()
	}
	valueRegexpCommand := fmt.Sprintf(releaseValueRegexpGlob, key)
	compiledRegexp, err := regexp.Compile(valueRegexpCommand)
	if err != nil {
		return "", errors.Annotate(err, "extract value from release info").Err()
	}
	matches := compiledRegexp.FindStringSubmatch(output)
	if len(matches) != 2 {
		return "", errors.Reason("extract value from release info: values is not found").Err()
	}
	value := matches[1]
	log.Debugf("Release info %q:%q", key, value)
	return value, nil
}

// ReleaseBoard reads release board info from lsb-release.
func ReleaseBoard(ctx context.Context, run components.Runner, log logger.Logger) (string, error) {
	board, err := ExtractValueFromleaseInfo(ctx, run, log, releaseBoardKey)
	if err != nil {
		return "", errors.Annotate(err, "release %q", releaseBoardKey).Err()
	}
	log.Debugf("Release %q: %q.", releaseBoardKey, board)
	return board, nil
}

// ReleaseTrack reads release track info from lsb-release.
func ReleaseTrack(ctx context.Context, run components.Runner, log logger.Logger) (string, error) {
	track, err := ExtractValueFromleaseInfo(ctx, run, log, releaseTrackKey)
	if err != nil {
		return "", errors.Annotate(err, "release %q", releaseTrackKey).Err()
	}
	log.Debugf("Release %q: %q.", releaseTrackKey, track)
	return track, nil
}
