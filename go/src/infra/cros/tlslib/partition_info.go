// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlslib provides the canonical implementation of a common TLS server.
package tlslib

import (
	"fmt"
)

const (
	partitionNumKernelA = "2"
	partitionNumKernelB = "4"
	partitionNumRootA   = "3"
	partitionNumRootB   = "5"
	partitionNumMiniOSA = "9"
	partitionNumMiniOSB = "10"
)

// partitionsInfo holds active/inactive root + kernel partition information.
type partitionInfo struct {
	// The active + inactive kernel device partitions (e.g. /dev/nvme0n1p2).
	activeKernel      string
	activeKernelNum   string
	inactiveKernel    string
	inactiveKernelNum string
	// The active + inactive root device partitions (e.g. /dev/nvme0n1p3).
	activeRoot   string
	inactiveRoot string
	// The A + B miniOS device partitions.
	miniOSA string
	miniOSB string
}

func getPartitionInfo(r rootDev) partitionInfo {
	// Determine the next kernel and root.
	rootDiskPartDelim := r.disk + r.partDelim
	switch r.partNum {
	case partitionNumRootA:
		return partitionInfo{
			activeKernel:      rootDiskPartDelim + partitionNumKernelA,
			activeKernelNum:   partitionNumKernelA,
			inactiveKernel:    rootDiskPartDelim + partitionNumKernelB,
			inactiveKernelNum: partitionNumKernelB,
			activeRoot:        rootDiskPartDelim + partitionNumRootA,
			inactiveRoot:      rootDiskPartDelim + partitionNumRootB,
			miniOSA:           rootDiskPartDelim + partitionNumMiniOSA,
			miniOSB:           rootDiskPartDelim + partitionNumMiniOSB,
		}
	case partitionNumRootB:
		return partitionInfo{
			activeKernel:      rootDiskPartDelim + partitionNumKernelB,
			activeKernelNum:   partitionNumKernelB,
			inactiveKernel:    rootDiskPartDelim + partitionNumKernelA,
			inactiveKernelNum: partitionNumKernelA,
			activeRoot:        rootDiskPartDelim + partitionNumRootB,
			inactiveRoot:      rootDiskPartDelim + partitionNumRootA,
			miniOSA:           rootDiskPartDelim + partitionNumMiniOSA,
			miniOSB:           rootDiskPartDelim + partitionNumMiniOSB,
		}
	default:
		panic(fmt.Sprintf("Unexpected root partition number of %s", r.partNum))
	}
}
