// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"infra/cros/recovery/internal/execs"
	"io/ioutil"
	"path/filepath"
	"time"

	"go.chromium.org/luci/common/errors"
)

// Permissions is the default file permissions for log files.
// Currently, we allow everyone to read and write and nobody to execute.
const permissions = 0b110_110_110

// DmesgExec grabs dmesg and persists the file into the log directory.
// DmesgExec fails if and only if the dmesg executable doesn't exist or returns nonzero.
func dmesgExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	output, err := run(ctx, time.Minute, `dmesg`)
	switch {
	case err == nil:
		// Output is non-empty and dmesg ran successfully. This exec is successful
		logRoot := info.RunArgs.LogRoot
		// TODO(gregorynisbet): Pick a better path to write to.
		// TODO(gregorynisbet): Don't ignore the error from writing the file locally.
		//                      However, failing to write the file locally should not cause the exec to fail.
		ioutil.WriteFile(filepath.Join(logRoot, "dmesg.txt"), []byte(output), permissions)
		// Write the number of bytes we collected to a separate file alongside dmesg.txt.
		// This allows us to know with complete certainty that we intentionally collected 0 bytes of output, for example.
		ioutil.WriteFile(
			filepath.Join(logRoot, "dmesg_bytes.txt"),
			[]byte(fmt.Sprintf("%d", len(output))),
			permissions,
		)
		return nil
	default:
		return errors.Annotate(err, "dmesg exec").Err()
	}
}

func init() {
	execs.Register("cros_dmesg", dmesgExec)
}
