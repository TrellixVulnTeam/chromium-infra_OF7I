// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufspb

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/util"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error messages for input validation
var (
	NilEntity                      string = "Invalid input - No Entity to add/update."
	EmptyID                        string = "Invalid input - Entity ID is empty."
	EmptyName                      string = "Invalid input - Entity Name is empty."
	InvalidMac                     string = "invalid mac address"
	ValidName                      string = "Name must match the regular expression `^[a-zA-Z0-9-)(_:.]{3,63}$`"
	HostnamePattern                string = "Name must match the regular expression `^[a-zA-Z0-9-.]{1,63}$`"
	InvalidCharacters              string = fmt.Sprintf("%s%s", "Invalid input - ", ValidName)
	InvalidHostname                string = fmt.Sprintf("%s%s", "Invalid input - ", HostnamePattern)
	InvalidTags                    string = "Invalid input - Tags must not include '='."
	InvalidPageSize                string = "Invalid input - PageSize should be non-negative."
	AssetNameFormat                string = "Invalid input - Entity Name pattern should be assets/{asset}."
	MachineNameFormat              string = "Invalid input - Entity Name pattern should be machines/{machine}."
	RackNameFormat                 string = "Invalid input - Entity Name pattern should be racks/{rack}."
	ChromePlatformNameFormat       string = "Invalid input - Entity Name pattern should be chromeplatforms/{chromeplatform}."
	CachingServiceNameFormat       string = "Invalid input - Entity Name pattern should be cachingservices/{hostname or ipv4}."
	MachineLSENameFormat           string = "Invalid input - Entity Name pattern should be machineLSEs/{machineLSE}."
	MachineLSEDeploymentNameFormat string = "Invalid input - Entity Name pattern should be machineLSEDeployments/{name}."
	VMNameFormat                   string = "Invalid input - Entity Name pattern should be vms/{vm}."
	RackLSENameFormat              string = "Invalid input - Entity Name pattern should be rackLSEs/{rackLSE}."
	NicNameFormat                  string = "Invalid input - Entity Name pattern should be nics/{nic}."
	KVMNameFormat                  string = "Invalid input - Entity Name pattern should be kvms/{kvm}."
	RPMNameFormat                  string = "Invalid input - Entity Name pattern should be rpms/{rpm}."
	DracNameFormat                 string = "Invalid input - Entity Name pattern should be dracs/{drac}."
	SwitchNameFormat               string = "Invalid input - Entity Name pattern should be switches/{switch}."
	SchedulingUnitNameFormat       string = "Invalid input - Entity Name pattern should be schedulingunits/{schedulingunit}."
	VlanNameFormat                 string = "Invalid input - Entity Name pattern should be vlans/{vlan}."
	MachineLSEPrototypeNameFormat  string = "Invalid input - Entity Name pattern should be machineLSEPrototypes/{machineLSEPrototype}."
	RackLSEPrototypeNameFormat     string = "Invalid input - Entity Name pattern should be rackLSEPrototypes/{rackLSEPrototype}."
	ResourceFormat                 string = "Invalid input - Entity Name pattern should be in a format of resource_names/XXX, resource_names includes machines/racks/vms/hosts/vlans."
	EmptyMachineName               string = "Invalid input - Machine name cannot be empty."
	EmptyHostName                  string = "Invalid input - Host name cannot be empty."
	EmptyRackName                  string = "Invalid input - Rack name cannot be empty."
	FilterFormat                   string = "Filter format Egs:\n" + "'machine=mac-1'\n" + "'machine=mac-1,mac-2'\n" + "'machine=mac-1 & nic=nic-1'\n" + "'machine=mac-1 & nic=nic-1 & kvm=kvm-1,kvm-2'"
	InvalidFilterFormat            string = fmt.Sprintf("%s%s", "Invalid input - ", FilterFormat)
	EmptyRequest                   string = "Empty Request"
)

var (
	emptyMachineDBSourceStatus         = status.New(codes.InvalidArgument, "Invalid argument - MachineDB source is empty")
	invalidHostInMachineDBSourceStatus = status.New(codes.InvalidArgument, "Invalid argument - Host in MachineDB source is empty/invalid")
)

// IDRegex regular expression for checking resource Name/ID
var IDRegex = regexp.MustCompile(`^[a-zA-Z0-9-)(_:.]{3,63}$`)

// HostnameRegex regular expression for checking hostname for host/vm
var HostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9-.]{1,63}$`)
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
var assetRegex = regexp.MustCompile(`assets\.*`)
var machineLSEDeploymentRegex = regexp.MustCompile(`machineLSEDeployments\.*`)
var schedulingUnitRegex = regexp.MustCompile(`schedulingunits\.*`)

// matches "cachingservices/{hostname or ipv4}"
var cachingServiceRegex = regexp.MustCompile(`cachingservices/[a-zA-Z0-9-.]{1,63}$`)

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

// FilterRegex is the regex for filter string for all List requests
//
// resource1=resourcename1
// resource1=resourcename1,resourcename2
// resource1=resourcename1 & resource2=resourcename21
// resource1=resourcename1,resourcename2 & resource2=resourcename21,resourcename22
// resource1=resourcename1 & resource2=resourcename21 & resource3=resourcename31
// machine=mac-1
// machine=mac-1,mac-2
// machine=mac-1 & nic=nic-1
// machine=mac-1 & nic=nic-1 & kvm=kvm-1,kvm-2
var FilterRegex = regexp.MustCompile(`^([a-z]*\=[a-zA-Z0-9-)(_:.\/]*)(\,[a-zA-Z0-9-)(_:.\/]*)*(\&([a-z]*\=[a-zA-Z0-9-)(_:.\/]*)(\,[a-zA-Z0-9-)(_:.\/]*)*)*$`)

// Validate validates input requests of RackRegistration.
func (r *RackRegistrationRequest) Validate() error {
	if r.Rack == nil {
		return status.Errorf(codes.InvalidArgument, "Rack "+NilEntity)
	}
	id := strings.TrimSpace(r.Rack.GetName())
	if id == "" {
		return status.Errorf(codes.InvalidArgument, "Rack "+EmptyName)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, "Rack "+InvalidCharacters)
	}
	if !util.ValidateTags(r.Rack.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.Rack.Name = id

	if r.GetRack().GetChromeBrowserRack() != nil {
		if r.GetRack().GetChromeBrowserRack().GetSwitchObjects() != nil {
			switches := r.GetRack().GetChromeBrowserRack().GetSwitchObjects()
			for _, s := range switches {
				id = strings.TrimSpace(s.GetName())
				if id == "" {
					return status.Errorf(codes.InvalidArgument, "Switch "+EmptyName)
				}
				if !IDRegex.MatchString(id) {
					errorMsg := fmt.Sprintf("Switch %s has invalid characters in the name.", id)
					return status.Errorf(codes.InvalidArgument, errorMsg+InvalidCharacters)
				}
			}
		}

		if r.GetRack().GetChromeBrowserRack().GetKvmObjects() != nil {
			kvms := r.GetRack().GetChromeBrowserRack().GetKvmObjects()
			for _, kvm := range kvms {
				id = strings.TrimSpace(kvm.GetName())
				if id == "" {
					return status.Errorf(codes.InvalidArgument, "KVM "+EmptyName)
				}
				if !IDRegex.MatchString(id) {
					errorMsg := fmt.Sprintf("KVM %s has invalid characters in the name.", id)
					return status.Errorf(codes.InvalidArgument, errorMsg+InvalidCharacters)
				}
			}
		}

		if r.GetRack().GetChromeBrowserRack().GetRpmObjects() != nil {
			rpms := r.GetRack().GetChromeBrowserRack().GetRpmObjects()
			for _, rpm := range rpms {
				id = strings.TrimSpace(rpm.GetName())
				if id == "" {
					return status.Errorf(codes.InvalidArgument, "RPM "+EmptyName)
				}
				if !IDRegex.MatchString(id) {
					errorMsg := fmt.Sprintf("RPM %s has invalid characters in the name.", id)
					return status.Errorf(codes.InvalidArgument, errorMsg+InvalidCharacters)
				}
			}
		}
	}
	return nil
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
	if !util.ValidateTags(r.ChromePlatform.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.ChromePlatformId = id
	return nil
}

// Validate validates input requests of UpdateChromePlatform.
func (r *UpdateChromePlatformRequest) Validate() error {
	if r.ChromePlatform == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if !util.ValidateTags(r.ChromePlatform.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(chromePlatformRegex, ChromePlatformNameFormat, r.ChromePlatform.GetName())
}

// Validate validates input requests of GetDHCPConfig.
func (r *GetDHCPConfigRequest) Validate() error {
	if r.GetHostname() == "" {
		return status.Errorf(codes.InvalidArgument, EmptyName)
	}
	return nil
}

// Validate validates input requests of GetChromePlatform.
func (r *GetChromePlatformRequest) Validate() error {
	return validateResourceName(chromePlatformRegex, ChromePlatformNameFormat, r.Name)
}

// Validate validates input requests of ListChromePlatforms.
func (r *ListChromePlatformsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	r.MachineLSEPrototypeId = id
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
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	r.RackLSEPrototypeId = id
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
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteRackLSEPrototype.
func (r *DeleteRackLSEPrototypeRequest) Validate() error {
	return validateResourceName(rackLSEPrototypeRegex, RackLSEPrototypeNameFormat, r.Name)
}

// Validate validates input requests of MachineRegistrationRequest.
func (r *MachineRegistrationRequest) Validate() error {
	if r.Machine == nil {
		return status.Errorf(codes.InvalidArgument, "Machine "+NilEntity)
	}
	id := strings.TrimSpace(r.Machine.GetName())
	if id == "" {
		return status.Errorf(codes.InvalidArgument, "Machine "+EmptyName)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, "Machine "+InvalidCharacters)
	}
	if !util.ValidateTags(r.Machine.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.Machine.Name = id
	r.Machine.SerialNumber = strings.TrimSpace(r.Machine.SerialNumber)

	if r.GetMachine().GetChromeBrowserMachine() != nil {
		if r.GetMachine().GetChromeBrowserMachine().GetNicObjects() != nil {
			nics := r.GetMachine().GetChromeBrowserMachine().GetNicObjects()
			for _, nic := range nics {
				id = strings.TrimSpace(nic.GetName())
				if id == "" {
					return status.Errorf(codes.InvalidArgument, "Nic "+EmptyName)
				}
				if !IDRegex.MatchString(id) {
					errorMsg := fmt.Sprintf("Nic %s has invalid characters in the name.", id)
					return status.Errorf(codes.InvalidArgument, errorMsg+InvalidCharacters)
				}
				if err := validateNic(nic); err != nil {
					return err
				}
			}
		}

		if r.GetMachine().GetChromeBrowserMachine().GetDracObject() != nil {
			drac := r.GetMachine().GetChromeBrowserMachine().GetDracObject()
			if drac != nil {
				id = strings.TrimSpace(drac.GetName())
				if id == "" {
					return status.Errorf(codes.InvalidArgument, "Drac "+EmptyName)
				}
				if !IDRegex.MatchString(id) {
					return status.Errorf(codes.InvalidArgument, "Drac "+InvalidCharacters)
				}
				if err := validateDrac(drac); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Validate validates input requests of UpdateMachine.
func (r *UpdateMachineRequest) Validate() error {
	if r.Machine == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if !util.ValidateTags(r.Machine.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.Machine.Name = strings.TrimSpace(r.Machine.GetName())
	r.Machine.SerialNumber = strings.TrimSpace(r.Machine.SerialNumber)
	return validateResourceName(machineRegex, MachineNameFormat, r.Machine.GetName())
}

// Validate validates input requests of GetMachine.
func (r *GetMachineRequest) Validate() error {
	return validateResourceName(machineRegex, MachineNameFormat, r.Name)
}

// Validate validates input requests of ListMachines.
func (r *ListMachinesRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteMachine.
func (r *DeleteMachineRequest) Validate() error {
	return validateResourceName(machineRegex, MachineNameFormat, r.Name)
}

// Validate validates input requests of RenameMachine.
func (r *RenameMachineRequest) Validate() error {
	if err := validateResourceName(machineRegex, MachineNameFormat, r.GetName()); err != nil {
		return err
	}
	return validateResourceName(machineRegex, MachineNameFormat, r.GetNewName())
}

// Validate validates input requests of UpdateRack.
func (r *UpdateRackRequest) Validate() error {
	if r.Rack == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if !util.ValidateTags(r.Rack.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(rackRegex, RackNameFormat, r.Rack.GetName())
}

// Validate validates input requests of GetRack.
func (r *GetRackRequest) Validate() error {
	return validateResourceName(rackRegex, RackNameFormat, r.Name)
}

// Validate validates input requests of ListRacks.
func (r *ListRacksRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	if !HostnameRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidHostname)
	}
	if r.MachineLSE.GetHostname() == "" {
		return status.Errorf(codes.InvalidArgument, "Hostname cannot be empty for a host. It must be same as name of the host")
	}
	if !HostnameRegex.MatchString(r.MachineLSE.GetHostname()) {
		return status.Errorf(codes.InvalidArgument, InvalidHostname)
	}
	if r.MachineLSE.GetMachines() == nil || len(r.MachineLSE.GetMachines()) == 0 {
		return status.Errorf(codes.InvalidArgument, EmptyMachineName)
	}
	if err := validateNetworkOption(r.GetNetworkOption()); err != nil {
		return err
	}
	for _, machineName := range r.MachineLSE.GetMachines() {
		if machineName == "" {
			return status.Errorf(codes.InvalidArgument, EmptyMachineName)
		}
	}
	if !util.ValidateTags(r.MachineLSE.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.MachineLSEId = id
	return nil
}

// Validate validates input requests of UpdateMachineLSE.
func (r *UpdateMachineLSERequest) Validate() error {
	if r.MachineLSE == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if !util.ValidateTags(r.MachineLSE.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	for k, v := range r.GetNetworkOptions() {
		if err := validateNetworkOption(v); err != nil {
			return errors.Annotate(err, "fail to validate host %s", k).Err()
		}
	}
	return validateResourceName(machineLSERegex, MachineLSENameFormat, r.MachineLSE.GetName())
}

// Validate validates input requests of GetMachineLSE.
func (r *GetMachineLSERequest) Validate() error {
	return validateResourceName(machineLSERegex, MachineLSENameFormat, r.Name)
}

// Validate validates input requests of ListMachineLSEs.
func (r *ListMachineLSEsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteMachineLSE.
func (r *DeleteMachineLSERequest) Validate() error {
	return validateResourceName(machineLSERegex, MachineLSENameFormat, r.Name)
}

// Validate validates input requests of CreateVM.
func (r *CreateVMRequest) Validate() error {
	if r.GetVm() == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if !HostnameRegex.MatchString(r.GetVm().GetName()) {
		return status.Errorf(codes.InvalidArgument, "VM name is invalid: %s", InvalidHostname)
	}
	id := strings.TrimSpace(r.GetVm().GetMachineLseId())
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyHostName)
	}
	if !HostnameRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, "Host name is invalid: %s", InvalidHostname)
	}
	if r.Vm.GetMacAddress() != "" {
		newMac, err := util.ParseMac(r.Vm.GetMacAddress())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		r.Vm.MacAddress = newMac
	}
	if err := validateNetworkOption(r.GetNetworkOption()); err != nil {
		return err
	}
	if !util.ValidateTags(r.Vm.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.GetVm().MachineLseId = id
	r.GetVm().Name = strings.TrimSpace(r.GetVm().Name)
	return nil
}

func validateNetworkOption(opt *NetworkOption) error {
	if opt == nil {
		return nil
	}
	if !opt.GetDelete() && opt.GetVlan() == "" && opt.GetIp() == "" {
		return status.Errorf(codes.InvalidArgument, "Network option doesn't set delete or vlan or ip.")
	}
	return nil
}

// Validate validates input requests of UpdateVM.
func (r *UpdateVMRequest) Validate() error {
	if r.GetVm() == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if r.Vm.GetMacAddress() != "" {
		newMac, err := util.ParseMac(r.Vm.GetMacAddress())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		r.Vm.MacAddress = newMac
	}
	if err := validateNetworkOption(r.GetNetworkOption()); err != nil {
		return err
	}
	if !util.ValidateTags(r.Vm.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(vmRegex, VMNameFormat, r.Vm.GetName())
}

// Validate validates input requests of DeleteVM.
func (r *DeleteVMRequest) Validate() error {
	return validateResourceName(vmRegex, VMNameFormat, r.Name)
}

// Validate validates input requests of GetVM.
func (r *GetVMRequest) Validate() error {
	return validateResourceName(vmRegex, VMNameFormat, r.Name)
}

// Validate validates input requests of ListVMs.
func (r *ListVMsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
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
	r.RackLSEId = id
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
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	if err := validateNic(r.GetNic()); err != nil {
		return err
	}
	if r.GetNic().GetMachine() == "" {
		return status.Errorf(codes.InvalidArgument, EmptyMachineName)
	}
	r.NicId = id
	return nil
}

func validateNic(nic *ufspb.Nic) error {
	if nic.GetMacAddress() == "" {
		return status.Errorf(codes.InvalidArgument, "nic macAddress cannot be empty")
	}
	if !util.ValidateTags(nic.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	newMac, err := util.ParseMac(nic.GetMacAddress())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	nic.MacAddress = newMac
	return nil
}

// Validate validates input requests of RenameNic.
func (r *RenameNicRequest) Validate() error {
	if err := validateResourceName(nicRegex, NicNameFormat, r.GetName()); err != nil {
		return err
	}
	return validateResourceName(nicRegex, NicNameFormat, r.GetNewName())
}

// Validate validates input requests of RenameSwitch.
func (r *RenameSwitchRequest) Validate() error {
	if err := validateResourceName(switchRegex, SwitchNameFormat, r.GetName()); err != nil {
		return err
	}
	return validateResourceName(switchRegex, SwitchNameFormat, r.GetNewName())
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

// Validate validates input requests of UpdateDutState
func (r *UpdateDutStateRequest) Validate() error {
	if r.DutState == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if r.GetDutState().GetId() == nil {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	dutID := strings.TrimSpace(r.GetDutState().GetId().GetValue())
	if dutID == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(dutID) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}

	dutHostname := strings.TrimSpace(r.GetDutState().GetHostname())
	if dutHostname == "" {
		return status.Errorf(codes.InvalidArgument, "Hostname cannot be empty when updating dut states for %q", dutID)
	}

	if r.GetDutMeta() != nil {
		if r.GetDutMeta().GetChromeosDeviceId() != dutID {
			return status.Errorf(codes.InvalidArgument, fmt.Sprintf("Mismatched dut ID in DUT meta with dut state: %q", dutID))
		}
		if r.GetDutMeta().GetHostname() != dutHostname {
			return status.Errorf(codes.InvalidArgument, fmt.Sprintf("Mismatched dut hostname in DUT meta (%q) with dut state %q", r.GetDutMeta().GetHostname(), dutHostname))
		}
	}
	if r.GetLabMeta() != nil {
		if r.GetLabMeta().GetChromeosDeviceId() != dutID {
			return status.Errorf(codes.InvalidArgument, fmt.Sprintf("Mismatched dut ID in lab meta with dut state: %q", dutID))
		}
		if r.GetLabMeta().GetHostname() != dutHostname {
			return status.Errorf(codes.InvalidArgument, fmt.Sprintf("Mismatched dut hostname in lab meta (%q) with dut state %q", r.GetLabMeta().GetHostname(), dutHostname))
		}
	}

	return nil
}

// Validate validates input requests of UpdateDeviceRecoveryData
func (r *UpdateDeviceRecoveryDataRequest) Validate() error {
	if r.DutState == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if err := r.validateDutId(); err != nil {
		return err
	}
	if err := r.validateHostnames(); err != nil {
		return err
	}
	if err := r.validateDutWifiRouterHostnames(); err != nil {
		return err
	}
	if err := r.validateBluetoothPeerHostnames(); err != nil {
		return err
	}
	return nil
}

func (r *UpdateDeviceRecoveryDataRequest) validateHostnames() error {
	dutHostname := strings.TrimSpace(r.GetDutState().GetHostname())
	// Hostnames are required. And request hostname should match dut state hostname
	if r.GetHostname() == "" {
		return status.Errorf(codes.InvalidArgument, "Empty request hostname (%q)", r.GetHostname())
	}
	if dutHostname == "" {
		return status.Errorf(codes.InvalidArgument, "Empty dut state hostname (%q)", dutHostname)
	}
	if r.GetHostname() != dutHostname {
		return status.Errorf(codes.InvalidArgument, "Mismatched request hostname (%q) with dut state hostname %q", r.GetHostname(), dutHostname)
	}
	return nil
}

func (r *UpdateDeviceRecoveryDataRequest) validateDutId() error {
	dutID := strings.TrimSpace(r.GetDutState().GetId().GetValue())
	if dutID == "" {
		return status.Errorf(codes.InvalidArgument, "Empty dut state id. %s", EmptyID)
	} else if dutID != r.GetChromeosDeviceId() {
		return status.Errorf(codes.InvalidArgument, "Mismatched chromeos device id(%q) with dut state id: %q", r.GetChromeosDeviceId(), dutID)
	}
	if !IDRegex.MatchString(dutID) {
		return status.Errorf(codes.InvalidArgument, "Invalid dut state id(%q). %s ", dutID, InvalidCharacters)
	}
	return nil
}

// validateHostnames checks for empty or duplicates in hostnames. Uses typ to construct error message.
func validateHostnames(hostnames []string, typ string) error {
	set := make(map[string]bool)
	for _, h := range hostnames {
		if len(strings.TrimSpace(h)) == 0 {
			return status.Errorf(codes.InvalidArgument, "Empty hostname in %s lab data: %v", typ, hostnames)
		}
		if set[h] {
			return status.Errorf(codes.InvalidArgument, "Duplicate hostname (%q) in %s lab data: %v", h, typ, hostnames)
		}
		set[h] = true
	}
	return nil
}

func (r *UpdateDeviceRecoveryDataRequest) validateDutWifiRouterHostnames() error {
	if r.GetLabData() == nil {
		return nil
	}
	var hostnames []string
	for _, r := range r.GetLabData().GetWifiRouters() {
		hostnames = append(hostnames, r.GetHostname())
	}
	return validateHostnames(hostnames, "WiFi")
}

func (r *UpdateDeviceRecoveryDataRequest) validateBluetoothPeerHostnames() error {
	if r.GetLabData() == nil {
		return nil
	}
	var hostnames []string
	for _, b := range r.GetLabData().GetBlueoothPeers() {
		hostnames = append(hostnames, b.GetHostname())
	}
	return validateHostnames(hostnames, "Blueooth peers")
}

// Validate validates input requests of GetDutStateRequest.
func (r *GetDutStateRequest) Validate() error {
	if r == nil {
		return status.Errorf(codes.InvalidArgument, EmptyRequest)
	}
	if r.ChromeosDeviceId == "" && r.Hostname == "" {
		return status.Errorf(codes.InvalidArgument, "Both Id and hostname are empty")
	}
	return nil
}

// Validate validates input requests of ListDutStates.
func (r *ListDutStatesRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of UpdateNic.
func (r *UpdateNicRequest) Validate() error {
	if r.Nic == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if r.Nic.GetMacAddress() != "" {
		newMac, err := util.ParseMac(r.Nic.GetMacAddress())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		r.Nic.MacAddress = newMac
	}
	if !util.ValidateTags(r.Nic.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(nicRegex, NicNameFormat, r.Nic.GetName())
}

// Validate validates input requests of GetNic.
func (r *GetNicRequest) Validate() error {
	return validateResourceName(nicRegex, NicNameFormat, r.Name)
}

// Validate validates input requests of ListNics.
func (r *ListNicsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	if r.GetKVM().GetRack() == "" {
		return status.Errorf(codes.InvalidArgument, EmptyRackName)
	}
	if r.KVM.GetMacAddress() != "" {
		newMac, err := util.ParseMac(r.KVM.GetMacAddress())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		r.KVM.MacAddress = newMac
	}
	if !util.ValidateTags(r.KVM.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.KVMId = id
	return nil
}

// Validate validates input requests of UpdateKVM.
func (r *UpdateKVMRequest) Validate() error {
	if r.KVM == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if r.KVM.GetMacAddress() != "" {
		newMac, err := util.ParseMac(r.KVM.GetMacAddress())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		r.KVM.MacAddress = newMac
	}
	if !util.ValidateTags(r.KVM.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(kvmRegex, KVMNameFormat, r.KVM.GetName())
}

// Validate validates input requests of GetKVM.
func (r *GetKVMRequest) Validate() error {
	return validateResourceName(kvmRegex, KVMNameFormat, r.Name)
}

// Validate validates input requests of ListKVMs.
func (r *ListKVMsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	if r.GetRPM().GetRack() == "" {
		return status.Errorf(codes.InvalidArgument, EmptyRackName)
	}
	if r.RPM.GetMacAddress() != "" {
		newMac, err := util.ParseMac(r.RPM.GetMacAddress())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		r.RPM.MacAddress = newMac
	}
	if !util.ValidateTags(r.RPM.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.RPMId = id
	return nil
}

// Validate validates input requests of UpdateRPM.
func (r *UpdateRPMRequest) Validate() error {
	if r.RPM == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if r.RPM.GetMacAddress() != "" {
		newMac, err := util.ParseMac(r.RPM.GetMacAddress())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		r.RPM.MacAddress = newMac
	}
	if !util.ValidateTags(r.RPM.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(rpmRegex, RPMNameFormat, r.RPM.GetName())
}

// Validate validates input requests of GetRPM.
func (r *GetRPMRequest) Validate() error {
	return validateResourceName(rpmRegex, RPMNameFormat, r.Name)
}

// Validate validates input requests of ListRPMs.
func (r *ListRPMsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	if err := validateDrac(r.GetDrac()); err != nil {
		return err
	}
	if r.GetDrac().GetMachine() == "" {
		return status.Errorf(codes.InvalidArgument, EmptyMachineName)
	}
	r.DracId = id
	return nil
}

func validateDrac(drac *ufspb.Drac) error {
	if drac.GetMacAddress() == "" {
		return status.Errorf(codes.InvalidArgument, "drac macAddress cannot be empty")
	}
	if !util.ValidateTags(drac.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	newMac, err := util.ParseMac(drac.GetMacAddress())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	drac.MacAddress = newMac
	return nil
}

// Validate validates input requests of UpdateDrac.
func (r *UpdateDracRequest) Validate() error {
	if r.Drac == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if r.Drac.GetMacAddress() != "" {
		newMac, err := util.ParseMac(r.Drac.GetMacAddress())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		r.Drac.MacAddress = newMac
	}
	if !util.ValidateTags(r.Drac.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(dracRegex, DracNameFormat, r.Drac.GetName())
}

// Validate validates input requests of GetDrac.
func (r *GetDracRequest) Validate() error {
	return validateResourceName(dracRegex, DracNameFormat, r.Name)
}

// Validate validates input requests of ListDracs.
func (r *ListDracsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	if r.GetSwitch().GetRack() == "" {
		return status.Errorf(codes.InvalidArgument, EmptyRackName)
	}
	if !util.ValidateTags(r.Switch.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.SwitchId = id
	return nil
}

// Validate validates input requests of UpdateSwitch.
func (r *UpdateSwitchRequest) Validate() error {
	if r.Switch == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if !util.ValidateTags(r.Switch.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(switchRegex, SwitchNameFormat, r.Switch.GetName())
}

// Validate validates input requests of GetSwitch.
func (r *GetSwitchRequest) Validate() error {
	return validateResourceName(switchRegex, SwitchNameFormat, r.Name)
}

// Validate validates input requests of ListSwitches.
func (r *ListSwitchesRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
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
	if r.Vlan.GetVlanAddress() == "" {
		return status.Errorf(codes.InvalidArgument, "Empty cidr block for vlan")
	}
	if !util.ValidateTags(r.Vlan.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.VlanId = id
	return nil
}

// Validate validates input requests of UpdateVlan.
func (r *UpdateVlanRequest) Validate() error {
	if r.Vlan == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if !util.ValidateTags(r.Vlan.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(vlanRegex, VlanNameFormat, r.Vlan.GetName())
}

// Validate validates input requests of GetVlan.
func (r *GetVlanRequest) Validate() error {
	return validateResourceName(vlanRegex, VlanNameFormat, r.Name)
}

// Validate validates input requests of ListVlans.
func (r *ListVlansRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteVlan.
func (r *DeleteVlanRequest) Validate() error {
	return validateResourceName(vlanRegex, VlanNameFormat, r.Name)
}

// Validate validates input requests of CreateAsset
func (r *CreateAssetRequest) Validate() error {
	if r.GetAsset() == nil {
		return status.Errorf(codes.InvalidArgument, "Empty asset")
	}
	name := strings.TrimSpace(r.GetAsset().GetName())
	if name == "" {
		return status.Errorf(codes.InvalidArgument, "Asset name missing")
	}
	if !assetRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument, "Invalid asset name %s", name)
	}
	if r.GetAsset().GetLocation() == nil {
		return status.Errorf(codes.InvalidArgument, "Asset location missing")
	}
	if r.GetAsset().GetLocation().GetZone() == ufspb.Zone_ZONE_UNSPECIFIED {
		return status.Errorf(codes.InvalidArgument, "Lab unspecified")
	}
	if r.GetAsset().GetLocation().GetRack() == "" {
		return status.Errorf(codes.InvalidArgument, "Rack missing")
	}
	r.GetAsset().Name = name
	return nil
}

// Validate validates input requests of UpdateAsset.
func (r *UpdateAssetRequest) Validate() error {
	if r.Asset == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	return validateResourceName(assetRegex, AssetNameFormat, r.Asset.GetName())
}

// Validate validates input requests of GetAsset.
func (r *GetAssetRequest) Validate() error {
	return validateResourceName(assetRegex, AssetNameFormat, r.Name)
}

// Validate validates input requests of ListAssets.
func (r *ListAssetsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteAsset.
func (r *DeleteAssetRequest) Validate() error {
	return validateResourceName(assetRegex, AssetNameFormat, r.Name)
}

// Validate validates input requests of GetChromeOSDeviceDataRequest.
func (r *GetChromeOSDeviceDataRequest) Validate() error {
	if r == nil {
		return status.Errorf(codes.InvalidArgument, EmptyRequest)
	}
	if r.ChromeosDeviceId == "" && r.Hostname == "" {
		return status.Errorf(codes.InvalidArgument, "Both Id and hostname are empty")
	}
	return nil
}

// Validate validates input requests of GetMachineLSEDeployment.
func (r *GetMachineLSEDeploymentRequest) Validate() error {
	return validateResourceName(machineLSEDeploymentRegex, MachineLSEDeploymentNameFormat, r.Name)
}

// Validate validates input requests of UpdateMachineLSEDeploymentRequest.
func (r *UpdateMachineLSEDeploymentRequest) Validate() error {
	if r == nil {
		return status.Errorf(codes.InvalidArgument, EmptyRequest)
	}
	if r.GetMachineLseDeployment().GetSerialNumber() == "" {
		return status.Errorf(codes.InvalidArgument, "cannot update a deployment record with empty serial number")
	}
	r.MachineLseDeployment.Hostname = strings.TrimSpace(r.MachineLseDeployment.Hostname)
	r.MachineLseDeployment.SerialNumber = strings.TrimSpace(r.MachineLseDeployment.SerialNumber)
	return nil
}

// Validate validates ListMachineLSEDeploymentsRequest.
func (r *ListMachineLSEDeploymentsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of CreateCachingService.
func (r *CreateCachingServiceRequest) Validate() error {
	if r.CachingService == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.CachingServiceId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !HostnameRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, fmt.Sprintf("name: %s", InvalidHostname))
	}
	if len(r.GetCachingService().GetServingSubnets()) == 0 {
		return status.Error(codes.InvalidArgument, "Empty serving subnets.")
	}
	switch n := r.GetCachingService().GetPrimaryNode(); {
	case n == "":
		return status.Errorf(codes.InvalidArgument, "Empty primary node name.")
	case !HostnameRegex.MatchString(n):
		return status.Errorf(codes.InvalidArgument, fmt.Sprintf("primaryNode: %s", InvalidHostname))
	}
	switch n := r.GetCachingService().GetSecondaryNode(); {
	case n == "":
		return status.Errorf(codes.InvalidArgument, "Empty secondary node name.")
	case !HostnameRegex.MatchString(n):
		return status.Errorf(codes.InvalidArgument, fmt.Sprintf("secondaryNode: %s", InvalidHostname))
	}
	r.CachingServiceId = id
	return nil
}

// Validate validates input requests of UpdateCachingService.
func (r *UpdateCachingServiceRequest) Validate() error {
	if r.CachingService == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	name := strings.TrimSpace(r.GetCachingService().GetName())
	if name == "" {
		return status.Errorf(codes.InvalidArgument, EmptyName)
	}
	if !cachingServiceRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument, CachingServiceNameFormat)
	}

	if n := r.GetCachingService().GetPrimaryNode(); n != "" && !HostnameRegex.MatchString(n) {
		return status.Errorf(codes.InvalidArgument, fmt.Sprintf("primaryNode: %s", InvalidHostname))
	}
	if n := r.GetCachingService().GetSecondaryNode(); n != "" && !HostnameRegex.MatchString(n) {
		return status.Errorf(codes.InvalidArgument, fmt.Sprintf("secondaryNode: %s", InvalidHostname))
	}
	return nil
}

// Validate validates input requests of GetCachingService.
func (r *GetCachingServiceRequest) Validate() error {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return status.Errorf(codes.InvalidArgument, EmptyName)
	}
	if !cachingServiceRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument, CachingServiceNameFormat)
	}
	return nil
}

// Validate validates input requests of ListCachingServices.
func (r *ListCachingServicesRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate validates input requests of DeleteCachingService.
func (r *DeleteCachingServiceRequest) Validate() error {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return status.Errorf(codes.InvalidArgument, EmptyName)
	}
	if !cachingServiceRegex.MatchString(name) {
		return status.Errorf(codes.InvalidArgument, CachingServiceNameFormat)
	}
	return nil
}

// Validate validates input requests of CreateSchedulingUnit.
func (r *CreateSchedulingUnitRequest) Validate() error {
	if r.SchedulingUnit == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	id := strings.TrimSpace(r.SchedulingUnitId)
	if id == "" {
		return status.Errorf(codes.InvalidArgument, EmptyID)
	}
	if !IDRegex.MatchString(id) {
		return status.Errorf(codes.InvalidArgument, InvalidCharacters)
	}
	if !util.ValidateTags(r.SchedulingUnit.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	r.SchedulingUnitId = id
	return nil
}

// Validate validates input requests of UpdateSchedulingUnit.
func (r *UpdateSchedulingUnitRequest) Validate() error {
	if r.SchedulingUnit == nil {
		return status.Errorf(codes.InvalidArgument, NilEntity)
	}
	if !util.ValidateTags(r.SchedulingUnit.GetTags()) {
		return status.Errorf(codes.InvalidArgument, InvalidTags)
	}
	return validateResourceName(schedulingUnitRegex, SchedulingUnitNameFormat, r.SchedulingUnit.GetName())
}

// Validate validates input requests of GetSchedulingUnit.
func (r *GetSchedulingUnitRequest) Validate() error {
	return validateResourceName(schedulingUnitRegex, SchedulingUnitNameFormat, r.Name)
}

// Validate validates input requests of ListSchedulingUnits.
func (r *ListSchedulingUnitsRequest) Validate() error {
	if err := ValidateFilter(r.Filter); err != nil {
		return err
	}
	return validatePageSize(r.PageSize)
}

// Validate the MachineLSE rename request.
func (r *RenameMachineLSERequest) Validate() error {
	if r.Name == "" {
		return status.Errorf(codes.InvalidArgument, EmptyName)
	}
	if r.NewName == "" {
		return status.Errorf(codes.InvalidArgument, "Missing new name to rename")
	}
	return nil
}

// Validate the Asset rename request.
func (r *RenameAssetRequest) Validate() error {
	if r.Name == "" {
		return status.Errorf(codes.InvalidArgument, EmptyName)
	}
	if r.NewName == "" {
		return status.Errorf(codes.InvalidArgument, "Missing new name to rename")
	}
	return nil
}

// Validate validates input requests of DeleteSchedulingUnit.
func (r *DeleteSchedulingUnitRequest) Validate() error {
	return validateResourceName(schedulingUnitRegex, SchedulingUnitNameFormat, r.Name)
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

// Validate validates input requests of UpdateConfigBundleRequest.
func (r *UpdateConfigBundleRequest) Validate() error {
	var cb payload.ConfigBundle
	if err := proto.Unmarshal(r.ConfigBundle, &cb); err != nil {
		return err
	}
	if cb.GetDesignList()[0].GetProgramId().GetValue() == "" {
		return status.Errorf(codes.InvalidArgument, "cannot update a ConfigBundle with empty program id")
	}
	if cb.GetDesignList()[0].GetId().GetValue() == "" {
		return status.Errorf(codes.InvalidArgument, "cannot update a ConfigBundle with empty design id")
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
			return status.Errorf(codes.InvalidArgument, fmt.Sprintf("%s ('%s')", InvalidCharacters, v))
		}
	}
	return nil
}

// ValidateFilter validates if the filter fomrat is correct
func ValidateFilter(filter string) error {
	if filter != "" {
		filter = fmt.Sprintf(strings.Replace(filter, " ", "", -1))
		if !FilterRegex.MatchString(filter) {
			return status.Errorf(codes.InvalidArgument, InvalidFilterFormat)
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

// Validate validates input requests of GetDeviceDataRequest.
func (r *GetDeviceDataRequest) Validate() error {
	if r == nil {
		return status.Errorf(codes.InvalidArgument, EmptyRequest)
	}
	if r.DeviceId == "" && r.Hostname == "" {
		return status.Errorf(codes.InvalidArgument, "Both Id and hostname are empty")
	}
	return nil
}
