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
	"path/filepath"
	"strconv"
	"time"

	build_api "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/test/api"
	lab_api "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/cros-tool-runner/internal/docker"
)

const (
	// Dut service container name template for .
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

	// Root directory for the cros-test artifacts inside docker.
	CrosTestRootDirInsideDocker = "/tmp/test"

	// Root directory for the cros-test-finder artifacts inside docker.
	CrosTestFinderRootDirInsideDocker = "/tmp/test"
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
			"-dut_address", endPointToString(dutSshInfo),
			"-cache_address", endPointToString(cacheServer.GetAddress()),
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

// CreateDutServicesForHostNetwork pulls and starts cros-dut services in host network.
func CreateDutServicesForHostNetwork(ctx context.Context, image *build_api.ContainerImageInfo, duts []*lab_api.Dut, dir, t string) ([]*docker.Docker, error) {
	const (
		DutServerPortRangeStart = 12300
		DutServerPortRangeEnd   = 12400
	)
	p, err := createImagePath(image)
	if err != nil {
		return nil, errors.Annotate(err, "create dut services for host network: failed to create image path").Err()
	}
	r, err := createRegistryName(image)
	if err != nil {
		return nil, errors.Annotate(err, "create dut services for host network: failed to create registry name").Err()
	}

	currentPort := DutServerPortRangeStart
	var dockerContainers []*docker.Docker
	defer func() {
		for _, d := range dockerContainers {
			log.Printf("Removing container %s", d.Name)
			d.Remove(ctx)
		}
	}()

	crosDutResultDirName := "/tmp/cros-dut"
	for _, dut := range duts {
		dutID := dut.Id.GetValue()

		containerName := fmt.Sprintf(crosDutContainerNameTemplate, dutID)
		if dut.CacheServer == nil {
			return nil, errors.Annotate(err, "create dut services for host network: cache server must be specified in DUT").Err()
		}
		success := false
		for port := currentPort; port < DutServerPortRangeEnd; port++ {
			d := &docker.Docker{
				Name:               containerName,
				RequestedImageName: p,
				Registry:           r,
				Token:              t,
				ExecCommand: []string{
					"cros-dut",
					"-dut_address", dutAddress(dut),
					"-cache_address", endPointToString(dut.CacheServer.GetAddress()),
					"-port", strconv.Itoa(port),
				},
				ServicePort: port,
				Detach:      true,
				Network:     "host",
				Volumes: []string{
					fmt.Sprintf("%s:%s", path.Join(dir, dutID), crosDutResultDirName),
				},
			}
			log.Printf("start cros-dut service for container %s at port %v", containerName, port)
			_, err = startService(ctx, d, true)
			// Keep trying until we find a port to start.
			if err == nil {
				success = true
				dockerContainers = append(dockerContainers, d)
				break
			}
		}
		if err != nil {
			return nil, errors.Annotate(err, "create dut services: failed to run cros-dut").Err()
		}
		if !success {
			return nil, errors.Reason("create dut services: no port available to run cros-dut").Err()
		}
	}
	result := dockerContainers
	dockerContainers = nil
	return result, nil
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
	// It is necessary to do sudo here because /tmp/test is owned by root inside docker
	// when docker mount /tmp/test. However, the user that is running cros-test is
	// chromeos-test inside docker. Hence, the user chromeos-test does not have write
	// permission in /tmp/test. Therefore, we need to change the owner of the directory.
	cmd := fmt.Sprintf("sudo --non-interactive chown -R chromeos-test:chromeos-test %s && cros-test", CrosTestRootDirInsideDocker)
	d := &docker.Docker{
		Name:               fmt.Sprintf(crosTestContainerNameTemplate, os.Getpid(), time.Now().Unix()),
		RequestedImageName: p,
		Registry:           r,
		Token:              t,
		ExecCommand: []string{
			"bash",
			"-c",

			cmd,
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s", crosTestDir, filepath.Join(CrosTestRootDirInsideDocker, "cros-test")),
			fmt.Sprintf("%s:%s", resultDir, filepath.Join(CrosTestRootDirInsideDocker, "results")),
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
			fmt.Sprintf("%s:%s", crosTestFinderDir, filepath.Join(CrosTestFinderRootDirInsideDocker, "cros-test-finder")),
		},
		Detach:  false,
		Network: networkName,
	}
	_, err = startService(ctx, d, true)
	return err
}

func dutAddress(dut *lab_api.Dut) string {
	if dut == nil {
		return ""
	}
	chromeOS := dut.GetChromeos()
	if chromeOS == nil {
		return ""
	}
	endPoint := chromeOS.GetSsh()
	return endPointToString(endPoint)
}

func endPointToString(endPoint *lab_api.IpEndpoint) string {
	if endPoint == nil {
		return ""
	}
	if endPoint.GetPort() == 0 {
		return fmt.Sprintf("%s:%v", endPoint.GetAddress(), DefaultDutAddressPort)
	}
	return fmt.Sprintf("%s:%v", endPoint.GetAddress(), endPoint.GetPort())
}
