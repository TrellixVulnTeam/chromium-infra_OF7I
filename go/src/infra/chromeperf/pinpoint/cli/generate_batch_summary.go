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
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"infra/chromeperf/output"
	"infra/chromeperf/pinpoint"
	"infra/chromeperf/pinpoint/cli/render"
	"infra/chromeperf/pinpoint/proto"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
)

const gerritBaseUrl = "https://chromium-review.googlesource.com/q/"

var nan = fmt.Sprintf("%f", math.NaN())

type generateBatchSummary struct {
	baseCommandRun
	presetsMixin
	batchFile, batchId string
}

func cmdGenerateBatchSummary(p Param) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "generate-batch-summary presets-file preset {batch-id | batch-file}",
		ShortDesc: text.Doc(`
			Summarizes a set of metrics (defined in the preset) for a batch of tests.
		`),
		LongDesc: text.Doc(`
		    Generates a .csv for each benchmark in the batch.
			Each story gets one row, and each metric defined in the preset
			gets three columns (p-value, Median (A), and Median (B)).
		`),
		CommandRun: wrapCommand(p, func() pinpointCommand {
			return &generateBatchSummary{}
		}),
	}
}

func (e *generateBatchSummary) RegisterFlags(p Param) {
	uc := e.baseCommandRun.RegisterFlags(p)
	e.presetsMixin.RegisterFlags(&e.Flags, uc)
	e.Flags.StringVar(&e.batchId, "batch-id", "", text.Doc(`
		The batch ID to analyze.
	`))
	e.Flags.StringVar(&e.batchFile, "batch-file", "", text.Doc(`
		The file containing a list of job IDs to analyze
	`))
}

type experimentResult struct {
	Report          *experimentReport
	Cfg, Story, URL string
}

func (e *generateBatchSummary) getJobs(ctx context.Context, c proto.PinpointClient) ([]*proto.Job, error) {
	if e.batchId == "" && e.batchFile == "" {
		return nil, fmt.Errorf("--batch-id or --batch-file must be specified!")
	}

	var jobs []*proto.Job
	if e.batchId != "" {
		req := &proto.ListJobsRequest{Filter: "batch_id=" + e.batchId}
		resp, err := c.ListJobs(ctx, req)
		if err != nil {
			return jobs, err
		}
		jobs = resp.Jobs
	} else if e.batchFile != "" {
		infile, err := os.Open(e.batchFile)
		if err != nil {
			return jobs, err
		}
		defer infile.Close()

		scanner := bufio.NewScanner(infile)
		for scanner.Scan() {
			jobId := scanner.Text()
			fmt.Println("Fetching " + jobId)
			req := &proto.GetJobRequest{Name: pinpoint.LegacyJobName(jobId)}
			j, err := c.GetJob(ctx, req)
			if err != nil {
				return jobs, errors.Annotate(err, "failed during GetJob").Err()
			}
			jobs = append(jobs, j)
		}
	}

	if len(jobs) == 0 {
		return jobs, fmt.Errorf("No jobs were run")
	}
	return jobs, nil
}

func (e *generateBatchSummary) waitAndDownloadJobArtifacts(ctx context.Context, c proto.PinpointClient, o io.Writer, jobs []*proto.Job) error {
	w := waitForJobMixin{
		wait:  true,
		quiet: false,
	}
	dr := downloadResultsMixin{
		downloadResults: false,
	}
	da := downloadArtifactsMixin{
		downloadArtifacts: true,
		selectArtifacts:   "test",
	}
	ae := analyzeExperimentMixin{
		analyzeExperiment: false,
	}
	return waitAndDownloadJobList(&e.baseCommandRun,
		w, dr, da, ae, ctx, o, c, jobs)
}

// Returns baseChangeConfig, expChangeConfig, error
func (e *generateBatchSummary) getChangeConfigs(j *proto.Job) (changeConfig, changeConfig, error) {
	manifest, err := loadManifestFromJob(e.baseCommandRun.workDir, j)
	if err != nil {
		return changeConfig{}, changeConfig{}, err
	}
	return manifest.Base, manifest.Experiment, nil
}

// Searches the job for the first output.json file.
// We use this to determine the benchmark and story.
func loadOutput(config *changeConfig, rootDir string) (*output.Output, error) {
	var r *output.Output = nil
	for _, a := range config.Artifacts {
		if a.Selector != "test" {
			continue
		}
		for _, f := range a.Files {
			dir := filepath.Join(rootDir, filepath.FromSlash(f.Path))
			if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}
				if d.Name() == "output.json" {
					jf, err := os.Open(path)
					if err != nil {
						return errors.Annotate(err, "failed loading file: %q", path).Err()
					}
					defer jf.Close()
					r, err = output.NewFromJSON(jf)
					return nil
				}
				return nil
			}); err != nil {
				return nil, err
			}
		}
	}
	if r == nil {
		return nil, fmt.Errorf("Could not find an output.json")
	}

	return r, nil
}

func (e *generateBatchSummary) getExperimentResults(jobs []*proto.Job) (map[string][]experimentResult, error) {
	resultMap := make(map[string][]experimentResult) // map[benchmark_name] -> list of experimentResult
	for _, j := range jobs {
		url, err := render.JobURL(j)
		if err != nil {
			return resultMap, err
		}
		jobId, err := render.JobID(j)
		if err != nil {
			return nil, err
		}
		manifest, err := loadManifestFromJob(e.baseCommandRun.workDir, j)
		if err != nil {
			return resultMap, err
		}
		rootDir := path.Join(e.baseCommandRun.workDir, jobId)
		report, err := analyzeExperiment(manifest, rootDir)
		if err != nil {
			return resultMap, err
		}

		// Derive benchmark, story, and pass/fail
		var benchmark, story, status string
		output, err := loadOutput(&manifest.Base, rootDir)
		if err != nil {
			fmt.Printf("Skipping job %s due to error %s", url, err)
			continue
		}
		if len(output.Tests) != 1 {
			fmt.Printf("Skipping job with number of benchmarks != 1")
			continue
		}
		for b, stories := range output.Tests {
			if len(stories) != 1 {
				break // Skip below
			}
			benchmark = string(b)
			for s, result := range stories {
				story = string(s)
				status = result.Actual
			}
		}
		if status == "" {
			fmt.Printf("Skipping job with number of stories != 1")
			continue
		}
		if status != "PASS" {
			// crbug.com/1218475 - reflect this in the .csvs
			fmt.Println("Skipping failing story " + story)
			continue
		}

		values := experimentResult{
			Report: report,
			Cfg:    manifest.Config,
			Story:  story,
			URL:    url,
		}
		resultMap[benchmark] = append(resultMap[benchmark], values)
	}
	return resultMap, nil
}

func getGerritURL(res string) string {
	if res == "" {
		return ""
	}
	return gerritBaseUrl + res
}

func (e *generateBatchSummary) generateABCSV(
	resultsDir string,
	baseChange, expChange changeConfig) error {
	outfilePath := path.Join(resultsDir, "cover.csv")
	fmt.Println("Generating cover " + outfilePath)
	outfile, err := os.Create(outfilePath)
	if err != nil {
		return err
	}
	defer outfile.Close()
	outcsv := csv.NewWriter(outfile)

	outcsv.Write([]string{"", "A", "B"})
	outcsv.Write([]string{"Commit", getGerritURL(baseChange.Commit), getGerritURL(expChange.Commit)})
	outcsv.Write([]string{"Applied Change", getGerritURL(baseChange.Change), getGerritURL(expChange.Change)})
	outcsv.Flush()
	return nil
}

// Generates one CSV per benchmark with the following format:
// (<benchmark name>.csv)
//    ,          ,      ,     , Metric0   ,           ,     , Metric 1  ,           , ...
// URL, DeviceCfg, Story, pval, Median (A), Median (B), pval, Median (A), Median (B), ...
// ...
func (e *generateBatchSummary) generateBenchmarkCSVs(resultsDir string, p preset, experimentResults map[string][]experimentResult) error {
	for b, storyResults := range experimentResults {
		_, found := (*p.BatchSummaryReportSpec)[b]
		if !found {
			fmt.Printf("Preset does not define metrics for benchmark " + b)
			continue
		}
		metrics := (*p.BatchSummaryReportSpec)[b].Metrics

		outfilePath := path.Join(resultsDir, b+".csv")
		fmt.Println("Generating report " + outfilePath)
		outfile, err := os.Create(outfilePath)
		if err != nil {
			return err
		}
		defer outfile.Close()
		outcsv := csv.NewWriter(outfile)

		line := []string{"URL", "Cfg", "Story", ""}
		for _, m := range *metrics {
			line = append(line, []string{m.Name, "", ""}...)
		}
		outcsv.Write(line)

		line = []string{"", "", ""}
		for range *metrics {
			line = append(line, []string{"pval", "Median (A)", "Median (B)"}...)
		}
		outcsv.Write(line)

		for _, result := range storyResults {
			line = []string{result.URL, result.Cfg, result.Story}
			for _, m := range *metrics {
				report, found := result.Report.Reports[metricNameKey(m.Name)]
				if found {
					line = append(line, fmt.Sprintf("%.6f", report.PValue))
					line = append(line, fmt.Sprintf("%.5f", report.Measurements[baseLabel].Median))
					line = append(line, fmt.Sprintf("%.5f", report.Measurements[expLabel].Median))
				} else {
					line = append(line, []string{nan, nan, nan}...)
				}
			}
			outcsv.Write(line)
		}
		outcsv.Flush()
	}
	return nil
}

// Generates a single CSV with the following format (no header):
// URL, AorB, Benchmark, DeviceCfg, Story, Metric, Value
func (e *generateBatchSummary) generateRawCSV(resultsDir string, p preset, experimentResults map[string][]experimentResult) error {
	outfilePath := path.Join(resultsDir, "raw.csv")
	fmt.Println("Generating report " + outfilePath)
	outfile, err := os.Create(outfilePath)
	if err != nil {
		return err
	}
	defer outfile.Close()
	outcsv := csv.NewWriter(outfile)

	outcsv.Write([]string{"URL", "AorB", "Benchmark", "Cfg", "Story",
		"Metric", "Value"})

	benchmarkKeys := []string{}
	for b := range experimentResults {
		benchmarkKeys = append(benchmarkKeys, b)
	}
	sort.Strings(benchmarkKeys)

	for _, b := range benchmarkKeys {
		storyResults := experimentResults[b]

		_, found := (*p.BatchSummaryReportSpec)[b]
		if !found {
			fmt.Printf("Preset does not define metrics for benchmark " + b)
			continue
		}
		metrics := (*p.BatchSummaryReportSpec)[b].Metrics

		for _, result := range storyResults {
			for _, m := range *metrics {

				report, found := result.Report.Reports[metricNameKey(m.Name)]
				if found {
					info := []string{result.URL, "A", b, result.Cfg, result.Story, m.Name}
					for _, val := range report.Measurements[baseLabel].Raw {
						outcsv.Write(append(info, fmt.Sprintf("%f", val)))
					}

					info = []string{result.URL, "B", b, result.Cfg, result.Story, m.Name}
					for _, val := range report.Measurements[expLabel].Raw {
						outcsv.Write(append(info, fmt.Sprintf("%f", val)))
					}
				}
			}
		}
		outcsv.Flush()
	}
	return nil
}

func (e *generateBatchSummary) analyzeArtifactsAndGenerateCSVs(outputDir string, p preset, batchId string, jobs []*proto.Job) error {
	// Analyze all artifacts
	baseChange, expChange, err := e.getChangeConfigs(jobs[0])
	if err != nil {
		return err
	}
	experimentResults, err := e.getExperimentResults(jobs)

	// Generate CSVs
	resultsDir := path.Join(outputDir, batchId+"_summaries")
	if err := removeExisting(resultsDir); err != nil {
		return errors.Annotate(err, "cannot download artifacts").Err()
	}
	err = os.Mkdir(resultsDir, 0755)
	if err != nil {
		return err
	}
	err = e.generateABCSV(resultsDir, baseChange, expChange)
	if err != nil {
		return err
	}
	err = e.generateBenchmarkCSVs(resultsDir, p, experimentResults)
	if err != nil {
		return err
	}
	err = e.generateRawCSV(resultsDir, p, experimentResults)
	if err != nil {
		return err
	}
	return nil
}

func (e *generateBatchSummary) Run(ctx context.Context, a subcommands.Application, args []string) error {
	c, err := e.pinpointClient(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to create a Pinpoint client").Err()
	}

	p, err := e.getPreset(ctx)
	if err != nil {
		return errors.Annotate(err, "unable to load preset").Err()
	}
	if p.BatchSummaryReportSpec == nil {
		return fmt.Errorf("Preset must be a batch_summary_report_spec")
	}

	// Determine the set of jobs in the batch and download all of their artifacts
	jobs, err := e.getJobs(ctx, c)
	if err != nil {
		return err
	}
	batchId := e.batchId
	if e.batchFile != "" {
		batchId = strings.TrimSuffix(filepath.Base(e.batchFile), filepath.Ext(e.batchFile))
	}
	err = e.waitAndDownloadJobArtifacts(ctx, c, a.GetOut(), jobs)
	if err != nil {
		return err
	}

	return e.analyzeArtifactsAndGenerateCSVs(e.baseCommandRun.workDir, p, batchId, jobs)
}
