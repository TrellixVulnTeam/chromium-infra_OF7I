// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
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
	out, err := exec.Command(paths.GetHostIdentifierPath).Output()
	if err != nil {
		return "", errors.Annotate(err, "get host identifier").Err()
	}
	s := string(out)
	return strings.TrimRight(s, "\n\t"), nil
}

// GetServiceAccountContent gets the content of the service account.
func GetServiceAccountContent() (string, error) {
	out, err := exec.Command(
		paths.DockerPath,
		"exec",
		"drone",
		"/bin/cat",
		"/creds/service_accounts/skylab-drone.json",
	).Output()
	if err != nil {
		return "", errors.Annotate(err, "get service account content").Err()
	}
	s := string(out)
	return strings.TrimRight(s, "\n\t"), nil
}
