// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufspb

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/unifiedfleet/app/util"
)

// Error messages for input validation
var (
	NilEntity                     string = "Invalid input - No Entity to add/update."
	EmptyID                       string = "Invalid input - Entity ID is empty."
	EmptyName                     string = "Invalid input - Entity Name is empty."
	ValidName                     string = "Name must contain only 4-63 characters, ASCII letters, numbers and special characters )(,_:."
	InvalidCharacters             string = fmt.Sprintf("%s,%s", "Invalid input -", ValidName)
	InvalidPageSize               string = "Invalid input - PageSize should be non-negative."
	MachineNameFormat             string = "Invalid input - Entity Name pattern should be machines/{machine}."
	RackNameFormat                string = "Invalid input - Entity Name pattern should be racks/{rack}."
	ChromePlatformNameFormat      string = "Invalid input - Entity Name pattern should be chromeplatforms/{chromeplatform}."
	MachineLSENameFormat          string = "Invalid input - Entity Name pattern should be machineLSEs/{machineLSE}."
	RackLSENameFormat             string = "Invalid input - Entity Name pattern should be rackLSEs/{rackLSE}."
	NicNameFormat                 string = "Invalid input - Entity Name pattern should be nics/{nic}."
	KVMNameFormat                 string = "Invalid input - Entity Name pattern should be kvms/{kvm}."
	RPMNameFormat                 string = "Invalid input - Entity Name pattern should be rpms/{rpm}."
	DracNameFormat                string = "Invalid input - Entity Name pattern should be dracs/{drac}."
	SwitchNameFormat              string = "Invalid input - Entity Name pattern should be switches/{switch}."
	VlanNameFormat                string = "Invalid input - Entity Name pattern should be vlans/{vlan}."
	MachineLSEPrototypeNameFormat string = "Invalid input - Entity Name pattern should be machineLSEPrototypes/{machineLSEPrototype}."
	RackLSEPrototypeNameFormat    string = "Invalid input - Entity Name pattern should be rackLSEPrototypes/{rackLSEPrototype}."
	ResourceFormat                string = "Invalid input - Entity Name pattern should be in a format of resource_names/XXX, resource_names includes machines/racks/vms/hosts/vlans."
)

var (
	emptyMachineDBSourceStatus         = status.New(codes.InvalidArgument, "Invalid argument - MachineDB source is empty")
	invalidHostInMachineDBSourceStatus = status.New(codes.InvalidArgument, "Invalid argument - Host in MachineDB source is empty/invalid")
)

// IDRegex regular expression for checking resource Name/ID
var IDRegex = regexp.MustCompile(`^[a-zA-Z0-9-)(,_:.]{4,63}$`)
var chromePlatformRegex = regexp.MustCompile(`chromeplatforms\.*`)
var machineRegex = regexp.MustCompile(`machines\.*`)
var rackRegex = regexp.MustCompile(`racks\.*`)
var machineLSERegex = regexp.MustCompile(`machineLSEs\.*`)
var rackLSERegex = regexp.MustCompile(`rackLSEs\.*`)
var nicRegex = regexp.MustCompile(`nics\.*`)
var kvmRegex = regexp.MustCompile(`kvms\.*`)
var rpmRegex = regexp.MustCompile(`rpms\.*`)
var dracRegex = regexp.MustCompile(`dracs\.*`)
var switchRegex = regexp.MustCompile(`switches\.*`)
var vlanRegex = regexp.MustCompile(`vlans\.*`)
var machineLSEPrototypeRegex = regexp.MustCompile(`machineLSEPrototypes\.*`)
var rackLSEPrototypeRegex = regexp.MustCompile(`rackLSEPrototypes\.*`)

// It's used to validate a host or vm in resource_name
var hostRegex = regexp.MustCompile(`hosts\.*`)
var vmRegex = regexp.MustCompile(`vms\.*`)
var resourceRegexs = []*regexp.Regexp{
	machineRegex,
	rackRegex,
	vlanRegex,
	hostRegex,
	vmRegex,
}

// Validate validates input requests of CreateChromePlatform.
func (r *CreateChromePlatformRequest) Validate() error {
	if r.ChromePlatform == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.ChromePlatformId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
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

// Validate validates input requests of CreateMachineLSEPrototype.
func (r *CreateMachineLSEPrototypeRequest) Validate() error {
	if r.MachineLSEPrototype == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.MachineLSEPrototypeId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateMachineLSEPrototype.
func (r *UpdateMachineLSEPrototypeRequest) Validate() error {
	if r.MachineLSEPrototype == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(machineLSEPrototypeRegex, MachineLSEPrototypeNameFormat, r.MachineLSEPrototype.GetName())
}

// Validate validates input requests of GetMachineLSEPrototype.
func (r *GetMachineLSEPrototypeRequest) Validate() error {
	return validateResourceName(machineLSEPrototypeRegex, MachineLSEPrototypeNameFormat, r.Name)
}

// Validate validates input requests of ListMachineLSEPrototypes.
func (r *ListMachineLSEPrototypesRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteMachineLSEPrototype.
func (r *DeleteMachineLSEPrototypeRequest) Validate() error {
	return validateResourceName(machineLSEPrototypeRegex, MachineLSEPrototypeNameFormat, r.Name)
}

// Validate validates input requests of CreateRackLSEPrototype.
func (r *CreateRackLSEPrototypeRequest) Validate() error {
	if r.RackLSEPrototype == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.RackLSEPrototypeId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateRackLSEPrototype.
func (r *UpdateRackLSEPrototypeRequest) Validate() error {
	if r.RackLSEPrototype == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(rackLSEPrototypeRegex, RackLSEPrototypeNameFormat, r.RackLSEPrototype.GetName())
}

// Validate validates input requests of GetRackLSEPrototype.
func (r *GetRackLSEPrototypeRequest) Validate() error {
	return validateResourceName(rackLSEPrototypeRegex, RackLSEPrototypeNameFormat, r.Name)
}

// Validate validates input requests of ListRackLSEPrototypes.
func (r *ListRackLSEPrototypesRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteRackLSEPrototype.
func (r *DeleteRackLSEPrototypeRequest) Validate() error {
	return validateResourceName(rackLSEPrototypeRegex, RackLSEPrototypeNameFormat, r.Name)
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
	if !IDRegex.MatchString(id) {
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
	if !IDRegex.MatchString(id) {
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
	if !IDRegex.MatchString(id) {
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
	if !IDRegex.MatchString(id) {
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

// Validate validates input requests of CreateNic.
func (r *CreateNicRequest) Validate() error {
	if r.Nic == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.NicId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateState.
func (r *UpdateStateRequest) Validate() error {
	if r.State == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	for _, rg := range resourceRegexs {
		if err := validateResourceName(rg, "invalid resource name format", r.State.GetResourceName()); err == nil {
			return nil
		}
	}
	return status.Errorf(codes.InvalidArgument, ResourceFormat)
}

// Validate validates input requests of GetState.
func (r *GetStateRequest) Validate() error {
	for _, rg := range resourceRegexs {
		if err := validateResourceName(rg, "invalid resource name format", r.GetResourceName()); err == nil {
			return nil
		}
	}
	return status.Errorf(codes.InvalidArgument, ResourceFormat)
}

// Validate validates input requests of UpdateNic.
func (r *UpdateNicRequest) Validate() error {
	if r.Nic == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(nicRegex, NicNameFormat, r.Nic.GetName())
}

// Validate validates input requests of GetNic.
func (r *GetNicRequest) Validate() error {
	return validateResourceName(nicRegex, NicNameFormat, r.Name)
}

// Validate validates input requests of ListNics.
func (r *ListNicsRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteNic.
func (r *DeleteNicRequest) Validate() error {
	return validateResourceName(nicRegex, NicNameFormat, r.Name)
}

// Validate validates input requests of CreateKVM.
func (r *CreateKVMRequest) Validate() error {
	if r.KVM == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.KVMId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateKVM.
func (r *UpdateKVMRequest) Validate() error {
	if r.KVM == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(kvmRegex, KVMNameFormat, r.KVM.GetName())
}

// Validate validates input requests of GetKVM.
func (r *GetKVMRequest) Validate() error {
	return validateResourceName(kvmRegex, KVMNameFormat, r.Name)
}

// Validate validates input requests of ListKVMs.
func (r *ListKVMsRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteKVM.
func (r *DeleteKVMRequest) Validate() error {
	return validateResourceName(kvmRegex, KVMNameFormat, r.Name)
}

// Validate validates input requests of CreateRPM.
func (r *CreateRPMRequest) Validate() error {
	if r.RPM == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.RPMId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateRPM.
func (r *UpdateRPMRequest) Validate() error {
	if r.RPM == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(rpmRegex, RPMNameFormat, r.RPM.GetName())
}

// Validate validates input requests of GetRPM.
func (r *GetRPMRequest) Validate() error {
	return validateResourceName(rpmRegex, RPMNameFormat, r.Name)
}

// Validate validates input requests of ListRPMs.
func (r *ListRPMsRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteRPM.
func (r *DeleteRPMRequest) Validate() error {
	return validateResourceName(rpmRegex, RPMNameFormat, r.Name)
}

// Validate validates input requests of CreateDrac.
func (r *CreateDracRequest) Validate() error {
	if r.Drac == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.DracId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateDrac.
func (r *UpdateDracRequest) Validate() error {
	if r.Drac == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(dracRegex, DracNameFormat, r.Drac.GetName())
}

// Validate validates input requests of GetDrac.
func (r *GetDracRequest) Validate() error {
	return validateResourceName(dracRegex, DracNameFormat, r.Name)
}

// Validate validates input requests of ListDracs.
func (r *ListDracsRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteDrac.
func (r *DeleteDracRequest) Validate() error {
	return validateResourceName(dracRegex, DracNameFormat, r.Name)
}

// Validate validates input requests of CreateSwitch.
func (r *CreateSwitchRequest) Validate() error {
	if r.Switch == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.SwitchId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateSwitch.
func (r *UpdateSwitchRequest) Validate() error {
	if r.Switch == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(switchRegex, SwitchNameFormat, r.Switch.GetName())
}

// Validate validates input requests of GetSwitch.
func (r *GetSwitchRequest) Validate() error {
	return validateResourceName(switchRegex, SwitchNameFormat, r.Name)
}

// Validate validates input requests of ListSwitches.
func (r *ListSwitchesRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteSwitch.
func (r *DeleteSwitchRequest) Validate() error {
	return validateResourceName(switchRegex, SwitchNameFormat, r.Name)
}

// Validate validates input requests of CreateVlan.
func (r *CreateVlanRequest) Validate() error {
	if r.Vlan == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.VlanId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	return nil
}

// Validate validates input requests of UpdateVlan.
func (r *UpdateVlanRequest) Validate() error {
	if r.Vlan == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(vlanRegex, VlanNameFormat, r.Vlan.GetName())
}

// Validate validates input requests of GetVlan.
func (r *GetVlanRequest) Validate() error {
	return validateResourceName(vlanRegex, VlanNameFormat, r.Name)
}

// Validate validates input requests of ListVlans.
func (r *ListVlansRequest) Validate() error {
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteVlan.
func (r *DeleteVlanRequest) Validate() error {
	return validateResourceName(vlanRegex, VlanNameFormat, r.Name)
}

func validateResourceName(resourceRegex *regexp.Regexp, resourceNameFormat, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return status.Errorf(codes.InvalidArgument, EmptyName)
	}
	if !resourceRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument, resourceNameFormat)
	}
	if !IDRegex.MatchString(util.RemovePrefix(name)) {
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

// ValidateResourceKey validates a key of a resource
//
// TODO(xixuan): add validation for all imported data
func ValidateResourceKey(resources interface{}, k string) error {
	vs := ParseResources(resources, k)
	for _, v := range vs {
		if !IDRegex.MatchString(v) {
			return status.Errorf(codes.InvalidArgument, InvalidCharacters)
		}
	}
	return nil
}

// ParseResources parse a list of resources and returns a string slice by key
func ParseResources(args interface{}, k string) []string {
	names := make([]string, 0)
	v := reflect.ValueOf(args)
	switch v.Kind() {
	case reflect.Ptr:
		names = append(names, parse(v.Elem(), k))
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			n := parse(v.Index(i).Elem(), k)
			if n != "" {
				names = append(names, n)
			}
		}
	}
	return names
}

func parse(v reflect.Value, k string) string {
	typeOfT := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if typeOfT.Field(i).Name == k {
			return f.Interface().(string)
		}
	}
	return ""
}
