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

func setDutPeripherals(p *inventory.Peripherals, c *inventory.HardwareCapabilities, d *lab.Peripherals) {
	if d == nil {
		return
	}

	p.AudioBoard = &falseValue
	if chameleon := d.GetChameleon(); chameleon != nil {
		for _, c := range chameleon.GetChameleonPeripherals() {
			cType := inventory.Peripherals_ChameleonType(c)
			if cType != inventory.Peripherals_CHAMELEON_TYPE_INVALID {
				p.Chameleon = &trueValue
				p.ChameleonType = []inventory.Peripherals_ChameleonType{cType}
				p.AudioBoard = &chameleon.AudioBoard
			}
		}
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
	}

	if touch := d.GetTouch(); touch != nil {
		p.Mimo = &(touch.Mimo)
	}

	carrier := inventory.HardwareCapabilities_Carrier(inventory.HardwareCapabilities_Carrier_value[d.GetCarrier()])
	c.Carrier = &carrier
	p.Camerabox = &(d.Camerabox)
}

func setManufacturingConfig(l *inventory.SchedulableLabels, m *manufacturing.Config) {
	l.Phase = (*inventory.SchedulableLabels_Phase)(&(m.DevicePhase))
	l.Cr50Phase = (*inventory.SchedulableLabels_CR50_Phase)(&(m.Cr50Phase))
	// TODO (guocb) cr50_key_env?
}

func setDeviceConfig(p *inventory.Peripherals, c *inventory.HardwareCapabilities, d *device.Config) {
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
		case device.Config_HARDWARE_FEATURE_BLUETOOTH:
			c.Bluetooth = &trueValue
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
		append("servo_type", p.GetServo().GetServoType()).
		append("servo_serial", p.GetServo().GetServoSerial())

	ostype := inventory.SchedulableLabels_OS_TYPE_CROS
	peri := inventory.Peripherals{}
	capa := inventory.HardwareCapabilities{}
	labels := inventory.SchedulableLabels{
		OsType:       &ostype,
		Platform:     &(data.LabConfig.GetDeviceConfigId().GetPlatformId().Value),
		Board:        &(data.LabConfig.GetDeviceConfigId().GetPlatformId().Value),
		Brand:        &(data.LabConfig.GetDeviceConfigId().GetBrandId().Value),
		Model:        &(data.LabConfig.GetDeviceConfigId().GetModelId().Value),
		Sku:          &(data.LabConfig.GetDeviceConfigId().GetVariantId().Value),
		Capabilities: &capa,
		Peripherals:  &peri,
	}
	setDutPeripherals(&peri, &capa, p)
	setDeviceConfig(&peri, &capa, data.GetDeviceConfig())
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
