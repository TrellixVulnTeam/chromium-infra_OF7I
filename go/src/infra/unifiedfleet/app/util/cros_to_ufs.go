// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	lab "go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/api/sheets/v4"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsCros "infra/unifiedfleet/api/v1/proto/chromeos/lab"
)

const (
	standardLSEPrototype   = "atl-lab:standard"
	labstationLSEPrototype = "atl-lab:labstation"
	cameraLSEPrototype     = "acs-lab:camera"
	wifiLSEPrototype       = "acs-lab:wificell"
)

// ToOSMachineLSEs converts cros inventory data to UFS LSEs for ChromeOS machines.
func ToOSMachineLSEs(labConfigs []*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig) []*ufspb.MachineLSE {
	lses := make([]*ufspb.MachineLSE, 0, len(labConfigs))
	for _, lc := range labConfigs {
		dut := lc.GetConfig().GetDut()
		deviceID := lc.GetConfig().GetId().GetValue()
		if dut != nil {
			lses = append(lses, DUTToLSE(dut, deviceID, lc.GetUpdatedTime()))
		} else {
			lses = append(lses, LabstationToLSE(lc.GetConfig().GetLabstation(), deviceID, lc.GetUpdatedTime()))
		}
	}
	return lses
}

// DUTToLSE converts a DUT spec to a UFS machine LSE
func DUTToLSE(dut *lab.DeviceUnderTest, deviceID string, updatedTime *timestamp.Timestamp) *ufspb.MachineLSE {
	hostname := dut.GetHostname()
	lse := &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
			ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
				DeviceLse: &ufspb.ChromeOSDeviceLSE{
					Device: &ufspb.ChromeOSDeviceLSE_Dut{
						Dut: copyDUT(dut),
					},
				},
			},
		},
	}
	return &ufspb.MachineLSE{
		Name:                hostname,
		MachineLsePrototype: getLSEPrototypeByLabConfig(dut),
		Hostname:            hostname,
		Machines:            []string{deviceID},
		UpdateTime:          updatedTime,
		Lse:                 lse,
		Lab:                 getLabByHostname(hostname).String(),
	}
}

// LabstationToLSE converts a DUT spec to a UFS machine LSE
func LabstationToLSE(l *lab.Labstation, deviceID string, updatedTime *timestamp.Timestamp) *ufspb.MachineLSE {
	hostname := l.GetHostname()
	lse := &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
			ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
				DeviceLse: &ufspb.ChromeOSDeviceLSE{
					Device: &ufspb.ChromeOSDeviceLSE_Labstation{
						Labstation: copyLabstation(l),
					},
				},
			},
		},
	}
	return &ufspb.MachineLSE{
		Name:                hostname,
		MachineLsePrototype: getLSEPrototypeByLabConfig(nil),
		Hostname:            hostname,
		Machines:            []string{deviceID},
		UpdateTime:          updatedTime,
		Lse:                 lse,
		Lab:                 getLabByHostname(hostname).String(),
	}
}

func copyDUT(dut *lab.DeviceUnderTest) *ufsCros.DeviceUnderTest {
	if dut == nil {
		return nil
	}
	s := proto.MarshalTextString(dut)
	var newDUT ufsCros.DeviceUnderTest
	proto.UnmarshalText(s, &newDUT)
	return &newDUT
}

func copyLabstation(l *lab.Labstation) *ufsCros.Labstation {
	if l == nil {
		return nil
	}
	s := proto.MarshalTextString(l)
	var newL ufsCros.Labstation
	proto.UnmarshalText(s, &newL)
	return &newL
}

func getLabByHostname(hostname string) ufspb.Lab {
	if strings.HasPrefix(hostname, "chromeos1") {
		return ufspb.Lab_LAB_CHROMEOS_SANTIAM
	}
	if strings.HasPrefix(hostname, "chromeos2") {
		return ufspb.Lab_LAB_CHROMEOS_ATLANTIS
	}
	if strings.HasPrefix(hostname, "chromeos4") {
		return ufspb.Lab_LAB_CHROMEOS_DESTINY
	}
	if strings.HasPrefix(hostname, "chromeos6") {
		return ufspb.Lab_LAB_CHROMEOS_PROMETHEUS
	}
	if strings.HasPrefix(hostname, "chromeos3") ||
		strings.HasPrefix(hostname, "chromeos5") ||
		strings.HasPrefix(hostname, "chromeos7") ||
		strings.HasPrefix(hostname, "chromeos9") ||
		strings.HasPrefix(hostname, "chromeos15") {
		return ufspb.Lab_LAB_CHROMEOS_LINDAVISTA
	}
	// Temporarily set all other OS labs to Unspecified
	return ufspb.Lab_LAB_UNSPECIFIED
}

func getLSEPrototypeByLabConfig(dut *lab.DeviceUnderTest) string {
	if dut == nil {
		return labstationLSEPrototype
	}
	// Only limit special LSE Prototypes to ACS lab
	if getLabByHostname(dut.GetHostname()) == ufspb.Lab_LAB_CHROMEOS_LINDAVISTA {
		if dut.GetPeripherals().GetWifi() != nil {
			return wifiLSEPrototype
		}
		if dut.GetPeripherals().GetCamerabox() {
			return cameraLSEPrototype
		}
	}
	return standardLSEPrototype
}

// GetOSMachineLSEPrototypes returns the pre-defined machine lse prototypes for ChromeOS machines.
func GetOSMachineLSEPrototypes() []*ufspb.MachineLSEPrototype {
	return []*ufspb.MachineLSEPrototype{
		{
			Name: standardLSEPrototype,
			PeripheralRequirements: []*ufspb.PeripheralRequirement{
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_SERVO,
					Min:            1,
					Max:            1,
				},
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_RPM,
					Min:            1,
					Max:            1,
				},
			},
			Tags: []string{"atl", "standard"},
		},
		{
			Name: labstationLSEPrototype,
			PeripheralRequirements: []*ufspb.PeripheralRequirement{
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_RPM,
					Min:            1,
					Max:            1,
				},
			},
			Tags: []string{"atl", "labstation"},
		},
		{
			Name: cameraLSEPrototype,
			PeripheralRequirements: []*ufspb.PeripheralRequirement{
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_SERVO,
					Min:            1,
					Max:            1,
				},
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_CAMERA,
					Min:            1,
					Max:            1,
				},
			},
			Tags: []string{"acs", "camera"},
		},
		{
			Name: wifiLSEPrototype,
			PeripheralRequirements: []*ufspb.PeripheralRequirement{
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_SERVO,
					Min:            1,
					Max:            1,
				},
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_WIFICELL,
					Min:            1,
					Max:            1,
				},
			},
			Tags: []string{"acs", "wificell"},
		},
	}
}

// Example: subnet 100.115.224.0 netmask 255.255.254.0 {
var dhcpdVlanRegexp = regexp.MustCompile(`subnet [0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3} netmask [0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3} {`)

// Example: 100.115.224.0
var ipRegexp = regexp.MustCompile(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`)

// Example: fixed-address 100.115.224.2;
var fixedAddressRegexp = regexp.MustCompile(`fixed-address [0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3};`)

// Example: host host1 {
var hostNameRegexp = regexp.MustCompile(`host .*{`)

// Example: hardware ethernet aa:00:00:00:00:00
var ethernetRegexp = regexp.MustCompile(`hardware ethernet .*;`)

// Example: aa:00:00:00:00:00
var macAddrRegexp = regexp.MustCompile(`([a-fA-F0-9]{2}\:){5}[a-fA-F0-9]{2}`)

// DHCPConf defines the format of response after parsing a ChromeOS dhcp conf file
type DHCPConf struct {
	ValidVlans       []*ufspb.Vlan
	ValidIPs         []*ufspb.IP
	ValidDHCPs       []*ufspb.DHCPConfig
	DHCPsWithoutVlan []*ufspb.DHCPConfig
	MismatchedVlans  []*ufspb.Vlan
	DuplicatedVlans  []*ufspb.Vlan
	DuplicatedIPs    []*ufspb.IP
}

// ParseOSDhcpdConf parses dhcpd.conf
func ParseOSDhcpdConf(conf string, topology map[string]*ufspb.Vlan) (*DHCPConf, error) {
	respIPs := make([]*ufspb.IP, 0)
	respVlans := make(map[string]*ufspb.Vlan, 0)
	dhcps := make([]*ufspb.DHCPConfig, 0)
	dhcpsWithoutVlan := make([]*ufspb.DHCPConfig, 0)
	ipMaps := make(map[string]*ufspb.IP, 0)
	mismatchedVlans := make([]*ufspb.Vlan, 0)
	duplicatedVlans := make([]*ufspb.Vlan, 0)
	duplicatedIPs := make([]*ufspb.IP, 0)

	lines := strings.Split(conf, "\n")
	// Record the hostname which is under scanning
	foundHostname := ""
	// Record the mac address which is under scanning
	foundMacAddress := ""
	for _, line := range lines {
		// Skip commented line
		if strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.Trim(line, "\r"))

		// Parse lines like "subnet 100.115.224.0 netmask 255.255.254.0"
		if dhcpdVlanRegexp.MatchString(line) {
			subnet, vlan, notMatchTopology := parseSubnetAndMaskLine(line, topology)
			if _, ok := respVlans[subnet]; ok {
				duplicatedVlans = append(duplicatedVlans, vlan)
				continue
			}
			respVlans[subnet] = vlan
			if notMatchTopology {
				mismatchedVlans = append(mismatchedVlans, vlan)
			}
			startIP, err := IPv4StrToInt(subnet)
			if err != nil {
				return nil, errors.Reason("fail to parse subnet %s to uint32", subnet).Err()
			}
			for i := 0; i < int(vlan.CapacityIp); i++ {
				ipV4Str := IPv4IntToStr(startIP)
				ip := &ufspb.IP{
					Id:      GetIPName(vlan.GetName(), ipV4Str),
					Ipv4:    startIP,
					Ipv4Str: ipV4Str,
					Vlan:    vlan.GetName(),
				}
				respIPs = append(respIPs, ip)
				ipMaps[ipV4Str] = ip
				startIP++
			}
			continue
		}

		// Parse lines like "fixed-address 100.115.224.2"
		if fixedAddressRegexp.MatchString(line) {
			ips := ipRegexp.FindAllString(line, -1)
			foundIP := ips[0]
			if foundHostname == "" {
				return nil, errors.Reason("no hostname for the address (%s) ", foundIP).Err()
			}
			dhcp := &ufspb.DHCPConfig{
				Hostname:   foundHostname,
				MacAddress: foundMacAddress,
				Ip:         foundIP,
			}
			// Reset
			foundHostname = ""
			foundMacAddress = ""
			ip, ok := ipMaps[foundIP]
			if !ok {
				dhcpsWithoutVlan = append(dhcpsWithoutVlan, dhcp)
			} else {
				if ip.Occupied {
					duplicatedIPs = append(duplicatedIPs, ip)
				}
				ip.Occupied = true
				dhcps = append(dhcps, dhcp)
			}
		}

		// Parse lines like "host host1 {"
		if hostNameRegexp.MatchString(line) {
			res := strings.Split(strings.TrimSpace(line[0:len(line)-1]), " ")
			if len(res) != 2 {
				return nil, errors.Reason("wrong format of hostname (%s)", line).Err()
			}
			foundHostname = res[1]
		}

		// Parse lines like "hardware ethernet aa:00:00:00:00:00;"
		if ethernetRegexp.MatchString(line) {
			macAddr := macAddrRegexp.FindAllString(line, -1)
			if len(macAddr) > 1 {
				return nil, errors.Reason("wrong format of ethernet (%s)", line).Err()
			}
			// mac address can be empty, e.g. "hardware ethernet ;"
			if len(macAddr) == 1 {
				foundMacAddress = macAddr[0]
			}
		}
	}

	// Also return the vlans pre-defined in topology but haven't been setup in dhcp conf.
	for k, v := range topology {
		if _, ok := respVlans[k]; !ok {
			respVlans[k] = v
		}
	}
	vlans := make([]*ufspb.Vlan, 0, len(respVlans))
	for _, v := range respVlans {
		vlans = append(vlans, v)
	}
	return &DHCPConf{
		ValidVlans:       vlans,
		ValidIPs:         respIPs,
		ValidDHCPs:       dhcps,
		DHCPsWithoutVlan: dhcpsWithoutVlan,
		MismatchedVlans:  mismatchedVlans,
		DuplicatedIPs:    duplicatedIPs,
		DuplicatedVlans:  duplicatedVlans,
	}, nil
}

// ParseATLTopology parse the topology of ATL lab based on a Google sheet
func ParseATLTopology(data *sheets.Spreadsheet) (map[string]*ufspb.Vlan, []*ufspb.Vlan) {
	resp := make(map[string]*ufspb.Vlan, 0)
	dupcatedVlan := make([]*ufspb.Vlan, 0)
	header := make([]string, 0)
	for i, row := range data.Sheets[0].Data[0].RowData {
		// Skip empty line
		if row.Values[0].FormattedValue == "" {
			continue
		}

		// Skip but parse header
		if i == 0 {
			for _, cell := range row.Values {
				header = append(header, cell.FormattedValue)
			}
			// Invalid sheet info
			if len(header) == 0 {
				break
			}
			continue
		}

		addr, vlan := parseTopologyRow(header, row.Values)
		// Skip rows without empty string in column "VLAN #" and "Address"
		if addr != "" && vlan.Name != "" {
			if _, ok := resp[addr]; ok {
				dupcatedVlan = append(dupcatedVlan, vlan)
				continue
			}
			resp[addr] = vlan
		}
	}
	return resp, dupcatedVlan
}

func parseSubnetAndMaskLine(line string, topology map[string]*ufspb.Vlan) (string, *ufspb.Vlan, bool) {
	ips := ipRegexp.FindAllString(line, -1)
	subnet := ips[0]
	cidr, capacity := parseCidrBlock(subnet, ips[1])
	notMatchTopology := false
	vlan, ok := topology[subnet]
	if ok {
		vlan.CapacityIp = int32(capacity - 2)
		if vlan.VlanAddress != cidr {
			notMatchTopology = true
		}
		vlan.VlanAddress = cidr
	} else {
		vlan = &ufspb.Vlan{
			// Use subnet as part of name and randomly assign vlan's name to CrOS lab
			Name:        GetCrOSLabName(subnet),
			VlanAddress: cidr,
			// OS lab-specific, 2 last ips are reserved
			CapacityIp: int32(capacity - 2),
		}
	}
	return subnet, vlan, notMatchTopology
}

func parseTopologyRow(header []string, rowValue []*sheets.CellData) (string, *ufspb.Vlan) {
	vlan := &ufspb.Vlan{}
	addr := ""
	mask := ""
	for j, cell := range rowValue {
		if j >= len(header) {
			break
		}
		switch header[j] {
		case "Subnet Name":
			vlan.Description = cell.FormattedValue
		case "VLAN #", "VLAN":
			if cell.FormattedValue != "" {
				vlan.Name = GetATLLabName(cell.FormattedValue)
			}
		case "Allocated Size":
			vlan.CapacityIp = int32(*cell.EffectiveValue.NumberValue)
		case "Address":
			addr = cell.FormattedValue
		case "Mask":
			mask = cell.FormattedValue
		}
		if addr != "" && mask != "" {
			vlan.VlanAddress = addr + mask
		}
	}
	return addr, vlan
}
