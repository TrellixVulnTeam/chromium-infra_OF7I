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
	"os"
	"path/filepath"
	"text/template"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/logging"
	"gopkg.in/yaml.v2"
)

// userConfig defines a type that represents some user-configured defaults for
// flags that are common to the Pinpoint CLI.
type userConfig struct {
	AnalyzeExperiment bool   `yaml:"analyze_experiment,omitempty"`
	CheckExperiment   bool   `yaml:"check_experiment,omitempty"`
	DownloadArtifacts bool   `yaml:"download_artifacts,omitempty"`
	DownloadResults   bool   `yaml:"download_results,omitempty"`
	Endpoint          string `yaml:"endpoint,omitempty"`
	OpenResults       bool   `yaml:"open_results,omitempty"`
	PresetsFile       string `yaml:"presets_file,omitempty"`
	Quiet             bool   `yaml:"quiet,omitempty"`
	ResultsDir        string `yaml:"results_dir,omitempty"`
	SelectArtifacts   string `yaml:"select_artifacts,omitempty"`
	Wait              bool   `yaml:"wait,omitempty"`
	WorkDir           string `yaml:"work_dir,omitempty"`
}

func getUserConfig(ctx context.Context, cfgFile string, p Param) userConfig {
	uc := userConfig{
		AnalyzeExperiment: false,
		CheckExperiment:   false,
		DownloadArtifacts: false,
		DownloadResults:   false,
		Endpoint:          p.DefaultServiceDomain,
		OpenResults:       false,
		PresetsFile:       ".pinpoint-presets.yaml",
		Quiet:             false,
		ResultsDir:        os.TempDir(),
		SelectArtifacts:   "test",
		Wait:              false,
		WorkDir:           os.TempDir(),
	}
	if len(cfgFile) == 0 {
		return uc
	}
	bs, err := os.ReadFile(cfgFile)
	if os.IsNotExist(err) {
		return uc
	}
	if err != nil {
		logging.Warningf(ctx, "failed reading %q: %s", cfgFile, err)
		return uc
	}
	err = yaml.Unmarshal(bs, &uc)
	if err != nil {
		logging.Warningf(ctx, "error parsing yaml: %s", err)
		return uc
	}
	return uc
}

// getUserConfigFilename looks up the `PINPOINT_USER_CONFIG` environment
// variable which would point to a YAML file defining the defaults.  If we can't
// find the environment variable, we'll use the default of
// $HOME/.pinpoint/config.yaml or the equivalent depending on the platform.
func getUserConfigFilename() string {
	envFile, found := os.LookupEnv("PINPOINT_USER_CONFIG")
	if found {
		return envFile
	}

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		// Because we cannot find a user config directory, we'll return an empty
		// string.
		return "pinpoint-config.yaml"
	}
	return filepath.Join(cfgDir, "pinpoint", "config.yaml")
}

type configRun struct {
	subcommands.CommandRunBase
	params     Param
	new, force bool
}

func (cr *configRun) RegisterFlags(p Param) {
	cfgFile := getUserConfigFilename()
	cr.GetFlags().BoolVar(&cr.new, "new", false, text.Doc(fmt.Sprintf(`
		Create a new config at: %s
	`, cfgFile)))
	cr.GetFlags().BoolVar(&cr.force, "force", false, text.Doc(fmt.Sprintf(`
		Force the creation of a new config, when provided with -new
		(still at %s)
	`, cfgFile)))
}

const cfgTempl = `Pinpoint CLI Configuration
(source: {{.Source}})
{{with .Cfg}}
--analyze-experiment={{.AnalyzeExperiment}}
--check-experiment={{.CheckExperiment}}
--download-artifacts={{.DownloadArtifacts}}
--download-results={{.DownloadResults}}
--endpoint={{.Endpoint}}
--open-results={{.OpenResults}}
--presets-file={{.PresetsFile}}{{end}}
--quiet={{.Quiet}}
--results-dir={{.ResultsDir}}
--select-artifacts={{.SelectArtifacts}}
--wait={{.Wait}}
--work-dir={{.WorkDir}}
`

func (cr *configRun) Run(ctx context.Context, a subcommands.Application, args []string) error {
	cfgFile := getUserConfigFilename()
	if cr.new {
		// FIXME: Support creating a new config file, support -force too.
	}
	cfg := getUserConfig(ctx, cfgFile, cr.params)
	o := template.Must(template.New("config").Parse(cfgTempl))
	if err := o.Execute(a.GetOut(), struct {
		Source string
		Cfg    userConfig
	}{cfgFile, cfg}); err != nil {
		return err
	}
	return nil
}

func cmdConfig(p Param) *subcommands.Command {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir, err = os.UserHomeDir()
		if err != nil {
			cfgDir = os.TempDir()
		}
	}
	return &subcommands.Command{
		UsageLine: "config",
		ShortDesc: "Show or create user-specific configuration",
		LongDesc: text.Doc(fmt.Sprintf(`
			Displays default configuration options for the user running the
			Pinpoint CLI.

			CONFIG LOCATION

			The tool will look for the configuration file in the following
			locations, in this order:

			- The PINPOINT_USER_CONFIG environment variable, pointing to a
			YAML config file.

			- In %s which is the default location for Pinpoint's user
			configuration.

			The options in the YAML control the defaults for flags that apply
			across a number of sub-commands.  These can still be overriden if
			the flags are provided to the commandline when invoking these
			sub-commands.

			SUPPORTED OPTIONS

			See
			https://source.chromium.org/chromium/infra/infra/+/main:go/src/infra/chromeperf/doc/pinpoint/cli/user-configs.md
			for more information.
		`, filepath.Join(cfgDir, "pinpoint", "config.yaml"))),
		CommandRun: wrapCommand(p, func() pinpointCommand {
			return &configRun{params: p}
		}),
	}
}
