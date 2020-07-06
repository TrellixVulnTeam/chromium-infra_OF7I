package utils

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"

	"infra/cmd/skylab/internal/site"
)

var appendUniqueDimensionsCases = []struct {
	name  string
	first []*swarming_api.SwarmingRpcsStringPair
	rest  []*swarming_api.SwarmingRpcsStringPair
	out   []*swarming_api.SwarmingRpcsStringPair
}{
	{
		name:  "append(nil, nil...)",
		first: nil,
		rest:  nil,
		out:   nil,
	},
	{
		name:  "append(empty, nil...)",
		first: []*swarming_api.SwarmingRpcsStringPair{},
		rest:  nil,
		out:   []*swarming_api.SwarmingRpcsStringPair{},
	},
	{
		name: "append([a, b], nil...)",
		first: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
		},
		rest: nil,
		out: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
		},
	},
	{
		name: "duplicate initial",
		first: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
			{Key: "a", Value: "b"},
		},
		rest: nil,
		out: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
			{Key: "a", Value: "b"},
		},
	},
	{
		name: "duplicate split between first and rest",
		first: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
		},
		rest: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
		},
		out: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
		},
	},
	{
		name:  "duplicate split between rest only",
		first: nil,
		rest: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
			{Key: "a", Value: "b"},
		},
		out: []*swarming_api.SwarmingRpcsStringPair{
			{Key: "a", Value: "b"},
		},
	},
}

func TestAppendUniqueDimensions(t *testing.T) {
	t.Parallel()
	for _, tt := range appendUniqueDimensionsCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := appendUniqueDimensions(tt.first, tt.rest...)
			if diff := cmp.Diff(tt.out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

var appendUniqueTagsCases = []struct {
	name  string
	first []string
	rest  []string
	out   []string
}{
	{
		name:  "empty",
		first: nil,
		rest:  nil,
		out:   nil,
	},
	{
		name:  "singleton (first)",
		first: []string{"a:b"},
		rest:  nil,
		out:   []string{"a:b"},
	},
	{
		name:  "singleton (rest)",
		first: nil,
		rest:  []string{"a:b"},
		out:   []string{"a:b"},
	},
	{
		name:  "duplicate",
		first: []string{"a:b", "a:b"},
		rest:  nil,
		out:   []string{"a:b", "a:b"},
	},
	{
		name:  "duplicate",
		first: []string{"a:b"},
		rest:  []string{"a:b"},
		out:   []string{"a:b"},
	},
	{
		name:  "duplicate",
		first: nil,
		rest:  []string{"a:b", "a:b"},
		out:   []string{"a:b"},
	},
	{
		name:  "leading colon",
		first: nil,
		rest:  []string{":1", ":2"},
		out:   []string{":1"},
	},
	{
		name:  "trailing colon",
		first: nil,
		rest:  []string{"1:", "2:"},
		out:   []string{"1:", "2:"},
	},
	{
		name:  "multiple colons",
		first: nil,
		rest:  []string{"1:2:3", "1:4"},
		out:   []string{"1:2:3"},
	},
}

func TestAppendUniqueTags(t *testing.T) {
	t.Parallel()
	for _, tt := range appendUniqueTagsCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := appendUniqueTags(tt.first, tt.rest...)
			if diff := cmp.Diff(tt.out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

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
