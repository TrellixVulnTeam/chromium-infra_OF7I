// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/model/caching"
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

//GetVMResource returns a Resource with VMEntity
func GetVMResource(vmID string) *Resource {
	return &Resource{
		Kind: inventory.VMKind,
		ID:   vmID,
		Entity: &inventory.VMEntity{
			ID: vmID,
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

// GetAssetResource returns a Resource with AssetEntity
func GetAssetResource(assetID string) *Resource {
	return &Resource{
		Kind: registration.AssetKind,
		ID:   assetID,
		Entity: &registration.AssetEntity{
			Name: assetID,
		},
	}
}

// GetCachingServiceResource returns a Resource with CSEntity.
func GetCachingServiceResource(cachingServiceID string) *Resource {
	return &Resource{
		Kind: caching.CachingServiceKind,
		ID:   cachingServiceID,
		Entity: &caching.CSEntity{
			ID: cachingServiceID,
		},
	}
}

// GetSchedulingUnitResource returns a Resource with SchedulingUnitEntity.
func GetSchedulingUnitResource(schedulingUnitID string) *Resource {
	return &Resource{
		Kind: inventory.SchedulingUnitKind,
		ID:   schedulingUnitID,
		Entity: &inventory.SchedulingUnitEntity{
			ID: schedulingUnitID,
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
	var NotFound = false
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
	var alreadyExists = false
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
	// Sort by port and compare. Stable sort is not needed as ports are unique.
	sort.Slice(a, func(i, j int) bool { return a[i].GetServoPort() > a[j].GetServoPort() })
	sort.Slice(b, func(i, j int) bool { return b[i].GetServoPort() > b[j].GetServoPort() })
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

func mergeTags(oldTags, newTags []string) []string {
	// Clean up all tags if new input tags are empty
	if newTags == nil || len(newTags) == 0 {
		return nil
	}
	for _, tag := range newTags {
		if tag != "" {
			oldTags = append(oldTags, tag)
		}
	}
	return oldTags
}

// TODO: merge this with mergeTags
func mergeIPs(olds, news []string) []string {
	// Clean up all tags if new input tags are empty
	if len(news) == 0 {
		return nil
	}
	for _, s := range news {
		if s != "" {
			olds = append(olds, s)
		}
	}
	return olds
}

func mergeZones(oldz, newz []ufspb.Zone) []ufspb.Zone {
	// Clean up all tags if new input zones are empty
	if len(newz) == 0 {
		return nil
	}
	zoneMap := make(map[string]ufspb.Zone, 0)
	for _, z := range append(oldz, newz...) {
		if z != ufspb.Zone_ZONE_UNSPECIFIED {
			zoneMap[z.String()] = z
		}
	}
	res := make([]ufspb.Zone, 0, len(zoneMap))
	for _, v := range zoneMap {
		res = append(res, v)
	}
	return res
}

func validateMacAddress(ctx context.Context, assetName, macAddr string) error {
	if macAddr == "" {
		return nil
	}
	nics, err := registration.QueryNicByPropertyName(ctx, "mac_address", macAddr, true)
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if nic.GetName() == assetName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "mac_address %s is already occupied by nic %s", macAddr, nic.GetName())
	}
	dracs, err := registration.QueryDracByPropertyName(ctx, "mac_address", macAddr, true)
	if err != nil {
		return err
	}
	for _, drac := range dracs {
		if drac.GetName() == assetName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "mac_address %s is already occupied by drac %s", macAddr, drac.GetName())
	}
	kvms, err := registration.QueryKVMByPropertyName(ctx, "mac_address", macAddr, true)
	if err != nil {
		return err
	}
	for _, kvm := range kvms {
		if kvm.GetName() == assetName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "mac_address %s is already occupied by KVM %s", macAddr, kvm.GetName())
	}
	rpms, err := registration.QueryRPMByPropertyName(ctx, "mac_address", macAddr, true)
	if err != nil {
		return err
	}
	for _, rpm := range rpms {
		if rpm.GetName() == assetName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "mac_address %s is already occupied by RPM %s", macAddr, rpm.GetName())
	}
	return nil
}

func validateDracSwitchPort(ctx context.Context, dracName, machineName string, switchInterface *ufspb.SwitchInterface) error {
	if switchInterface.GetSwitch() == "" || switchInterface.GetPortName() == "" || machineName == "" {
		return nil
	}
	switchID := switchInterface.GetSwitch()
	switchPort := switchInterface.GetPortName()
	nics, _, err := ListNics(ctx, -1, "", fmt.Sprintf("switch=%s & switchPort=%s", switchID, switchPort), false)
	if err != nil {
		return err
	}
	for _, nic := range nics {
		// Nic and drac can share the same switch port within a machine.
		if nic.GetMachine() == machineName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "switch port %s of %s is already occupied by nic %s", switchPort, switchID, nic.GetName())
	}
	dracs, _, err := ListDracs(ctx, -1, "", fmt.Sprintf("switch=%s & switchPort=%s", switchID, switchPort), false)
	if err != nil {
		return err
	}
	for _, drac := range dracs {
		if drac.GetName() == dracName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "switch port %s of %s is already occupied by drac %s", switchPort, switchID, drac.GetName())
	}
	return nil
}

func validateNicSwitchPort(ctx context.Context, nicName, machineName string, switchInterface *ufspb.SwitchInterface) error {
	if switchInterface.GetSwitch() == "" || switchInterface.GetPortName() == "" || machineName == "" {
		return nil
	}
	switchID := switchInterface.GetSwitch()
	switchPort := switchInterface.GetPortName()
	nics, _, err := ListNics(ctx, -1, "", fmt.Sprintf("switch=%s & switchPort=%s", switchID, switchPort), false)
	if err != nil {
		return err
	}
	for _, nic := range nics {
		if nic.GetName() == nicName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "switch port %s of %s is already occupied by nic %s", switchPort, switchID, nic.GetName())
	}
	dracs, _, err := ListDracs(ctx, -1, "", fmt.Sprintf("switch=%s & switchPort=%s", switchID, switchPort), false)
	if err != nil {
		return err
	}
	for _, drac := range dracs {
		// Nic and drac can share the same switch port within a machine.
		if drac.GetMachine() == machineName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "switch port %s of %s is already occupied by drac %s", switchPort, switchID, drac.GetName())
	}
	return nil
}

func validateKVMPort(ctx context.Context, assetName string, kvmInterface *ufspb.KVMInterface) error {
	if kvmInterface.GetKvm() == "" || kvmInterface.GetPortName() == "" {
		return nil
	}
	kvmID := kvmInterface.GetKvm()
	kvmPort := kvmInterface.GetPortName()
	machines, _, err := ListMachines(ctx, -1, "", fmt.Sprintf("kvm=%s & kvmport=%s", kvmID, kvmPort), false, false)
	if err != nil {
		return err
	}
	for _, machine := range machines {
		if machine.GetName() == assetName {
			continue
		}
		return status.Errorf(codes.InvalidArgument, "kvm port %s of %s is already occupied by machine %s", kvmPort, kvmID, machine.GetName())
	}
	return nil
}

func validateReservedIPs(ctx context.Context, vlan *ufspb.Vlan) error {
	if len(vlan.GetReservedIps()) == 0 {
		return nil
	}
	ips, _, _, _, err := util.ParseVlan(vlan.GetName(), vlan.GetVlanAddress(), vlan.GetFreeStartIpv4Str(), vlan.GetFreeEndIpv4Str())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Invalid %s: %s", vlan.GetVlanAddress(), err.Error())
	}
	ipMap := make(map[string]bool)
	for _, ip := range ips {
		ipMap[ip.GetIpv4Str()] = true
	}
	for _, ip := range vlan.GetReservedIps() {
		if ip == "" {
			continue
		}
		if !ipMap[ip] {
			return status.Errorf(codes.InvalidArgument, "ip %s doesn't belong to vlan %s", ip, vlan.GetVlanAddress())
		}
		dhcps, err := configuration.QueryDHCPConfigByPropertyName(ctx, "ipv4", ip)
		if err != nil {
			return err
		}
		for _, dhcp := range dhcps {
			return status.Errorf(codes.InvalidArgument, "ip %s is already occupied by hostname %s", dhcp.GetIp(), dhcp.GetHostname())
		}
	}
	return nil
}

func resetStateFilter(filterMap map[string][]interface{}) map[string][]interface{} {
	if v, ok := filterMap["state"]; ok {
		s := util.ToUFSState(fmt.Sprintf("%s", v[0]))
		filterMap["state"] = []interface{}{s.String()}
	}
	return filterMap
}

func resetOSFilter(filterMap map[string][]interface{}) map[string][]interface{} {
	if v, ok := filterMap["os"]; ok {
		for i, id := range v {
			v[i] = strings.ToLower(id.(string))
		}
		filterMap["os"] = v
	}
	return filterMap
}

func resetZoneFilter(filterMap map[string][]interface{}) map[string][]interface{} {
	if v, ok := filterMap["zone"]; ok {
		for i, vz := range v {
			v[i] = util.ToUFSZone(fmt.Sprintf("%s", vz)).String()
		}
		filterMap["zone"] = v
	}
	return filterMap
}

func resetAssetTypeFilter(filterMap map[string][]interface{}) map[string][]interface{} {
	if v, ok := filterMap["type"]; ok {
		for i, vt := range v {
			v[i] = util.ToAssetType(fmt.Sprintf("%s", vt)).String()
			fmt.Println(v[i])
		}
		filterMap["type"] = v
	}
	return filterMap
}

func resetSchedulingUnitTypeFilter(filterMap map[string][]interface{}) map[string][]interface{} {
	if v, ok := filterMap["type"]; ok {
		for i, vt := range v {
			v[i] = util.ToSchedulingUnitType(fmt.Sprintf("%s", vt)).String()
			fmt.Println(v[i])
		}
		filterMap["type"] = v
	}
	return filterMap
}

func resetDeviceTypeFilter(filterMap map[string][]interface{}) map[string][]interface{} {
	if v, ok := filterMap[util.DeviceTypeFilterName]; ok {
		for i, vz := range v {
			v[i] = util.ToUFSDeviceType(fmt.Sprintf("%s", vz)).String()
		}
		filterMap[util.DeviceTypeFilterName] = v
	}
	return filterMap
}

func parseIntTypeFilter(filterMap map[string][]interface{}, filterName string) (map[string][]interface{}, error) {
	if v, ok := filterMap[filterName]; ok {
		for i, vz := range v {
			intNum, err := strconv.ParseInt(fmt.Sprintf("%s", vz), 10, 32)
			if err != nil {
				return filterMap, err
			}
			v[i] = intNum
		}
		filterMap[filterName] = v
	}
	return filterMap, nil
}
