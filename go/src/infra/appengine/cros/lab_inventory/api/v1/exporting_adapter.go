// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package api

import (
	"fmt"
	"strings"

	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"infra/libs/skylab/inventory"
)

var (
	trueValue  bool = true
	falseValue bool = false
)

var arcBoardMap = map[string]bool{
	"asuka":                   true,
	"atlas":                   true,
	"auron_paine":             true,
	"auron_yuna":              true,
	"banon":                   true,
	"bob":                     true,
	"caroline":                true,
	"caroline-ndktranslation": true,
	"cave":                    true,
	"celes":                   true,
	"chell":                   true,
	"coral":                   true,
	"cyan":                    true,
	"edgar":                   true,
	"elm":                     true,
	"eve":                     true,
	"fizz":                    true,
	"gandof":                  true,
	"gnawty":                  true,
	"grunt":                   true,
	"hana":                    true,
	"hatch":                   true,
	"jacuzzi":                 true,
	"kalista":                 true,
	"kefka":                   true,
	"kevin":                   true,
	"kukui":                   true,
	"lars":                    true,
	"lulu":                    true,
	"nami":                    true,
	"nautilus":                true,
	"nocturne":                true,
	"octopus":                 true,
	"pyro":                    true,
	"rammus":                  true,
	"reef":                    true,
	"reks":                    true,
	"relm":                    true,
	"samus":                   true,
	"sand":                    true,
	"sarien":                  true,
	"sarien-kvm":              true,
	"scarlet":                 true,
	"sentry":                  true,
	"setzer":                  true,
	"snappy":                  true,
	"soraka":                  true,
	"squawks":                 true,
	"sumo":                    true,
	"terra":                   true,
	"ultima":                  true,
	"veyron_fievel":           true,
	"veyron_jaq":              true,
	"veyron_jerry":            true,
	"veyron_mickey":           true,
	"veyron_mighty":           true,
	"veyron_minnie":           true,
	"veyron_speedy":           true,
	"veyron_tiger":            true,
	"wizpig":                  true,
}

var appMap = map[string]bool{
	"hotrod": true,
}

type attributes []*inventory.KeyValue

func (a *attributes) append(key string, value string) *attributes {
	if value == "" {
		return a
	}
	*a = append(*a, &inventory.KeyValue{
		Key:   &key,
		Value: &value,
	})
	return a
}

func setDutPeripherals(p *inventory.Peripherals, c *inventory.HardwareCapabilities, hint *inventory.TestCoverageHints, d *lab.Peripherals) {
	if d == nil {
		return
	}

	p.AudioBoard = &falseValue
	if chameleon := d.GetChameleon(); chameleon != nil {
		for _, c := range chameleon.GetChameleonPeripherals() {
			cType := inventory.Peripherals_ChameleonType(c)
			if cType != inventory.Peripherals_CHAMELEON_TYPE_INVALID {
				p.Chameleon = &trueValue
				p.ChameleonType = append(p.ChameleonType, cType)
			}
		}
		p.AudioBoard = &chameleon.AudioBoard
	}

	p.Huddly = &falseValue
	if cameras := d.GetConnectedCamera(); cameras != nil {
		for _, c := range cameras {
			switch c.GetCameraType() {
			case lab.CameraType_CAMERA_HUDDLY:
				p.Huddly = &trueValue
			case lab.CameraType_CAMERA_PTZPRO2:
				p.Ptzpro2 = &trueValue
			}
		}
	}

	if audio := d.GetAudio(); audio != nil {
		p.AudioBox = &(audio.AudioBox)
		c.Atrus = &(audio.Atrus)
	}

	if wifi := d.GetWifi(); wifi != nil {
		p.Wificell = &(wifi.Wificell)
		if wifi.GetAntennaConn() == lab.Wifi_CONN_CONDUCTIVE {
			p.Conductive = &trueValue
		} else {
			p.Conductive = &falseValue
		}
		if wifi.GetRouter() == lab.Wifi_ROUTER_802_11AX {
			p.Router_802_11Ax = &trueValue
		}
	}

	if touch := d.GetTouch(); touch != nil {
		p.Mimo = &(touch.Mimo)
	}

	carrierKey := fmt.Sprintf("CARRIER_%s", strings.ToUpper(d.GetCarrier()))
	carrier := inventory.HardwareCapabilities_Carrier(inventory.HardwareCapabilities_Carrier_value[carrierKey])
	c.Carrier = &carrier
	p.Camerabox = &(d.Camerabox)

	hint.ChaosDut = &(d.Chaos)
	for _, c := range d.GetCable() {
		switch c.GetType() {
		case lab.CableType_CABLE_AUDIOJACK:
			hint.TestAudiojack = &trueValue
		case lab.CableType_CABLE_USBAUDIO:
			hint.TestUsbaudio = &trueValue
		case lab.CableType_CABLE_USBPRINTING:
			hint.TestUsbprinting = &trueValue
		case lab.CableType_CABLE_HDMIAUDIO:
			hint.TestHdmiaudio = &trueValue
		}
	}
}

func setDutPools(labels *inventory.SchedulableLabels, hint *inventory.TestCoverageHints, inputPools []string) {
	for _, p := range inputPools {
		v, ok := inventory.SchedulableLabels_DUTPool_value[p]
		if ok {
			labels.CriticalPools = append(labels.CriticalPools, inventory.SchedulableLabels_DUTPool(v))
		} else {
			labels.SelfServePools = append(labels.SelfServePools, p)
		}

		if _, ok := appMap[p]; ok {
			hint.HangoutApp = &trueValue
			hint.MeetApp = &trueValue
		}
	}
}

func setManufacturingConfig(l *inventory.SchedulableLabels, m *manufacturing.Config) {
	if m == nil {
		return
	}
	l.Phase = (*inventory.SchedulableLabels_Phase)(&(m.DevicePhase))
	l.Cr50Phase = (*inventory.SchedulableLabels_CR50_Phase)(&(m.Cr50Phase))
	cr50Env := ""
	switch m.Cr50KeyEnv {
	case manufacturing.Config_CR50_KEYENV_PROD:
		cr50Env = "prod"
	case manufacturing.Config_CR50_KEYENV_DEV:
		cr50Env = "dev"
	}
	if cr50Env != "" {
		l.Cr50RoKeyid = &cr50Env
	}
}

func setDeviceConfig(labels *inventory.SchedulableLabels, d *device.Config) {
	p := labels.GetPeripherals()
	c := labels.GetCapabilities()
	if d == nil {
		return
	}
	c.GpuFamily = &(d.GpuFamily)
	var graphics string
	switch d.Graphics {
	case device.Config_GRAPHICS_GL:
		graphics = "gl"
	case device.Config_GRAPHICS_GLE:
		graphics = "gles"
	}
	c.Graphics = &graphics

	for _, f := range d.GetHardwareFeatures() {
		switch f {
		case device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD:
			c.Detachablebase = &trueValue
		case device.Config_HARDWARE_FEATURE_BLUETOOTH:
			c.Bluetooth = &trueValue
		case device.Config_HARDWARE_FEATURE_FINGERPRINT:
			c.Fingerprint = &trueValue
		case device.Config_HARDWARE_FEATURE_FLASHROM:
			c.Flashrom = &trueValue
		case device.Config_HARDWARE_FEATURE_HOTWORDING:
			c.Hotwording = &trueValue
		case device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY:
			c.InternalDisplay = &trueValue
		case device.Config_HARDWARE_FEATURE_LUCID_SLEEP:
			c.Lucidsleep = &trueValue
		case device.Config_HARDWARE_FEATURE_WEBCAM:
			c.Webcam = &trueValue
		case device.Config_HARDWARE_FEATURE_STYLUS:
			p.Stylus = &trueValue
		case device.Config_HARDWARE_FEATURE_TOUCHPAD:
			c.Touchpad = &trueValue
		case device.Config_HARDWARE_FEATURE_TOUCHSCREEN:
			c.Touchscreen = &trueValue
		}
	}
	var power string
	switch pr := d.GetPower(); pr {
	case device.Config_POWER_SUPPLY_AC_ONLY:
		power = "AC_only"
	case device.Config_POWER_SUPPLY_BATTERY:
		power = "battery"
	}
	c.Power = &power

	if st := d.GetStorage(); st != device.Config_STORAGE_UNSPECIFIED {
		// Extract the storge type, e.g. "STORAGE_SSD" -> "ssd".
		storage := strings.ToLower(strings.SplitAfterN(st.String(), "_", 2)[1])
		c.Storage = &storage
	}

	if videoAcc := d.GetVideoAccelerationSupports(); videoAcc != nil {
		var acc []inventory.HardwareCapabilities_VideoAcceleration
		for _, v := range videoAcc {
			acc = append(acc, inventory.HardwareCapabilities_VideoAcceleration(v))
		}
		c.VideoAcceleration = acc
	}

	// Set CTS_ABI & CTS_CPU.
	switch d.GetCpu() {
	case device.Config_X86, device.Config_X86_64:
		labels.CtsAbi = []inventory.SchedulableLabels_CTSABI{
			inventory.SchedulableLabels_CTS_ABI_X86,
		}
		labels.CtsCpu = []inventory.SchedulableLabels_CTSCPU{
			inventory.SchedulableLabels_CTS_CPU_X86,
		}
	case device.Config_ARM, device.Config_ARM64:
		labels.CtsAbi = []inventory.SchedulableLabels_CTSABI{
			inventory.SchedulableLabels_CTS_ABI_ARM,
		}
		labels.CtsCpu = []inventory.SchedulableLabels_CTSCPU{
			inventory.SchedulableLabels_CTS_CPU_ARM,
		}
	}
}

func setHwidData(l *inventory.SchedulableLabels, h *HwidData) {
	l.HwidSku = &(h.Sku)
	l.Variant = []string{
		h.GetVariant(),
	}
}

func setDutStateHelper(s lab.PeripheralState, target **bool) {
	switch s {
	case lab.PeripheralState_WORKING:
		*target = &trueValue
	case lab.PeripheralState_NOT_CONNECTED:
		*target = &falseValue
	}
}
func setDutState(p *inventory.Peripherals, s *lab.DutState) {
	setDutStateHelper(s.GetServo(), &(p.Servo))
	setDutStateHelper(s.GetChameleon(), &(p.Chameleon))
	setDutStateHelper(s.GetAudioLoopbackDongle(), &(p.AudioLoopbackDongle))
}

// AdaptToV1DutSpec adapts ExtendedDeviceData to inventory.DeviceUnderTest of
// inventory v1 defined in
// https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/libs/skylab/inventory/device.proto
func AdaptToV1DutSpec(data *ExtendedDeviceData) (*inventory.DeviceUnderTest, error) {
	p := data.LabConfig.GetDut().GetPeripherals()
	var attrs attributes
	attrs.
		append("HWID", data.GetLabConfig().GetManufacturingId().GetValue()).
		append("powerunit_hostname", p.GetRpm().GetPowerunitName()).
		append("powerunit_outlet", p.GetRpm().GetPowerunitOutlet()).
		append("serial_number", data.GetLabConfig().GetSerialNumber()).
		append("servo_host", p.GetServo().GetServoHostname()).
		append("servo_port", fmt.Sprintf("%v", p.GetServo().GetServoPort())).
		append("servo_serial", p.GetServo().GetServoSerial()).
		append("servo_type", p.GetServo().GetServoType())

	ostype := inventory.SchedulableLabels_OS_TYPE_CROS
	peri := inventory.Peripherals{}
	capa := inventory.HardwareCapabilities{}
	hint := inventory.TestCoverageHints{}
	_, arc := arcBoardMap[data.LabConfig.GetDeviceConfigId().GetPlatformId().GetValue()]
	// Use GetXXX in case any object is nil.
	platform := data.LabConfig.GetDeviceConfigId().GetPlatformId().GetValue()
	brand := data.LabConfig.GetDeviceConfigId().GetBrandId().GetValue()
	model := data.LabConfig.GetDeviceConfigId().GetModelId().GetValue()
	variant := data.LabConfig.GetDeviceConfigId().GetVariantId().GetValue()
	labels := inventory.SchedulableLabels{
		Arc:               &arc,
		OsType:            &ostype,
		Platform:          &(platform),
		Board:             &(platform),
		Brand:             &(brand),
		Model:             &(model),
		Sku:               &(variant),
		Capabilities:      &capa,
		Peripherals:       &peri,
		TestCoverageHints: &hint,
	}
	setDutPools(&labels, &hint, data.GetLabConfig().GetDut().GetPools())
	setDutPeripherals(&peri, &capa, &hint, p)
	setDeviceConfig(&labels, data.GetDeviceConfig())
	setManufacturingConfig(&labels, data.GetManufacturingConfig())
	setHwidData(&labels, data.GetHwidData())
	setDutState(&peri, data.GetDutState())

	dut := inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:           &(data.LabConfig.GetId().Value),
			SerialNumber: &(data.LabConfig.SerialNumber),
			Hostname:     &(data.LabConfig.GetDut().Hostname),
			Attributes:   attrs,
			Labels:       &labels,
		},
	}
	return &dut, nil
}
