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
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/cros-tool-runner/internal/testfinder"
)

type runTestFinderCmd struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath     string
	outputPath    string
	imagesPath    string
	dockerKeyFile string
}

// TestFinder execute cros-test-finder to find tests.
func TestFinder(authOpts auth.Options) *subcommands.Command {
	const testFinderDesc = `Test Finder,

	Tool used to finds all tests that match the criteria from a list of chromiumos.test.api.TestCase.

	Example:
	cros-tool-runner test-finder -images docker-images.json -input test_finder_request.json -output test_finder_result.json
	`

	c := &runTestFinderCmd{}
	return &subcommands.Command{
		UsageLine: "test-finder -images md_container.jsonpb -input input.json -output output.json",
		ShortDesc: "Test finder finds all tests that match the criteria from a list of chromiumos.test.api.TestCase",
		LongDesc:  testFinderDesc,
		CommandRun: func() subcommands.CommandRun {
			c.authFlags.Register(&c.Flags, authOpts)
			// Used to provide input by files.
			c.Flags.StringVar(&c.inputPath, "input", "", "The input file contains a jsonproto representation of test finder requests (CrosToolRunnerTestFinderRequest)")
			c.Flags.StringVar(&c.outputPath, "output", "", "The output file contains a jsonproto representation of test finder responses (CrosToolRunnerTestResponse)")
			c.Flags.StringVar(&c.imagesPath, "images", "", "The input file contains a jsonproto representation of containers metadata (ContainerMetadata)")
			c.Flags.StringVar(&c.dockerKeyFile, "docker_key_file", "", "The input file contains the docker auth key")
			return c
		},
	}
}

// Run executes the tool.
func (c *runTestFinderCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
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
		log.Printf("Failed in running test finder: %s", err)
		return 1
	}
	if err := saveTestFinderOutput(out, c.outputPath); err != nil {
		log.Printf("Failed to save test finder output: %s", err)
	}
	return 0
}

func (c *runTestFinderCmd) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env, token string) (*api.CrosToolRunnerTestFinderResponse, error) {
	ctx, err := useSystemAuth(ctx, &c.authFlags)
	if err != nil {
		return nil, errors.Annotate(err, "inner run: read system auth").Err()
	}
	req, err := readTestFinderRequest(c.inputPath)
	if err != nil {
		return nil, errors.Annotate(err, "inner run: failed to read test finder request").Err()
	}

	cm, err := readContainersMetadata(c.imagesPath)
	if err != nil {
		return nil, errors.Annotate(err, "inner run: failed to read containter metadata").Err()
	}
	lookupKey := req.ContainerMetadataKey

	crosTestFinderContainer := findContainer(cm, lookupKey, testfinder.CrosTestFinderName)
	result, err := testfinder.Run(ctx, req, crosTestFinderContainer, token)
	return result, errors.Annotate(err, "inner run: failed to find tests").Err()
}

// readTestFinderRequest reads the jsonproto at path input request data.
func readTestFinderRequest(p string) (*api.CrosToolRunnerTestFinderRequest, error) {
	in := &api.CrosToolRunnerTestFinderRequest{}
	r, err := os.Open(p)
	if err != nil {
		return nil, errors.Annotate(err, "inner run: read test finder request %q", p).Err()
	}

	umrsh := jsonpb.Unmarshaler{AllowUnknownFields: true}
	err = umrsh.Unmarshal(r, in)
	return in, errors.Annotate(err, "inner run: read test finder request %q", p).Err()
}

// saveTestFinderOutput saves output data to the file.
func saveTestFinderOutput(out *api.CrosToolRunnerTestFinderResponse, outputPath string) error {
	if outputPath != "" && out != nil {
		dir := filepath.Dir(outputPath)
		// Create the directory if it doesn't exist.
		if err := os.MkdirAll(dir, 0777); err != nil {
			return errors.Annotate(err, "save test finder output: failed to create directory while saving output").Err()
		}
		f, err := os.Create(outputPath)
		if err != nil {
			return errors.Annotate(err, "save test finder output: failed to create file while saving output").Err()
		}
		defer f.Close()
		marshaler := jsonpb.Marshaler{}
		if err := marshaler.Marshal(f, out); err != nil {
			return errors.Annotate(err, "save test finder output: failed to marshal result while saving output").Err()
		}
		if err := f.Close(); err != nil {
			return errors.Annotate(err, "save test finder output: failed to close file while saving output").Err()
		}
	}
	return nil
}
