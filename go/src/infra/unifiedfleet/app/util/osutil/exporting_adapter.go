// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package osutil

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"

	"infra/cros/lab_inventory/deviceconfig"
	"infra/libs/skylab/inventory"
	ufspb "infra/unifiedfleet/api/v1/models"
	device "infra/unifiedfleet/api/v1/models/chromeos/device"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	manufacturing "infra/unifiedfleet/api/v1/models/chromeos/manufacturing"
)

var (
	trueValue            bool   = true
	falseValue           bool   = false
	emptyString          string = ""
	invServoStateUnknown        = inventory.PeripheralState_UNKNOWN
)

var noArcBoardMap = map[string]bool{
	"banjo":            true,
	"buddy":            true,
	"candy":            true,
	"dell":             true,
	"enguarde":         true,
	"expresso":         true,
	"falco":            true,
	"fizz-labstation":  true,
	"fizz-moblab":      true,
	"gale":             true,
	"gnawty":           true,
	"guado":            true,
	"guado_labstation": true,
	"heli":             true,
	"kip":              true,
	"link":             true,
	"monroe":           true,
	"ninja":            true,
	"nyan_big":         true,
	"nyan_blaze":       true,
	"nyan_kitty":       true,
	"orco":             true,
	"panther":          true,
	"peach_pit":        true,
	"peppy":            true,
	"quawks":           true,
	"rikku":            true,
	"sumo":             true,
	"swanky":           true,
	"tidus":            true,
	"tricky":           true,
	"veyron_jack":      true,
	"veyron_mickey":    true,
	"veyron_rialto":    true,
	"veyron_speedy":    true,
	"whirlwind":        true,
	"winky":            true,
	"zako":             true,
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

func setDutPeripherals(labels *inventory.SchedulableLabels, d *chromeosLab.Peripherals) {
	if d == nil {
		return
	}

	p := labels.Peripherals
	c := labels.Capabilities
	hint := labels.TestCoverageHints

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
			case chromeosLab.CameraType_CAMERA_HUDDLY:
				p.Huddly = &trueValue
			case chromeosLab.CameraType_CAMERA_PTZPRO2:
				p.Ptzpro2 = &trueValue
			}
		}
	}

	if audio := d.GetAudio(); audio != nil {
		p.AudioBox = &(audio.AudioBox)
		p.AudioCable = &(audio.AudioCable)
		c.Atrus = &(audio.Atrus)
	}

	if wifi := d.GetWifi(); wifi != nil {
		p.Wificell = &(wifi.Wificell)
		if wifi.GetAntennaConn() == chromeosLab.Wifi_CONN_CONDUCTIVE {
			p.Conductive = &trueValue
		} else {
			p.Conductive = &falseValue
		}
		if wifi.GetRouter() == chromeosLab.Wifi_ROUTER_802_11AX {
			p.Router_802_11Ax = &trueValue
		} else {
			p.Router_802_11Ax = &falseValue
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
		case chromeosLab.CableType_CABLE_AUDIOJACK:
			hint.TestAudiojack = &trueValue
		case chromeosLab.CableType_CABLE_USBAUDIO:
			hint.TestUsbaudio = &trueValue
		case chromeosLab.CableType_CABLE_USBPRINTING:
			hint.TestUsbprinting = &trueValue
		case chromeosLab.CableType_CABLE_HDMIAUDIO:
			hint.TestHdmiaudio = &trueValue
		}
	}

	if servo := d.GetServo(); servo != nil {
		servoType := servo.GetServoType()
		p.ServoType = &servoType
		setServoTopology(p, servo.GetServoTopology())
	}

	if facing := d.GetCameraboxInfo().GetFacing(); facing != chromeosLab.Camerabox_FACING_UNKNOWN {
		v1Facing := inventory.Peripherals_CameraboxFacing(facing)
		p.CameraboxFacing = &v1Facing
	}
	if light := d.GetCameraboxInfo().GetLight(); light != chromeosLab.Camerabox_LIGHT_UNKNOWN {
		v1Light := inventory.Peripherals_CameraboxLight(light)
		p.CameraboxLight = &v1Light
	}

	p.SmartUsbhub = &(d.SmartUsbhub)
}

func setServoTopology(p *inventory.Peripherals, st *chromeosLab.ServoTopology) {
	var t *inventory.ServoTopology
	if st != nil {
		stString := proto.MarshalTextString(st)
		t = &inventory.ServoTopology{}
		proto.UnmarshalText(stString, t)
	}
	p.ServoTopology = t
}

func setDutPools(labels *inventory.SchedulableLabels, inputPools []string) {
	for _, p := range inputPools {
		v, ok := inventory.SchedulableLabels_DUTPool_value[p]
		if ok {
			labels.CriticalPools = append(labels.CriticalPools, inventory.SchedulableLabels_DUTPool(v))
		} else {
			labels.SelfServePools = append(labels.SelfServePools, p)
		}

		if _, ok := appMap[p]; ok {
			labels.TestCoverageHints.HangoutApp = &trueValue
			labels.TestCoverageHints.MeetApp = &trueValue
		}
	}
}

func setManufacturingConfig(l *inventory.SchedulableLabels, m *manufacturing.Config) {
	if m == nil {
		return
	}
	l.Phase = (*inventory.SchedulableLabels_Phase)(&(m.DevicePhase))
	wifiChip := m.GetWifiChip()
	l.WifiChip = &wifiChip
	hwidComponent := m.GetHwidComponent()
	l.HwidComponent = hwidComponent
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

func setHwidData(l *inventory.SchedulableLabels, h *ufspb.HwidData) {
	sku := h.GetSku()
	l.HwidSku = &sku
	l.Variant = []string{
		h.GetVariant(),
	}
}

func setLicenses(l *inventory.SchedulableLabels, lic []*chromeosLab.License) {
	l.Licenses = make([]*inventory.License, len(lic))
	for i, v := range lic {
		var t inventory.LicenseType
		switch v.Type {
		case chromeosLab.LicenseType_LICENSE_TYPE_MS_OFFICE_STANDARD:
			t = inventory.LicenseType_LICENSE_TYPE_MS_OFFICE_STANDARD
		case chromeosLab.LicenseType_LICENSE_TYPE_WINDOWS_10_PRO:
			t = inventory.LicenseType_LICENSE_TYPE_WINDOWS_10_PRO
		default:
			t = inventory.LicenseType_LICENSE_TYPE_UNSPECIFIED
		}
		l.Licenses[i] = &inventory.License{
			Type:       &t,
			Identifier: &v.Identifier,
		}
	}
}

func setDutStateHelper(s chromeosLab.PeripheralState) *bool {
	var val bool
	if s == chromeosLab.PeripheralState_UNKNOWN || s == chromeosLab.PeripheralState_NOT_CONNECTED {
		val = false
	} else {
		val = true
	}
	return &val
}

func setPeripheralState(s chromeosLab.PeripheralState) *inventory.PeripheralState {
	target := inventory.PeripheralState_UNKNOWN
	if s != chromeosLab.PeripheralState_UNKNOWN {
		target = inventory.PeripheralState(s)
	}
	return &target
}

func setCr50Configs(l *inventory.SchedulableLabels, s *chromeosLab.DutState) {
	switch s.GetCr50Phase() {
	case chromeosLab.DutState_CR50_PHASE_PVT:
		l.Cr50Phase = inventory.SchedulableLabels_CR50_PHASE_PVT.Enum()
	case chromeosLab.DutState_CR50_PHASE_PREPVT:
		l.Cr50Phase = inventory.SchedulableLabels_CR50_PHASE_PREPVT.Enum()
	default:
		l.Cr50Phase = inventory.SchedulableLabels_CR50_PHASE_INVALID.Enum()
	}

	cr50Env := ""
	switch s.GetCr50KeyEnv() {
	case chromeosLab.DutState_CR50_KEYENV_PROD:
		cr50Env = "prod"
	case chromeosLab.DutState_CR50_KEYENV_DEV:
		cr50Env = "dev"
	}
	l.Cr50RoKeyid = &cr50Env
}

func setHardwareState(s chromeosLab.HardwareState) *inventory.HardwareState {
	target := inventory.HardwareState_HARDWARE_UNKNOWN
	if s != chromeosLab.HardwareState_HARDWARE_UNKNOWN {
		target = inventory.HardwareState(s)
	}
	return &target
}

func setDutState(l *inventory.SchedulableLabels, s *chromeosLab.DutState) {
	p := l.Peripherals
	p.ServoState = setPeripheralState(s.GetServo())
	p.Servo = setDutStateHelper(s.GetServo())
	p.Chameleon = setDutStateHelper(s.GetChameleon())
	p.AudioLoopbackDongle = setDutStateHelper(s.GetAudioLoopbackDongle())
	p.ServoUsbState = setHardwareState(s.GetServoUsbState())
	p.StorageState = setHardwareState(s.GetStorageState())
	p.RpmState = setPeripheralState(s.GetRpmState())

	if n := s.GetWorkingBluetoothBtpeer(); n > 0 {
		p.WorkingBluetoothBtpeer = &n
	}
	setCr50Configs(l, s)
}

func createDutLabels(devConfig *device.Config, osType *inventory.SchedulableLabels_OSType) *inventory.SchedulableLabels {
	devcfgID := devConfig.GetId()
	_, ok := noArcBoardMap[devcfgID.GetPlatformId().GetValue()]
	arc := !ok
	// Use GetXXX in case any object is nil.
	platform := devcfgID.GetPlatformId().GetValue()
	brand := devcfgID.GetBrandId().GetValue()
	model := devcfgID.GetModelId().GetValue()
	variant := devcfgID.GetVariantId().GetValue()

	labels := inventory.SchedulableLabels{
		Arc:               &arc,
		OsType:            osType,
		Platform:          &platform,
		Board:             &platform,
		Brand:             &brand,
		Model:             &model,
		Sku:               &variant,
		Capabilities:      &inventory.HardwareCapabilities{},
		Peripherals:       &inventory.Peripherals{},
		TestCoverageHints: &inventory.TestCoverageHints{},
	}

	ecTypeCros := inventory.SchedulableLabels_EC_TYPE_CHROME_OS
	mappedPlatform := deviceconfig.BoardToPlatformMap[platform]

	boardsHasCrosEc := stringset.NewFromSlice(crosEcTypeBoards...)
	if boardsHasCrosEc.Has(platform) || boardsHasCrosEc.Has(mappedPlatform) {
		labels.EcType = &ecTypeCros
	}
	return &labels
}

// AdaptToV1DutSpec adapts ChromeOSDeviceData to inventory.DeviceUnderTest of
// inventory v1 defined in
// https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/libs/skylab/inventory/device.proto
func AdaptToV1DutSpec(data *ufspb.ChromeOSDeviceData) (dut *inventory.DeviceUnderTest, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Reason("Recovered from %v\n%s", r, debug.Stack()).Err()
		}
	}()

	if data.GetLabConfig() == nil {
		return nil, errors.Reason("chromeosdevicedata is nil to adapt").Err()
	}
	if data.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		return adaptV2DutToV1DutSpec(data)
	}
	if data.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		return adaptV2LabstationToV1DutSpec(data)
	}
	panic("We should never reach here!")
}

func adaptV2DutToV1DutSpec(data *ufspb.ChromeOSDeviceData) (*inventory.DeviceUnderTest, error) {
	lse := data.GetLabConfig()
	machine := data.GetMachine()
	devConfig := data.GetDeviceConfig()
	dut := lse.GetChromeosMachineLse().GetDeviceLse().GetDut()
	p := dut.GetPeripherals()
	sn := machine.GetSerialNumber()
	var attrs attributes
	attrs.
		append("HWID", machine.GetChromeosMachine().GetHwid()).
		append("powerunit_hostname", p.GetRpm().GetPowerunitName()).
		append("powerunit_outlet", p.GetRpm().GetPowerunitOutlet()).
		append("serial_number", sn).
		append("servo_host", p.GetServo().GetServoHostname()).
		append("servo_port", fmt.Sprintf("%v", p.GetServo().GetServoPort())).
		append("servo_serial", p.GetServo().GetServoSerial()).
		append("servo_type", p.GetServo().GetServoType()).
		append("servo_setup", p.GetServo().GetServoSetup().String()[len("SERVO_SETUP_"):])

	osType := inventory.SchedulableLabels_OS_TYPE_INVALID
	if board := devConfig.GetId().GetPlatformId().GetValue(); board != "" {
		var found bool
		if osType, found = boardToOsTypeMapping[board]; !found {
			osType = inventory.SchedulableLabels_OS_TYPE_CROS
		}
	}

	labels := createDutLabels(devConfig, &osType)

	setDutPools(labels, dut.GetPools())
	setLicenses(labels, dut.GetLicenses())
	setDutPeripherals(labels, p)
	setDutState(labels, data.GetDutState())
	setDeviceConfig(labels, devConfig)
	setManufacturingConfig(labels, data.GetManufacturingConfig())
	setHwidData(labels, data.GetHwidData())

	id := machine.GetName()
	hostname := lse.GetName()
	deviceUnderTest := &inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:           &id,
			SerialNumber: &sn,
			Hostname:     &hostname,
			Attributes:   attrs,
			Labels:       labels,
		},
	}
	return deviceUnderTest, nil
}

func adaptV2LabstationToV1DutSpec(data *ufspb.ChromeOSDeviceData) (*inventory.DeviceUnderTest, error) {
	lse := data.GetLabConfig()
	machine := data.GetMachine()
	devConfig := data.GetDeviceConfig()
	l := lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation()
	sn := machine.GetSerialNumber()

	var attrs attributes
	attrs.
		append("HWID", machine.GetChromeosMachine().GetHwid()).
		append("powerunit_hostname", l.GetRpm().GetPowerunitName()).
		append("powerunit_outlet", l.GetRpm().GetPowerunitOutlet()).
		append("serial_number", sn)
	osType := inventory.SchedulableLabels_OS_TYPE_LABSTATION
	labels := createDutLabels(devConfig, &osType)
	// Hardcode labstation labels.
	labels.Platform = &emptyString
	acOnly := "AC_only"
	carrierInvalid := inventory.HardwareCapabilities_CARRIER_INVALID
	labels.Capabilities = &inventory.HardwareCapabilities{
		Atrus:           &falseValue,
		Bluetooth:       &falseValue,
		Carrier:         &carrierInvalid,
		Detachablebase:  &falseValue,
		Fingerprint:     &falseValue,
		Flashrom:        &falseValue,
		GpuFamily:       &emptyString,
		Graphics:        &emptyString,
		Hotwording:      &falseValue,
		InternalDisplay: &falseValue,
		Lucidsleep:      &falseValue,
		Modem:           &emptyString,
		Power:           &acOnly,
		Storage:         &emptyString,
		Telephony:       &emptyString,
		Webcam:          &falseValue,
		Touchpad:        &falseValue,
		Touchscreen:     &falseValue,
	}
	cr50PhaseInvalid := inventory.SchedulableLabels_CR50_PHASE_INVALID
	labels.Cr50Phase = &cr50PhaseInvalid
	labels.Cr50RoKeyid = &emptyString
	labels.Cr50RoVersion = &emptyString
	labels.Cr50RwKeyid = &emptyString
	labels.Cr50RwVersion = &emptyString
	ecTypeInvalid := inventory.SchedulableLabels_EC_TYPE_INVALID
	labels.EcType = &ecTypeInvalid
	labels.WifiChip = &emptyString

	labels.Peripherals = &inventory.Peripherals{
		AudioBoard:          &falseValue,
		AudioBox:            &falseValue,
		AudioLoopbackDongle: &falseValue,
		Chameleon:           &falseValue,
		ChameleonType:       []inventory.Peripherals_ChameleonType{inventory.Peripherals_CHAMELEON_TYPE_INVALID},
		Conductive:          &falseValue,
		Huddly:              &falseValue,
		Mimo:                &falseValue,
		Servo:               &falseValue,
		ServoState:          &invServoStateUnknown,
		SmartUsbhub:         &falseValue,
		Stylus:              &falseValue,
		Camerabox:           &falseValue,
		Wificell:            &falseValue,
		Router_802_11Ax:     &falseValue,
	}

	labels.TestCoverageHints = &inventory.TestCoverageHints{
		ChaosDut:        &falseValue,
		ChaosNightly:    &falseValue,
		Chromesign:      &falseValue,
		HangoutApp:      &falseValue,
		MeetApp:         &falseValue,
		RecoveryTest:    &falseValue,
		TestAudiojack:   &falseValue,
		TestHdmiaudio:   &falseValue,
		TestUsbaudio:    &falseValue,
		TestUsbprinting: &falseValue,
		UsbDetect:       &falseValue,
		UseLid:          &falseValue,
	}
	setHwidData(labels, data.GetHwidData())
	labels.Variant = nil
	setDutPools(labels, l.GetPools())
	id := machine.GetName()
	hostname := lse.GetName()
	deviceUnderTest := &inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:           &id,
			SerialNumber: &sn,
			Hostname:     &hostname,
			Attributes:   attrs,
			Labels:       labels,
		},
	}
	return deviceUnderTest, nil
}
