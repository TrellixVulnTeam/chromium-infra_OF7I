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
		servos := []*lab.Servo{
			makeServo("labstation1", "ser1", 2),
			makeServo("labstation1", "ser2", 3),
			makeServo("labstation1", "ser3", 4),
		}
		l := mockLabstation("labstation1", "labstation1_id")
		l.GetLabstation().Servos = servos
		_, err := AddDevices(ctx, []*lab.ChromeOSDevice{l}, false)
		So(err, ShouldBeNil)
		Convey("Delete servos for a labstation", func() {
			ls := getLabConfigByHostname(ctx, t, "labstation1")
			So(ls.GetLabstation().GetServos(), ShouldHaveLength, len(servos))
			_, err := UpdateLabstations(ctx, "labstation1", []string{"ser1", "ser2"})
			So(err, ShouldBeNil)
			newLs := getLabConfigByHostname(ctx, t, "labstation1")
			newServos := newLs.GetLabstation().GetServos()
			So(newServos, ShouldHaveLength, 1)
			So(newServos[0].GetServoSerial(), ShouldEqual, "ser3")
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
		Convey("When save new device or update device with out servo", func() {
			Convey("No duplicates", func() {
				servo := makeServo("host1", "Ser1", 1)
				err := checkDuplicatePort(servo, nil, servos)
				So(err, ShouldBeNil)
				err = checkDuplicateSerial(servo, nil, servos)
				So(err, ShouldBeNil)
			})
			Convey("has duplicate by port", func() {
				servo := makeServo("host1", "Ser1", 3)
				err := checkDuplicatePort(servo, nil, servos)
				So(err, ShouldNotBeNil)
				err = checkDuplicateSerial(servo, nil, servos)
				So(err, ShouldBeNil)
			})
			Convey("has duplicate by serial number", func() {
				servo := makeServo("host1", "Ser3", 1)
				err := checkDuplicatePort(servo, nil, servos)
				So(err, ShouldBeNil)
				err = checkDuplicateSerial(servo, nil, servos)
				So(err, ShouldNotBeNil)
			})
		})
		Convey("When update device with existed servo", func() {
			Convey("Servo host not changed", func() {
				Convey("Will not run check when servo did not changed at all, even duplicates exist", func() {
					oldServo := makeServo("host1", "Ser2", 2)
					servo := makeServo("host1", "Ser2", 2)
					err := checkDuplicatePort(servo, oldServo, servos)
					So(err, ShouldBeNil)
					err = checkDuplicateSerial(servo, oldServo, servos)
					So(err, ShouldBeNil)
				})
				Convey("No duplicates when servo changes to unique port and serial number", func() {
					oldServo := makeServo("host1", "Ser1", 1)
					servo := makeServo("host1", "Ser9", 9)
					err := checkDuplicatePort(servo, oldServo, servos)
					So(err, ShouldBeNil)
					err = checkDuplicateSerial(servo, oldServo, servos)
					So(err, ShouldBeNil)
				})
				Convey("The port changed to existed one", func() {
					oldServo := makeServo("host1", "Ser1", 1)
					servo := makeServo("host1", "Ser1", 2)
					// will run the check
					err := checkDuplicatePort(servo, oldServo, servos)
					So(err, ShouldNotBeNil)
					// will not run the check because serial number was not changed
					err = checkDuplicateSerial(servo, oldServo, servos)
					So(err, ShouldBeNil)
				})
				Convey("The serial number changed to existed one", func() {
					oldServo := makeServo("host1", "Ser1", 1)
					servo := makeServo("host1", "Ser2", 1)
					// will not run the check because serial number was not changed
					err := checkDuplicatePort(servo, oldServo, servos)
					So(err, ShouldBeNil)
					// will run the check
					err = checkDuplicateSerial(servo, oldServo, servos)
					So(err, ShouldNotBeNil)
				})
			})
			Convey("Servo host has changed", func() {
				// All cases will run check
				Convey("Servo port and serial did not changed, when duplicates exist", func() {
					oldServo := makeServo("host2", "Ser2", 2)
					servo := makeServo("host1", "Ser2", 2)
					err := checkDuplicatePort(servo, oldServo, servos)
					So(err, ShouldNotBeNil)
					err = checkDuplicateSerial(servo, oldServo, servos)
					So(err, ShouldNotBeNil)
				})
				Convey("Servo port and serial did not changed, when duplicates not exist", func() {
					oldServo := makeServo("host2", "Ser9", 9)
					servo := makeServo("host1", "Ser9", 9)
					err := checkDuplicatePort(servo, oldServo, servos)
					So(err, ShouldBeNil)
					err = checkDuplicateSerial(servo, oldServo, servos)
					So(err, ShouldBeNil)
				})
				Convey("The port changed to existed one", func() {
					oldServo := makeServo("host2", "Ser1", 1)
					servo := makeServo("host1", "Ser1", 2)
					err := checkDuplicatePort(servo, oldServo, servos)
					So(err, ShouldNotBeNil)
					err = checkDuplicateSerial(servo, oldServo, servos)
					So(err, ShouldBeNil)
				})
				Convey("The serial number changed to existed one", func() {
					oldServo := makeServo("host2", "Ser1", 1)
					servo := makeServo("host1", "Ser2", 1)
					err := checkDuplicatePort(servo, oldServo, servos)
					So(err, ShouldBeNil)
					err = checkDuplicateSerial(servo, oldServo, servos)
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}
