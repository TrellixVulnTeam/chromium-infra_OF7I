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
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"gopkg.in/yaml.v2"
)

type batchSummaryReportMetric struct {
	Name string `yaml:"name"`
}

type batchSummaryReportSpec struct {
	Metrics *[]batchSummaryReportMetric `yaml:"metrics"`
}

type telemetryExperimentJobSpec struct {
	Config         string `yaml:"config"`
	StorySelection struct {
		Story     string   `yaml:"story,omitempty"`
		StoryTags []string `yaml:"story_tags,omitempty"`
	} `yaml:"story_selection,omitempty"`
	Benchmark     string   `yaml:"benchmark"`
	Measurement   string   `yaml:"measurement"`
	GroupingLabel string   `yaml:"grouping_label"`
	ExtraArgs     []string `yaml:"extra_args"`
}

type telemetryBatchExperiment struct {
	Benchmark   string   `yaml:"benchmark"`
	Configs     []string `yaml:"configs"`
	Measurement string   `yaml:"measurement,omitempty"`
	Stories     []string `yaml:"stories,omitempty"`
	StoryTags   []string `yaml:"story_tags,omitempty"`
	ExtraArgs   []string `yaml:"extra_args"`
}

type preset struct {
	BatchSummaryReportSpec   *map[string]batchSummaryReportSpec `yaml:"batch_summary_report_spec,omitempty"`
	TelemetryBatchExperiment *[]telemetryBatchExperiment        `yaml:"telemetry_batch_experiment,omitempty"`
	TelemetryExperiment      *telemetryExperimentJobSpec        `yaml:"telemetry_experiment,omitempty"`
}

type presetDb struct {
	Presets map[string]preset `yaml:"presets"`
}

type presetNotFoundError struct {
	name string
}

func newPresetNotFound(name string) *presetNotFoundError {
	return &presetNotFoundError{name}
}

func (p *presetNotFoundError) Error() string {
	return fmt.Sprintf("preset not found: %q", p.name)
}

func validateTelemetryJobPreset(p preset) error {
	if p.TelemetryExperiment == nil && p.TelemetryBatchExperiment == nil {
		return nil
	}
	if p.TelemetryExperiment != nil && p.TelemetryBatchExperiment != nil {
		return fmt.Errorf("exactly one experiment type should be defined")
	} else if p.TelemetryExperiment != nil {
		if (len(p.TelemetryExperiment.StorySelection.Story) > 0 &&
			len(p.TelemetryExperiment.StorySelection.StoryTags) > 0) ||
			(len(p.TelemetryExperiment.StorySelection.Story) == 0 &&
				len(p.TelemetryExperiment.StorySelection.StoryTags) == 0) {
			return fmt.Errorf(text.Doc(`
				telemetry experiments must only have exactly one of story or
				story_tags in story_selection
			`))
		}
		if len(p.TelemetryExperiment.Config) == 0 {
			return fmt.Errorf(text.Doc(`
				telemetry experiments must have a non-empty config
			`))
		}
	} else if p.TelemetryBatchExperiment != nil {
		for i := range *p.TelemetryBatchExperiment {
			if len((*p.TelemetryBatchExperiment)[i].Stories) == 0 &&
				len((*p.TelemetryBatchExperiment)[i].StoryTags) == 0 {
				return fmt.Errorf("at least one story or story tag should be defined for each benchmark")
			}
			if len((*p.TelemetryBatchExperiment)[i].Configs) == 0 {
				return fmt.Errorf("at least one config should be defined for each benchmark")
			}
		}
	}
	return nil
}

func validateBatchSummaryPreset(p preset) error {
	if p.BatchSummaryReportSpec == nil {
		return nil
	}
	return nil // All states are valid, so far.
}

func (pdb *presetDb) GetPreset(pName string) (preset, error) {
	p, found := pdb.Presets[pName]
	if !found {
		return p, newPresetNotFound(pName)
	}

	// We need to validate that the preset is well-formed.  We're doing this
	// late because we don't want to stop forward progress at loading time.
	e := validateTelemetryJobPreset(p)
	if e != nil {
		return p, e
	}
	e = validateBatchSummaryPreset(p)
	if e != nil {
		return p, e
	}

	return p, nil
}

func loadPresets(pfile io.Reader) (*presetDb, error) {
	if pfile == nil {
		return nil, errors.Reason("pfile must not be nil").Err()
	}
	pd := &presetDb{}
	d := yaml.NewDecoder(pfile)
	d.SetStrict(true)
	if err := d.Decode(pd); err != nil {
		return nil, errors.Annotate(err, "failed loading presets").Err()
	}

	return pd, nil
}

type presetsMixin struct {
	// presetFile is bound to the -preset-file flag.
	presetFile string

	// presetName is bound to the -preset flag.
	presetName string
}

func (pm *presetsMixin) RegisterFlags(flags *flag.FlagSet, uc userConfig) {
	flags.StringVar(&pm.presetFile, "presets-file", uc.PresetsFile, text.Doc(`
		File to look up when loading preset job configurations.
	`))
	flags.StringVar(&pm.presetName, "preset", "", text.Doc(`
		Name of the preset to select. Will not use a named preset if empty.
	`))
}

func (pm *presetsMixin) getPreset(ctx context.Context) (preset, error) {
	if pm.presetName == "" {
		return preset{}, nil
	}

	b, err := os.ReadFile(pm.presetFile)
	if err != nil {
		logging.Warningf(ctx, "failed reading preset file %q", pm.presetFile)
		return preset{}, errors.Annotate(err, "failed reading preset file %q", pm.presetFile).Err()
	}

	pdb, err := loadPresets(bytes.NewReader(b))
	if err != nil {
		return preset{}, errors.Annotate(err, "potentially malformed presets file %q", pm.presetFile).Err()
	}

	p, err := pdb.GetPreset(pm.presetName)
	if err != nil {
		return p, errors.Annotate(err, "failed getting preset %q", pm.presetName).Err()
	}

	return p, nil
}
