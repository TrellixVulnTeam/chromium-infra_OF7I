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

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/logging/gologger"
)

func application(p Param) *cli.Application {
	return &cli.Application{
		Name:  "pinpoint",
		Title: "A CLI client for pinpoint.",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			cmdTelemetryExperiment(p),
			cmdListJobs(p),
			subcommands.CmdHelp,
		},
		EnvVars: map[string]subcommands.EnvVarDefinition{
			"PINPOINT_CACHE_DIR": {
				Advanced:  true,
				ShortDesc: "Directory used for caching configs and settings.",
			},
		},
	}
}

// Param includes the parameters to use for the CLI application.
type Param struct {
	DefaultServiceDomain, OIDCProviderURL string
}

// Main invokes the subcommands for the application.
func Main(p Param, args []string) int {
	return subcommands.Run(application(p), nil)
}
