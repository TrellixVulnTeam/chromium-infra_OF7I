// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"io/ioutil"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// MakeTempFile makes a temporary file.
// TODO(gregorynisbet): Move to separate package.
func MakeTempFile(content string) (string, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", errors.Annotate(err, "makeTempFile").Err()
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return "", errors.Annotate(err, "makeTempFile").Err()
	}
	if err := ioutil.WriteFile(name, []byte(content), 0o077); err != nil {
		return "", errors.Annotate(err, "makeTempFile").Err()
	}
	return name, nil
}

// TrimOutput trims trailing whitespace from command output.
// TODO(gregorynisbet): Move to separate package.
func TrimOutput(output []byte) string {
	if len(output) == 0 {
		return ""
	}
	return strings.TrimRight(string(output), "\n\t")
}
