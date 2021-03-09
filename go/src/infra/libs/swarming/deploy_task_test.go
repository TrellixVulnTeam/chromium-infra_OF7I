package swarming

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDeployTaskCommand(t *testing.T) {
	t.Parallel()
	Convey("deployTaskCommand", t, func() {
		tc := &TaskCreator{
			session:        "TestSession",
			LogdogService:  "logdog://fake-logdog.appspot.com",
			LUCIProject:    "chromeos",
			logdogTaskCode: "logdoc_code",
		}
		Convey("deployTaskCommand - Happy path", func() {
			actions := []string{"do-nothing", "do-something"}
			cmd := tc.deployTaskCommand(actions)
			So(cmd, ShouldContain, SSWPath)
			So(cmd, ShouldContain, "-actions")
			So(cmd, ShouldContain, strings.Join(actions, ","))
			So(cmd, ShouldContain, "-logdog-annotation-url")
			So(cmd, ShouldContain, tc.LogdogURL())
		})
		Convey("deployTaskCommand - Without actions", func() {
			cmd := tc.deployTaskCommand([]string{})
			So(cmd, ShouldContain, SSWPath)
			So(cmd, ShouldNotContain, "-actions")
			So(cmd, ShouldContain, "-logdog-annotation-url")
			So(cmd, ShouldContain, tc.LogdogURL())
		})
		Convey("deployTaskCommand - Without logdog", func() {
			tc.LogdogService = ""
			actions := []string{"do-nothing", "do-something"}
			cmd := tc.deployTaskCommand(actions)
			So(cmd, ShouldContain, SSWPath)
			So(cmd, ShouldContain, "-actions")
			So(cmd, ShouldContain, strings.Join(actions, ","))
			So(cmd, ShouldNotContain, "-logdog-annotation-url")
		})
	})
}

func TestDeployDUTTask(t *testing.T) {
	t.Parallel()
	Convey("deployDutTask", t, func() {
		tc := &TaskCreator{
			session:                "TestSession",
			LogdogService:          "fake-logdog.appspot.com",
			LUCIProject:            "chromeos",
			SwarmingServiceAccount: "testServiceAccount@testmail.com",
			logdogTaskCode:         "logdoc_code",
		}
		Convey("deployDUTTask - Happy path", func() {
			req := tc.deployDUTTask("test-1", "testDUT", "testPool", "testUser", 30, []string{"reboot"}, []string{"test:yes"}, map[string]string{"bluetooth": "NO"})
			So(req.EvaluateOnly, ShouldBeFalse)
			So(req.Name, ShouldEqual, "Deploy")
			So(req.Priority, ShouldEqual, DeployTaskPriority)
			So(req.Tags, ShouldContain, "task:Deploy")
			So(req.Tags, ShouldContain, "admin_session:TestSession")
			So(req.Tags, ShouldContain, "deploy_task:testDUT")
			So(req.Tags, ShouldContain, "log_location:"+tc.LogdogURL())
			So(req.TaskSlices, ShouldHaveLength, 1)
			So(req.TaskSlices[0].Properties.Command, ShouldContain, SSWPath)
			So(req.TaskSlices[0].Properties.Command, ShouldContain, "-actions")
			So(req.TaskSlices[0].Properties.Command, ShouldContain, "-logdog-annotation-url")
			So(req.TaskSlices[0].Properties.Command, ShouldContain, tc.LogdogURL())
			So(req.User, ShouldEqual, "testUser")
			So(req.ServiceAccount, ShouldEqual, "testServiceAccount@testmail.com")
		})
		Convey("deployDUTTask - Missing logdog service and actions", func() {
			tc.LogdogService = ""
			req := tc.deployDUTTask("test-1", "testDUT", "testPool", "testUser", 30, []string{}, []string{}, nil)
			So(req.EvaluateOnly, ShouldBeFalse)
			So(req.Name, ShouldEqual, "Deploy")
			So(req.Priority, ShouldEqual, DeployTaskPriority)
			So(req.Tags, ShouldContain, "task:Deploy")
			So(req.Tags, ShouldContain, "admin_session:TestSession")
			So(req.Tags, ShouldContain, "deploy_task:testDUT")
			So(req.Tags, ShouldNotContain, "log_location:"+tc.LogdogURL())
			So(req.TaskSlices, ShouldHaveLength, 1)
			So(req.TaskSlices[0].Properties.Command, ShouldContain, SSWPath)
			So(req.TaskSlices[0].Properties.Command, ShouldNotContain, "-actions")
			So(req.TaskSlices[0].Properties.Command, ShouldNotContain, "-logdog-annotation-url")
			So(req.TaskSlices[0].Properties.Command, ShouldNotContain, tc.LogdogURL())
			So(req.User, ShouldEqual, "testUser")
			So(req.ServiceAccount, ShouldEqual, "testServiceAccount@testmail.com")
		})
	})
}
