// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"context"
	"fmt"
	"infra/chromeperf/pinpoint/proto"
	"os"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	. "github.com/smartystreets/goconvey/convey"
	grpc "google.golang.org/grpc"
)

type startedJob struct {
	Benchmark     string
	Configuration string
	Story         string
	StoryTags     []string
}

type fakePinpointClient struct {
	Jobs []startedJob
	mu   sync.Mutex
}

func less(x, y startedJob) bool {
	if x.Benchmark != y.Benchmark {
		return x.Benchmark < y.Benchmark
	} else if x.Configuration != y.Configuration {
		return x.Configuration < y.Configuration
	} else if x.Story != y.Story {
		return x.Story < y.Story
	} else if len(x.StoryTags) != len(y.StoryTags) {
		return len(x.StoryTags) < len(y.StoryTags)
	} else {
		for i := range x.StoryTags {
			if x.StoryTags[i] != y.StoryTags[i] {
				return x.StoryTags[i] < y.StoryTags[i]
			}
		}
	}
	return false
}

func (c *fakePinpointClient) ScheduleJob(ctx context.Context, in *proto.ScheduleJobRequest, opts ...grpc.CallOption) (*proto.Job, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := new(proto.Job)
	out.Name = "jobs/legacy-4242"
	started_job := startedJob{
		Benchmark:     in.Job.GetTelemetryBenchmark().Benchmark,
		Configuration: in.Job.Config,
		Story:         in.Job.GetTelemetryBenchmark().GetStory(),
	}
	if in.Job.GetTelemetryBenchmark().GetStoryTags() != nil {
		started_job.StoryTags = in.Job.GetTelemetryBenchmark().GetStoryTags().StoryTags
	} else {
		started_job.StoryTags = []string{}
	}
	c.Jobs = append(c.Jobs, started_job)
	return out, nil
}

func (c *fakePinpointClient) GetJob(ctx context.Context, in *proto.GetJobRequest, opts ...grpc.CallOption) (*proto.Job, error) {
	return nil, nil
}

func (c *fakePinpointClient) ListJobs(ctx context.Context, in *proto.ListJobsRequest, opts ...grpc.CallOption) (*proto.ListJobsResponse, error) {
	return nil, nil
}

func (c *fakePinpointClient) CancelJob(ctx context.Context, in *proto.CancelJobRequest, opts ...grpc.CallOption) (*proto.Job, error) {
	return nil, nil
}

func TestBatchKickoff(t *testing.T) {
	t.Parallel()
	Convey("A batch config should kick off a set of jobs", t, func() {
		batch_experiments := []telemetryBatchExperiment{
			{
				Benchmark: "desktop",
				Configs:   []string{"linux", "win"},
				Stories:   []string{"dsA", "dsB"},
				StoryTags: []string{},
			},
			{
				Benchmark: "mobile",
				Configs:   []string{"pixel"},
				Stories:   []string{"msA"},
				StoryTags: []string{"tagA", "tagB"},
			},
		}
		runner := experimentTelemetryRun{}
		experiment := proto.Experiment{}

		c := &fakePinpointClient{}

		var wg_errs sync.WaitGroup
		wg_errs.Add(1)
		errC := make(chan error)
		go handleErrors(context.Background(), &wg_errs, errC)

		var wg sync.WaitGroup
		runBatchJob(&runner, &wg, errC, context.Background(), os.Stdout, c, "", batch_experiments, &experiment)
		wg.Wait()
		close(errC)
		wg_errs.Wait()

		expected := []startedJob{
			{
				Benchmark:     "desktop",
				Configuration: "linux",
				Story:         "dsA",
				StoryTags:     []string{},
			},
			{
				Benchmark:     "desktop",
				Configuration: "linux",
				Story:         "dsB",
				StoryTags:     []string{},
			},
			{
				Benchmark:     "desktop",
				Configuration: "win",
				Story:         "dsA",
				StoryTags:     []string{},
			},
			{
				Benchmark:     "desktop",
				Configuration: "win",
				Story:         "dsB",
				StoryTags:     []string{},
			},
			{
				Benchmark:     "mobile",
				Configuration: "pixel",
				Story:         "msA",
				StoryTags:     []string{},
			},
			{
				Benchmark:     "mobile",
				Configuration: "pixel",
				Story:         "",
				StoryTags:     []string{"tagA", "tagB"},
			},
		}
		fmt.Printf("%s\n", c.Jobs)
		fmt.Printf("%s\n", expected)
		So(cmp.Equal(c.Jobs, expected, cmpopts.SortSlices(less)), ShouldBeTrue)
	})
}

func TestCLIFlagOverriding(t *testing.T) {
	t.Parallel()
	Convey("CLI args must override presets", t, func() {
		batch_experiments := []telemetryBatchExperiment{
			{
				Benchmark:   "desktop",
				Configs:     []string{"linux", "win"},
				Stories:     []string{"dsA", "dsB"},
				StoryTags:   []string{},
				Measurement: "FCP",
			},
			{
				Benchmark:   "mobile",
				Configs:     []string{"pixel"},
				Stories:     []string{"msA"},
				StoryTags:   []string{"tagA", "tagB"},
				Measurement: "LCP",
			},
		}
		runner := experimentTelemetryRun{}
		runner.configurations = []string{"config1", "config2"}
		runner.stories = []string{"story1, story2"}
		runner.storyTags = []string{"tag1", "tag2"}
		runner.measurement = "measurement"
		runner.benchmark = "benchmark"
		applyFlags(&runner, &batch_experiments, []string{"extra_arg1", "extra_arg2"})
		expected := []telemetryBatchExperiment{
			{
				Benchmark:   "benchmark",
				Configs:     []string{"config1", "config2"},
				Stories:     []string{"story1, story2"},
				StoryTags:   []string{"tag1", "tag2"},
				Measurement: "measurement",
				ExtraArgs:   []string{"extra_arg1", "extra_arg2"},
			},
			{
				Benchmark:   "benchmark",
				Configs:     []string{"config1", "config2"},
				Stories:     []string{"story1, story2"},
				StoryTags:   []string{"tag1", "tag2"},
				Measurement: "measurement",
				ExtraArgs:   []string{"extra_arg1", "extra_arg2"},
			},
		}
		fmt.Println(cmp.Diff(batch_experiments, expected))
		So(cmp.Equal(batch_experiments, expected), ShouldBeTrue)
	})

	Convey("A single CLI override must not impact other preset params", t, func() {
		batch_experiments := []telemetryBatchExperiment{
			{
				Benchmark:   "desktop",
				Configs:     []string{"linux", "win"},
				Stories:     []string{"dsA", "dsB"},
				StoryTags:   []string{},
				Measurement: "FCP",
			},
			{
				Benchmark:   "mobile",
				Configs:     []string{"pixel"},
				Stories:     []string{"msA"},
				StoryTags:   []string{"tagA", "tagB"},
				Measurement: "LCP",
			},
		}
		runner := experimentTelemetryRun{}
		runner.configurations = []string{"config1", "config2"}
		applyFlags(&runner, &batch_experiments, nil)
		expected := []telemetryBatchExperiment{
			{
				Benchmark:   "desktop",
				Configs:     []string{"config1", "config2"},
				Stories:     []string{"dsA", "dsB"},
				StoryTags:   []string{},
				Measurement: "FCP",
			},
			{
				Benchmark:   "mobile",
				Configs:     []string{"config1", "config2"},
				Stories:     []string{"msA"},
				StoryTags:   []string{"tagA", "tagB"},
				Measurement: "LCP",
			},
		}
		fmt.Println(cmp.Diff(batch_experiments, expected))
		So(cmp.Equal(batch_experiments, expected), ShouldBeTrue)
	})
}

func TestGetTelemetryBatchExperiment(t *testing.T) {
	t.Parallel()

	Convey("A valid experiment is generated from only CLI flags", t, func() {
		p := preset{}
		runner := experimentTelemetryRun{}
		runner.configurations = []string{"config1", "config2"}
		runner.stories = []string{"story1, story2"}
		runner.storyTags = []string{"tag1", "tag2"}
		runner.measurement = "measurement"
		runner.benchmark = "benchmark"
		actual, _ := getTelemetryBatchExperiment(&runner, nil, p)
		expected := []telemetryBatchExperiment{
			{
				Benchmark:   "benchmark",
				Configs:     []string{"config1", "config2"},
				Stories:     []string{"story1, story2"},
				StoryTags:   []string{"tag1", "tag2"},
				Measurement: "measurement",
			},
		}
		So(actual, ShouldResemble, expected)
	})

	Convey("A valid experiment is generated from a single-run preset and CLI flags", t, func() {
		experiment := telemetryExperimentJobSpec{
			Benchmark:   "benchmark",
			Config:      "cfg",
			Measurement: "measurement",
			ExtraArgs:   []string{"arg1"},
		}
		experiment.StorySelection.Story = "story"
		p := preset{
			TelemetryExperiment: &experiment,
		}
		runner := experimentTelemetryRun{}
		runner.configurations = []string{"config1", "config2"}
		actual, _ := getTelemetryBatchExperiment(&runner, nil, p)
		expected := []telemetryBatchExperiment{
			{
				Benchmark:   "benchmark",
				Configs:     []string{"config1", "config2"},
				Stories:     []string{"story"},
				Measurement: "measurement",
				ExtraArgs:   []string{"arg1"},
			},
		}
		So(actual, ShouldResemble, expected)
	})

	Convey("A valid experiment is generated from a multi-run preset and CLI flags", t, func() {
		batch_experiments := []telemetryBatchExperiment{
			{
				Benchmark:   "desktop",
				Configs:     []string{"linux", "win"},
				Stories:     []string{"dsA", "dsB"},
				StoryTags:   []string{},
				Measurement: "FCP",
			},
			{
				Benchmark:   "mobile",
				Configs:     []string{"pixel"},
				Stories:     []string{"msA"},
				StoryTags:   []string{"tagA", "tagB"},
				Measurement: "LCP",
			},
		}
		p := preset{
			TelemetryBatchExperiment: &batch_experiments,
		}
		runner := experimentTelemetryRun{}
		runner.configurations = []string{"config1", "config2"}
		actual, _ := getTelemetryBatchExperiment(&runner, nil, p)
		expected := []telemetryBatchExperiment{
			{
				Benchmark:   "desktop",
				Configs:     []string{"config1", "config2"},
				Stories:     []string{"dsA", "dsB"},
				StoryTags:   []string{},
				Measurement: "FCP",
			},
			{
				Benchmark:   "mobile",
				Configs:     []string{"config1", "config2"},
				Stories:     []string{"msA"},
				StoryTags:   []string{"tagA", "tagB"},
				Measurement: "LCP",
			},
		}
		So(actual, ShouldResemble, expected)
	})
}
