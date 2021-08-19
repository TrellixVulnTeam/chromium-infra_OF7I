// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package main implements a distributed worker model for uploading debug
// symbols to the crash service. This package will be called by recipes through
// CIPD and will perform the buisiness logic of the builder.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

const (
	// Default server URLs for the crash service.
	prodUploadUrl    = "https://prod-crashsymbolcollector-pa.googleapis.com/v1"
	stagingUploadUrl = "https://staging-crashsymbolcollector-pa.googleapis.com/v1"
	// Time in milliseconds to sleep before retrying the task.
	sleepTimeMs = 100
)

// Flags passed in the CLI to describe the builder configuration.
var (
	gsPath      string
	workerCount int
	retryCount  int
	isStaging   bool
	dryRun      bool
)

// taskConfig will contain the information needed to complete the upload task.
type taskConfig struct {
	symbolPath string
	retryCount string
	dryRun     bool
	isStaging  bool
}

// channels will contain the forward configChannel and backwards retryChannel
// that the upload worker will use. The forward channel will have an information
// flow going from the main loop(driver) to the worker. The backwards channel is
// the opposite.
type channels struct {
	configChannel chan taskConfig
	retryChannel  chan taskConfig
}

// initFlags initializes the CLI flags to be used by the builder.
func initFlags() {
	flag.StringVar(&gsPath, "gsPath", "localhost", ("Url pointing to the GS " +
		"bucket storing the tarball."))
	flag.IntVar(&workerCount, "workerCount", -1, ("Number of worker threads" +
		" to spawn."))
	flag.IntVar(&retryCount, "retryCount", -1, ("Number of total upload retries" +
		" allowed."))
	flag.BoolVar(&isStaging, "isStaging", false, ("Specifies if the builder" +
		" should push to the staging crash service or prod."))
	flag.BoolVar(&dryRun, "dryRun", false, ("Specified whether network" +
		" operations should be dry ran."))
	flag.Parse()
}

// uploadWorker will perform the upload of the symbol file to the crash service.
func uploadWorker(chans channels) error {
	// Fetch the local file from the unpacked tarball.

	// Open up an https request to the crash service.

	// Verify if the file has been uploaded already.

	// Upload the file.

	// Return with appropriate status code.
	// TODO(b/197010274): remove skeleton code.
	return nil
}

// fetchTarball will download the tarball from google storage which contains all
// of the .sym files to be uploaded. Once downloaded it will return the local
// filepath to tarball.
func fetchTarball(gsPath string) (string, error) {
	// TODO(b/197010274): remove skeleton code.
	return "./path", nil
}

// unpackTarball will take the local path of the fetched tarball and then unpack
// it. It will then return a list of file paths pointing to the unpacked .sym
// files.
func unpackTarball(localPath string) ([]string, error) {
	// TODO(b/197010274): remove skeleton code.
	return []string{"./path"}, nil
}

// generateConfigs will take a list of strings with containing the paths to the
// unpacked symbol files. It will return a list of generated task configs
// alongside the communication channels to be used.
func generateConfigs(symbolFiles []string) ([]taskConfig, *channels, error) {
	// TODO(b/197010274): remove skeleton code.
	return nil, nil, nil
}

// doUpload is the main loop that will spawn goroutines that will handle the
// upload tasks. Should the worker fail it's upload and we have retries left,
// send the task to the end of the channel's buffer.
func doUpload(tasks []taskConfig, chans *channels, retryCount int,
	isStaging, dryRun bool) (int, error) {
	// TODO(b/197010274): remove skeleton code.
	return 0, nil
}

// main is the function to be called by the CLE execution.
func main() {
	// Initialize and collect CLI flags.
	initFlags()

	// TODO(b/197010274): remove skeleton code.
	fmt.Print(gsPath, "\n")
	fmt.Print(workerCount, "\n")
	fmt.Print(retryCount, "\n")
	fmt.Print(isStaging, "\n")
	fmt.Print(dryRun, "\n")

	tarballPath, err := fetchTarball(gsPath)

	if err != nil {
		log.Fatal(err)
	}

	symbolFiles, err := unpackTarball(tarballPath)

	if err != nil {
		log.Fatal(err)
	}

	tasks, chans, err := generateConfigs(symbolFiles)

	retcode, err := doUpload(tasks, chans, retryCount, isStaging, dryRun)

	if err != nil {
		log.Fatal(err)
	}

	// TODO(b/197010274): remove skeleton code.
	// Return:
	// 		0: Success, all symbols uploaded, no retries.
	// 		1: Failure, more failures occurred than retries were allotted
	// 		2: Warning, all symbols eventually uploaded. Retries were needed.
	os.Exit(retcode)
}
