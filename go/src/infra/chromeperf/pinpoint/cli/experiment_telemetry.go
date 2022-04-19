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
	"infra/chromeperf/pinpoint/cli/render"
	"io"
	"os"
	"path"
	"sync"

	"infra/chromeperf/pinpoint/proto"

	"github.com/google/uuid"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/sync/parallel"
)

// TODO(crbug/1230880): Increase concurrency after we solve the underlying Datastore issue.
const MaxScheduleConcurrency = 1

const defaultJobPriority = 0
const batchJobPriority = 10

type experimentTelemetryRun struct {
	experimentBaseRun
	benchmark, measurement string
	stories, storyTags     []string
	initialAttemptCount    int
}

func cmdTelemetryExperiment(p Param) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "experiment-telemetry-start <-flag...> -- <extra telemetry args>",
		ShortDesc: "starts a telemetry a/b experiment",
		LongDesc: text.Doc(`
		experiment-telemetry-start schedules an A/B experiment between two
		builds (a base and experiment) generating results. Alternatively, a set
		of A/B experiments can be kicked off using a preset (see below). The extra
		telemetry arguments are passed to the invocation of the benchmark
		runner as-is.  To differentiate flags for the subcommand, you can use
		'--':

		experiment-telemetry-start -benchmark=... -- --enable_features ...

		The extra telemetry args will be treated as a space-separated list.

		Comparing at different commits:
			experiment-telemetry-start -benchmark=... -base-commit <...> -exp-commit <...>

		Applying non-chromium/src patches:
			experiment-telemetry-start -project=v8 ...

		Waiting for and downloading results:
			experiment-telemetry-start -benchmark=... -wait -download-results

		PRESETS

		See https://source.chromium.org/chromium/infra/infra/+/master:go/src/infra/chromeperf/doc/pinpoint/cli/job-presets.md
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
	e.Flags.Var(flag.CommaList(&e.stories), "stories", text.Doc(`
		See "story".
	`))
	e.Flags.Var(e.Flags.Lookup("stories").Value, "story", text.Doc(`
		A story (or comma-separated list of stories) to run.
	`))
	e.Flags.Var(flag.CommaList(&e.storyTags), "story-tags", text.Doc(`
		A comma-separated list of telemetry story tags.
	`))
	e.Flags.StringVar(&e.measurement, "measurement", "", text.Doc(`
		The measurement to pick out.
		When empty defaults to all measurements produced by the benchmark (optional).
	`))
	e.Flags.IntVar(&e.initialAttemptCount, "attempts", 10, text.Doc(`
		The number of A and B iterations to execute.
	`))
}

// Generates the set of jobs to be run from a preset and CLI flags
func getTelemetryBatchExperiments(e *experimentTelemetryRun,
	ctx context.Context,
	p preset) ([]telemetryBatchExperiment, error) {
	var batch_experiments []telemetryBatchExperiment
	if p.TelemetryBatchExperiment != nil {
		batch_experiments = *p.TelemetryBatchExperiment
	} else if p.TelemetryExperiment != nil {
		single_experiment := telemetryBatchExperiment{
			Benchmark:   p.TelemetryExperiment.Benchmark,
			Configs:     []string{p.TelemetryExperiment.Config},
			Measurement: p.TelemetryExperiment.Measurement,
			ExtraArgs:   p.TelemetryExperiment.ExtraArgs,
		}
		if len(p.TelemetryExperiment.StorySelection.Story) > 0 {
			single_experiment.Stories = []string{p.TelemetryExperiment.StorySelection.Story}
		} else if len(p.TelemetryExperiment.StorySelection.StoryTags) > 0 {
			single_experiment.StoryTags = p.TelemetryExperiment.StorySelection.StoryTags
		}
		batch_experiments = []telemetryBatchExperiment{single_experiment}
	} else {
		// The job is specified in command line flags.
		if len(e.configurations) == 0 || len(e.benchmark) == 0 ||
			(len(e.stories) == 0 && len(e.storyTags) == 0) {
			// We can't generate a complete job.
			return make([]telemetryBatchExperiment, 0), nil
		}
		// This single entry will be populated by applyFlags
		batch_experiments = make([]telemetryBatchExperiment, 1)
	}
	extra_args := e.Flags.Args()
	applyFlags(e, &batch_experiments, extra_args)
	return batch_experiments, nil
}

func getExperimentPriority(p preset) int32 {
	if p.TelemetryBatchExperiment != nil {
		return batchJobPriority
	}
	return defaultJobPriority // Use default priority for non-batch jobs
}

func applyFlags(e *experimentTelemetryRun,
	batch_experiments *[]telemetryBatchExperiment,
	extraArgs []string) {
	for i := range *batch_experiments {
		if len(e.configurations) > 0 {
			(*batch_experiments)[i].Configs = e.configurations
		}
		if len(e.stories) > 0 {
			(*batch_experiments)[i].Stories = e.stories
		}
		if len(e.storyTags) > 0 {
			(*batch_experiments)[i].StoryTags = e.storyTags
		}
		if len(e.measurement) > 0 {
			(*batch_experiments)[i].Measurement = e.measurement
		}
		if len(e.benchmark) > 0 {
			(*batch_experiments)[i].Benchmark = e.benchmark
		}
		if len(extraArgs) > 0 {
			(*batch_experiments)[i].ExtraArgs = extraArgs
		}
	}
}

func getExperiment(e *experimentTelemetryRun) *proto.Experiment {
	exp := proto.Experiment{
		BaseCommit: &proto.GitilesCommit{
			Host:    e.gitilesHost,
			Project: e.repository,
			GitHash: e.baseCommit,
		},
		ExperimentCommit: &proto.GitilesCommit{
			Host:    e.gitilesHost,
			Project: e.repository,
			GitHash: e.expCommit,
		},
	}
	if e.baseCL.clNum > 0 {
		exp.BasePatch = &proto.GerritChange{
			Host:     e.gerritHost,
			Project:  e.repository,
			Change:   e.baseCL.clNum,
			Patchset: e.baseCL.patchSet,
		}
	}
	if e.expCL.clNum > 0 {
		exp.ExperimentPatch = &proto.GerritChange{
			Host:     e.gerritHost,
			Project:  e.repository,
			Change:   e.expCL.clNum,
			Patchset: e.expCL.patchSet,
		}
	}
	return &exp
}

func newTelemetryBenchmark(benchmark, measurement, story string, storyTags, extraArgs []string) *proto.TelemetryBenchmark {
	tb := &proto.TelemetryBenchmark{
		Benchmark:   benchmark,
		Measurement: measurement,
	}
	if len(story) > 0 {
		tb.StorySelection = &proto.TelemetryBenchmark_Story{
			Story: story,
		}
	}
	if len(storyTags) > 0 {
		tb.StorySelection = &proto.TelemetryBenchmark_StoryTags{
			StoryTags: &proto.TelemetryBenchmark_StoryTagList{
				StoryTags: storyTags,
			},
		}
	}
	tb.ExtraArgs = extraArgs
	return tb
}

var (
	botCfgs = map[string]string{
		"Android Nexus5X WebView Perf": "performance_webview_test_suite",
		"android-go_webview-perf":      "performance_webview_test_suite",
		"android-pixel2_webview-perf":  "performance_webview_test_suite",
		"android-pixel4_webview-perf":  "performance_webview_test_suite",
		"Android Nexus5 Perf":          "performance_test_suite_android_chrome",
		"android-pixel4a_power-perf":   "performance_test_suite_android_clank_chrome",
		"android-pixel2-perf":          "performance_test_suite_android_clank_monochrome_64_32_bundle",
		"android-pixel4-perf":          "performance_test_suite_android_clank_trichrome_bundle",
		"android-pixel2_weblayer-perf": "performance_weblayer_test_suite",
		"android-pixel4_weblayer-perf": "performance_weblayer_test_suite",
		"lacros-eve-perf":              "performance_test_suite_eve",
	}
)

func getTarget(cfg string) string {
	if ret, ok := botCfgs[cfg]; ok {
		return ret
	}
	return "performance_test_suite"
}

func scheduleTelemetryJob(e *experimentTelemetryRun,
	ctx context.Context,
	c proto.PinpointClient,
	batch_id string,
	initial_attempt_count int,
	experiment *proto.Experiment,
	bot_cfg, benchmark, measurement, story string,
	storyTags, extraArgs []string, priority int32) (*proto.Job, error) {
	js := &proto.JobSpec{
		BatchId:        batch_id,
		ComparisonMode: proto.JobSpec_PERFORMANCE,
		Config:         bot_cfg,
		Target:         getTarget(bot_cfg),
		JobKind: &proto.JobSpec_Experiment{
			Experiment: experiment,
		},
		Arguments: &proto.JobSpec_TelemetryBenchmark{
			TelemetryBenchmark: newTelemetryBenchmark(
				benchmark, measurement, story, storyTags, extraArgs),
		},
		Priority:            priority,
		InitialAttemptCount: int32(initial_attempt_count),
	}

	if e.issue.issueID != 0 {
		js.MonorailIssue = &proto.MonorailIssue{
			Project: e.issue.project,
			IssueId: e.issue.issueID,
		}
	}
	j, err := c.ScheduleJob(ctx, &proto.ScheduleJobRequest{Job: js})
	if err != nil {
		job_debug := fmt.Sprintf("Bot: %s Benchmark: %s Story: %s StoryTags: %s", bot_cfg, benchmark, story, storyTags)
		return nil, errors.Annotate(err, "failed to ScheduleJob for "+job_debug).Err()
	}
	jobURL, err := render.JobURL(j)
	if err != nil {
		return j, err
	}
	fmt.Printf("Pinpoint job scheduled: %s\n", jobURL)
	return j, nil
}

func runBatchJob(e *experimentTelemetryRun,
	ctx context.Context,
	o io.Writer,
	c proto.PinpointClient,
	batch_id string,
	batch_experiments []telemetryBatchExperiment,
	experiment *proto.Experiment,
	priority int32) ([]*proto.Job, error) {

	outpath := path.Join(e.baseCommandRun.workDir, batch_id+".txt")
	outfile, err := os.Create(outpath)
	if err != nil {
		return nil, err
	}
	defer outfile.Close()

	var jobsMu sync.Mutex
	jobs := []*proto.Job{}

	err = parallel.WorkPool(MaxScheduleConcurrency, func(workC chan<- func() error) {
		for _, config := range batch_experiments {
			config := config
			for _, bot_config := range config.Configs {
				bot_config := bot_config
				for _, story := range config.Stories {
					story := story
					workC <- func() error {
						j, err := scheduleTelemetryJob(e, ctx,
							c, batch_id, e.initialAttemptCount,
							experiment, bot_config, config.Benchmark,
							config.Measurement, story,
							[]string{}, config.ExtraArgs, priority)
						if err != nil {
							return err
						}
						jobsMu.Lock()
						defer jobsMu.Unlock()
						jobs = append(jobs, j)
						return nil
					}
				}
				if len(config.StoryTags) > 0 {
					workC <- func() error {
						j, err := scheduleTelemetryJob(e, ctx,
							c, batch_id, e.initialAttemptCount,
							experiment, bot_config, config.Benchmark,
							config.Measurement, "",
							config.StoryTags, config.ExtraArgs, priority)
						if err != nil {
							return err
						}
						jobsMu.Lock()
						defer jobsMu.Unlock()
						jobs = append(jobs, j)
						return nil
					}
				}
			}
		}
	})
	for _, j := range jobs {
		jobID, err := render.JobID(j)
		if err == nil {
			outfile.WriteString(jobID + "\n")
		}
	}
	fmt.Fprintln(o, "Also wrote new jobs to "+outpath)
	return jobs, err
}

func (e *experimentTelemetryRun) Run(ctx context.Context, a subcommands.Application, args []string) error {
	c, err := e.pinpointClient(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to create a Pinpoint client").Err()
	}

	p, err := e.getPreset(ctx)
	if err != nil {
		return errors.Annotate(err, "unable to load preset").Err()
	}
	if e.presetsMixin.presetName != "" && p.TelemetryExperiment == nil && p.TelemetryBatchExperiment == nil {
		return fmt.Errorf("Preset must be a telemetry_batch_experiment or telemetry_experiment")
	}

	if (e.baseCommit == "HEAD" || e.expCommit == "HEAD") &&
		(len(e.configurations) > 1 || len(e.stories) > 1 || p.TelemetryBatchExperiment != nil) {
		return fmt.Errorf("--base-commit and --exp-commit must be explicitly defined to be something other than HEAD when running a batch of jobs.")
	}

	batch_experiments, err := getTelemetryBatchExperiments(e, ctx, p)
	if err != nil {
		return err
	}
	if len(batch_experiments) == 0 {
		return fmt.Errorf("No jobs specified to start. Provide a preset or benchmark + config + (story or story tag).")
	}

	experiment := getExperiment(e)
	batch_id := uuid.New().String()
	fmt.Fprintf(a.GetOut(), "Created job batch: %s\n", batch_id)
	defer fmt.Fprintf(a.GetOut(), "Finished actions for batch: %s\n", batch_id)

	jobs, err := runBatchJob(e, ctx, a.GetOut(), c, batch_id,
		batch_experiments, experiment, getExperimentPriority(p))
	if err != nil {
		return errors.Annotate(err, "Failed to start all jobs: ").Err()
	}

	err = waitAndDownloadJobList(&e.baseCommandRun,
		e.waitForJobMixin, e.downloadResultsMixin,
		e.downloadArtifactsMixin, e.analyzeExperimentMixin, ctx, a.GetOut(), c, jobs)
	if err != nil {
		return errors.Annotate(err, "Failed to wait and download jobs: ").Err()
	}

	return nil
}
