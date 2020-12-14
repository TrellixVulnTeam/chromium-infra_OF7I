// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlslib provides the canonical implementation of a common TLS server.
package tlslib

import (
	"fmt"
	"log"
	"path"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"golang.org/x/crypto/ssh"
)

type dlcSlot string

const (
	dlcSlotA dlcSlot = "dlc_a"
	dlcSlotB dlcSlot = "dlc_b"
)

const (
	dlcCacheDir    = "/var/cache/dlc"
	dlcImage       = "dlc.img"
	dlcLibDir      = "/var/lib/dlcservice/dlc"
	dlcPackage     = "package"
	dlcVerified    = "verified"
	dlcserviceUtil = "dlcservice_util"
)

func getActiveDLCSlot(r rootDev) dlcSlot {
	switch r.partNum {
	case partitionNumRootA:
		return dlcSlotA
	case partitionNumRootB:
		return dlcSlotB
	default:
		panic(fmt.Sprintf("Invalid partition number %s", r.partNum))
	}
}

func getInactiveDLCSlot(r rootDev) dlcSlot {
	switch slot := getActiveDLCSlot(r); slot {
	case dlcSlotA:
		return dlcSlotB
	case dlcSlotB:
		return dlcSlotA
	default:
		panic(fmt.Sprintf("Invalid DLC slot %s", slot))
	}
}

func isDLCVerified(c *ssh.Client, spec *tls.ProvisionDutRequest_DLCSpec, slot dlcSlot) (bool, error) {
	dlcID := spec.GetId()
	verified, err := pathExists(c, path.Join(dlcLibDir, dlcID, string(slot), dlcVerified))
	if err != nil {
		return false, fmt.Errorf("is DLC verfied: failed to check if DLC %s is verified, %s", dlcID, err)
	}
	return verified, nil
}

// clearInactiveDLCVerifiedMarks will clear the verified marks for all DLCs in the inactive slots.
func clearInactiveDLCVerifiedMarks(c *ssh.Client, r rootDev) error {
	// Stop dlcservice daemon in order to not interfere with clearing inactive verified DLCs.
	if err := runCmd(c, "stop dlcservice"); err != nil {
		log.Printf("clear inactive verified DLC marks: failed to stop dlcservice daemon, %s", err)
	}
	defer func() {
		if err := runCmd(c, "start dlcservice"); err != nil {
			log.Printf("clear inactive verified DLC marks: failed to start dlcservice daemon, %s", err)
		}
	}()

	inactiveSlot := getInactiveDLCSlot(r)
	err := runCmd(c, fmt.Sprintf("rm -f %s", path.Join(dlcCacheDir, "*", "*", string(inactiveSlot), dlcVerified)))
	if err != nil {
		return fmt.Errorf("clear inactive verified DLC marks: failed remove inactive verified DLCs, %s", err)
	}

	return nil
}
