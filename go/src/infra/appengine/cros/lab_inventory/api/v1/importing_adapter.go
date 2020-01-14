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

func importServo(servo **lab.Servo, key string, value string) error {
	if *servo == nil {
		*servo = new(lab.Servo)
	}
	s := *servo
	switch key {
	case "servo_host":
		s.ServoHostname = value
	case "servo_port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return errors.Reason("invalid servo port: %s", value).Err()
		}
		s.ServoPort = int32(port)
	case "servo_serial":
		s.ServoSerial = value
	case "servo_type":
		s.ServoType = value
	}
	return nil
}

func importRpm(rpm **lab.RPM, key string, value string) {
	r := *rpm
	if r == nil {
		r = new(lab.RPM)
	}
	switch key {
	case "powerunit_hostname":
		r.PowerunitName = value
	case "powerunit_outlet":
		r.PowerunitOutlet = value
	}
}

func importAttributes(attrs []*inventory.KeyValue) (hwid string, servo *lab.Servo, rpm *lab.RPM, err error) {
	for _, attr := range attrs {
		value := attr.GetValue()
		switch key := attr.GetKey(); key {
		case "HWID":
			hwid = value
		case "servo_host", "servo_port", "servo_serial", "servo_type":
			err = importServo(&servo, key, value)
			if err != nil {
				return
			}
		case "powerunit_hostname", "powerunit_outlet":
			importRpm(&rpm, key, value)
		}
	}
	return
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
		BrandId: &dev_proto.BrandId{
			Value: strings.ToLower(labels.GetBrand()),
		},
	}
}

func getPeripherals(peripherals *inventory.Peripherals, capabilities *inventory.HardwareCapabilities) *lab.Peripherals {
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
		Carrier:   capabilities.GetCarrier().String(),
		Camerabox: peripherals.GetCamerabox(),
	}
	getConnectedCamera(&p, peripherals)
	return &p
}

func createDut(devices *[]*lab.ChromeOSDevice, servoHostRegister servoHostRegister, olddata *inventory.CommonDeviceSpecs) error {
	hwid, servo, rpm, err := importAttributes(olddata.GetAttributes())
	if err != nil {
		return err
	}

	oldPeri := olddata.Labels.Peripherals
	oldCapa := olddata.Labels.Capabilities
	peri := getPeripherals(oldPeri, oldCapa)
	if servo != nil {
		servoHostRegister.addServo(servo)
		peri.Servo = servo
	}
	if rpm != nil {
		peri.Rpm = rpm
	}

	newDut := lab.ChromeOSDevice{
		Id:              &lab.ChromeOSDeviceID{Value: olddata.GetId()},
		SerialNumber:    olddata.GetSerialNumber(),
		ManufacturingId: &manufacturing.ConfigID{Value: hwid},

		DeviceConfigId: getDeviceConfigID(olddata.GetLabels()),
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{
				Hostname:    olddata.GetHostname(),
				Peripherals: peri,
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
	hwid, _, _, err := importAttributes(olddata.GetAttributes())
	if err != nil {
		return err
	}
	servoHostRegister[hostname] = &lab.ChromeOSDevice{
		Id:              &lab.ChromeOSDeviceID{Value: olddata.GetId()},
		SerialNumber:    olddata.GetSerialNumber(),
		DeviceConfigId:  getDeviceConfigID(olddata.GetLabels()),
		ManufacturingId: &manufacturing.ConfigID{Value: hwid},

		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{
				Hostname: hostname,
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
