// Copyright 2020 The Chromium Authors.
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
	"fmt"
	"log"

	"infra/chromeperf/pinpoint"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/flag"
	"google.golang.org/protobuf/encoding/prototext"
)

type experimentTelemetryRun struct {
	experimentBaseRun
	benchmark, story, measurement string
	storyTags                     []string
}

func cmdTelemetryExperiment(p Param) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "experiment-telemetry-start <--flag...> -- <extra telemetry args>",
		ShortDesc: "starts a telemetry a/b experiment",
		LongDesc: text.Doc(`
			Starts an A/B experiment between two builds (a base and experiment) generating results.
			The extra telemetry arguments are passed to the invocation of the benchmark runner as-is.
			To differentiate flags for the subcommand, you can use '--':

			experiment-telemetry-start --benchmark=... -- --enable_features ...

			The extra telemetry args will be treated as a space-separated list.
		`),
		CommandRun: func() subcommands.CommandRun {
			e := &experimentTelemetryRun{}
			e.RegisterDefaultFlags(p)
			e.Flags.StringVar(&e.benchmark, "benchmark", "", text.Doc(`
				A telemetry benchmark.
			`))
			e.Flags.StringVar(&e.story, "story", "", text.Doc(`
				A story to run.
				Mutually exclusive with -story-tags.
			`))
			e.Flags.Var(flag.CommaList(&e.storyTags), "story-tags", text.Doc(`
				A comma-separated list of telemetry story tags.
				Mutually exclusive with -story.
			`))
			e.Flags.StringVar(&e.measurement, "measurement", "", text.Doc(`
				The measurement to pick out.
				When empty defaults to all measurements produced by the benchmark (optional).
			`))
			return e
		},
	}
}

const (
	chromiumGitilesHost    = "chromium.googlesource.com"
	chromiumGerritHost     = "chromium-review.googlesource.com"
	chromiumGerritProject  = "chromium/src"
	chromiumGitilesProject = "chromium/src"
)

func newTelemetryBenchmark(benchmark, measurement, story string, storyTags, extraArgs []string) *pinpoint.TelemetryBenchmark {
	tb := &pinpoint.TelemetryBenchmark{
		Benchmark:   benchmark,
		Measurement: measurement,
	}
	if len(story) > 0 {
		tb.StorySelection = &pinpoint.TelemetryBenchmark_Story{
			Story: story,
		}
	}
	if len(storyTags) > 0 {
		tb.StorySelection = &pinpoint.TelemetryBenchmark_StoryTags{
			StoryTags: &pinpoint.TelemetryBenchmark_StoryTagList{
				StoryTags: storyTags,
			},
		}
	}
	tb.ExtraArgs = extraArgs
	return tb
}

func (e *experimentTelemetryRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if (len(e.story) == 0 && len(e.storyTags) == 0) || (len(e.story) > 0 && len(e.storyTags) > 0) {
		fmt.Fprintln(a.GetErr(), "ERROR: pick one of -story or -story-tags")
		e.GetFlags().Usage()
		return 1
	}
	ctx := cli.GetContext(a, e, env)
	if err := e.initFactory(ctx); err != nil {

	}
	c, err := e.pinpointClientFactory.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(a.GetErr(), "ERROR: Failed to create a Pinpoint client: %s", err)
		return 1
	}
	js := &pinpoint.JobSpec{
		ComparisonMode: pinpoint.JobSpec_PERFORMANCE,
		Config:         e.configuration,

		// This is hard-coded for Chromium Telemetry.
		Target: "performance_test_suite",
		JobKind: &pinpoint.JobSpec_Experiment{
			Experiment: &pinpoint.Experiment{
				BaseCommit: &pinpoint.GitilesCommit{
					Host:    chromiumGitilesHost,
					Project: chromiumGitilesProject,
					GitHash: e.baseCommit,
				},
				ExperimentPatch: &pinpoint.GerritChange{
					Host:     chromiumGerritHost,
					Project:  chromiumGerritProject,
					Change:   e.expCL.clNum,
					Patchset: e.expCL.patchSet,
				},
			},
		},
		Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
			TelemetryBenchmark: newTelemetryBenchmark(
				e.benchmark, e.measurement, e.story, e.storyTags, e.Flags.Args()),
		},
	}
	exp := js.GetExperiment()

	if e.issue.issueID != 0 {
		js.MonorailIssue = &pinpoint.MonorailIssue{
			Project: e.issue.project,
			IssueId: e.issue.issueID,
		}
	}
	if e.baseCL.clNum > 0 {
		exp.BasePatch = &pinpoint.GerritChange{
			Host:     chromiumGerritHost,
			Project:  chromiumGerritProject,
			Change:   e.baseCL.clNum,
			Patchset: e.baseCL.patchSet,
		}
	}
	if len(e.expCommit) > 0 {
		exp.ExperimentCommit = &pinpoint.GitilesCommit{
			Host:    chromiumGitilesHost,
			Project: chromiumGitilesProject,
			GitHash: e.expCommit,
		}
	}
	j, err := c.ScheduleJob(ctx, &pinpoint.ScheduleJobRequest{Job: js})
	if err != nil {
		log.Printf("ERROR: Failed: %s", err)
		return 1
	}
	fmt.Fprintf(a.GetOut(), "Job: %s", prototext.Format(j))
	return 0
}
