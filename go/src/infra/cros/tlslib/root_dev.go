// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlslib provides the canonical implementation of a common TLS server.
package tlslib

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

// rootDev holds root device related information.
type rootDev struct {
	disk      string
	partDelim string
	partNum   string
}

var rePartitionNumber = regexp.MustCompile(`.*([0-9]+)`)

func getRootDev(c *ssh.Client) (rootDev, error) {
	var r rootDev
	// Example 1: "/dev/nvme0n1p3"
	// Example 2: "/dev/sda3"
	curRoot, err := runCmdOutput(c, "rootdev -s")
	if err != nil {
		return r, fmt.Errorf("get root device: failed to get current root, %s", err)
	}
	curRoot = strings.TrimSpace(curRoot)

	// Example 1: "/dev/nvme0n1"
	// Example 2: "/dev/sda"
	rootDisk, err := runCmdOutput(c, "rootdev -s -d")
	if err != nil {
		return r, fmt.Errorf("get root device: failed to get root disk, %s", err)
	}
	r.disk = strings.TrimSpace(rootDisk)

	// Handle /dev/mmcblk0pX, /dev/sdaX, etc style partitions.
	// Example 1: "3"
	// Example 2: "3"
	match := rePartitionNumber.FindStringSubmatch(curRoot)
	if match == nil {
		return r, fmt.Errorf("get root device: failed to match partition number from %s", curRoot)
	}
	r.partNum = match[1]
	switch r.partNum {
	case partitionNumRootA, partitionNumRootB:
		break
	default:
		return r, fmt.Errorf("get root device: invalid partition number %s", r.partNum)
	}

	// Example 1: "p3"
	// Example 2: "3"
	rootPartNumWithDelim := strings.TrimPrefix(curRoot, r.disk)

	// Example 1: "p"
	// Example 2: ""
	r.partDelim = strings.TrimSuffix(rootPartNumWithDelim, r.partNum)

	return r, nil
}
