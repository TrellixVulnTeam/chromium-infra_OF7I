// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufspb

import (
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{4,63}$`)

// Validate validates input requests of CreateMachine.
func (r *CreateMachineRequest) Validate() error {
	if r.Machine == nil {
		return status.Errorf(codes.InvalidArgument, "Invalid input - no Machine to add/update")
	}
	name := strings.TrimSpace(r.Machine.GetName())
	if name == "" {
		name = strings.TrimSpace(r.MachineId)
		if name == "" {
			return status.Errorf(codes.InvalidArgument, "Invalid input - Machine ID is empty")
		}
	}
	if !nameRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument,
			"Invalid input - Machine ID must contain only 4-63 characters, ASCII letters, numbers, dash and underscore.")
	}
	return nil
}

// Validate validates input requests of UpdateMachine.
func (r *UpdateMachineRequest) Validate() error {
	if r.Machine == nil {
		return status.Errorf(codes.InvalidArgument, "Invalid input - no Machine to add/update")
	}
	name := strings.TrimSpace(r.Machine.GetName())
	if name == "" {
		return status.Errorf(codes.InvalidArgument, "Invalid input - Machine ID is empty")
	}
	if !nameRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument,
			"Invalid input - Machine ID must contain only 4-63 characters, ASCII letters, numbers, dash and underscore.")
	}
	return nil
}
