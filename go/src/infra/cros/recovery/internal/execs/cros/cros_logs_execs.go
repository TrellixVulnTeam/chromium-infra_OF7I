// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
)

// Permissions is the default file permissions for log files.
// Currently, we allow everyone to read and write and nobody to execute.
const defaultFilePermissions fs.FileMode = 0666

// DmesgExec grabs dmesg and persists the file into the log directory.
// DmesgExec fails if and only if the dmesg executable doesn't exist or returns nonzero.
func dmesgExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	log := info.NewLogger()
	logRoot := info.GetLogRoot()
	output, err := run(ctx, time.Minute, "dmesg", "-H")
	if err != nil {
		return errors.Annotate(err, "dmesg exec").Err()
	}
	// Output is non-empty and dmesg ran successfully. This exec is successful
	f := filepath.Join(logRoot, "dmesg")
	log.Debugf("dmesg path to safe: %s", f)
	ioutil.WriteFile(f, []byte(output), defaultFilePermissions)
	// Write the number of bytes we collected to a separate file alongside dmesg.txt.
	// This allows us to know with complete certainty that we intentionally collected 0 bytes of output, for example.
	fc := filepath.Join(logRoot, "dmesg_bytes_count")
	ioutil.WriteFile(fc, []byte(fmt.Sprintf("%d", len(output))), defaultFilePermissions)
	return nil
}

// copyFileToLogExec grabs the file from the host and copy to the log directory.
//
// For now implementation support only servo-host.
// TODO: Extend support to collect logs from any host.
func copyFileToLogExec(ctx context.Context, info *execs.ExecInfo) error {
	resource := info.RunArgs.DUT.ServoHost.Name
	run := info.NewRunner(resource)
	log := info.NewLogger()
	logRoot := info.GetLogRoot()
	argMap := info.GetActionArgs(ctx)
	fullPath := strings.TrimSpace(argMap.AsString(ctx, "filepath", ""))
	if fullPath == "" {
		return errors.Reason("copy file to logs: filepath is empty or not provided").Err()
	}
	if _, err := run(ctx, time.Minute, "test", "-f", fullPath); err != nil {
		return errors.Annotate(err, "copy file to logs: the file is not exist or it is directory").Err()
	}
	newName := strings.TrimSpace(argMap.AsString(ctx, "filename", ""))
	if newName == "" {
		newName = filepath.Base(fullPath)
	}
	if newName == "" {
		return errors.Reason("copy file to logs: filename is empty and could not extracted from filepath").Err()
	}
	// Logs will be saved to the resource folder.
	if argMap.AsBool(ctx, "use_host_dir", false) {
		logRoot := filepath.Join(logRoot, resource)
		if err := exec.CommandContext(ctx, "mkdir", "-p", logRoot).Run(); err != nil {
			return errors.Annotate(err, "copy file to logs").Err()
		}
	}
	destFile := filepath.Join(logRoot, newName)
	log.Debugf("Try to collect servod log %q to %q!", fullPath, destFile)
	err := info.CopyFrom(ctx, resource, fullPath, destFile)
	return errors.Annotate(err, "copy file to logs").Err()
}

func init() {
	execs.Register("cros_dmesg", dmesgExec)
	execs.Register("cros_copy_file_to_log", copyFileToLogExec)
}
