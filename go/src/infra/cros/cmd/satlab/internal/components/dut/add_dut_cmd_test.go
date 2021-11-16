// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHasServo(t *testing.T) {
	t.Parallel()
	satlabId := "satlab123"
	Convey("Register servo for labstation", t, func() {
		ad := &addDUT{
			shivasAddDUT: shivasAddDUT{
				servo:       "servo_1",
				servoSerial: "servo_serial",
			},
		}
		if yes := ad.setupServoArguments(satlabId); !yes {
			t.Errorf("Expected servo is not detected but expected!")
		}
		So(ad.qualifiedServo, ShouldEqual, "satlab-satlab123-servo_1")
		So(ad.servoDockerContainerName, ShouldEqual, "")
	})
	Convey("Register servo for container", t, func() {
		ad := &addDUT{
			shivasAddDUT: shivasAddDUT{
				servo:       "",
				servoSerial: "servo_serial",
			},
		}
		if yes := ad.setupServoArguments(satlabId); !yes {
			t.Errorf("Expected servo is not detected but expected!")
		}
		So(ad.qualifiedServo, ShouldEqual, "satlab-satlab123--docker_servod:9999")
		So(ad.servoDockerContainerName, ShouldEqual, "satlab-satlab123--docker_servod")
	})
	Convey("Servo-less setup", t, func() {
		ad := &addDUT{
			shivasAddDUT: shivasAddDUT{
				servo:       "",
				servoSerial: "",
			},
		}
		if yes := ad.setupServoArguments(satlabId); yes {
			t.Errorf("Expected servo is detected but not expected!")
		}
		So(ad.qualifiedServo, ShouldEqual, "")
		So(ad.servoDockerContainerName, ShouldEqual, "")
	})
}

func TestSetDeployActions(t *testing.T) {
	t.Parallel()
	Convey("Full deploy with servo", t, func() {
		c := &addDUT{
			fullDeploy: true,
			shivasAddDUT: shivasAddDUT{
				deployActions: []string{"hi"},
			},
		}
		hasServo := true
		c.setDeployActions(hasServo)

		So(c.deploySkipDownloadImage, ShouldBeFalse)
		So(c.deploySkipInstallOS, ShouldBeFalse)
		So(c.deploySkipInstallFirmware, ShouldBeFalse)
		So(c.deploySkipRecoveryMode, ShouldBeFalse)
		So(c.deployActions, ShouldResemble, []string{"hi", "verify-recovery-mode"})
	})
	Convey("Not full deploy with servo", t, func() {
		c := &addDUT{
			fullDeploy: false,
			shivasAddDUT: shivasAddDUT{
				deployActions: []string{"hi"},
			},
		}
		hasServo := true
		c.setDeployActions(hasServo)

		So(c.deploySkipDownloadImage, ShouldBeTrue)
		So(c.deploySkipInstallOS, ShouldBeTrue)
		So(c.deploySkipInstallFirmware, ShouldBeTrue)
		So(c.deploySkipRecoveryMode, ShouldBeTrue)
		So(c.deployActions, ShouldResemble, []string{"hi"})
	})
	Convey("Full deploy without servo", t, func() {
		c := &addDUT{
			fullDeploy: true,
			shivasAddDUT: shivasAddDUT{
				deployActions: []string{"hi"},
			},
		}
		hasServo := false
		c.setDeployActions(hasServo)

		So(c.deploySkipDownloadImage, ShouldBeTrue)
		So(c.deploySkipInstallOS, ShouldBeTrue)
		So(c.deploySkipInstallFirmware, ShouldBeTrue)
		So(c.deploySkipRecoveryMode, ShouldBeTrue)
		So(c.deployActions, ShouldResemble, []string{"hi"})
	})
	Convey("Not full deploy without servo", t, func() {
		c := &addDUT{
			fullDeploy: false,
			shivasAddDUT: shivasAddDUT{
				deployActions: []string{"hi"},
			},
		}
		hasServo := false
		c.setDeployActions(hasServo)

		So(c.deploySkipDownloadImage, ShouldBeTrue)
		So(c.deploySkipInstallOS, ShouldBeTrue)
		So(c.deploySkipInstallFirmware, ShouldBeTrue)
		So(c.deploySkipRecoveryMode, ShouldBeTrue)
		So(c.deployActions, ShouldResemble, []string{"hi"})
	})
}
