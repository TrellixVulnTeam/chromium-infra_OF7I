// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	build_api "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/test/api"
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/cros-tool-runner/internal/docker"
)

const (
	// Dut service running port, docker info.
	crosDutContainerNameTemplate = "cros-dut-%s"

	// Provision service running port, docker info.
	crosProvisionContainerNameTemplate = "cros-provision-%s"

	// Cros Test container name template.
	crosTestContainerNameTemplate = "cros-test-%d_%d"

	// File names used to interact with cros-provision CLI.
	InputFileName  = "in.json"
	OutputFileName = "out.json"

	// Default dut address port
	DefaultDutAddressPort = "22"
)

// CreateDutService pulls and starts cros-dut service.
func CreateDutService(ctx context.Context, image *build_api.ContainerImageInfo, dutName, networkName string, cacheServer *lab_api.CacheServer, dutSshInfo *lab_api.IpEndpoint, dir string, t string) (*docker.Docker, error) {
	p, err := createImagePath(image)
	if err != nil {
		log.Printf("create cros-dut service: %s", err)
	}
	r, err := createRegistryName(image)
	if err != nil {
		log.Printf("create cros-dut service: %s", err)
	}
	dutPortToBeUsed := DefaultDutAddressPort
	if dutSshInfo.GetPort() != 0 {
		dutPortToBeUsed = string(dutSshInfo.GetPort())
	}
	crosDutResultDirName := "/tmp/cros-dut"
	d := &docker.Docker{
		Name:               fmt.Sprintf(crosDutContainerNameTemplate, dutName),
		RequestedImageName: p,
		Registry:           r,
		Token:              t,
		// Fallback version used in case when main image fail to pull.
		FallbackImageName: "gcr.io/chromeos-bot/cros-dut:fallback",
		ExecCommand: []string{
			"cros-dut",
			"-dut_address", dutSshInfo.GetAddress() + ":" + dutPortToBeUsed,
			"-cache_address", cacheServer.GetAddress().GetAddress() + ":" + string(cacheServer.GetAddress().GetPort()),
			"-port", "80",
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s", dir, crosDutResultDirName),
		},
		ServicePort: 80,
		Detach:      true,
		Network:     networkName,
	}
	return startService(ctx, d, false)
}

// RunProvisionCLI pulls and starts cros-provision as CLI.
func RunProvisionCLI(ctx context.Context, image *build_api.ContainerImageInfo, networkName string, req *api.CrosProvisionRequest, dir string, t string) (*docker.Docker, error) {
	// Create directory to provide input files and collect output files.
	// The directory will also has logs of the provisioning.
	if err := createProvisionInput(req, dir); err != nil {
		return nil, errors.Reason("run provision").Err()
	}
	// Path on the drone where service put the logs by default.
	dockerResultDirName := "/tmp/provisionservice"
	p, err := createImagePath(image)
	if err != nil {
		return nil, errors.Reason("failed to create image for run provision").Err()
	}
	r, err := createRegistryName(image)
	if err != nil {
		return nil, errors.Reason("failed to create registry path run provision").Err()
	}
	dutName := req.Dut.Id.GetValue()
	d := &docker.Docker{
		Name:               fmt.Sprintf(crosProvisionContainerNameTemplate, dutName),
		RequestedImageName: p,
		Registry:           r,
		Token:              t,
		// Fallback version used in case when main image fail to pull.
		FallbackImageName: "gcr.io/chromeos-bot/cros-provision:fallback",
		ExecCommand: []string{
			"cros-provision",
			"cli",
			"-input", path.Join(dockerResultDirName, InputFileName),
			"-output", path.Join(dockerResultDirName, OutputFileName),
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s", dir, dockerResultDirName),
		},
		Detach:  false,
		Network: networkName,
	}
	return startService(ctx, d, true)
}

// RunTestCLI pulls and runs cros-test as CLI.
func RunTestCLI(ctx context.Context, image *build_api.ContainerImageInfo, networkName, inputFileName, crosTestDir, resultDir string, t string) error {
	p, err := createImagePath(image)
	if err != nil {
		return errors.Annotate(err, "failed to create image for cros-test").Err()
	}
	r, err := createRegistryName(image)
	if err != nil {
		return errors.Annotate(err, "failed to create registry path for cros-test").Err()
	}
	d := &docker.Docker{
		Name:               fmt.Sprintf(crosTestContainerNameTemplate, os.Getpid(), time.Now().Unix()),
		RequestedImageName: p,
		Registry:           r,
		Token:              t,
		// Fallback version used in case when main image fail to pull.
		FallbackImageName: "gcr.io/chromeos-bot/cros-test:fallback",
		ExecCommand: []string{
			"bash",
			"-c",
			"sudo chown -R chromeos-test:chromeos-test /tmp/test && cros-test",
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s", crosTestDir, "/tmp/test/cros-test"),
			fmt.Sprintf("%s:%s", resultDir, "/tmp/test/results"),
		},
		Detach:  false,
		Network: networkName,
	}
	_, err = startService(ctx, d, true)
	return err
}

// RunTestFinderCLI pulls and runs cros-test-finder as CLI.
func RunTestFinderCLI(ctx context.Context, image *build_api.ContainerImageInfo, networkName, crosTestFinderDir string, t string) error {
	p, err := createImagePath(image)
	if err != nil {
		return errors.Annotate(err, "failed to create image for cros-test").Err()
	}
	r, err := createRegistryName(image)
	if err != nil {
		return errors.Annotate(err, "failed to create reigstry path for cros-test-finder").Err()
	}
	// The files or directories used by cros-test-finder container is set up this way.
	// File or Directory inside the container   Source
	// ++++++++++++++++++++++++++++++++++++++   +++++++++++++++++++++++++++++++++++++++++++
	// /tmp/test/cros-test-finder               Mount /tmp/test/cros-test-finder during run
	// /tmp/test/cros-test-finder/request.json  Generated before execute cros-test-finder
	// /tmp/test/cros-test-finder/result.json   Generated by cros-test-finder
	// /usr/bin/cros-test-finder                Included in container image
	// /tmp/test/metadata                       Included in container image
	d := &docker.Docker{
		Name:               fmt.Sprintf(crosTestContainerNameTemplate, os.Getpid(), time.Now().Unix()),
		RequestedImageName: p,
		Registry:           r,
		Token:              t,
		// Fallback version used in case when main image fail to pull.
		FallbackImageName: "gcr.io/chromeos-bot/cros-test-finder:fallback",
		ExecCommand: []string{
			"cros-test-finder",
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s", crosTestFinderDir, "/tmp/test/cros-test-finder"),
		},
		Detach:  false,
		Network: networkName,
	}
	_, err = startService(ctx, d, true)
	return err
}
