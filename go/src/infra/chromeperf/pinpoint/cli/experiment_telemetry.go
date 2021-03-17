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
	"context"
	"fmt"

	"infra/chromeperf/pinpoint"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/protobuf/encoding/prototext"
)

type experimentTelemetryRun struct {
	experimentBaseRun
	benchmark, story, measurement string
	storyTags                     []string
}

func cmdTelemetryExperiment(p Param) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "experiment-telemetry-start <-flag...> -- <extra telemetry args>",
		ShortDesc: "starts a telemetry a/b experiment",
		LongDesc: text.Doc(`
			Starts an A/B experiment between two builds (a base and experiment) generating results.
			The extra telemetry arguments are passed to the invocation of the benchmark runner as-is.
			To differentiate flags for the subcommand, you can use '--':

			    experiment-telemetry-start -benchmark=... -- --enable_features ...

			The extra telemetry args will be treated as a space-separated list.

			Comparing at different commits:

 			    experiment-telemetry-start -benchmark=... -base-commit <...> -exp-commit <...>

			Applying non-chromium/src patches:

			    experiment-telemetry-start -project=v8 ...

		`),
		CommandRun: wrapCommand(p, func() pinpointCommand {
			return &experimentTelemetryRun{}
		}),
	}
}

func (e *experimentTelemetryRun) RegisterFlags(p Param) {
	e.experimentBaseRun.RegisterFlags(p)
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
}

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

func (e *experimentTelemetryRun) Run(ctx context.Context, a subcommands.Application, args []string) error {
	if (len(e.story) == 0 && len(e.storyTags) == 0) || (len(e.story) > 0 && len(e.storyTags) > 0) {
		e.GetFlags().Usage()
		return errors.Reason("pick one of -story or -story-tags").Err()
	}
	c, err := e.pinpointClient(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to create a Pinpoint client").Err()
	}

	js := &pinpoint.JobSpec{
		ComparisonMode: pinpoint.JobSpec_PERFORMANCE,
		Config:         e.configuration,

		// This is hard-coded for Chromium Telemetry.
		Target: "performance_test_suite",
		JobKind: &pinpoint.JobSpec_Experiment{
			Experiment: &pinpoint.Experiment{
				BaseCommit: &pinpoint.GitilesCommit{
					Host:    e.gitilesHost,
					Project: e.repository,
					GitHash: e.baseCommit,
				},
				ExperimentCommit: &pinpoint.GitilesCommit{
					Host:    e.gitilesHost,
					Project: e.repository,
					GitHash: e.expCommit,
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
			Host:     e.gerritHost,
			Project:  e.repository,
			Change:   e.baseCL.clNum,
			Patchset: e.baseCL.patchSet,
		}
	}
	if e.expCL.clNum > 0 {
		exp.ExperimentPatch = &pinpoint.GerritChange{
			Host:     e.gerritHost,
			Project:  e.repository,
			Change:   e.expCL.clNum,
			Patchset: e.expCL.patchSet,
		}
	}
	j, err := c.ScheduleJob(ctx, &pinpoint.ScheduleJobRequest{Job: js})
	if err != nil {
		return errors.Annotate(err, "failed to ScheduleJob").Err()
	}
	jobURL, err := legacyJobURL(j)
	var out string
	if err != nil {
		logging.Errorf(ctx, "ERROR: %s", err)
		out = prototext.Format(j)
	} else {
		out = jobURL
	}
	fmt.Fprintf(a.GetOut(), "Pinpoint job scheduled: %s\n", out)
	return nil
}
