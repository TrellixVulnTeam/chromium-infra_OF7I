// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// kernelInfo holds info about kernel and root partitions.
type kernelInfo struct {
	name            string
	kernelPartition int
	rootPartition   int
}

var (
	// ChromeOS devices has two kernels and separate root partitions to boot.
	kernelA = &kernelInfo{name: "KERN-A", kernelPartition: 2, rootPartition: 3}
	kernelB = &kernelInfo{name: "KERN-B", kernelPartition: 4, rootPartition: 5}
)

// kernelPriorityChangePattern is the leading 3 or 5 in the output of rootdev -s -d.
var kernelPriorityChangePattern = regexp.MustCompile(`(\d)`)

// IsKernelPriorityChanged check if kernel priority changed and is waiting for reboot to apply the change.
func IsKernelPriorityChanged(ctx context.Context, run execs.Runner) (bool, error) {
	// Determine if we have an update that pending on reboot by check if
	// the current inactive kernel has priority for the next boot.
	// Check which partition is set for the next boot. If that is not active Kernel then system expect reboot.
	diskBlockResult, err := run(ctx, time.Minute, "rootdev -s -d")
	if err != nil {
		return false, errors.Annotate(err, "is kernel priority changed").Err()
	}
	log.Debugf(ctx, "Booted disk block: %q.", diskBlockResult)
	// Get the name of root partition on the resource.
	diskRoot, err := run(ctx, time.Minute, "rootdev -s")
	if err != nil {
		return false, errors.Annotate(err, "is kernel priority changed").Err()
	}
	log.Debugf(ctx, "Booted root disk: %q.", diskRoot)
	diskSuffix := strings.TrimPrefix(diskRoot, diskBlockResult)
	// Find first number. We expected number 3 or 5.
	parts := kernelPriorityChangePattern.FindStringSubmatch(diskSuffix)
	if len(parts) < 2 || parts[1] == "" {
		return false, errors.Reason("is kernel priority changed: fail to read value from %s", diskSuffix).Err()
	}
	activeRootPartition, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return false, errors.Annotate(err, "is kernel priority changed: fail extract root partition number for %q", diskSuffix).Err()
	}
	log.Debugf(ctx, "Booted root partion: %d.", activeRootPartition)
	var activeKernel, nextKernel *kernelInfo
	if kernelA.rootPartition == int(activeRootPartition) {
		activeKernel, nextKernel = kernelA, kernelB
	} else if kernelB.rootPartition == int(activeRootPartition) {
		activeKernel, nextKernel = kernelB, kernelA
	} else {
		return false, errors.Reason("is kernel priority changed: fail found kernel for root partition %q", diskRoot).Err()
	}
	log.Debugf(ctx, "Active kernel:%s , partition %d.", activeKernel.name, activeKernel.kernelPartition)
	log.Debugf(ctx, "Next kernel:%s , partition %d.", nextKernel.name, nextKernel.kernelPartition)
	// Help function to read boot priority for kernel.
	getKernelBootPriority := func(k *kernelInfo) (int, error) {
		v, kErr := run(ctx, time.Minute, fmt.Sprintf("cgpt show -n -i %d -P %s", k.kernelPartition, diskBlockResult))
		if kErr != nil {
			return 0, errors.Annotate(err, "kernel boot priority %q", k.name).Err()
		}
		p, kErr := strconv.ParseInt(v, 10, 32)
		if kErr != nil {
			return 0, errors.Annotate(err, "kernel boot priority %q: parse %q", k.name, v).Err()
		}
		return int(p), nil
	}
	kap, err := getKernelBootPriority(kernelA)
	if err != nil {
		return false, errors.Annotate(err, "is kernel priority changed").Err()
	}
	log.Debugf(ctx, "Kernel %q has priority %d.", kernelA.name, kap)
	kbp, err := getKernelBootPriority(kernelB)
	if err != nil {
		return false, errors.Annotate(err, "is kernel priority changed").Err()
	}
	log.Debugf(ctx, "Kernel %q has priority %d.", kernelB.name, kbp)
	// The kernel with higher priority is next kernel to boot.
	// If kernel with higher priority is not equal active kernel then next boot
	// kernel will be changed.
	if kap > kbp {
		return activeKernel != kernelA, nil
	}
	return activeKernel != kernelB, nil
}

const bootIDFile = "/proc/sys/kernel/random/boot_id"

// KernelBootId extracts and return unique ID associated with the current boot.
//
// If returns the same value then reboot was not performed.
func KernelBootId(ctx context.Context, run execs.Runner) (string, error) {
	noIdMsg := "no boot_id available"
	cmd := fmt.Sprintf("if [ -f %s ]; then cat %s; else echo %q; fi", bootIDFile, bootIDFile, noIdMsg)
	v, err := run(ctx, time.Minute, cmd)
	if err != nil {
		return "", errors.Annotate(err, "kernel boot id").Err()
	}
	if v == noIdMsg {
		return "", nil
	}
	return v, nil
}
