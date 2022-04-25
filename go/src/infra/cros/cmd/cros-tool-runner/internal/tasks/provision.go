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
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"golang.org/x/sync/errgroup"

	"infra/cros/cmd/cros-tool-runner/internal/provision"
)

type runCmd struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath     string
	outputPath    string
	imagesPath    string
	dockerKeyFile string

	in                 *api.CrosToolRunnerProvisionRequest
	crosDutImage       *build_api.ContainerImageInfo
	crosProvisionImage *build_api.ContainerImageInfo
}

// Provision executes the provisioning for requested devices.
func Provision(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "provision -input input.json -output output.json",
		ShortDesc: "Run provisioning for ChromeOS devices",
		LongDesc: `Run provisioning for ChromeOS devices

Tool used to perfrom provisioning OS, components and FW to ChromeOS device specified by ProvisionState.

Example:
cros-tool-runner provision -images docker-images.json -input provision_request.json -output provision_result.json
`,
		CommandRun: func() subcommands.CommandRun {
			c := &runCmd{}
			c.authFlags.Register(&c.Flags, authOpts)
			// Used to provide input by files.
			c.Flags.StringVar(&c.inputPath, "input", "", "The input file contains a jsonproto representation of provision requests (CrosToolRunnerProvisionRequest)")
			c.Flags.StringVar(&c.outputPath, "output", "", "The output file contains a jsonproto representation of provision responses (CrosToolRunnerProvisionResponse)")
			c.Flags.StringVar(&c.imagesPath, "images", "", "The input file contains a jsonproto representation of containers metadata (ContainerMetadata)")
			c.Flags.StringVar(&c.dockerKeyFile, "docker_key_file", "", "The input file contains the docker auth key")
			return c
		},
	}
}

// Run executes the tool.
func (c *runCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	token := ""
	var err error
	if c.dockerKeyFile != "" {
		if token, err = dockerAuth(ctx, c.dockerKeyFile); err != nil {
			log.Printf("failed in docker auth: %s", err)
			return 1
		}
	}

	returnCode := 0
	out, err := c.innerRun(ctx, a, args, env, token)
	// Unexpected error will counted as incorrect request data.
	// all expected cases has to generate responses.
	if err != nil && len(out.GetResponses()) == 0 {
		log.Printf("run: add error to output, %s", err)
		out.Responses = []*api.CrosProvisionResponse{
			{
				Outcome: &api.CrosProvisionResponse_Failure{
					Failure: &api.InstallFailure{
						Reason: api.InstallFailure_REASON_INVALID_REQUEST,
					},
				},
			},
		}
		returnCode = 1
	}

	// Always try to save output.
	if err := saveOutput(out, c.outputPath); err != nil {
		log.Printf("run: error while saving output, %s", err)
		returnCode = 1
	}
	return returnCode
}

func (c *runCmd) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env, token string) (*api.CrosToolRunnerProvisionResponse, error) {
	out := &api.CrosToolRunnerProvisionResponse{}
	ctx, err := useSystemAuth(ctx, &c.authFlags)
	if err != nil {
		return out, errors.Annotate(err, "inner run: read system auth").Err()
	}
	req, err := readProvisionRequest(c.inputPath)
	if err != nil {
		return out, errors.Annotate(err, "inner run").Err()
	}

	cm, err := readContainersMetadata(c.imagesPath)
	if err != nil {
		return out, errors.Annotate(err, "inner run").Err()
	}

	// TODO(otabek): Listen signal to cancel execution by client.

	// errgroup returns the first error but doesn't stop execution of other goroutines.
	g, ctx := errgroup.WithContext(ctx)
	provisionResults := make([]*api.CrosProvisionResponse, len(req.GetDevices()))
	// Each DUT will run in parallel execution.
	for i, device := range req.GetDevices() {
		i, device := i, device
		g.Go(func() error {
			result := provision.Run(ctx,
				device,
				findContainer(cm, device.GetContainerMetadataKey(), "cros-dut"),
				findContainer(cm, device.GetContainerMetadataKey(), "cros-provision"),
				token)
			provisionResults[i] = result.Out
			return result.Err
		})
	}
	err = g.Wait()
	// Read all generated results for the output.
	for _, result := range provisionResults {
		out.Responses = append(out.Responses, result)
	}
	return out, errors.Annotate(err, "inner run").Err()
}

func isEmptyEndPoint(i *lab_api.IpEndpoint) bool {
	return i == nil || i.GetAddress() == "" || i.GetPort() <= 0
}

// readProvisionRequest reads the jsonproto at path input request data.
func readProvisionRequest(p string) (*api.CrosToolRunnerProvisionRequest, error) {
	in := &api.CrosToolRunnerProvisionRequest{}
	r, err := os.Open(p)
	if err != nil {
		return nil, errors.Annotate(err, "read provision request %q", p).Err()
	}

	umrsh := jsonpb.Unmarshaler{AllowUnknownFields: true}
	err = umrsh.Unmarshal(r, in)
	return in, errors.Annotate(err, "read provision request %q", p).Err()
}

// saveOutput saves output data to the file.
func saveOutput(out *api.CrosToolRunnerProvisionResponse, outputPath string) error {
	if outputPath != "" && out != nil {
		dir := filepath.Dir(outputPath)
		// Create the directory if it doesn't exist.
		if err := os.MkdirAll(dir, 0777); err != nil {
			return errors.Annotate(err, "save output").Err()
		}
		f, err := os.Create(outputPath)
		if err != nil {
			return errors.Annotate(err, "save output").Err()
		}
		defer f.Close()
		marshaler := jsonpb.Marshaler{}
		if err := marshaler.Marshal(f, out); err != nil {
			return errors.Annotate(err, "save output").Err()
		}
		if err := f.Close(); err != nil {
			return errors.Annotate(err, "save output").Err()
		}
	}
	return nil
}
