// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package provision run provisioning for DUT.
package provision

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/cros-provision/internal/docker"
)

const (
	// TLW port.
	tlwPort = 7151

	// Dut service running port, docker info.
	dutServerDockerImageRepo = "gcr.io/chromeos-bot/dutserver"
	dutServerNameTemplate    = "dut-service-%s"

	// Provision service running port, docker info.
	provisionServerDockerImageRepo = "gcr.io/chromeos-bot/provisionserver"
	provisionServerNameTemplate    = "provision-service-%s"

	// File names used to interact with provision-server CLI.
	inputFileName  = "in.json"
	outputFileName = "out.json"
)

// Result holds result data.
type Result struct {
	Out *api.DutOutput
	Err error
}

// Run runs provisioning software dependencies per DUT.
func Run(ctx context.Context, in *api.DutInput, localAddr string) *Result {
	res := &Result{
		Out: &api.DutOutput{
			Id: in.GetId(),
			Outcome: &api.DutOutput_Failure{
				Failure: &api.InstallFailure{
					Reason: api.InstallFailure_REASON_PROVISIONING_FAILED,
				},
			},
		},
	}
	if in == nil || in.GetProvisionState() == nil {
		res.Err = errors.Reason("run provision: DUT input is empty").Err()
		return res
	}
	if localAddr == "" {
		res.Err = errors.Reason("run provision: local address is not provided").Err()
		return res
	}
	dutName := in.GetId().GetValue()
	log.Printf("Starting provisioning of %q, with: %s", dutName, in.GetProvisionState())
	tlwAddr := fmt.Sprintf("%s:%d", localAddr, tlwPort)

	// Create separate network to run docker independent.
	networkName := dutName
	if err := docker.CreateNetwork(ctx, networkName); err != nil {
		res.Err = errors.Annotate(err, "run provision").Err()
		return res
	}
	defer docker.RemoveNetwork(ctx, networkName)

	// Prepare required DUT-service and provision services.
	dutService, err := createDutService(ctx, in.GetDutService(), dutName, tlwAddr, networkName)
	defer dutService.Remove(ctx)
	if err != nil {
		res.Err = errors.Annotate(err, "run provision").Err()
		return res
	}

	// Create directory to provide input files and collect output files.
	// The directory will also has logs of the provisioning.
	dir, err := createInput(in.GetProvisionState())
	defer func() {
		if dir != "" {
			os.RemoveAll(dir)
		}
	}()
	if err != nil {
		res.Err = errors.Annotate(err, "run provision").Err()
		return res
	}
	provisionService, err := createProvisionService(ctx, in.GetProvisionService(), dutName, tlwAddr, networkName, dutService, dir)
	defer provisionService.Remove(ctx)

	if err != nil {
		res.Err = errors.Annotate(err, "run provision").Err()
		return res
	}
	resultFileName := path.Join(dir, outputFileName)
	if _, err := os.Stat(resultFileName); os.IsNotExist(err) {
		res.Err = errors.Reason("run provision: result not found").Err()
		return res
	}
	out, err := readOutput(resultFileName)
	if err != nil {
		res.Err = errors.Annotate(err, "run provision").Err()
		return res
	}
	log.Printf("Result file %s: found. %s", dutName, out)
	if f := out.GetFailure(); f != nil {
		res.Out.Outcome = &api.DutOutput_Failure{
			Failure: f,
		}
		res.Err = errors.Annotate(err, "run provision").Err()
	} else {
		res.Out.Outcome = &api.DutOutput_Success{
			Success: &api.InstallSuccess{},
		}
		res.Err = nil
	}
	return res
}

// createDutService pulls and starts dut-service.
func createDutService(ctx context.Context, di *api.DutInput_DockerImage, dutName, tlwAddr, networkName string) (*docker.Docker, error) {
	d := &docker.Docker{
		Name:               fmt.Sprintf(dutServerNameTemplate, dutName),
		RequestedImageName: docker.CreateImageNameFromInputInfo(di, dutServerDockerImageRepo, docker.DefaultImageTag),
		FallbackImageName:  docker.CreateImageName(dutServerDockerImageRepo, docker.DefaultImageTag),
		ExecCommand: []string{
			"dutserver",
			"-dut_name", dutName,
			"-wiring_address", tlwAddr,
			"-port", "80",
		},
		ServicePort: 80,
		Detach:      true,
		Network:     networkName,
	}
	return startService(ctx, d)
}

// createProvisionService pulls and starts provision-service.
func createProvisionService(ctx context.Context, di *api.DutInput_DockerImage, dutName, tlwAddr, networkName string, dutService *docker.Docker, tpmDir string) (*docker.Docker, error) {
	dockerResultDirName := "/tmp/provisionservice"
	d := &docker.Docker{
		Name:               fmt.Sprintf(provisionServerNameTemplate, dutName),
		RequestedImageName: docker.CreateImageNameFromInputInfo(di, provisionServerDockerImageRepo, docker.DefaultImageTag),
		FallbackImageName:  docker.CreateImageName(provisionServerDockerImageRepo, docker.DefaultImageTag),
		ExecCommand: []string{
			"provisionserver",
			"cli",
			"-dut-name", dutName,
			"-dut-service-address", fmt.Sprintf("%s:%d", dutService.Name, dutService.ServicePort),
			"-wiring-service-address", tlwAddr,
			"-in-json", path.Join(dockerResultDirName, inputFileName),
			"-out-json", path.Join(dockerResultDirName, outputFileName),
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s", tpmDir, dockerResultDirName),
		},
		Network: networkName,
	}
	return startService(ctx, d)
}

func startService(ctx context.Context, d *docker.Docker) (*docker.Docker, error) {
	if err := d.Remove(ctx); err != nil {
		log.Printf("Fail to clean up container %q. Error: %s", d.Name, err)
	}
	if err := d.PullImage(ctx); err != nil {
		return d, errors.Annotate(err, "start service").Err()
	}
	if err := d.Run(ctx); err != nil {
		return d, errors.Annotate(err, "start service").Err()
	}
	return d, nil
}

// createInput created input file fro provision-service.
func createInput(state *api.ProvisionState) (string, error) {
	dir, err := ioutil.TempDir("", "provision-result")
	if err != nil {
		log.Printf("Fail create dir: %s", err)
		return "", errors.Annotate(err, "create input").Err()
	}
	inputFilePath := path.Join(dir, inputFileName)
	f, err := os.Create(inputFilePath)
	if err != nil {
		return dir, errors.Annotate(err, "create input").Err()
	}
	defer f.Close()

	marshaler := jsonpb.Marshaler{}
	if err := marshaler.Marshal(f, state); err != nil {
		return dir, errors.Annotate(err, "create input").Err()
	}
	err = f.Close()
	return dir, errors.Annotate(err, "create input").Err()
}

// readOutput reads output file generated by provision-service.
func readOutput(filePath string) (*api.InstallCrosResponse, error) {
	r, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Annotate(err, "read output").Err()
	}
	out := &api.InstallCrosResponse{}
	err = jsonpb.Unmarshal(r, out)
	return out, errors.Annotate(err, "read output").Err()
}
