// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package io

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
	"infra/libs/sshpool"
)

// These constants avoid magic numbers in the code. E.g. if we ever
// decide to change the default port etc, this will be the single
// place to change.
const (
	// the command use for creating as well as extracting the data
	// that is being copied
	tarCmd = "tar"

	// the port number to be used for creating SSH connections to
	// the remote device.
	defaultSSHPort = 22

	// permissions for use during potential destination directory
	// creation.
	dirPermission = os.FileMode(0755)
)

// CopyFileFrom copies a single file from remote device to local
// machine. req contains the complete path of the source file on the
// remote machine, and the complete path of the destination directory
// on the local machine where the source file will be copied. The
// destination path is just the directory name, and does not include
// the filename.
func CopyFileFrom(ctx context.Context, pool *sshpool.Pool, req *tlw.CopyRequest) error {
	if err := validateInputParams(ctx, pool, req); err != nil {
		return errors.Annotate(err, "copy file from").Err()
	}
	if err := ensureDirExists(ctx, req.PathDestination); err != nil {
		return errors.Annotate(err, "copy file from").Err()
	}

	addr := net.JoinHostPort(req.Resource, strconv.Itoa(defaultSSHPort))
	client, err := pool.Get(addr)
	if err != nil {
		return errors.Annotate(err, "copy file from: failed to get client for %q from pool", addr).Err()
	}
	defer pool.Put(addr, client)
	session, err := client.NewSession()
	if err != nil {
		return errors.Annotate(err, "copy file from: failed to create SSH session").Err()
	}
	defer session.Close()

	remoteSrc := req.PathSource
	remoteFileName := filepath.Base(remoteSrc)

	// On the remote device, read the input file and create a
	// compressed tar archive. Then write it to stdout. Here the
	// '-C' flag changes the current directory to the location of
	// the source file. This ensures that the tar archive includes
	// paths relative only to this directory.
	rCmd := fmt.Sprintf("%s -c --gzip -C %s %s", tarCmd, filepath.Dir(remoteSrc), remoteFileName)
	p, err := session.StdoutPipe()
	if err != nil {
		return errors.Annotate(err, "copy file from: error with obtaining the stdout pipe").Err()
	}
	if err := session.Start(rCmd); err != nil {
		return errors.Annotate(err, "copy file from: error with starting the remote command %q", rCmd).Err()
	}

	destFileName := filepath.Join(req.PathDestination, remoteFileName)
	log.Debug(ctx, "Copy file from: %q path to new file.", destFileName)
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Annotate(err, "copy file from: error with creating temporary dir %q", tmpDir).Err()
	}
	defer os.RemoveAll(tmpDir)

	// Read from stdin and extract the contents to tmpDir. Here,
	// the '-C' flag changes the working directory to tmpDir and
	// ensures that the output is placed there.
	lCmd := exec.CommandContext(ctx, tarCmd, "-x", "--gzip", "-C", tmpDir)
	lCmd.Stdin = p
	if err := lCmd.Run(); err != nil {
		return errors.Annotate(err, "copy file from: error with running the local command").Err()
	}
	var tmpLocalFile = filepath.Join(tmpDir, remoteFileName)
	if err := os.Rename(tmpLocalFile, destFileName); err != nil {
		return errors.Annotate(err, "copy file from: moving local file %q to %q failed", tmpLocalFile, destFileName).Err()
	}
	log.Debug(ctx, "Copy file from: successfully moved %q to %q.", tmpLocalFile, destFileName)
	return nil
}

func validateInputParams(ctx context.Context, pool *sshpool.Pool, req *tlw.CopyRequest) error {
	if pool == nil {
		return errors.New("validate input params: ssh pool is not initialized")
	} else if req.Resource == "" {
		return errors.New("validate input params: resource is empty")
	} else if req.PathSource == "" {
		return errors.New("validate input params: source path is empty")
	} else if req.PathDestination == "" {
		return errors.New("validate input params: destination path is empty")
	}
	log.Debug(ctx, "Source for transfer: %q.", req.PathSource)
	log.Debug(ctx, "Destination for transfer: %q.", req.PathDestination)
	log.Debug(ctx, "Resource: %q.", req.Resource)
	return nil
}

// ensureDirExists checks whether the directory 'd' exists, or not. If
// it does not exist, the function creates it. It returns any error
// encountered during checking or creating the directory.
func ensureDirExists(ctx context.Context, d string) error {
	s, err := os.Stat(d)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug(ctx, "Ensure directory exists: creating directory %q.", d)
			return os.MkdirAll(d, dirPermission)
		}
		// This means that 'err' is not known to report
		// whether or not the file or directory already
		// exists. Hence we cannot proceed with checking
		// whether the directory pre-exists, or creating
		// directory.
		return errors.Annotate(err, "ensure directory exists: cannot determine if %q exists", d).Err()
	}
	if s.IsDir() {
		log.Debug(ctx, "Ensure directory exists: directory %q already exists.", d)
		return nil
	}
	return fmt.Errorf("ensure directory exists: cannot create directory %q, it is a pre-existing file", d)
}
