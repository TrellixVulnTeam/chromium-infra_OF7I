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

// Error messages for input validation
const (
	NilEntity         string = "Invalid input - No Entity to add/update."
	EmptyID           string = "Invalid input - Entity ID is empty."
	InvalidCharacters string = "Invalid input - Entity ID must contain only 4-63 characters, ASCII letters, numbers, dash and underscore."
	InvalidPageSize   string = "Invalid input - PageSize should be non-negative."
)

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{4,63}$`)

// Validate validates input requests of CreateMachine.
func (r *CreateMachineRequest) Validate() error {
	if r.Machine == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	name := strings.TrimSpace(r.Machine.GetName())
	if name == "" {
		name = r.MachineId
	}
	return validateResourceName(name)
}

// Validate validates input requests of UpdateMachine.
func (r *UpdateMachineRequest) Validate() error {
	if r.Machine == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(r.Machine.GetName())
}

// Validate validates input requests of GetMachine.
func (r *GetMachineRequest) Validate() error {
	return validateResourceName(r.Name)
}

// Validate validates input requests of ListMachines.
func (r *ListMachinesRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteMachine.
func (r *DeleteMachineRequest) Validate() error {
	return validateResourceName(r.Name)
}

func validateResourceName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !nameRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

func validatePageSize(pageSize int32) error {
	if pageSize < 0 {
		return status.Errorf(codes.InvalidArgument, InvalidPageSize)
	}
	return nil
}
