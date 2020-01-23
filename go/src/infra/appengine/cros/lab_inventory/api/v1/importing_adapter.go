// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package api

// Adapts the data defined by proto
// https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/libs/skylab/inventory/device.proto
// to data defined by
// https://chromium.googlesource.com/chromiumos/infra/proto/src/lab/device.proto
import (
	"strconv"
	"strings"

	dev_proto "go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"go.chromium.org/luci/common/errors"
	"infra/libs/skylab/inventory"
)

// A mapping from servo host name to servo host proto message.
type servoHostRegister map[string]*lab.ChromeOSDevice

func (r servoHostRegister) addServo(servo *lab.Servo) {
	hostname := servo.GetServoHostname()
	if hostname == "" {
		return
	}
	// FIXME(guocb) Try to load the labstation from datastore first. Otherwise
	// it may be overwritten and lost servos.
	if _, existing := r[hostname]; !existing {
		r[hostname] = &lab.ChromeOSDevice{
			Device: &lab.ChromeOSDevice_Labstation{
				Labstation: &lab.Labstation{
					Hostname: hostname,
				},
			},
		}
	}
	servoHost := r[hostname].GetLabstation()
	servoHost.Servos = append(servoHost.Servos, servo)
}

func (r servoHostRegister) getAllLabstations() []*lab.ChromeOSDevice {
	labstations := make([]*lab.ChromeOSDevice, 0, len(r))
	for _, v := range r {
		labstations = append(labstations, v)
	}
	return labstations
}

// ImportFromV1DutSpecs adapts v1 inventory data to v2 format.
func ImportFromV1DutSpecs(oldSpecs []*inventory.CommonDeviceSpecs) (devices []*lab.ChromeOSDevice, labstations []*lab.ChromeOSDevice, dutStates []*lab.DutState, err error) {
	servoHostRegister := servoHostRegister{}
	errs := errors.MultiError{}
	for _, olddata := range oldSpecs {
		if err := createCrosDevice(&devices, servoHostRegister, olddata); err != nil {
			errs = append(errs, errors.Annotate(err, "import spec for %s", olddata.GetHostname()).Err())
		}
		createDutState(&dutStates, olddata)
	}
	if len(errs) != 0 {
		err = errs
	}
	return devices, servoHostRegister.getAllLabstations(), dutStates, err
}

func createCrosDevice(results *[]*lab.ChromeOSDevice, servoHostRegister servoHostRegister, olddata *inventory.CommonDeviceSpecs) error {
	if osType := olddata.GetLabels().GetOsType(); osType == inventory.SchedulableLabels_OS_TYPE_LABSTATION {
		if err := createLabstation(servoHostRegister, olddata); err != nil {
			return err
		}
	} else {
		// Convert all other os_type (INVALID, ANDROID, CROS, MOBLAB, JETSTREAM)
		// to DUT.
		if err := createDut(results, servoHostRegister, olddata); err != nil {
			return err
		}
	}
	return nil
}

func importServo(servo *lab.Servo, key string, value string) error {
	switch key {
	case "servo_host":
		servo.ServoHostname = value
		if value == "" {
			return errors.Reason("invalid servo hostname: '%s'", value).Err()
		}
	case "servo_port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return errors.Reason("invalid servo port: %s", value).Err()
		}
		servo.ServoPort = int32(port)
	case "servo_serial":
		servo.ServoSerial = value
	case "servo_type":
		servo.ServoType = value
	}
	return nil
}

func importRpm(rpm *lab.RPM, key string, value string) {
	switch key {
	case "powerunit_hostname":
		rpm.PowerunitName = value
	case "powerunit_outlet":
		rpm.PowerunitOutlet = value
	}
}

func importAttributes(attrs []*inventory.KeyValue) (string, *lab.Servo, *lab.RPM) {
	skipServo := false
	var hwid string
	var servo lab.Servo
	var rpm lab.RPM
	for _, attr := range attrs {
		value := attr.GetValue()
		switch key := attr.GetKey(); key {
		case "HWID":
			hwid = value
		case "servo_host", "servo_port", "servo_serial", "servo_type":
			if err := importServo(&servo, key, value); err != nil {
				skipServo = true
			}
		case "powerunit_hostname", "powerunit_outlet":
			importRpm(&rpm, key, value)
		}
	}
	if skipServo {
		return hwid, nil, &rpm
	}
	return hwid, &servo, &rpm
}

func getChameleonType(oldperi *inventory.Peripherals) []lab.ChameleonType {
	oldtypes := oldperi.GetChameleonType()
	newtype := make([]lab.ChameleonType, len(oldtypes))
	for i, v := range oldtypes {
		newtype[i] = lab.ChameleonType(v)
	}
	return newtype
}

func getAntennaConn(peri *inventory.Peripherals) lab.Wifi_AntennaConnection {
	if peri.GetConductive() {
		return lab.Wifi_CONN_CONDUCTIVE
	}
	return lab.Wifi_CONN_OTA
}

func getConnectedCamera(peri *lab.Peripherals, oldPeri *inventory.Peripherals) {
	if oldPeri.GetHuddly() {
		peri.ConnectedCamera = append(peri.ConnectedCamera, &lab.Camera{
			CameraType: lab.CameraType_CAMERA_HUDDLY,
		})
	}
	if oldPeri.GetPtzpro2() {
		peri.ConnectedCamera = append(peri.ConnectedCamera, &lab.Camera{
			CameraType: lab.CameraType_CAMERA_PTZPRO2,
		})
	}
}

func getDeviceConfigID(labels *inventory.SchedulableLabels) *dev_proto.ConfigId {
	return &dev_proto.ConfigId{
		PlatformId: &dev_proto.PlatformId{
			Value: strings.ToLower(labels.GetBoard()),
		},
		ModelId: &dev_proto.ModelId{
			Value: strings.ToLower(labels.GetModel()),
		},
		VariantId: &dev_proto.VariantId{
			// Use sku (an integer) instead of HwidSKU (a string).
			Value: strings.ToLower(labels.GetSku()),
		},
	}
}

func getPeripherals(l *inventory.SchedulableLabels) *lab.Peripherals {
	peripherals := l.GetPeripherals()
	capabilities := l.GetCapabilities()
	testHints := l.GetTestCoverageHints()
	p := lab.Peripherals{
		Chameleon: &lab.Chameleon{
			AudioBoard:           peripherals.GetAudioBoard(),
			ChameleonPeripherals: getChameleonType(peripherals),
		},
		Audio: &lab.Audio{
			AudioBox: peripherals.GetAudioBox(),
			Atrus:    capabilities.GetAtrus(),
		},
		Wifi: &lab.Wifi{
			Wificell:    peripherals.GetWificell(),
			AntennaConn: getAntennaConn(peripherals),
		},
		Touch: &lab.Touch{
			Mimo: peripherals.GetMimo(),
		},
		Carrier:   parseCarrier(capabilities.GetCarrier()),
		Camerabox: peripherals.GetCamerabox(),
		Chaos:     testHints.GetChaosDut(),
	}
	getCables(&p, testHints)
	getConnectedCamera(&p, peripherals)
	return &p
}

func parseCarrier(c inventory.HardwareCapabilities_Carrier) string {
	return strings.ToLower(c.String()[len("CARRIER_"):])
}

func getCables(p *lab.Peripherals, testHints *inventory.TestCoverageHints) {
	if testHints.GetTestAudiojack() {
		p.Cable = append(p.Cable, &lab.Cable{
			Type: lab.CableType_CABLE_AUDIOJACK,
		})
	}
	if testHints.GetTestUsbaudio() {
		p.Cable = append(p.Cable, &lab.Cable{
			Type: lab.CableType_CABLE_USBAUDIO,
		})
	}
	if testHints.GetTestUsbprinting() {
		p.Cable = append(p.Cable, &lab.Cable{
			Type: lab.CableType_CABLE_USBPRINTING,
		})
	}
	if testHints.GetTestHdmiaudio() {
		p.Cable = append(p.Cable, &lab.Cable{
			Type: lab.CableType_CABLE_HDMIAUDIO,
		})
	}
}

func getPools(l *inventory.SchedulableLabels) []string {
	var pools []string
	for _, p := range l.GetCriticalPools() {
		pools = append(pools, inventory.SchedulableLabels_DUTPool_name[int32(p)])
	}
	for _, p := range l.GetSelfServePools() {
		pools = append(pools, p)
	}
	return pools
}

func createDut(devices *[]*lab.ChromeOSDevice, servoHostRegister servoHostRegister, olddata *inventory.CommonDeviceSpecs) error {
	hwid, servo, rpm := importAttributes(olddata.GetAttributes())

	peri := getPeripherals(olddata.Labels)
	if servo != nil {
		servoHostRegister.addServo(servo)
		peri.Servo = servo
	}
	if rpm != nil {
		peri.Rpm = rpm
	}

	pools := getPools(olddata.GetLabels())
	newDut := lab.ChromeOSDevice{
		Id:              &lab.ChromeOSDeviceID{Value: olddata.GetId()},
		SerialNumber:    olddata.GetSerialNumber(),
		ManufacturingId: &manufacturing.ConfigID{Value: hwid},

		DeviceConfigId: getDeviceConfigID(olddata.GetLabels()),
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{
				Hostname:    olddata.GetHostname(),
				Peripherals: peri,
				Pools:       pools,
			},
		},
	}
	*devices = append(*devices, &newDut)
	return nil
}

func createLabstation(servoHostRegister servoHostRegister, olddata *inventory.CommonDeviceSpecs) error {
	hostname := olddata.GetHostname()
	if _, existing := servoHostRegister[hostname]; existing {
		return nil
	}
	hwid, servo, rpm := importAttributes(olddata.GetAttributes())
	servoHostRegister[hostname] = &lab.ChromeOSDevice{
		Id:              &lab.ChromeOSDeviceID{Value: olddata.GetId()},
		SerialNumber:    olddata.GetSerialNumber(),
		DeviceConfigId:  getDeviceConfigID(olddata.GetLabels()),
		ManufacturingId: &manufacturing.ConfigID{Value: hwid},

		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{
				Hostname: hostname,
				Rpm:      rpm,
				Servos:   []*lab.Servo{servo},
				Pools:    getPools(olddata.GetLabels()),
			},
		},
	}
	return nil
}

func boolToDutState(state bool) lab.PeripheralState {
	if state {
		return lab.PeripheralState_WORKING
	}
	return lab.PeripheralState_NOT_CONNECTED
}

func createDutState(states *[]*lab.DutState, olddata *inventory.CommonDeviceSpecs) {
	if ostype := olddata.GetLabels().GetOsType(); ostype == inventory.SchedulableLabels_OS_TYPE_LABSTATION {
		return
	}
	peri := olddata.GetLabels().GetPeripherals()
	*states = append(*states, &lab.DutState{
		Id:                  &lab.ChromeOSDeviceID{Value: olddata.GetId()},
		Servo:               boolToDutState(peri.GetServo()),
		Chameleon:           boolToDutState(peri.GetChameleon()),
		AudioLoopbackDongle: boolToDutState(peri.GetAudioLoopbackDongle()),
	})
}
