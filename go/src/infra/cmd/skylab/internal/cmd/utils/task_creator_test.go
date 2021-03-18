package utils

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"

	"infra/cmd/skylab/internal/site"
)

var combineTagsCases = []struct {
	name       string
	toolName   string
	logDogURL  string
	customTags []string
	out        []string
}{
	{
		"test1",
		"tool1",
		"",
		nil,
		[]string{
			"skylab-tool:tool1",
			"luci_project:Env1",
			"pool:ChromeOSSkylab",
			"admin_session:session1",
		},
	},
	{
		"test2",
		"tool2",
		"log2",
		[]string{},
		[]string{
			"skylab-tool:tool2",
			"luci_project:Env1",
			"pool:ChromeOSSkylab",
			"admin_session:session1",
			"log_location:log2",
		},
	},
	{
		"test3",
		"tool3",
		"",
		[]string{
			"mytag:val3",
		},
		[]string{
			"skylab-tool:tool3",
			"luci_project:Env1",
			"pool:ChromeOSSkylab",
			"admin_session:session1",
			"mytag:val3",
		},
	},
	{
		"test4",
		"tool4",
		"log4",
		[]string{
			"mytag:val4",
		},
		[]string{
			"skylab-tool:tool4",
			"luci_project:Env1",
			"pool:ChromeOSSkylab",
			"admin_session:session1",
			"log_location:log4",
			"mytag:val4",
		},
	},
}

func TestCombineTags(t *testing.T) {
	t.Parallel()
	for _, tt := range combineTagsCases {
		t.Run(tt.name, func(t *testing.T) {
			tc := &TaskCreator{
				Environment: site.Environment{
					LUCIProject: "Env1",
				},
				session: "session1",
			}
			got := tc.combineTags(tt.toolName, tt.logDogURL, tt.customTags)
			if diff := cmp.Diff(tt.out, got); diff != "" {
				t.Errorf("%s output mismatch (-want +got): %s\n", tt.name, diff)
			}
		})
	}
}

func TestDeployTask(t *testing.T) {
	t.Parallel()
	Convey("Test deploytask of task creator", t, func() {
		Convey("Verify deploy task has the highest priority", func() {
			So(deployTaskPriority, ShouldBeLessThan, defaultTaskPriority)
		})
		Convey("Verify deploy task request is correct formated", func() {
			tc := &TaskCreator{
				Client:      nil,
				Environment: site.Dev,
				session:     "session0",
			}
			r := tc.getDeployTaskRequest("fake_dut_id", "fake_actions")
			So(r.Name, ShouldEqual, "deploy")
			So(r.TaskSlices, ShouldHaveLength, 1)
			command := strings.Join(r.TaskSlices[0].Properties.Command, " ")
			So(command, ShouldContainSubstring, "/opt/infra-tools/skylab_swarming_worker -actions fake_actions -logdog-annotation-url")
			So(command, ShouldContainSubstring, "-task-name deploy")
			for _, d := range r.TaskSlices[0].Properties.Dimensions {
				switch d.Key {
				case "pool":
					So(d.Value, ShouldEqual, "ChromeOSSkylab")
				case "dut_id":
					So(d.Value, ShouldEqual, "fake_dut_id")
				}
			}
			So("skylab-tool:deploy", ShouldBeIn, r.Tags)
			So("admin_session:session0", ShouldBeIn, r.Tags)
			So("deploy_task:fake_dut_id", ShouldBeIn, r.Tags)
			So("pool:ChromeOSSkylab", ShouldBeIn, r.Tags)
		})
	})
}

func TestGetLeaseCommand(t *testing.T) {
	t.Parallel()
	Convey("Create command for lease tasks ", t, func() {
		Convey("Task with update DUT state to needs_repair", func() {
			cmd := getLeaseCommand(true)
			So(cmd[0], ShouldEqual, "/bin/sh")
			So(cmd[1], ShouldEqual, "-c")
			So(cmd[2], ShouldEqual, `/opt/infra-tools/skylab_swarming_worker -task-name set_needs_repair; while true; do sleep 60; echo Zzz...; done`)
		})
		Convey("Task without update DUT state to needs_repair", func() {
			cmd := getLeaseCommand(false)
			So(cmd[0], ShouldEqual, "/bin/sh")
			So(cmd[1], ShouldEqual, "-c")
			So(cmd[2], ShouldEqual, `while true; do sleep 60; echo Zzz...; done`)
		})
	})
}
