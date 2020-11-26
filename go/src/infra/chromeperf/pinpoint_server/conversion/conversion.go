package conversion

import (
	"fmt"
	"net/url"
	"strings"

	"infra/chromeperf/pinpoint"

	"go.chromium.org/luci/common/errors"
)

func convertGerritChangeToURL(c *pinpoint.GerritChange) (string, error) {
	if c.Host == "" || c.Project == "" || c.Change == 0 || c.Patchset == 0 {
		return "", errors.Reason("all of host, project, change, and patchset must be provided").Err()
	}
	return fmt.Sprintf("https://%s/c/%s/+/%d/%d", c.Host, c.Project, c.Change, c.Patchset), nil
}

// ConvertToValues turns a pinpoint.JobSpec into a url.Values which can then be HTTP form-encoded for the legacy
// Pinpoint API.
func ConvertToValues(job *pinpoint.JobSpec, userEmail string) (url.Values, error) {
	v := url.Values{}
	v.Set("user", userEmail)

	// Lift the configuration into a key.
	v.Set("configuration", job.Config)

	// Always set the target into a key.
	v.Set("target", job.Target)

	// We're turning a floating point comparison magnitude to a string.
	if job.ComparisonMagnitude != 0.0 {
		v.Set("comparison_magnitude", fmt.Sprintf("%f", job.ComparisonMagnitude))
	}

	// Handle the spec for git hashes.
	switch jk := job.JobKind.(type) {
	case *pinpoint.JobSpec_Bisection:
		// Transcode the mode to the ones Pinpoint currently supports.
		switch job.GetComparisonMode() {
		case pinpoint.JobSpec_COMPARISON_MODE_UNSPECIFIED:
			fallthrough
		case pinpoint.JobSpec_PERFORMANCE:
			// The legacy API uses "performance" for performance bisections, and "try" for experiments
			v.Set("comparison_mode", "performance")
		case pinpoint.JobSpec_FUNCTIONAL:
			v.Set("comparison_mode", "functional")
		default:
			return nil, errors.Reason("Unknown comparison mode provided: %v", job.GetComparisonMode()).Err()
		}

		// We ignore the repository here, because legacy Pinpoint's API didn't support those.
		v.Set("start_git_hash", jk.Bisection.CommitRange.StartGitHash)
		v.Set("end_git_hash", jk.Bisection.CommitRange.EndGitHash)

		if jk.Bisection.Patch != nil {
			p := jk.Bisection.Patch
			patchURL, err := convertGerritChangeToURL(p)
			if err != nil {
				return nil, errors.Annotate(err, "invalid patch provided: %v", err).Err()
			}
			v.Set("patch", patchURL)
		}
	case *pinpoint.JobSpec_Experiment:
		// Transcode the mode to the ones Pinpoint currently supports.
		switch job.GetComparisonMode() {
		case pinpoint.JobSpec_COMPARISON_MODE_UNSPECIFIED:
			fallthrough
		case pinpoint.JobSpec_PERFORMANCE:
			// The legacy API uses "performance" for performance bisections, and "try" for experiments
			v.Set("comparison_mode", "try")
		case pinpoint.JobSpec_FUNCTIONAL:
			// We're failing gracefully here in cases where we're still proxying to the legacy API.
			// In the future, we should support functional experiments too.
			return nil, errors.Reason("functional experiments not supported by legacy API").Err()
		default:
			return nil, errors.Reason("Unknown comparison mode provided: %v", job.GetComparisonMode()).Err()
		}

		// The legacy Pinpoint API doesn't support specifying the Gitiles host/project as it assumes the only
		// the Chromium codebase is being worked on.
		v.Set("base_git_hash", jk.Experiment.BaseCommit.GitHash)
		experimentPatchURL, err := convertGerritChangeToURL(jk.Experiment.ExperimentPatch)
		if err != nil {
			return nil, errors.Annotate(err, "invalid experiment patch: %v", err).Err()
		}
		v.Set("patch", experimentPatchURL)

		// Even if these are ignored, we add them anyway.
		if jk.Experiment.ExperimentCommit != nil {
			v.Set("experiment_git_hash", jk.Experiment.ExperimentCommit.GitHash)
		}
		if jk.Experiment.BasePatch != nil {
			basePatchURL, err := convertGerritChangeToURL(jk.Experiment.BasePatch)
			if err != nil {
				return nil, errors.Annotate(err, "invalid base patch: %v", err).Err()
			}
			v.Set("base_patch", basePatchURL)
		}
	}

	// Process the benchmark arguments.
	switch args := job.Arguments.(type) {
	case *pinpoint.JobSpec_TelemetryBenchmark:
		tb := args.TelemetryBenchmark
		v.Set("benchmark", tb.Benchmark)
		v.Set("metric", tb.Measurement)
		v.Set("grouping_label", tb.GroupingLabel)
		switch s := tb.StorySelection.(type) {
		case *pinpoint.TelemetryBenchmark_Story:
			v.Set("story", s.Story)
			break
		case *pinpoint.TelemetryBenchmark_StoryTags:
			v.Set("story_tags", strings.Join(s.StoryTags.StoryTags, ","))
			break
		default:
			return nil, errors.Reason("Unsupported story_selection in TelemetryBenchmark").
				InternalReason("story_selection is %v", s).Err()
		}
	case *pinpoint.JobSpec_GtestBenchmark:
		gb := args.GtestBenchmark
		v.Set("benchmark", gb.Benchmark)
		v.Set("trace", gb.Test)
		v.Set("chart", gb.Measurement)
	default:
		return nil, errors.Reason("unsupported arguments in JobSpec").
			InternalReason("args type is %v", args).Err()
	}
	return v, nil
}
