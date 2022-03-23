// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
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
	output, err := run(ctx, time.Minute, `dmesg`)
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

func init() {
	execs.Register("cros_dmesg", dmesgExec)
}
