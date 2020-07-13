// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	lab "go.chromium.org/chromiumos/infra/proto/go/lab"
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
	if strings.HasPrefix(hostname, "chromeos2") || strings.HasPrefix(hostname, "chromeos4") || strings.HasPrefix(hostname, "chromeos6") {
		return ufspb.Lab_LAB_CHROMEOS_ATLANTIS
	}
	if strings.HasPrefix(hostname, "chromeos1") {
		return ufspb.Lab_LAB_CHROMEOS_SANTIAM
	}
	// It's probably wrong as it doesn't consider other ChromeOS lab. Temporarily set all other labs to lindavista
	return ufspb.Lab_LAB_CHROMEOS_LINDAVISTA
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
		},
	}
}

// ParseATLTopology parse the topology of ATL lab based on a Google sheet
func ParseATLTopology(data *sheets.Spreadsheet) map[string]*ufspb.Vlan {
	resp := make(map[string]*ufspb.Vlan, 0)
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
		if addr != "" && vlan.Name != "" {
			resp[addr] = vlan
		}
	}
	return resp
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
		case "VLAN #":
			vlan.Name = GetATLLabName(cell.FormattedValue)
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
