// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/retry"
	"infra/cros/recovery/logger"
)

const (
	DefaultPingCount = 3
)

// IsPingable checks whether the resource is pingable
func IsPingable(ctx context.Context, count int, ping components.Pinger) error {
	err := ping(ctx, count)
	return errors.Annotate(err, "is pingable").Err()
}

// IsNotPingable checks whether the resource is not pingable
func IsNotPingable(ctx context.Context, count int, ping components.Pinger, log logger.Logger) error {
	if err := ping(ctx, count); err != nil {
		log.Debugf("Resource is not pingable, but expected!")
		return nil
	}
	return errors.Reason("not pingable: is pingable").Err()
}

// IsSSHable checks whether the resource is sshable
func IsSSHable(ctx context.Context, run components.Runner) error {
	_, err := run(ctx, time.Minute, "true")
	return errors.Annotate(err, "is sshable").Err()
}

const (
	PingRetryInteval = 5 * time.Second
	SSHRetryInteval  = 10 * time.Second
)

// WaitUntilPingable waiting resource to be pingable.
func WaitUntilPingable(ctx context.Context, waitTime, waitInterval time.Duration, countPerAttempt int, ping components.Pinger, log logger.Logger) error {
	log.Debugf("Start ping for the next %s.", waitTime)
	return retry.WithTimeout(ctx, waitInterval, waitTime, func() error {
		return IsPingable(ctx, countPerAttempt, ping)
	}, "wait to ping")
}

// WaitUntilNotPingable waiting resource to be not pingable.
func WaitUntilNotPingable(ctx context.Context, waitTime, waitInterval time.Duration, countPerAttempt int, ping components.Pinger, log logger.Logger) error {
	return retry.WithTimeout(ctx, waitInterval, waitTime, func() error {
		return IsNotPingable(ctx, countPerAttempt, ping, log)
	}, "wait to be not pingable")
}

// WaitUntilSSHable waiting resource to be sshable.
func WaitUntilSSHable(ctx context.Context, waitTime, waitInterval time.Duration, run components.Runner, log logger.Logger) error {
	log.Debugf("Start SSH check for the next %s.", waitTime)
	return retry.WithTimeout(ctx, waitInterval, waitTime, func() error {
		return IsSSHable(ctx, run)
	}, "wait to ssh access")
}
