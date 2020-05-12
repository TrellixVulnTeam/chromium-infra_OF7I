// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufspb

import (
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/unifiedfleet/app/util"
)

// Error messages for input validation
const (
	NilEntity                string = "Invalid input - No Entity to add/update."
	EmptyID                  string = "Invalid input - Entity ID is empty."
	EmptyName                string = "Invalid input - Entity Name is empty."
	InvalidCharacters        string = "Invalid input - Entity ID must contain only 4-63 characters, ASCII letters, numbers, dash and underscore."
	InvalidPageSize          string = "Invalid input - PageSize should be non-negative."
	MachineNameFormat        string = "Invalid input - Entity Name pattern should be machines/{machine}."
	RackNameFormat           string = "Invalid input - Entity Name pattern should be racks/{rack}."
	ChromePlatformNameFormat string = "Invalid input - Entity Name pattern should be chromeplatforms/{chromeplatform}."
	MachineLSENameFormat     string = "Invalid input - Entity Name pattern should be machineLSEs/{machineLSE}."
	RackLSENameFormat        string = "Invalid input - Entity Name pattern should be rackLSEs/{rackLSE}."
)

var (
	emptyMachineDBSourceStatus         = status.New(codes.InvalidArgument, "Invalid argument - MachineDB source is empty")
	invalidHostInMachineDBSourceStatus = status.New(codes.InvalidArgument, "Invalid argument - Host in MachineDB source is empty/invalid")
)

var idRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{4,63}$`)
var chromePlatformRegex = regexp.MustCompile(`chromeplatforms\.*`)
var machineRegex = regexp.MustCompile(`machines\.*`)
var rackRegex = regexp.MustCompile(`racks\.*`)
var machineLSERegex = regexp.MustCompile(`machineLSEs\.*`)
var rackLSERegex = regexp.MustCompile(`rackLSEs\.*`)

// Validate validates input requests of CreateChromePlatform.
func (r *CreateChromePlatformRequest) Validate() error {
	if r.ChromePlatform == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.ChromePlatformId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !idRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateChromePlatform.
func (r *UpdateChromePlatformRequest) Validate() error {
	if r.ChromePlatform == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(chromePlatformRegex, ChromePlatformNameFormat, r.ChromePlatform.GetName())
}

// Validate validates input requests of GetChromePlatform.
func (r *GetChromePlatformRequest) Validate() error {
	return validateResourceName(chromePlatformRegex, ChromePlatformNameFormat, r.Name)
}

// Validate validates input requests of ListChromePlatforms.
func (r *ListChromePlatformsRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteChromePlatform.
func (r *DeleteChromePlatformRequest) Validate() error {
	return validateResourceName(chromePlatformRegex, ChromePlatformNameFormat, r.Name)
}

// Validate validates input requests of CreateMachine.
func (r *CreateMachineRequest) Validate() error {
	if r.Machine == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.MachineId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !idRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateMachine.
func (r *UpdateMachineRequest) Validate() error {
	if r.Machine == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(machineRegex, MachineNameFormat, r.Machine.GetName())
}

// Validate validates input requests of GetMachine.
func (r *GetMachineRequest) Validate() error {
	return validateResourceName(machineRegex, MachineNameFormat, r.Name)
}

// Validate validates input requests of ListMachines.
func (r *ListMachinesRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteMachine.
func (r *DeleteMachineRequest) Validate() error {
	return validateResourceName(machineRegex, MachineNameFormat, r.Name)
}

// Validate validates input requests of CreateRack.
func (r *CreateRackRequest) Validate() error {
	if r.Rack == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.RackId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !idRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateRack.
func (r *UpdateRackRequest) Validate() error {
	if r.Rack == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(rackRegex, RackNameFormat, r.Rack.GetName())
}

// Validate validates input requests of GetRack.
func (r *GetRackRequest) Validate() error {
	return validateResourceName(rackRegex, RackNameFormat, r.Name)
}

// Validate validates input requests of ListRacks.
func (r *ListRacksRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteRack.
func (r *DeleteRackRequest) Validate() error {
	return validateResourceName(rackRegex, RackNameFormat, r.Name)
}

// Validate validates input requests of CreateMachineLSE.
func (r *CreateMachineLSERequest) Validate() error {
	if r.MachineLSE == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.MachineLSEId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !idRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateMachineLSE.
func (r *UpdateMachineLSERequest) Validate() error {
	if r.MachineLSE == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(machineLSERegex, MachineLSENameFormat, r.MachineLSE.GetName())
}

// Validate validates input requests of GetMachineLSE.
func (r *GetMachineLSERequest) Validate() error {
	return validateResourceName(machineLSERegex, MachineLSENameFormat, r.Name)
}

// Validate validates input requests of ListMachineLSEs.
func (r *ListMachineLSEsRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteMachineLSE.
func (r *DeleteMachineLSERequest) Validate() error {
	return validateResourceName(machineLSERegex, MachineLSENameFormat, r.Name)
}

// Validate validates input requests of CreateRackLSE.
func (r *CreateRackLSERequest) Validate() error {
	if r.RackLSE == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.RackLSEId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !idRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateRackLSE.
func (r *UpdateRackLSERequest) Validate() error {
	if r.RackLSE == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(rackLSERegex, RackLSENameFormat, r.RackLSE.GetName())
}

// Validate validates input requests of GetRackLSE.
func (r *GetRackLSERequest) Validate() error {
	return validateResourceName(rackLSERegex, RackLSENameFormat, r.Name)
}

// Validate validates input requests of ListRackLSEs.
func (r *ListRackLSEsRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteRackLSE.
func (r *DeleteRackLSERequest) Validate() error {
	return validateResourceName(rackLSERegex, RackLSENameFormat, r.Name)
}

func validateResourceName(resourceRegex *regexp.Regexp, resourceNameFormat, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return status.Errorf(codes.InvalidArgument, EmptyName)
	}
	if !resourceRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument, resourceNameFormat)
	}
	if !idRegex.MatchString(util.RemovePrefix(name)) {
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

// ValidateMachineDBSource validates the MachineDBSource
func ValidateMachineDBSource(machinedb *MachineDBSource) error {
	if machinedb == nil {
		return emptyMachineDBSourceStatus.Err()
	}
	if machinedb.GetHost() == "" {
		return invalidHostInMachineDBSourceStatus.Err()
	}
	return nil
}
