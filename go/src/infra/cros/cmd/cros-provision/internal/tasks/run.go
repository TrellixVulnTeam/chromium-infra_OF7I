// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"
	config_go "go.chromium.org/chromiumos/config/go"
	"go.chromium.org/chromiumos/config/go/test/api"
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/lucictx"
	"golang.org/x/sync/errgroup"

	"infra/cmdsupport/cmdlib"
	"infra/cros/cmd/cros-provision/internal/provision"
)

// Run executes the provisioning for requested devices.
func Run(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "run [-in-json in_path.json] [-out-json out_path.json]",
		ShortDesc: "Run provisioning for ChromeOS devices",
		LongDesc: `Run provisioning for ChromeOS devices

Tool used to perfrom provisioning OS, components and FW to ChromeOS device specified by ProvisionState.

Supporting two ways to execute provisioning.
1) Provide ProvisionCliInput as jsonproto structure in JSON file. (provision_cli.proto)
	Usage: cros-provision run -in-json your_in.json -out-json your_out.json
2) Provide all required detail in command line interface.
	Usage: cros-provision run -dut-name your_device_name -cros chrome_os_dir_path [-prevent-reboot] [-dut-service-docker-image repo:tag] [-provision-service-docker-image repo:tag]
`,
		CommandRun: func() subcommands.CommandRun {
			c := &runCmd{}
			c.authFlags.Register(&c.Flags, authOpts)
			// Used to provide input by files.
			c.Flags.StringVar(&c.inputPath, "in-json", "", "Path that contains JSON file. Used together with '-out-json'.")
			c.Flags.StringVar(&c.outputPath, "out-json", "", "Path that will contain JSON file. Used together with '-in-json'.")

			// Used to provide by manually.
			c.Flags.StringVar(&c.dutName, "dut-name", "", "Name of provisioning device.")
			c.Flags.StringVar(&c.crosImagePath, "cros", "", "Path for system image directory.")
			c.Flags.BoolVar(&c.preventReboot, "prevent-reboot", false, "Prevent reboot during install system image.")
			c.Flags.StringVar(&c.dutServiceDockerImage, "dut-service-docker-image", "", "The name of dut-service docker image. Example: 'gcr.io/chromeos-bot/dutserver:your_version'.")
			c.Flags.StringVar(&c.provisionServiceDockerImage, "provision-service-docker-image", "", "The name of provision-service docker image. Example: 'gcr.io/chromeos-bot/provisionserver:your_version'.")
			return c
		},
	}
}

type runCmd struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string

	dutName                     string
	crosImagePath               string
	preventReboot               bool
	dutServiceDockerImage       string
	provisionServiceDockerImage string
}

// Run executes the tool.
func (c *runCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	out, err := c.innerRun(ctx, a, args, env)
	if err := saveOutput(out, c.outputPath); err != nil {
		log.Printf("Run: %s", err)
	}
	printOutput(out, a)
	if err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *runCmd) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) (*api.ProvisionCliOutput, error) {
	var out *api.ProvisionCliOutput
	ctx, err := useSystemAuth(ctx, &c.authFlags)
	if err != nil {
		return out, errors.Annotate(err, "inner run: read system auth").Err()
	}
	localAddr, err := readLocalAddress()
	if err != nil {
		return out, errors.Annotate(err, "inner run: read local addr").Err()
	}
	req, err := c.newProvisionInput()
	if err != nil {
		return out, errors.Annotate(err, "inner run").Err()
	}

	// TODO(otabek): Listen signal to cancel execution by client.

	out = &api.ProvisionCliOutput{}
	// errgroup returns the first error but doesn't stop execution of other goroutines.
	g, ctx := errgroup.WithContext(ctx)
	provisionResults := make([]*api.DutOutput, len(req.GetDutInputs()))
	// Each DUT will run in parallel execution.
	for i, dutInfo := range req.GetDutInputs() {
		i, dutInfo := i, dutInfo
		g.Go(func() error {
			result := provision.Run(ctx, dutInfo, localAddr)
			provisionResults[i] = result.Out
			return result.Err
		})
	}
	err = g.Wait()
	// Read all generated results for the output.
	for _, result := range provisionResults {
		out.DutOutputs = append(out.DutOutputs, result)
	}
	return out, errors.Annotate(err, "inner run").Err()
}

// Read input data to create the input request.
func (c *runCmd) newProvisionInput() (*api.ProvisionCliInput, error) {
	if c.inputPath != "" {
		return readInput(c.inputPath)
	}
	return c.createInputRequest()
}

// TODO(otabek): Add support for other options.
func (c *runCmd) createInputRequest() (*api.ProvisionCliInput, error) {
	if c.dutName == "" {
		return nil, errors.Reason("create input request: dut name is not provided").Err()
	}
	return &api.ProvisionCliInput{
		DutInputs: []*api.DutInput{
			{
				Id:               &lab_api.Dut_Id{Value: c.dutName},
				DutService:       createDockerImage(c.dutServiceDockerImage),
				ProvisionService: createDockerImage(c.provisionServiceDockerImage),
				ProvisionState: &api.ProvisionState{
					SystemImage:   createSystemImage(c.crosImagePath),
					PreventReboot: c.preventReboot,
				},
			},
		},
	}, nil
}

func createDockerImage(name string) *api.DutInput_DockerImage {
	if name == "" {
		return nil
	}
	parts := strings.Split(name, ":")
	r := &api.DutInput_DockerImage{
		RepositoryPath: parts[0],
		Tag:            "latest",
	}
	if len(parts) > 1 {
		r.Tag = parts[1]
	}
	return r
}

func createSystemImage(imagePath string) *api.ProvisionState_SystemImage {
	if imagePath == "" {
		return nil
	}
	pathType := config_go.StoragePath_GS
	if !strings.HasPrefix(imagePath, "gs://") {
		pathType = config_go.StoragePath_LOCAL
	}
	return &api.ProvisionState_SystemImage{
		SystemImagePath: &config_go.StoragePath{
			HostType: pathType,
			Path:     imagePath,
		},
	}
}

// readInput reads the jsonproto at path input data.
func readInput(inputPath string) (*api.ProvisionCliInput, error) {
	in := &api.ProvisionCliInput{}
	r, err := os.Open(inputPath)
	if err != nil {
		return nil, errors.Annotate(err, "read input").Err()
	}
	err = jsonpb.Unmarshal(r, in)
	return in, errors.Annotate(err, "read input").Err()
}

// saveOutput saves output data to the file.
func saveOutput(out *api.ProvisionCliOutput, outputPath string) error {
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

func printOutput(out *api.ProvisionCliOutput, a subcommands.Application) {
	if out != nil {
		s, err := json.MarshalIndent(out, "", "\t")
		if err != nil {
			log.Printf("Output: fail to print info. Error: %s", err)
		} else {
			log.Println("Output:")
			fmt.Fprintf(a.GetOut(), "%s\n", s)
		}
	}
}

// readLocalAddress read local IP of the host.
func readLocalAddress() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Annotate(err, "read local address").Err()
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return "", errors.Annotate(err, "read local address").Err()
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			// TODO(otabek): Add option to work with IPv6 if we switched to it.
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.Reason("read local address: fail to find").Err()
}

func useSystemAuth(ctx context.Context, authFlags *authcli.Flags) (context.Context, error) {
	authOpts, err := authFlags.Options()
	if err != nil {
		return nil, errors.Annotate(err, "switching to system auth").Err()
	}

	authCtx, err := lucictx.SwitchLocalAccount(ctx, "system")
	if err == nil {
		// If there's a system account use it (the case of running on Swarming).
		// Otherwise default to user credentials (the local development case).
		authOpts.Method = auth.LUCIContextMethod
		return authCtx, nil
	}
	log.Printf("System account not found, err %s.\nFalling back to user credentials for auth.\n", err)
	return ctx, nil
}
