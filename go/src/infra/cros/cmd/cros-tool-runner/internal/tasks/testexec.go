// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"
	build_api "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/cros-tool-runner/internal/common"
	"infra/cros/cmd/cros-tool-runner/internal/testexec"
)

const longDesc = `Run tests,

Tool used to executing tests.

Example:
cros-tool-runner test -images docker-images.json -input test_request.json -output test_result.json
`

type runTestCmd struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath     string
	outputPath    string
	imagesPath    string
	dockerKeyFile string

	in            *api.CrosToolRunnerTestRequest
	crosDutImage  *build_api.ContainerImageInfo
	crosTestImage *build_api.ContainerImageInfo
}

// TestExec executes the tests.
func Test(authOpts auth.Options) *subcommands.Command {
	c := &runTestCmd{}
	return &subcommands.Command{
		UsageLine: "test -input input.json -output output.json",
		ShortDesc: "Run tests",
		LongDesc:  longDesc,
		CommandRun: func() subcommands.CommandRun {
			c.authFlags.Register(&c.Flags, authOpts)
			// Used to provide input by files.
			c.Flags.StringVar(&c.inputPath, "input", "", "The input file contains a jsonproto representation of test requests (CrosToolRunnerTestRequest)")
			c.Flags.StringVar(&c.outputPath, "output", "", "The output file contains a jsonproto representation of test responses (CrosToolRunnerTestResponse)")
			c.Flags.StringVar(&c.imagesPath, "images", "", "The input file contains a jsonproto representation of containers metadata (ContainerMetadata)")
			c.Flags.StringVar(&c.dockerKeyFile, "docker_key_file", "", "The input file contains the docker auth key")
			return c
		},
	}
}

// Run executes the tool.
func (c *runTestCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	token := ""
	var err error
	if c.dockerKeyFile != "" {
		if token, err = dockerAuth(ctx, c.dockerKeyFile); err != nil {
			log.Printf("Failed in docker auth: %s", err)
			return 1
		}
	}
	out, err := c.innerRun(ctx, a, args, env, token)
	// Unexpected error will counted as incorrect request data.
	// all expected cases has to generate responses.
	if err != nil {
		log.Printf("Failed in running cros-test: %s", err)
		return 1
	}
	if err := saveTestOutput(out, c.outputPath); err != nil {
		log.Printf("Failed to save test output: %s", err)
	}
	return 0
}

func (c *runTestCmd) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env, token string) (*api.CrosToolRunnerTestResponse, error) {
	ctx, err := useSystemAuth(ctx, &c.authFlags)
	if err != nil {
		return nil, errors.Annotate(err, "inner run: read system auth").Err()
	}
	req, err := readTestRequest(c.inputPath)
	if err != nil {
		return nil, errors.Annotate(err, "inner run: failed to read test request").Err()
	}

	device := req.GetPrimaryDut()
	if device == nil {
		return nil, errors.New("inner run: requests does not specify primary device")
	}

	cm, err := readContainersMetadata(c.imagesPath)
	if err != nil {
		return nil, errors.Annotate(err, "inner run: failed to read containter metadata").Err()
	}
	lookupKey := device.ContainerMetadataKey

	crosTestContainer := findContainer(cm, lookupKey, "cros-test")
	crosDUTContainer := findContainer(cm, lookupKey, "cros-dut")
	result, err := testexec.Run(ctx, req, crosTestContainer, crosDUTContainer, token)
	return result, errors.Annotate(err, "inner run: failed to run tests").Err()
}

// readTestRequest reads the jsonproto at path input request data.
func readTestRequest(p string) (*api.CrosToolRunnerTestRequest, error) {
	in := &api.CrosToolRunnerTestRequest{}
	r, err := os.Open(p)
	if err != nil {
		return nil, errors.Annotate(err, "inner run: read test request %q", p).Err()
	}

	umrsh := common.JsonPbUnmarshaler()
	err = umrsh.Unmarshal(r, in)
	return in, errors.Annotate(err, "inner run: read test request %q", p).Err()
}

// saveTestOutput saves output data to the file.
func saveTestOutput(out *api.CrosToolRunnerTestResponse, outputPath string) error {
	if outputPath != "" && out != nil {
		dir := filepath.Dir(outputPath)
		// Create the directory if it doesn't exist.
		if err := os.MkdirAll(dir, 0777); err != nil {
			return errors.Annotate(err, "save test output: failed to create directory while saving output").Err()
		}
		f, err := os.Create(outputPath)
		if err != nil {
			return errors.Annotate(err, "save test output: failed to create file while saving output").Err()
		}
		defer f.Close()
		marshaler := jsonpb.Marshaler{}
		if err := marshaler.Marshal(f, out); err != nil {
			return errors.Annotate(err, "save test output: failed to marshal result while saving output").Err()
		}
		if err := f.Close(); err != nil {
			return errors.Annotate(err, "save test output: failed to close file while saving output").Err()
		}
	}
	return nil
}
