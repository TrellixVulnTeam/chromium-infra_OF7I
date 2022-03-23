// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// The path to get the value of the Flag 2
	VPD_CACHE = `/mnt/stateful_partition/unencrypted/cache/vpd/full-v2.txt`
)

// isEnrollmentInCleanState confirms that the device's enrollment state is clean
//
// Verify that the device's enrollment state is clean.
//
// There are two "flags" that generate 3 possible enrollment states here.
// Flag 1 - The presence of install attributes file in
//          /home/.shadow/install_attributes.pb
//
// Flag 2 - The value of "check_enrollment" from VPD. Can be obtained by
//          reading the cache file in
//          /mnt/stateful_partition/unencrypted/cache/vpd/full-v2.txt
//
// The states:
// State 1 - Device is enrolled, means flag 1 is true and in flag 2 check_enrollment=1
// State 2 - Device is consumer owned, means flag 1 is true and in flag 2 check_enrollment=0
// State 3 - Device is enrolled and has been powerwashed, means flag 1 is
//           false. If the value in flag 2 is check_enrollment=1 then the
//           device will perform forced re-enrollment check and depending
//           on the response from the server might force the device to enroll
//           again. If the value is check_enrollment=0, then device can be
//           used like a new device.
//
// We consider state 1, and first scenario(check_enrollment=1) of state 3
// as unacceptable state here as they may interfere with normal tests.
func isEnrollmentInCleanStateExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	command := fmt.Sprintf(`grep "check_enrollment" %s`, VPD_CACHE)
	result, err := run(ctx, time.Minute, command)
	if err == nil {
		log.Debugf(ctx, "Enrollment state in VPD cache: %s", result)
		if result != `"check_enrollment"="0"` {
			return errors.Reason("enrollment in clean state: failed, The device is enrolled, it may interfere with some tests").Err()
		}
		return nil
	}
	// In any case it returns a non zero value, it means we can't verify enrollment state, but we cannot say the device is enrolled
	// Only trigger the enrollment in clean state when we can confirm the device is enrolled.
	log.Errorf(ctx, "Unexpected error occurred during verify enrollment state in VPD cache, skipping verify process.")
	return nil
}

func init() {
	execs.Register("cros_is_enrollment_in_clean_state", isEnrollmentInCleanStateExec)
}
