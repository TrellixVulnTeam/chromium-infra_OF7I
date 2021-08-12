// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package io

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

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

// CopyDirectoryFrom copies a single directory, and all the files
// contained within it on the remote machine to the local machine. req
// contains the complete path of the source directory on the remote
// machine, and the complete path of the destination directory on the
// local machine where the source directory will be copied. The
// destination path is the directory within which the source directory
// will be copied.
func CopyDirectoryFrom(ctx context.Context, pool *sshpool.Pool, req *tlw.CopyRequest) error {
	if err := copyFromHelper(ctx, pool, req, true); err != nil {
		return errors.Annotate(err, "copy directory from").Err()
	}
	return nil
}

// CopyFileFrom copies a single file from remote device to local
// machine. req contains the complete path of the source file on the
// remote machine, and the complete path of the destination directory
// on the local machine where the source file will be copied. The
// destination path is just the directory name, and does not include
// the filename.
func CopyFileFrom(ctx context.Context, pool *sshpool.Pool, req *tlw.CopyRequest) error {
	if err := copyFromHelper(ctx, pool, req, false); err != nil {
		return errors.Annotate(err, "copy file from").Err()
	}
	return nil
}

// CopyFileTo copies a single file from local machine to remote
// device. req contains the complete path of the source file on the
// local machine, and the complete path of the destination directory
// on the remote device where the source file will be copied.
func CopyFileTo(ctx context.Context, pool *sshpool.Pool, req *tlw.CopyRequest) error {
	if err := validateInputParams(ctx, pool, req); err != nil {
		return errors.Annotate(err, "copy file to").Err()
	}
	if err := checkFileExists(ctx, req.PathSource); err != nil {
		return errors.Annotate(err, "copy file to: error while checking whether the source file exists").Err()
	}

	addr := net.JoinHostPort(req.Resource, strconv.Itoa(defaultSSHPort))
	client, err := pool.Get(addr)
	if err != nil {
		return errors.Annotate(err, "copy file to: failed to get client %q from pool", addr).Err()
	}
	defer pool.Put(addr, client)
	session, err := client.NewSession()
	if err != nil {
		return errors.Annotate(err, "copy file to: failed to create SSH session").Err()
	}
	defer session.Close()

	// Read the input path on the local machine and create a
	// compressed tar archive. Then write it to stdout. Here, the '-C'
	// flag changes the working directory to the location where the
	// input exists. This ensures that the archive includes paths only
	// relative to this directory.
	lCmd := exec.CommandContext(ctx, tarCmd, "-c", "--gzip", "-C", filepath.Dir(req.PathSource), filepath.Base(req.PathSource))
	p, err := lCmd.StdoutPipe()
	if err != nil {
		return errors.Annotate(err, "copy file to: could not obtain the stdout pipe").Err()
	}
	if err := lCmd.Start(); err != nil {
		return errors.Annotate(err, "copy file to: could not execute local command %q", lCmd).Err()
	}
	defer lCmd.Wait()
	p2, err2 := session.StdinPipe()
	if err2 != nil {
		return errors.Annotate(err, "copy file to: error with obtaining stdin pipe for the SSH Session").Err()
	}
	uploadErrors := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	// the tar-archive that was created above has been written to the
	// stdout of the process on local machine. Now, we copy this to
	// the stdin of the ssh session so that the tar extraction process
	// on the remote machine can read the archive off its stdin and
	// extract it to the file system on the remote machine.
	go func(wg1 *sync.WaitGroup) {
		defer wg1.Done()
		if _, err := io.Copy(p2, p); err != nil {
			uploadErrors <- errors.Annotate(err, "copy file to: error with copying contents from local stdout to remote stdin").Err()
		}
		defer p2.Close()
	}(&wg)

	// Read the stdin on the remote device and extract to the
	// destination path. The '-C' flag changes the current directory
	// to the destination path, and ensures that the output is placed
	// there.
	rCmd := fmt.Sprintf("%s -x --gzip -C %s", tarCmd, req.PathDestination)
	wg.Add(1)
	go func(wg2 *sync.WaitGroup) {
		defer wg2.Done()
		if err := session.Start(rCmd); err != nil {
			uploadErrors <- errors.Annotate(err, "copy file to: remote device could not read the uploaded contents").Err()
		} else if err := session.Wait(); err != nil {
			uploadErrors <- errors.Annotate(err, "copy file to: remote command did not exit cleanly").Err()
		}
	}(&wg)
	wg.Wait()

	select {
	case e, ok := <-uploadErrors:
		if ok {
			return errors.Annotate(e, "copy file to").Err()
		} else {
			// No one is closing the channel, but we want
			// to defensively handle this case.
			return nil
		}
	default:
		return nil
	}
}

// copyFromHelper copies contents of the remote source path to a local
// destination path. req contains the complete path of the source on
// the remote machine that needs to be copied to destination on the
// local machine. The function can handle both, a single file, as well
// as a single directory, as the source.
func copyFromHelper(ctx context.Context, pool *sshpool.Pool, req *tlw.CopyRequest, isDir bool) error {
	if err := validateInputParams(ctx, pool, req); err != nil {
		return errors.Annotate(err, "copy from helper").Err()
	}
	if err := ensureDirExists(ctx, req.PathDestination, true); err != nil {
		return errors.Annotate(err, "copy from helper").Err()
	}

	addr := net.JoinHostPort(req.Resource, strconv.Itoa(defaultSSHPort))
	client, err := pool.Get(addr)
	if err != nil {
		return errors.Annotate(err, "copy from helper: failed to get client for %q from pool", addr).Err()
	}
	defer pool.Put(addr, client)
	session, err := client.NewSession()
	if err != nil {
		return errors.Annotate(err, "copy from helper: failed to create SSH session").Err()
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
		return errors.Annotate(err, "copy from helper: error with obtaining the stdout pipe").Err()
	}
	if err := session.Start(rCmd); err != nil {
		return errors.Annotate(err, "copy from helper: error with starting the remote command %q", rCmd).Err()
	}

	destFileName := filepath.Join(req.PathDestination, remoteFileName)
	log.Debug(ctx, "copy from helper: %q path to new file.", destFileName)
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Annotate(err, "copy from helper: error with creating temporary dir %q", tmpDir).Err()
	}
	defer os.RemoveAll(tmpDir)

	// Read from stdin and extract the contents to tmpDir. Here,
	// the '-C' flag changes the working directory to tmpDir and
	// ensures that the output is placed there.
	lCmd := exec.CommandContext(ctx, tarCmd, "-x", "--gzip", "-C", tmpDir)
	lCmd.Stdin = p
	if err := lCmd.Run(); err != nil {
		return errors.Annotate(err, "copy from helper: error with running the local command").Err()
	}
	var tmpLocalFile = filepath.Join(tmpDir, remoteFileName)

	// The source for this copy operation resides on the remote
	// machine. We cannot examine this source on the remote machine to
	// confirm whether it is a directory or a file. However, after the
	// copy operations has been completed, we will now verify that the
	// source for this copy operation is of the appropriate type.

	// If source is a file, while it is expected to be a directory,
	// delete the temporary files and raise an error.
	if isDir && checkFileExists(ctx, tmpLocalFile) == nil {
		if err := os.RemoveAll(tmpLocalFile); err != nil {
			return errors.Annotate(err, "copy from helper: expected a directory but found a file, error while removing temporary file %q", tmpLocalFile).Err()
		} else {
			return errors.Reason("copy from helper: expected a directory, but found a file %q", remoteFileName).Err()
		}
	}
	// If source is a directory, while it is expected to be a file,
	// delete the temporary files and raise an error.
	if !isDir && ensureDirExists(ctx, tmpLocalFile, false) == nil {
		if err := os.RemoveAll(tmpLocalFile); err != nil {
			return errors.Annotate(err, "copy from helper: expected a file but found a directory, error while removing temporary location %q", tmpLocalFile).Err()
		} else {
			return errors.Reason("copy from helper: expected a file, but found a directory %q", remoteFileName).Err()
		}
	}

	if err := os.Rename(tmpLocalFile, destFileName); err != nil {
		return errors.Annotate(err, "copy from helper: moving local file %q to %q failed", tmpLocalFile, destFileName).Err()
	}
	log.Debug(ctx, "copy from helper: successfully moved %q to %q.", tmpLocalFile, destFileName)
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

// ensureDirExists checks whether the directory 'dirPath' exists, or
// not. If 'createDir' is true, the function will create the directory
// if it does not exist. It returns any error encountered during
// checking the directory, or creating it.
func ensureDirExists(ctx context.Context, dirPath string, createDir bool) error {
	s, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug(ctx, "Ensure directory exists: creating directory %q.", dirPath)
			if createDir {
				if err := os.MkdirAll(dirPath, dirPermission); err != nil {
					return errors.Annotate(err, "ensure directory exists: cannot create %q", dirPath).Err()
				}
				return nil
			}
			return errors.Reason("ensure directory exists: directory %q does not exist", dirPath).Err()
		}
		// This means that 'err' is not known to report whether or not
		// the file or directory already exists. Hence we cannot
		// proceed with checking whether the directory pre-exists, or
		// creating directory.
		return errors.Annotate(err, "ensure directory exists: cannot determine if %q exists", dirPath).Err()
	}
	if s.IsDir() {
		log.Debug(ctx, "Ensure directory %q exists: confirmed.", dirPath)
		return nil
	}
	return errors.Reason("ensure directory exists: cannot create directory %q, it is a pre-existing file", dirPath).Err()
}

// ensureFileExists checks whether the file 'filePath' exists, or
// not. If it does not exist, the function returns an error.
func checkFileExists(ctx context.Context, filePath string) error {
	s, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			errors.Annotate(err, "check file exists: file %q does not exist", filePath).Err()
		}
		// This means that 'err' is not known to report whether or not
		// the file or directory already exists. Hence we cannot
		// proceed with checking whether the file exists or not.
		return errors.Annotate(err, "check file exists: cannot determine if %q exists", filePath).Err()
	}
	if s.IsDir() {
		return errors.Annotate(err, "check file exists: %q is a directory, not a file", filePath).Err()
	}
	return nil
}
