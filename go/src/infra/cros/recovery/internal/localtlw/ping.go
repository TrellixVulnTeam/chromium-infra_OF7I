// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package localtlw

import (
	"bytes"
	"os/exec"
	"strconv"

	"go.chromium.org/luci/common/errors"
)

// ping represent simple network verification by ping by hostname.
func ping(addr string, count int) error {
	if addr == "" {
		return errors.Reason("ping: addr is empty").Err()
	}
	cmd := exec.Command("ping",
		addr,
		"-c",
		strconv.Itoa(count), // How many times will ping.
		"-W",
		"1", // How long wait for response.
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.Annotate(err, stderr.String()).Err()
	}
	return nil
}
