// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package changehistory

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
)

func TestUpdateLabstation(t *testing.T) {
	t.Parallel()
	Convey("Log the change of labstation", t, func() {
		oldLabstation := lab.ChromeOSDevice{
			Id: &lab.ChromeOSDeviceID{Value: "UUID:1234"},
			Device: &lab.ChromeOSDevice_Labstation{
				Labstation: &lab.Labstation{
					Hostname: "labstation1",
					Servos: []*lab.Servo{
						{
							ServoHostname: "labstation1",
							ServoPort:     8888,
							ServoSerial:   "SERVO1",
							ServoType:     "v3",
						},
						{
							ServoHostname: "labstation1",
							ServoPort:     8889,
							ServoSerial:   "SERVO2",
							ServoType:     "v3",
						},
					},
				},
			},
		}
		// An identical copy of `oldLabstation`.
		newLabstation := lab.ChromeOSDevice{
			Id: &lab.ChromeOSDeviceID{Value: "UUID:1234"},
			Device: &lab.ChromeOSDevice_Labstation{
				Labstation: &lab.Labstation{
					Hostname: "labstation1",
					Servos: []*lab.Servo{
						{
							ServoHostname: "labstation1",
							ServoPort:     8888,
							ServoSerial:   "SERVO1",
							ServoType:     "v3",
						},
						{
							ServoHostname: "labstation1",
							ServoPort:     8889,
							ServoSerial:   "SERVO2",
							ServoType:     "v3",
						},
					},
				},
			},
		}

		Convey("no change", func() {
			changes := LogChromeOSDeviceChanges(&oldLabstation, &oldLabstation)
			So(changes, ShouldBeEmpty)
		})

		Convey("change servo port", func() {
			newLabstation.GetLabstation().GetServos()[0].ServoPort = 9999
			changes := LogChromeOSDeviceChanges(&oldLabstation, &newLabstation)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].OldValue, ShouldContainSubstring, "8888")
			So(changes[0].NewValue, ShouldContainSubstring, "9999")
		})

		Convey("remove a servo", func() {
			newLabstation.GetLabstation().Servos = newLabstation.GetLabstation().GetServos()[:1]
			changes := LogChromeOSDeviceChanges(&oldLabstation, &newLabstation)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "servo.SERVO2")
			So(changes[0].OldValue, ShouldContainSubstring, "SERVO2")
			So(changes[0].NewValue, ShouldEqual, "<nil>")
		})

		Convey("add a servo", func() {
			servos := newLabstation.GetLabstation().GetServos()
			servos = append(servos, &lab.Servo{
				ServoHostname: "labstation1",
				ServoPort:     8890,
				ServoSerial:   "SERVO3",
				ServoType:     "v4",
			})
			newLabstation.GetLabstation().Servos = servos
			changes := LogChromeOSDeviceChanges(&oldLabstation, &newLabstation)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].OldValue, ShouldEqual, "<nil>")
			So(changes[0].NewValue, ShouldContainSubstring, "SERVO3")
		})

		Convey("replace a servo with 'bigger' serial number", func() {
			// Replace a servo => remove a servo + add another servo.
			newLabstation.GetLabstation().GetServos()[0].ServoSerial = "SERVO4"
			changes := LogChromeOSDeviceChanges(&oldLabstation, &newLabstation)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].OldValue, ShouldContainSubstring, "SERVO1")
			So(changes[0].NewValue, ShouldEqual, "<nil>")
			So(changes[1].OldValue, ShouldEqual, "<nil>")
			So(changes[1].NewValue, ShouldContainSubstring, "SERVO4")
		})

		Convey("replace a servo with 'smaller' serial number", func() {
			// Replace a servo => remove a servo + add another servo.
			newLabstation.GetLabstation().GetServos()[0].ServoSerial = "SERVO0"
			changes := LogChromeOSDeviceChanges(&oldLabstation, &newLabstation)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].OldValue, ShouldEqual, "<nil>")
			So(changes[0].NewValue, ShouldContainSubstring, "SERVO0")
			So(changes[1].OldValue, ShouldContainSubstring, "SERVO1")
			So(changes[1].NewValue, ShouldEqual, "<nil>")
		})
	})
}

func TestUpdateDut(t *testing.T) {
	t.Parallel()
	Convey("Log the change of dut", t, func() {
		servo1 := lab.Servo{
			ServoHostname: "labstation1",
			ServoPort:     8888,
			ServoSerial:   "SERVO1",
			ServoType:     "v3",
		}
		servo2 := servo1

		chameleon1 := lab.Chameleon{
			ChameleonPeripherals: []lab.ChameleonType{lab.ChameleonType_CHAMELEON_TYPE_HDMI},
			AudioBoard:           true,
		}
		chameleon2 := chameleon1

		rpm1 := lab.RPM{
			PowerunitName:   "powerunit_name",
			PowerunitOutlet: "powerunit_outlet",
		}
		rpm2 := rpm1

		camera1 := lab.Camera{CameraType: lab.CameraType_CAMERA_HUDDLY}
		camera2 := camera1

		audio1 := lab.Audio{AudioBox: true, Atrus: true}
		audio2 := audio1

		wifi1 := lab.Wifi{Wificell: true, AntennaConn: lab.Wifi_CONN_OTA}
		wifi2 := wifi1

		touch1 := lab.Touch{Mimo: true}
		touch2 := touch1

		oldDut := lab.ChromeOSDevice{
			Id: &lab.ChromeOSDeviceID{Value: "UUID:1234"},
			Device: &lab.ChromeOSDevice_Dut{
				Dut: &lab.DeviceUnderTest{
					Hostname: "dut1",
					Peripherals: &lab.Peripherals{
						Servo:           &servo1,
						Chameleon:       &chameleon1,
						Rpm:             &rpm1,
						ConnectedCamera: []*lab.Camera{&camera1},
						Audio:           &audio1,
						Wifi:            &wifi1,
						Touch:           &touch1,
						Carrier:         "carrier1",
						Camerabox:       true,
					},
				},
			},
		}
		// An identical copy of `oldDut`.
		newDut := lab.ChromeOSDevice{
			Id: &lab.ChromeOSDeviceID{Value: "UUID:1234"},
			Device: &lab.ChromeOSDevice_Dut{
				Dut: &lab.DeviceUnderTest{
					Hostname: "dut1",
					Peripherals: &lab.Peripherals{
						Servo:           &servo2,
						Chameleon:       &chameleon2,
						Rpm:             &rpm2,
						ConnectedCamera: []*lab.Camera{&camera2},
						Audio:           &audio2,
						Wifi:            &wifi2,
						Touch:           &touch2,
						Carrier:         "carrier1",
						Camerabox:       true,
					},
				},
			},
		}

		Convey("no change", func() {
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldBeEmpty)
		})

		Convey("remove all peripherals", func() {
			newDut.GetDut().Peripherals = nil
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "peripherals")
			So(changes[0].NewValue, ShouldEqual, "<nil>")
		})

		Convey("change servo", func() {
			newDut.GetDut().GetPeripherals().GetServo().ServoPort = 9999
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "servo.SERVO1")
			So(changes[0].OldValue, ShouldContainSubstring, "8888")
			So(changes[0].NewValue, ShouldContainSubstring, "9999")
		})

		Convey("change chameleon", func() {
			newDut.GetDut().GetPeripherals().GetChameleon().ChameleonPeripherals = []lab.ChameleonType{
				lab.ChameleonType_CHAMELEON_TYPE_VGA,
			}
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "chameleon")
			So(changes[0].OldValue, ShouldContainSubstring, "TYPE_HDMI")
			So(changes[0].NewValue, ShouldContainSubstring, "TYPE_VGA")
		})

		Convey("change RPM", func() {
			newDut.GetDut().GetPeripherals().GetRpm().PowerunitName = "new powerunit"
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "powerunit_name")
			So(changes[0].OldValue, ShouldContainSubstring, "powerunit_name")
			So(changes[0].NewValue, ShouldContainSubstring, "new powerunit")
		})

		Convey("remove all connected camera", func() {
			newDut.GetDut().GetPeripherals().ConnectedCamera = nil
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "connected_camera")
			So(changes[0].OldValue, ShouldContainSubstring, "HUDDLY")
			So(changes[0].NewValue, ShouldEqual, "[]")
		})

		Convey("change audio", func() {
			newDut.GetDut().GetPeripherals().GetAudio().AudioBox = false
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "audio_box")
			So(changes[0].OldValue, ShouldContainSubstring, "true")
			So(changes[0].NewValue, ShouldContainSubstring, "false")
		})

		Convey("change wifi", func() {
			newDut.GetDut().GetPeripherals().GetWifi().Wificell = false
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "wificell")
			So(changes[0].OldValue, ShouldContainSubstring, "true")
			So(changes[0].NewValue, ShouldContainSubstring, "false")
		})

		Convey("change touch", func() {
			newDut.GetDut().GetPeripherals().GetTouch().Mimo = false
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "mimo")
			So(changes[0].OldValue, ShouldContainSubstring, "true")
			So(changes[0].NewValue, ShouldContainSubstring, "false")
		})

		Convey("change carrier", func() {
			newDut.GetDut().GetPeripherals().Carrier = "carrier2"
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "carrier")
			So(changes[0].OldValue, ShouldContainSubstring, "carrier1")
			So(changes[0].NewValue, ShouldContainSubstring, "carrier2")
		})

		Convey("change camerabox", func() {
			newDut.GetDut().GetPeripherals().Camerabox = false
			changes := LogChromeOSDeviceChanges(&oldDut, &newDut)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].Label, ShouldEqual, "camerabox")
			So(changes[0].OldValue, ShouldContainSubstring, "true")
			So(changes[0].NewValue, ShouldContainSubstring, "false")
		})
	})
}
