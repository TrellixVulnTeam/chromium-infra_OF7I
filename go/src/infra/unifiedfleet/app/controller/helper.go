// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	chromeosLab "infra/unifiedfleet/api/v1/proto/chromeos/lab"
	"infra/unifiedfleet/app/model/configuration"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

//Generalized error messages for resources in the system
var (
	NotFoundErrorMessage      string = "There is no %s with %sID %s in the system.\n"
	AlreadyExistsErrorMessage string = "%s %s already exists in the system.\n"
)

// Resource contains the fleet entity to be checked and the ID and Kind
type Resource struct {
	Kind   string
	ID     string
	Entity ufsds.FleetEntity
}

type getFieldFunc func(string) (string, error)

// GetChromePlatformResource returns a Resource with ChromePlatformEntity
func GetChromePlatformResource(chromePlatformID string) *Resource {
	return &Resource{
		Kind: configuration.ChromePlatformKind,
		ID:   chromePlatformID,
		Entity: &configuration.ChromePlatformEntity{
			ID: chromePlatformID,
		},
	}
}

// GetMachineLSEProtoTypeResource returns a Resource with MachineLSEProtoTypeEntity
func GetMachineLSEProtoTypeResource(machineLSEProtoTypeID string) *Resource {
	return &Resource{
		Kind: configuration.MachineLSEPrototypeKind,
		ID:   machineLSEProtoTypeID,
		Entity: &configuration.MachineLSEPrototypeEntity{
			ID: machineLSEProtoTypeID,
		},
	}
}

// GetRackLSEProtoTypeResource returns a Resource with RackLSEProtoTypeEntity
func GetRackLSEProtoTypeResource(rackLSEProtoTypeID string) *Resource {
	return &Resource{
		Kind: configuration.RackLSEPrototypeKind,
		ID:   rackLSEProtoTypeID,
		Entity: &configuration.RackLSEPrototypeEntity{
			ID: rackLSEProtoTypeID,
		},
	}
}

//GetMachineResource returns a Resource with MachineEntity
func GetMachineResource(machineID string) *Resource {
	return &Resource{
		Kind: registration.MachineKind,
		ID:   machineID,
		Entity: &registration.MachineEntity{
			ID: machineID,
		},
	}
}

//GetMachineLSEResource returns a Resource with MachineLSEEntity
func GetMachineLSEResource(machinelseID string) *Resource {
	return &Resource{
		Kind: inventory.MachineLSEKind,
		ID:   machinelseID,
		Entity: &inventory.MachineLSEEntity{
			ID: machinelseID,
		},
	}
}

//GetRackResource returns a Resource with RackEntity
func GetRackResource(rackID string) *Resource {
	return &Resource{
		Kind: registration.RackKind,
		ID:   rackID,
		Entity: &registration.RackEntity{
			ID: rackID,
		},
	}
}

//GetKVMResource returns a Resource with KVMEntity
func GetKVMResource(kvmID string) *Resource {
	return &Resource{
		Kind: registration.KVMKind,
		ID:   kvmID,
		Entity: &registration.KVMEntity{
			ID: kvmID,
		},
	}
}

// GetRPMResource returns a Resource with RPMEntity
func GetRPMResource(rpmID string) *Resource {
	return &Resource{
		Kind: registration.RPMKind,
		ID:   rpmID,
		Entity: &registration.RPMEntity{
			ID: rpmID,
		},
	}
}

// GetSwitchResource returns a Resource with SwitchEntity
func GetSwitchResource(switchID string) *Resource {
	return &Resource{
		Kind: registration.SwitchKind,
		ID:   switchID,
		Entity: &registration.SwitchEntity{
			ID: switchID,
		},
	}
}

// GetNicResource returns a Resource with NicEntity
func GetNicResource(nicID string) *Resource {
	return &Resource{
		Kind: registration.NicKind,
		ID:   nicID,
		Entity: &registration.NicEntity{
			ID: nicID,
		},
	}
}

// GetDracResource returns a Resource with DracEntity
func GetDracResource(dracID string) *Resource {
	return &Resource{
		Kind: registration.DracKind,
		ID:   dracID,
		Entity: &registration.DracEntity{
			ID: dracID,
		},
	}
}

// GetVlanResource returns a Resource with VlanEntity
func GetVlanResource(vlanID string) *Resource {
	return &Resource{
		Kind: configuration.VlanKind,
		ID:   vlanID,
		Entity: &configuration.VlanEntity{
			ID: vlanID,
		},
	}
}

// ResourceExist checks if the given resources exists in the datastore
//
// Returns error if any one resource does not exist in the system.
// Appends error messages to the the given error message for resources
// that does not exist in the datastore.
func ResourceExist(ctx context.Context, resources []*Resource, errorMsg *strings.Builder) error {
	if len(resources) == 0 {
		return nil
	}
	if errorMsg == nil {
		errorMsg = &strings.Builder{}
	}
	var NotFound bool = false
	checkEntities := make([]ufsds.FleetEntity, 0, len(resources))
	for _, resource := range resources {
		logging.Debugf(ctx, "checking resource existence: %#v", resource)
		checkEntities = append(checkEntities, resource.Entity)
	}
	exists, err := ufsds.Exists(ctx, checkEntities)
	if err == nil {
		for i := range checkEntities {
			if !exists[i] {
				NotFound = true
				errorMsg.WriteString(fmt.Sprintf(NotFoundErrorMessage, resources[i].Kind, resources[i].Kind, resources[i].ID))
			}
		}
	} else {
		logging.Errorf(ctx, "Failed to check existence: %s", err)
		return status.Errorf(codes.Internal, err.Error())
	}
	if NotFound {
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// resourceAlreadyExists checks if the given resources already exists in the datastore
//
// Returns error if any of the resource already exists.
// Appends error messages to the the given error message for resources
// that already exist in the datastore.
func resourceAlreadyExists(ctx context.Context, resources []*Resource, errorMsg *strings.Builder) error {
	if len(resources) == 0 {
		return nil
	}
	if errorMsg == nil {
		errorMsg = &strings.Builder{}
	}
	var alreadyExists bool = false
	checkEntities := make([]ufsds.FleetEntity, 0, len(resources))
	for _, resource := range resources {
		logging.Debugf(ctx, "checking resource existence: %#v", resource)
		checkEntities = append(checkEntities, resource.Entity)
	}
	exists, err := ufsds.Exists(ctx, checkEntities)
	if err == nil {
		for i := range checkEntities {
			if exists[i] {
				alreadyExists = true
				errorMsg.WriteString(fmt.Sprintf(AlreadyExistsErrorMessage, resources[i].Kind, resources[i].ID))
			}
		}
	} else {
		logging.Errorf(ctx, "Failed to check existence: %s", err)
		return status.Errorf(codes.Internal, err.Error())
	}
	if alreadyExists {
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// testServoEq checks if the 2 slice of servos are equal
func testServoEq(a, b []*chromeosLab.Servo) bool {
	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !proto.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func deleteByPage(ctx context.Context, toDelete []string, pageSize int, deletFunc func(ctx context.Context, resourceNames []string) *ufsds.OpResults) *ufsds.OpResults {
	var allRes ufsds.OpResults
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(toDelete))
		res := deletFunc(ctx, toDelete[i:end])
		allRes = append(allRes, *res...)
		if i+pageSize >= len(toDelete) {
			break
		}
	}
	return &allRes
}

// TODO(eshwarn) : Use pattern matching instead of strings.split and add unit test
func getFilterMap(filter string, f getFieldFunc) (map[string][]interface{}, error) {
	filterMap := make(map[string][]interface{})
	filter = strings.TrimSpace(filter)
	conditions := strings.Split(filter, "&")
	if len(conditions) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid filter format %s - Filter Egs: \"machine=cx-1,cx-2 & machinelseprototype=mx-1\"", filter)
	}
	for _, condition := range conditions {
		condition = strings.TrimSpace(condition)
		keyValue := strings.Split(condition, "=")
		if len(keyValue) < 2 {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid filter name format %s - Filter Egs: \"machine=cx-1,cx-2\"", condition)
		}
		field, err := f(keyValue[0])
		if err != nil {
			return nil, err
		}
		s := strings.Split(keyValue[1], ",")
		if len(s) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid filter value format %s- Filter Egs: \"machine=cx-1,cx-2\"", keyValue[1])
		}
		values := make([]interface{}, len(s))
		for i, v := range s {
			values[i] = strings.TrimSpace(v)
		}
		filterMap[field] = values
	}
	return filterMap, nil
}

// deleteDHCPHelper deletes ip configs for a given hostname
//
// Can be used in a transaction
func deleteDHCPHelper(ctx context.Context, hostname string) error {
	dhcp, err := configuration.GetDHCPConfig(ctx, hostname)
	if util.IsInternalError(err) {
		return errors.Annotate(err, "Fail to query dhcpHost").Err()
	}
	if err == nil && dhcp != nil {
		if err := deleteHostHelper(ctx, dhcp); err != nil {
			return err
		}
	}
	return nil
}

// Delete all ip-related configs
//
// Can be used in a transaction
func deleteHostHelper(ctx context.Context, dhcp *ufspb.DHCPConfig) error {
	logging.Debugf(ctx, "Found existing dhcp configs for host %s", dhcp.GetHostname())
	logging.Debugf(ctx, "Deleting dhcp %s (%s)", dhcp.GetHostname(), dhcp.GetIp())
	if err := configuration.DeleteDHCP(ctx, dhcp.GetHostname()); err != nil {
		return errors.Annotate(err, fmt.Sprintf("Fail to delete dhcp: hostname %q, ip %q", dhcp.GetHostname(), dhcp.GetIp())).Err()
	}
	ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
	if err != nil {
		return errors.Annotate(err, fmt.Sprintf("Fail to query ip by ipv4 str: %q", dhcp.GetIp())).Err()
	}
	if ips == nil {
		return nil
	}
	ips[0].Occupied = false
	logging.Debugf(ctx, "Update ip %s to non-occupied", ips[0].GetIpv4Str())
	if _, err := configuration.BatchUpdateIPs(ctx, ips); err != nil {
		return errors.Annotate(err, fmt.Sprintf("Fail to update ip: %q (ipv4: %q, vlan %q)", ips[0].GetId(), ips[0].GetIpv4Str(), ips[0].GetVlan())).Err()
	}
	return nil
}

// Find free ip and update ip-related configs
//
// Can be used in a transaction
func addHostHelper(ctx context.Context, vlanName, hostName, macAddress string) error {
	ips, err := getFreeIP(ctx, vlanName, 1)
	if err != nil {
		return errors.Annotate(err, "Failed to find new IP to for host %s", hostName).Err()
	}
	if ips[0].GetIpv4Str() == "" {
		return errors.New(fmt.Sprintf("No empty ip is found. Found ip: %q, vlan %q", ips[0].GetId(), ips[0].GetVlan()))
	}
	logging.Debugf(ctx, "Get free ip %s", ips[0].GetIpv4Str())
	ips[0].Occupied = true
	if _, err := configuration.BatchUpdateIPs(ctx, ips); err != nil {
		return errors.Annotate(err, "Failed to update IP %s (%s)", ips[0].GetId(), ips[0].GetIpv4Str()).Err()
	}
	if _, err := configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{
		{
			Hostname:   hostName,
			Ip:         ips[0].GetIpv4Str(),
			MacAddress: macAddress,
		},
	}); err != nil {
		return errors.Annotate(err, "Failed to update dhcp configs for host %s and mac address %s", hostName, macAddress).Err()
	}
	return nil
}
