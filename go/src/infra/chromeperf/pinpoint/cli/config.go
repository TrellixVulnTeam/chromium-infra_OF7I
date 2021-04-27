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
	"io/ioutil"
	"os"
	"path/filepath"

	"go.chromium.org/luci/common/logging"
	"gopkg.in/yaml.v2"
)

// userConfig defines a type that represents some user-configured defaults for
// flags that are common to the Pinpoint CLI.
type userConfig struct {
	Endpoint        string `yaml:"endpoint,omitempty"`
	Wait            bool   `yaml:"wait,omitempty"`
	DownloadResults bool   `yaml:"download_results,omitempty"`
	OpenResults     bool   `yaml:"open_results,omitempty"`
	TempDir         string `yaml:"temp_dir,omitempty"`
	Quiet           bool   `yaml:"quiet,omitempty"`
}

func getUserConfig(ctx context.Context, cfgFile string, p Param) userConfig {
	uc := userConfig{
		Endpoint:        p.DefaultServiceDomain,
		Wait:            false,
		DownloadResults: false,
		OpenResults:     false,
		TempDir:         os.TempDir(),
		Quiet:           false,
	}
	if len(cfgFile) == 0 {
		return uc
	}
	bs, err := ioutil.ReadFile(cfgFile)
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
		return ""
	}
	return filepath.Join(cfgDir, "pinpoint", "config.yaml")
}
