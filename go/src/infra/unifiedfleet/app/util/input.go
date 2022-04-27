// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	descriptorpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/info"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc/metadata"

	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
)

const (
	// AssetCollection refers to the prefix of the corresponding resource.
	AssetCollection string = "assets"
	// MachineCollection refers to the prefix of the corresponding resource.
	MachineCollection string = "machines"
	// RackCollection refers to the prefix of the corresponding resource.
	RackCollection string = "racks"
	// VMCollection refers to the prefix of the corresponding resource.
	VMCollection string = "vms"
	// ChromePlatformCollection refers to the prefix of the corresponding resource.
	ChromePlatformCollection string = "chromeplatforms"
	// MachineLSECollection refers to the prefix of the corresponding resource.
	MachineLSECollection string = "machineLSEs"
	// MachineLSEDeploymentCollection refers to the prefix of the corresponding resource.
	MachineLSEDeploymentCollection string = "machineLSEDeployments"
	// HostCollection refers to the prefix of the corresponding resource.
	HostCollection string = "hosts"
	// RackLSECollection refers to the prefix of the corresponding resource.
	RackLSECollection string = "rackLSEs"
	// NicCollection refers to the prefix of the corresponding resource.
	NicCollection string = "nics"
	// KVMCollection refers to the prefix of the corresponding resource.
	KVMCollection string = "kvms"
	// RPMCollection refers to the prefix of the corresponding resource.
	RPMCollection string = "rpms"
	// DracCollection refers to the prefix of the corresponding resource.
	DracCollection string = "dracs"
	// SwitchCollection refers to the prefix of the corresponding resource.
	SwitchCollection string = "switches"
	// VlanCollection refers to the prefix of the corresponding resource.
	VlanCollection string = "vlans"
	// MachineLSEPrototypeCollection refers to the prefix of the corresponding resource.
	MachineLSEPrototypeCollection string = "machineLSEPrototypes"
	// RackLSEPrototypeCollection refers to the prefix of the corresponding resource.
	RackLSEPrototypeCollection string = "rackLSEPrototypes"
	// DHCPCollection refers to the prefix of the dhcp config id in change history
	DHCPCollection string = "dhcps"
	// IPCollection refers to the prefix of the ip id in change history
	IPCollection string = "ips"
	// StateCollection refers to the prefix of the states id in change history
	StateCollection string = "states"
	// CachingServiceCollection refers to the prefix of the CachingService.
	CachingServiceCollection string = "cachingservices"
	// DutStateCollection refers to the prefix of the DutStates id in change history
	DutStateCollection string = "dutstates"
	// SchedulingUnitCollection refers to the prefix of the SchedulingUnit.
	SchedulingUnitCollection string = "schedulingunits"

	// DefaultImporter refers to the user of the cron job importer
	DefaultImporter string = "crimson-importer"

	defaultPageSize int32 = 100
	// MaxPageSize maximum page size for list operations
	MaxPageSize int32 = 1000

	// NoHostPrefix is the prefix string for generating a fake non-existing hostname
	NoHostPrefix string = "no-host-yet-"
)

var collectionsRe = regexp.MustCompile(`\/{[a-zA-Z0-9]*}$`)

// Filter names for indexed properties in datastore for different entities
var (
	ZoneFilterName                 string = "zone"
	RackFilterName                 string = "rack"
	MachineFilterName              string = "machine"
	HostFilterName                 string = "host"
	NicFilterName                  string = "nic"
	DracFilterName                 string = "drac"
	KVMFilterName                  string = "kvm"
	KVMPortFilterName              string = "kvmport"
	MacAddressFilterName           string = "mac"
	RPMFilterName                  string = "rpm"
	RPMPortFilterName              string = "rpmport"
	SwitchFilterName               string = "switch"
	SwitchPortFilterName           string = "switchport"
	ServoFilterName                string = "servo"
	ServoTypeFilterName            string = "servotype"
	TagFilterName                  string = "tag"
	ChromePlatformFilterName       string = "platform"
	MachinePrototypeFilterName     string = "machineprototype"
	RackPrototypeFilterName        string = "rackprototype"
	VlanFilterName                 string = "vlan"
	StateFilterName                string = "state"
	IPV4FilterName                 string = "ipv4"
	IPV4StringFilterName           string = "ipv4str"
	OccupiedFilterName             string = "occupied"
	ManufacturerFilterName         string = "man"
	FreeVMFilterName               string = "free"
	ResourceTypeFilterName         string = "resourcetype"
	OSVersionFilterName            string = "osversion"
	OSFilterName                   string = "os"
	VirtualDatacenterFilterName    string = "vdc"
	ModelFilterName                string = "model"
	BuildTargetFilterName          string = "target"
	BoardFilterName                string = "board"
	DeviceTypeFilterName           string = "devicetype"
	PhaseFilterName                string = "phase"
	AssetTypeFilterName            string = "assettype"
	SubnetsFilterName              string = "subnets"
	PoolsFilterName                string = "pools"
	DeploymentIdentifierFilterName string = "deploymentidentifier"
	DeploymentEnvFilterName        string = "deploymentenv"
	MachineLSEsFilterName          string = "duts"
	TypeFilterName                 string = "type"
	SerialNumberFilterName         string = "serialnumber"
	BbnumFilterName                string = "bbnum"
	AssetTagFilterName             string = "assettag"
)

const separator string = "/"

// Namespace namespace to be set by clients in context metadata
// This will be used to set the actual datastore namespace in the context
var (
	// OSNamespace os namespace to be set in client context metadata. OS data is stored in os namespace in the datastore.
	OSNamespace = "os"
	// BrowserNamespace browser namespace to be set in client context metadata. Browser data is stored in default namespace in the datastore.
	BrowserNamespace = "browser"
	//Namespace key in the incoming context metadata
	Namespace = "namespace"
)

// ClientToDatastoreNamespace refers a map between client namespace(set in context metadata) to actual datastore namespace
var ClientToDatastoreNamespace = map[string]string{
	BrowserNamespace: "",          // browser data is stored in default namespace
	OSNamespace:      OSNamespace, // os data in os namespace
}

// ValidClientNamespaceStr returns a valid str list for client namespace(set in incoming context metadata) strings.
func ValidClientNamespaceStr() []string {
	ks := make([]string, 0, len(ClientToDatastoreNamespace))
	for k := range ClientToDatastoreNamespace {
		ks = append(ks, k)
	}
	return ks
}

// IsClientNamespace checks if a string refers to a valid client namespace.
func IsClientNamespace(namespace string) bool {
	_, ok := ClientToDatastoreNamespace[namespace]
	return ok
}

// SetupDatastoreNamespace sets the datastore namespace in the context to access the correct namespace in the datastore
func SetupDatastoreNamespace(ctx context.Context, namespace string) (context.Context, error) {
	return info.Namespace(ctx, namespace)
}

// GetDatastoreNamespace returns the namespace used in context
func GetDatastoreNamespace(ctx context.Context) string {
	return info.GetNamespace(ctx)
}

// GetIncomingCtxNamespace parses namespace in incoming context passed to UFS
//
// Only when user specify namespace as OSNamespace, the returned namespace
// is OSNamespace. Any other case will cause namespace=BrowserNamespace.
func GetIncomingCtxNamespace(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return BrowserNamespace
	}
	namespace, ok := md[Namespace]
	if !ok {
		return BrowserNamespace
	}
	return namespace[0]
}

// GetNamespaceFromCtx parses namespace from ctx, either a normal context
// or an incoming context.
func GetNamespaceFromCtx(ctx context.Context) string {
	outNs := GetIncomingCtxNamespace(ctx)
	datastoreNs := GetDatastoreNamespace(ctx)
	if outNs == OSNamespace || datastoreNs == OSNamespace {
		return OSNamespace
	}
	return BrowserNamespace
}

// GetPageSize gets the correct page size for List pagination
func GetPageSize(pageSize int32) int32 {
	switch {
	case pageSize == 0:
		return defaultPageSize
	case pageSize > MaxPageSize:
		return MaxPageSize
	default:
		return pageSize
	}
}

// GetResourcePrefix gets the resource prefix given to the proto message.
//
// Returns the resource prefix for a given proto message.
// See also: https://blog.golang.org/protobuf-apiv2
func GetResourcePrefix(message proto.Message) (string, error) {
	m := proto.MessageReflect(message)
	x, ok := m.Descriptor().Options().(*descriptorpb.MessageOptions)
	if !ok {
		return "", errors.Reason("Unable to read Message Options").Err()
	}
	y, _ := proto.GetExtension(x, annotations.E_Resource)
	z, ok := y.(*annotations.ResourceDescriptor)
	if !ok {
		return "", errors.Reason("Resource descriptor not found in proto message").Err()
	}
	prefix := collectionsRe.ReplaceAllString(z.Pattern[0], "")
	return prefix, nil
}

// FormatInputNames formats a given array of resource names
func FormatInputNames(names []string) []string {
	var res []string
	for _, n := range names {
		if n != "" {
			res = append(res, RemovePrefix(n))
		}
	}
	return res
}

// FormatDHCPHostname formats a name which will be a dhcp host
func FormatDHCPHostname(old string) string {
	return strings.ToLower(old)
}

// FormatDHCPHostnames formats a given array of resource names which could be used as dhcp hostnames
func FormatDHCPHostnames(names []string) []string {
	for i, n := range names {
		names[i] = FormatDHCPHostname(n)
	}
	return names
}

// FormatDeploymentRecord initialize a MachineLSE deployment record object by hostname and given serial number.
func FormatDeploymentRecord(hostname, serialNumber string) *ufspb.MachineLSEDeployment {
	if hostname == "" {
		hostname = GetHostnameWithNoHostPrefix(serialNumber)
	}
	return &ufspb.MachineLSEDeployment{
		Hostname:             hostname,
		SerialNumber:         serialNumber,
		DeploymentIdentifier: "",
		ConfigsToPush:        nil,
	}
}

// GetHostnameWithNoHostPrefix generates a hostname with NoHostPrefix.
func GetHostnameWithNoHostPrefix(suffix string) string {
	return fmt.Sprintf("%s%s", NoHostPrefix, suffix)
}

// RemovePrefix extracts string appearing after a "/"
func RemovePrefix(name string) string {
	// Get substring after a string.
	name = strings.TrimSpace(name)
	pos := strings.Index(name, separator)
	if pos == -1 {
		return name
	}
	adjustedPos := pos + len(separator)
	if adjustedPos >= len(name) {
		return name
	}
	return name[adjustedPos:]
}

// AddPrefix adds the prefix for a given resource name
func AddPrefix(collection, entity string) string {
	return fmt.Sprintf("%s%s%s", collection, separator, entity)
}

// GetPrefix returns the prefix for a resource name
func GetPrefix(resourceName string) string {
	s := strings.Split(strings.TrimSpace(resourceName), separator)
	if len(s) < 1 {
		return ""
	}
	return s[0]
}

// GetRackHostname returns a rack host name.
func GetRackHostname(rackName string) string {
	return fmt.Sprintf("%s-host", rackName)
}

// FormatResourceName formats the resource name
func FormatResourceName(old string) string {
	str := strings.Replace(old, " ", "_", -1)
	return strings.Replace(str, ",", "_", -1)
}

// StrToUFSState refers a map between a string to a UFS defined state map.
var StrToUFSState = map[string]string{
	"registered":           "STATE_REGISTERED",
	"deployed_pre_serving": "STATE_DEPLOYED_PRE_SERVING",
	"deployed_testing":     "STATE_DEPLOYED_TESTING",
	"serving":              "STATE_SERVING",
	"needs_reset":          "STATE_NEEDS_RESET",
	"needs_repair":         "STATE_NEEDS_REPAIR",
	"repair_failed":        "STATE_REPAIR_FAILED",
	"disabled":             "STATE_DISABLED",
	"reserved":             "STATE_RESERVED",
	"decommissioned":       "STATE_DECOMMISSIONED",
	"deploying":            "STATE_DEPLOYING",
	"ready":                "STATE_READY",
}

// StateToDescription refers a map between a State to its description.
var StateToDescription = map[string]string{
	"registered":           "Needs deploy",
	"deployed_pre_serving": "Deployed but not placed in prod",
	"deployed_testing":     "Deployed to the prod, but for testing",
	"serving":              "Deployed to the prod, serving",
	"needs_reset":          "Deployed to the prod, but required cleanup and verify",
	"needs_repair":         "Deployed to the prod, needs repair",
	"repair_failed":        "Deployed to the prod, failed to be repaired in previous step and requires new repair attempt",
	"disabled":             "Deployed to the prod, but disabled",
	"reserved":             "Deployed to the prod, but reserved (e.g. locked)",
	"decommissioned":       "Decommissioned from the prod, but still lives in UFS record",
	"deploying":            "Deploying the resource with required configs just before it is READY",
	"ready":                "Resource is ready for use or free to use",
}

// IsUFSState checks if a string refers to a valid UFS state.
func IsUFSState(state string) bool {
	_, ok := StrToUFSState[state]
	return ok
}

// ValidStateStr returns a valid str list for state strings.
func ValidStateStr() []string {
	ks := make([]string, 0, len(StrToUFSState))
	for k := range StrToUFSState {
		ks = append(ks, k)
	}
	return ks
}

// RemoveStatePrefix removes the "state_" prefix from the string
func RemoveStatePrefix(state string) string {
	state = strings.ToLower(state)
	if idx := strings.Index(state, "state_"); idx != -1 {
		state = state[idx+len("state_"):]
	}
	return state
}

// ToUFSState converts state string to a UFS state enum.
func ToUFSState(state string) ufspb.State {
	state = RemoveStatePrefix(state)
	v, ok := StrToUFSState[state]
	if !ok {
		return ufspb.State_STATE_UNSPECIFIED
	}
	return ufspb.State(ufspb.State_value[v])
}

// StrToUFSZone refers a map between a string to a UFS defined map.
var StrToUFSZone = genStrToUFSZone()

func genStrToUFSZone() map[string]string {
	res := make(map[string]string, len(ufspb.Zone_name))
	for _, value := range ufspb.Zone_name {
		// Generate the mapping by removing the prefix and converting text to lower case.
		res[strings.ToLower(strings.TrimPrefix(value, "ZONE_"))] = value
	}
	return res
}

// IsUFSZone checks if a string refers to a valid UFS zone.
func IsUFSZone(zone string) bool {
	_, ok := StrToUFSZone[zone]
	return ok
}

// IsAssetType checks if a strings is a valid asset type
func IsAssetType(assetType string) bool {
	for _, x := range ValidAssetTypeStr() {
		if x == assetType {
			return true
		}
	}
	return false
}

// ToAssetType returns an AssetType object corresponding to string
func ToAssetType(assetType string) ufspb.AssetType {
	aType := RemoveGivenPrefix(assetType, "assettype_")
	for k, v := range ufspb.AssetType_value {
		if strings.EqualFold(k, aType) {
			return ufspb.AssetType(v)
		}
	}
	return ufspb.AssetType_UNDEFINED
}

// ValidAssetTypeStr returns a valid str list for AssetTypes
func ValidAssetTypeStr() []string {
	keys := make([]string, 0, len(ufspb.AssetType_name))
	for k, v := range ufspb.AssetType_name {
		// 0 is UNDEFINED
		if k != 0 {
			keys = append(keys, strings.ToLower(v))
		}
	}
	return keys
}

// ValidDeploymentEnvStr returns a valid str list for DeploymentEnv
func ValidDeploymentEnvStr() []string {
	keys := make([]string, 0, len(ufspb.DeploymentEnv_name))
	for k, v := range ufspb.DeploymentEnv_name {
		// 0 is UNDEFINED
		if k != 0 {
			keys = append(keys, strings.ToLower(v))
		}
	}
	return keys
}

// ToUFSDeploymentEnv converts string to a UFS DeploymentEnv enum.
func ToUFSDeploymentEnv(env string) ufspb.DeploymentEnv {
	v, ok := ufspb.DeploymentEnv_value[strings.ToUpper(env)]
	if !ok {
		return ufspb.DeploymentEnv_DEPLOYMENTENV_UNDEFINED
	}
	return ufspb.DeploymentEnv(v)
}

// ValidZoneStr returns a valid str list for zone strings.
func ValidZoneStr() []string {
	ks := make([]string, 0, len(StrToUFSZone))
	for k := range StrToUFSZone {
		ks = append(ks, k)
	}
	return ks
}

// RemoveZonePrefix removes the "zone_" prefix from the string
func RemoveZonePrefix(zone string) string {
	zone = strings.ToLower(zone)
	if idx := strings.Index(zone, "zone_"); idx != -1 {
		zone = zone[idx+len("zone_"):]
	}
	return zone
}

// ToUFSZone converts zone string to a UFS zone enum.
func ToUFSZone(zone string) ufspb.Zone {
	zone = RemoveZonePrefix(zone)
	v, ok := StrToUFSZone[zone]
	if !ok {
		return ufspb.Zone_ZONE_UNSPECIFIED
	}
	return ufspb.Zone(ufspb.Zone_value[v])
}

// StrToUFSDeviceType refers a map between a string to a UFS defined map.
var StrToUFSDeviceType = map[string]string{
	"chromebook":  "DEVICE_CHROMEBOOK",
	"labstation":  "DEVICE_LABSTATION",
	"servo":       "DEVICE_SERVO",
	"unspecified": "CHROME_OS_DEVICE_TYPE_UNSPECIFIED",
}

// ValidDeviceTypeStr returns a valid str list for devicetype strings.
func ValidDeviceTypeStr() []string {
	ks := make([]string, 0, len(StrToUFSDeviceType))
	for k := range StrToUFSDeviceType {
		ks = append(ks, k)
	}
	return ks
}

// RemoveGivenPrefix removes the prefix from the string
func RemoveGivenPrefix(msg, prefix string) string {
	msg = strings.ToLower(msg)
	if idx := strings.Index(msg, prefix); idx != -1 {
		msg = msg[idx+len(prefix):]
	}
	return msg
}

// ToUFSDeviceType converts devicetype string to a UFS devicetype enum.
func ToUFSDeviceType(devicetype string) ufspb.ChromeOSDeviceType {
	devicetype = RemoveGivenPrefix(devicetype, "device_")
	v, ok := StrToUFSDeviceType[devicetype]
	if !ok {
		return ufspb.ChromeOSDeviceType_CHROME_OS_DEVICE_TYPE_UNSPECIFIED
	}
	return ufspb.ChromeOSDeviceType(ufspb.ChromeOSDeviceType_value[v])
}

// StrToChameleonType refers a map between a string to a ChameleonType map.
var StrToChameleonType = map[string]string{
	"invalid": "CHAMELEON_TYPE_INVALID",
	"dp":      "CHAMELEON_TYPE_DP",
	"dphdmi":  "CHAMELEON_TYPE_DP_HDMI",
	"vga":     "CHAMELEON_TYPE_VGA",
	"hdmi":    "CHAMELEON_TYPE_HDMI",
}

// IsChameleonType checks if a string refers to a valid ChameleonType.
func IsChameleonType(chameleonType string) bool {
	_, ok := StrToChameleonType[chameleonType]
	return ok
}

// ValidChameleonTypeStr returns a valid str list for Chameleontype strings.
func ValidChameleonTypeStr() []string {
	ks := make([]string, 0, len(StrToChameleonType))
	for k := range StrToChameleonType {
		ks = append(ks, k)
	}
	return ks
}

// ToChameleonType converts devicetype string to a Chameleon type enum.
func ToChameleonType(chameleonType string) chromeosLab.ChameleonType {
	chameleonType = RemoveGivenPrefix(chameleonType, "chameleon_type_")
	v, ok := StrToChameleonType[chameleonType]
	if !ok {
		return chromeosLab.ChameleonType_CHAMELEON_TYPE_INVALID
	}
	return chromeosLab.ChameleonType(chromeosLab.ChameleonType_value[v])
}

// StrToCameraType refers a map between a string to a CameraType map.
var StrToCameraType = map[string]string{
	"invalid": "CAMERA_INVALID",
	"huddly":  "CAMERA_HUDDLY",
	"ptzpro2": "CAMERA_PTZPRO2",
}

// IsCameraType checks if a string refers to a valid CameraType.
func IsCameraType(cameraType string) bool {
	_, ok := StrToCameraType[cameraType]
	return ok
}

// ValidCameraTypeStr returns a valid str list for CameraType strings.
func ValidCameraTypeStr() []string {
	ks := make([]string, 0, len(StrToCameraType))
	for k := range StrToCameraType {
		ks = append(ks, k)
	}
	return ks
}

// ToCameraType converts cameraType string to a Camera type enum.
func ToCameraType(cameraType string) chromeosLab.CameraType {
	cameraType = RemoveGivenPrefix(cameraType, "camera_")
	v, ok := StrToCameraType[cameraType]
	if !ok {
		return chromeosLab.CameraType_CAMERA_INVALID
	}
	return chromeosLab.CameraType(chromeosLab.CameraType_value[v])
}

// StrToAntennaConnection refers a map between a string to a AntennaConnection map.
var StrToAntennaConnection = map[string]string{
	"unknown":    "CONN_UNKNOWN",
	"conductive": "CONN_CONDUCTIVE",
	"ota":        "CONN_OTA",
}

// IsAntennaConnection checks if a string refers to a valid AntennaConnection.
func IsAntennaConnection(antennaConnection string) bool {
	_, ok := StrToAntennaConnection[antennaConnection]
	return ok
}

// ValidAntennaConnectionStr returns a valid str list for AntennaConnection strings.
func ValidAntennaConnectionStr() []string {
	ks := make([]string, 0, len(StrToAntennaConnection))
	for k := range StrToAntennaConnection {
		ks = append(ks, k)
	}
	return ks
}

// ToAntennaConnection converts antennaConnection string to a Wifi_AntennaConnection enum.
func ToAntennaConnection(antennaConnection string) chromeosLab.Wifi_AntennaConnection {
	antennaConnection = RemoveGivenPrefix(antennaConnection, "conn_")
	v, ok := StrToAntennaConnection[antennaConnection]
	if !ok {
		return chromeosLab.Wifi_CONN_UNKNOWN
	}
	return chromeosLab.Wifi_AntennaConnection(chromeosLab.Wifi_AntennaConnection_value[v])
}

// StrToRouter refers a map between a string to a Router map.
var StrToRouter = map[string]string{
	"unspecified": "ROUTER_UNSPECIFIED",
	"80211ax":     "ROUTER_802_11AX",
}

// IsRouter checks if a string refers to a valid Router.
func IsRouter(router string) bool {
	_, ok := StrToRouter[router]
	return ok
}

// ValidRouterStr returns a valid str list for Router strings.
func ValidRouterStr() []string {
	ks := make([]string, 0, len(StrToRouter))
	for k := range StrToRouter {
		ks = append(ks, k)
	}
	return ks
}

// ToRouter converts router string to a Wifi_Router enum.
func ToRouter(router string) chromeosLab.Wifi_Router {
	router = RemoveGivenPrefix(router, "router_")
	v, ok := StrToRouter[router]
	if !ok {
		return chromeosLab.Wifi_ROUTER_UNSPECIFIED
	}
	return chromeosLab.Wifi_Router(chromeosLab.Wifi_Router_value[v])
}

// StrToCableType refers a map between a string to a CableType map.
var StrToCableType = map[string]string{
	"invalid":     "CABLE_INVALID",
	"audiojack":   "CABLE_AUDIOJACK",
	"usbaudio":    "CABLE_USBAUDIO",
	"usbprinting": "CABLE_USBPRINTING",
	"hdmiaudio":   "CABLE_HDMIAUDIO",
}

// IsCableType checks if a string refers to a valid CableType.
func IsCableType(cableType string) bool {
	_, ok := StrToCableType[cableType]
	return ok
}

// ValidCableTypeStr returns a valid str list for CableType strings.
func ValidCableTypeStr() []string {
	ks := make([]string, 0, len(StrToCableType))
	for k := range StrToCableType {
		ks = append(ks, k)
	}
	return ks
}

// ToCableType converts cableType string to a Cable type enum.
func ToCableType(cableType string) chromeosLab.CableType {
	cableType = RemoveGivenPrefix(cableType, "cable_")
	v, ok := StrToCableType[cableType]
	if !ok {
		return chromeosLab.CableType_CABLE_INVALID
	}
	return chromeosLab.CableType(chromeosLab.CableType_value[v])
}

// StrToFacing refers a map between a string to a Facing map.
var StrToFacing = map[string]string{
	"unknown": "FACING_UNKNOWN",
	"back":    "FACING_BACK",
	"front":   "FACING_FRONT",
}

// IsFacing checks if a string refers to a valid Facing.
func IsFacing(facing string) bool {
	_, ok := StrToFacing[facing]
	return ok
}

// ValidFacingStr returns a valid str list for Facing strings.
func ValidFacingStr() []string {
	ks := make([]string, 0, len(StrToFacing))
	for k := range StrToFacing {
		ks = append(ks, k)
	}
	return ks
}

// ToFacing converts facing string to a Camerabox_Facing enum.
func ToFacing(facing string) chromeosLab.Camerabox_Facing {
	facing = RemoveGivenPrefix(facing, "facing_")
	v, ok := StrToFacing[facing]
	if !ok {
		return chromeosLab.Camerabox_FACING_UNKNOWN
	}
	return chromeosLab.Camerabox_Facing(chromeosLab.Camerabox_Facing_value[v])
}

// StrToLight refers a map between a string to a Light map.
var StrToLight = map[string]string{
	"unknown": "LIGHT_UNKNOWN",
	"led":     "LIGHT_LED",
	"noled":   "LIGHT_NOLED",
}

// IsLight checks if a string refers to a valid Light.
func IsLight(light string) bool {
	_, ok := StrToLight[light]
	return ok
}

// ValidLightStr returns a valid str list for Light strings.
func ValidLightStr() []string {
	ks := make([]string, 0, len(StrToLight))
	for k := range StrToLight {
		ks = append(ks, k)
	}
	return ks
}

// ToLight converts light string to a Camerabox_Light enum.
func ToLight(light string) chromeosLab.Camerabox_Light {
	light = RemoveGivenPrefix(light, "light_")
	v, ok := StrToLight[light]
	if !ok {
		return chromeosLab.Camerabox_LIGHT_UNKNOWN
	}
	return chromeosLab.Camerabox_Light(chromeosLab.Camerabox_Light_value[v])
}

// StrToLicenseType refers a map between a string to a LicenseType map.
var StrToLicenseType = map[string]string{
	"invalid":      "LICENSE_TYPE_UNSPECIFIED",
	"windows10pro": "LICENSE_TYPE_WINDOWS_10_PRO",
	"msofficestd":  "LICENSE_TYPE_MS_OFFICE_STANDARD",
}

// IsLicenseType checks if a string refers to a valid LicenseType.
func IsLicenseType(licenseType string) bool {
	_, ok := StrToLicenseType[licenseType]
	return ok
}

// ValidLicenseTypeStr returns a valid str list for LicenseType strings.
func ValidLicenseTypeStr() []string {
	ks := make([]string, 0, len(StrToLicenseType))
	for k := range StrToLicenseType {
		ks = append(ks, k)
	}
	return ks
}

// ToLicenseType converts licenseType string to a License type enum.
func ToLicenseType(licenseType string) chromeosLab.LicenseType {
	licenseType = RemoveGivenPrefix(licenseType, "license_")
	v, ok := StrToLicenseType[licenseType]
	if !ok {
		return chromeosLab.LicenseType_LICENSE_TYPE_UNSPECIFIED
	}
	return chromeosLab.LicenseType(chromeosLab.LicenseType_value[v])
}

// StrToSchedulingUnitType refers a map between a string to a UFS defined map.
var StrToSchedulingUnitType = map[string]string{
	"invalid":    "SCHEDULING_UNIT_TYPE_INVALID",
	"all":        "SCHEDULING_UNIT_TYPE_ALL",
	"individual": "SCHEDULING_UNIT_TYPE_INDIVIDUAL",
}

// ValidSchedulingUnitTypeStr returns a valid str list for SchedulingUnitType strings.
func ValidSchedulingUnitTypeStr() []string {
	ks := make([]string, 0, len(StrToSchedulingUnitType))
	for k := range StrToSchedulingUnitType {
		ks = append(ks, k)
	}
	return ks
}

// IsSchedulingUnitType checks if a string refers to a valid SchedulingUnitType.
func IsSchedulingUnitType(schedulingUnitType string) bool {
	_, ok := StrToSchedulingUnitType[schedulingUnitType]
	return ok
}

// ToSchedulingUnitType converts SchedulingUnitType string to a UFS SchedulingUnitType enum.
func ToSchedulingUnitType(schedulingUnitType string) ufspb.SchedulingUnitType {
	schedulingUnitType = RemoveGivenPrefix(schedulingUnitType, "scheduling_unit_type_")
	v, ok := StrToSchedulingUnitType[schedulingUnitType]
	if !ok {
		return ufspb.SchedulingUnitType_SCHEDULING_UNIT_TYPE_INVALID
	}
	return ufspb.SchedulingUnitType(ufspb.SchedulingUnitType_value[v])
}

// List of regexps for recognizing assets stored with googlers or out of lab.
var googlers = []*regexp.Regexp{
	regexp.MustCompile(`container`),
	regexp.MustCompile(`desk`),
	regexp.MustCompile(`testbed`),
}

// LabToZone converts deprecated Lab type to Zone
func LabToZone(lab string) ufspb.Zone {
	if strings.Contains(lab, "mtv1950-testing") {
		return ufspb.Zone_ZONE_MTV1950_TESTING
	}
	switch oslabRegexp.FindString(lab) {
	case "chromeos1":
		return ufspb.Zone_ZONE_CHROMEOS1
	case "chromeos2":
		return ufspb.Zone_ZONE_CHROMEOS2
	case "chromeos3":
		return ufspb.Zone_ZONE_CHROMEOS3
	case "chromeos4":
		return ufspb.Zone_ZONE_CHROMEOS4
	case "chromeos5":
		return ufspb.Zone_ZONE_CHROMEOS5
	case "chromeos6":
		return ufspb.Zone_ZONE_CHROMEOS6
	case "chromeos7":
		return ufspb.Zone_ZONE_CHROMEOS7
	case "chromeos15":
		return ufspb.Zone_ZONE_CHROMEOS15
	default:
		for _, r := range googlers {
			if r.MatchString(lab) {
				return ufspb.Zone_ZONE_CROS_GOOGLER_DESK
			}
		}
		return ufspb.Zone_ZONE_UNSPECIFIED
	}
}

// ToUFSDept returns the dept name based on zone string.
func ToUFSDept(zone string) string {
	ufsZone := ToUFSZone(zone)
	if IsInBrowserZone(ufsZone.String()) {
		return Browser
	}
	return CrOS
}

// GetStateDescription returns the description for the state
func GetStateDescription(state string) string {
	state = RemoveStatePrefix(state)
	v, ok := StateToDescription[state]
	if !ok {
		return ""
	}
	return v
}

// GetSuffixAfterSeparator extracts the string appearing after the separator
//
// returns the suffix after the first found separator
func GetSuffixAfterSeparator(name, seprator string) string {
	name = strings.TrimSpace(name)
	pos := strings.Index(name, seprator)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(seprator)
	if adjustedPos >= len(name) {
		return ""
	}
	return name[adjustedPos:]
}

// ServoV3HostnameRegex is used to identify servo V3 hosts.
var ServoV3HostnameRegex = regexp.MustCompile(`.*-servo`)

// PoolNameRegex ensures that a pool name is valid.
var PoolNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// Invalid characters for tags field. Used by Contains/ContainsAny method.
var invalidTagChars string = "="

// ValidateTags checks if the tags contain only valid characters
func ValidateTags(tags []string) bool {
	for _, t := range tags {
		if strings.ContainsAny(t, invalidTagChars) {
			return false
		}
	}
	return true
}

// StrToUFSAttachedDeviceType refers a map between a string to a UFS defined map.
var StrToUFSAttachedDeviceType = map[string]string{
	"apple_phone":    "ATTACHED_DEVICE_TYPE_APPLE_PHONE",
	"android_phone":  "ATTACHED_DEVICE_TYPE_ANDROID_PHONE",
	"apple_tablet":   "ATTACHED_DEVICE_TYPE_APPLE_TABLET",
	"android_tablet": "ATTACHED_DEVICE_TYPE_ANDROID_TABLET",
	"devboard":       "ATTACHED_DEVICE_TYPE_DEVBOARD",
	"jetstream":      "ATTACHED_DEVICE_TYPE_JETSTREAM",
}

// ValidAttachedDeviceTypeStr returns a valid str list for attached device type strings.
func ValidAttachedDeviceTypeStr() []string {
	ks := make([]string, 0, len(StrToUFSAttachedDeviceType))
	for k := range StrToUFSAttachedDeviceType {
		ks = append(ks, k)
	}
	return ks
}

// IsAttachedDeviceType checks if a string refers to a valid AttachedDeviceType.
func IsAttachedDeviceType(deviceType string) bool {
	_, ok := StrToUFSAttachedDeviceType[deviceType]
	return ok
}

// ToUFSAttachedDeviceType converts type string to a UFS attached device type enum.
func ToUFSAttachedDeviceType(deviceType string) ufspb.AttachedDeviceType {
	v, ok := StrToUFSAttachedDeviceType[deviceType]
	if !ok {
		return ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_UNSPECIFIED
	}
	return ufspb.AttachedDeviceType(ufspb.AttachedDeviceType_value[v])
}
