// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package datastore

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/appengine/gaetesting"
)

func makeServo(servoHost, serial string, port int) *lab.Servo {
	return &lab.Servo{
		ServoHostname: servoHost,
		ServoPort:     int32(port),
		ServoSerial:   serial,
		ServoType:     "v3",
	}
}

func TestUpdateLabstations(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("Test labstation updates", t, func() {
		checkServosCount := func(expectCount int) {
			ls := getLabConfigByHostname(ctx, t, "labstation1")
			So(ls.GetLabstation().GetServos(), ShouldHaveLength, expectCount)
		}
		checkServoExistBySerialNumber := func(sn string) {
			ls := getLabConfigByHostname(ctx, t, "labstation1")
			state := ""
			for _, servo := range ls.GetLabstation().GetServos() {
				if servo.GetServoSerial() == sn {
					state = sn
					break
				}
			}
			So(state, ShouldEqual, sn)
		}
		dut1 := mockDut("dut1", "ID_1", "labstation2")
		dut1.GetDut().GetPeripherals().Servo.ServoPort = 9901
		dut1.GetDut().GetPeripherals().Servo.ServoSerial = "sn-1"
		dut2 := mockDut("dut2", "ID_2", "labstation1")
		dut2.GetDut().GetPeripherals().Servo.ServoPort = 9902
		dut2.GetDut().GetPeripherals().Servo.ServoSerial = "ser2"
		dut3 := mockDut("dut3", "ID_3", "labstation1")
		dut3.GetDut().GetPeripherals().Servo.ServoPort = 9903
		dut3.GetDut().GetPeripherals().Servo.ServoSerial = "sn-3"
		lab1 := mockLabstation("labstation1", "labstation1_id")
		lab2 := mockLabstation("labstation2", "labstation2_id")
		_, err := AddDevices(ctx, []*lab.ChromeOSDevice{lab1, lab2, dut1, dut2, dut3}, false)
		So(err, ShouldBeNil)

		r := newServoHostRegistryFromProtoMsgs(ctx, nil)
		testlab, err := r.getServoHost(ctx, "labstation1")
		So(err, ShouldBeNil)
		testlab.Servos = []*lab.Servo{
			makeServo("labstation1", "ser1", 2),
			makeServo("labstation1", "ser2", 3),
			makeServo("labstation1", "ser3", 4),
		}
		initServoCount := len(testlab.GetServos())
		err = r.saveToDatastore(ctx)
		So(err, ShouldBeNil)

		Convey("Delete servos for a labstation", func() {
			Convey("Happy path", func() {
				checkServosCount(initServoCount)
				_, err := UpdateLabstations(ctx, "labstation1", []string{"ser1", "ser2"}, nil)
				So(err, ShouldBeNil)
				checkServosCount(1)
				checkServoExistBySerialNumber("ser3")
			})
		})

		Convey("Add servos from DUTs", func() {
			Convey("The Dut not exist", func() {
				checkServosCount(initServoCount)
				_, err := UpdateLabstations(ctx, "labstation1", nil, []string{"host1"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "No such host: host1")
				checkServosCount(initServoCount)
			})
			Convey("The Dut servo used another labstation", func() {
				checkServosCount(initServoCount)
				_, err := UpdateLabstations(ctx, "labstation1", nil, []string{"dut1"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "DUT does not use selected labstation as servo_hostname")
				checkServosCount(initServoCount)
			})
			Convey("Duplicate by serial number", func() {
				checkServosCount(initServoCount)
				_, err := UpdateLabstations(ctx, "labstation1", nil, []string{"dut2"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "The labstation labstation1 already has servo with serial number: ser2")
				checkServosCount(initServoCount)
			})
			Convey("Skip empty hostnames", func() {
				checkServosCount(initServoCount)
				_, err := UpdateLabstations(ctx, "labstation1", nil, []string{""})
				So(err, ShouldBeNil)
				checkServosCount(initServoCount)
			})
			Convey("happy path", func() {
				checkServosCount(initServoCount)
				_, err := UpdateLabstations(ctx, "labstation1", nil, []string{"dut3"})
				So(err, ShouldBeNil)
				checkServosCount(initServoCount + 1)
				checkServoExistBySerialNumber("ser1")
				checkServoExistBySerialNumber("ser2")
				checkServoExistBySerialNumber("ser3")
				checkServoExistBySerialNumber("sn-3")
			})
		})
	})
}

func TestCheckDuplicates(t *testing.T) {
	t.Parallel()
	Convey("Check duplicates by port", t, func() {
		servos := []*lab.Servo{
			makeServo("host1", "Ser2", 2),
			makeServo("host1", "Ser3", 3),
			makeServo("host1", "Ser4", 4),
			makeServo("host1", "Ser5", 5),
		}
		Convey("No duplicates", func() {
			servo := makeServo("host1", "Ser1", 1)
			err := checkDuplicatePort(servo, servos)
			So(err, ShouldBeNil)
			err = checkDuplicateSerial(servo, servos)
			So(err, ShouldBeNil)
		})
		Convey("has duplicate by port", func() {
			servo := makeServo("host1", "Ser1", 3)
			err := checkDuplicatePort(servo, servos)
			So(err, ShouldNotBeNil)
			err = checkDuplicateSerial(servo, servos)
			So(err, ShouldBeNil)
		})
		Convey("has duplicate by serial number", func() {
			servo := makeServo("host1", "Ser3", 1)
			err := checkDuplicatePort(servo, servos)
			So(err, ShouldBeNil)
			err = checkDuplicateSerial(servo, servos)
			So(err, ShouldNotBeNil)
		})
	})
}
