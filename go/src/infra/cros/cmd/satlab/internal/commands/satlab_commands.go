// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/satlab/internal/paths"
)

// Decision is a classification of a line in a file.
// Lines may be kept, modified, or deleted.
// Functions that process lines of text are split conceptually
// into a decision which classifies lines and a transformation
// which only applies to selected lines.
type Decision int

const (
	Unknown Decision = iota
	Keep
	Reject
	// Modify is used only by replacing things.
	Modify
)

// GetHostIdentifier gets the host identifier value.
func GetDockerHostBoxIdentifier() (string, error) {
	fmt.Fprintf(os.Stderr, "Get host identifier: run %s\n", paths.GetHostIdentifierScript)
	out, err := exec.Command(paths.GetHostIdentifierScript).Output()
	// Immediately normalize the satlab prefix to lowercase. It will save a lot of
	// trouble later.
	return strings.ToLower(TrimOutput(out)), errors.Annotate(err, "get host identifier").Err()
}

// GetServiceAccountContent gets the content of the service account.
func GetServiceAccountContent() (string, error) {
	args := []string{
		paths.DockerPath,
		"exec",
		"drone",
		"/bin/cat",
		"/creds/service_accounts/skylab-drone.json",
	}
	fmt.Fprintf(os.Stderr, "Get drone credential: run %s\n", args)
	out, err := exec.Command(args[0], args...).Output()
	return TrimOutput(out), errors.Annotate(err, "get service account content").Err()
}
