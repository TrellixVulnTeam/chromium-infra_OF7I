// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package main implements a distributed worker model for uploading debug
// symbols to the crash service. This package will be called by recipes through
// CIPD and will perform the buisiness logic of the builder.
// TODO(b/197010274): Add meaningful logging, with timing, to builder.
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	lgs "go.chromium.org/luci/common/gcloud/gs"
	"infra/cros/internal/gs"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// Default server URLs for the crash service.
	prodUploadUrl    = "https://prod-crashsymbolcollector-pa.googleapis.com/v1"
	stagingUploadUrl = "https://staging-crashsymbolcollector-pa.googleapis.com/v1"
	// Time in milliseconds to sleep before retrying the task.
	sleepTimeMs = 100
)

// Regex used when finding symbol files.
var fileRegex = regexp.MustCompile(`([\w-]+.so.sym)$`)

// taskConfig will contain the information needed to complete the upload task.
type taskConfig struct {
	symbolPath string
	retryCount int
	dryRun     bool
	isStaging  bool
}

type uploadDebugSymbols struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	gsPath      string
	workerCount int
	retryCount  int
	channelSize int
	isStaging   bool
	dryRun      bool
}

func getCmdUploadDebugSymbols(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "upload <options>",
		ShortDesc: "Upload debug symbols to crash.",
		CommandRun: func() subcommands.CommandRun {
			b := &uploadDebugSymbols{}
			b.authFlags = authcli.Flags{}
			b.authFlags.Register(b.GetFlags(), authOpts)
			b.Flags.StringVar(&b.gsPath, "gs-path", "", ("[Required] Url pointing to the GS " +
				"bucket storing the tarball."))
			b.Flags.IntVar(&b.workerCount, "worker-count", 64, ("Number of worker threads" +
				" to spawn."))
			b.Flags.IntVar(&b.retryCount, "retry-count", 200, ("Number of total upload retries" +
				" allowed."))
			b.Flags.IntVar(&b.channelSize, "channel-size", 200, ("Number of task configs allowed" +
				" in the channel buffer at once."))
			b.Flags.BoolVar(&b.isStaging, "is-staging", false, ("Specifies if the builder" +
				" should push to the staging crash service or prod."))
			b.Flags.BoolVar(&b.dryRun, "dry-run", false, ("Specified whether network" +
				" operations should be dry ran."))
			return b
		}}
}

// generateClient handles the authentication of the user then generation of the
// client to be used by the gs module.
func generateClient(ctx context.Context, authOpts auth.Options) (*gs.ProdClient, error) {
	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		return nil, err
	}

	gsClient, err := gs.NewProdClient(ctx, authedClient)
	if err != nil {
		return nil, err
	}
	return gsClient, err
}

// downloadTgz will download the tarball from google storage which contains all
// of the symbol files to be uploaded. Once downloaded it will return the local
// filepath to tarball.
func downloadTgz(client gs.Client, gsPath, tgzPath string) error {
	return client.Download(lgs.Path(gsPath), tgzPath)
}

// uploadWorker will perform the upload of the symbol file to the crash service.
func uploadWorker(configChannel chan taskConfig) error {
	// Fetch the local file from the unpacked tarball.

	// Open up an https request to the crash service.

	// Verify if the file has been uploaded already.

	// Upload the file.

	// Return with appropriate status code.
	// TODO(b/197010274): remove skeleton code.
	return nil
}

// unzipTgz will take the local path of the fetched tarball and then unpack
// it. It will then return a list of file paths pointing to the unpacked symbol
// files.
func unzipTgz(inputPath, outputPath string) error {
	srcReader, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer srcReader.Close()

	destWriter, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer destWriter.Close()

	gzipReader, err := gzip.NewReader(srcReader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	_, err = io.Copy(destWriter, gzipReader)

	return err
}

// unpackTarball will take the local path of the fetched tarball and then unpack
// it. It will then return a list of file paths pointing to the unpacked symbol
// files. Searches for .so.sym files.
func unpackTarball(inputPath, outputDir string) ([]string, error) {
	retArray := []string{}

	// Open locally stored .tar file.
	srcReader, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer srcReader.Close()

	tarReader := tar.NewReader(srcReader)

	// Iterate through the tar file saving only the debug symbols.
	for {
		header, err := tarReader.Next()
		// End of file reached, terminate the loop smoothly.
		if err == io.EOF {
			break
		}
		// An error occurred fetching the next header.
		if err != nil {
			return nil, err
		}
		// The header indicates it's a file. Store and save the file if it is a symbol file.
		if header.FileInfo().Mode().IsRegular() {
			// Check if the file is a symbol file.
			filename := fileRegex.FindString(header.Name)
			if filename == "" {
				continue
			}

			destFilePath := filepath.Join(outputDir, filename)
			destFile, err := os.Create(destFilePath)
			if err != nil {
				return nil, err
			}

			retArray = append(retArray, destFilePath)

			// Write contents of the symbol file to local storage.
			_, err = io.Copy(destFile, tarReader)
			if err != nil {
				return nil, err
			}
		}
	}

	return retArray, err
}

// generateConfigs will take a list of strings with containing the paths to the
// unpacked symbol files. It will return a list of generated task configs
// alongside the communication channels to be used.
func generateConfigs(symbolFiles []string, dryRun, isStaging bool) []taskConfig {

	tasks := make([]taskConfig, len(symbolFiles))

	// Generate task configurations.
	for index, filepath := range symbolFiles {
		tasks[index] = taskConfig{filepath, 0, dryRun, isStaging}
	}

	return tasks
}

// doUpload is the main loop that will spawn goroutines that will handle the
// upload tasks. Should its worker fail it's upload and we have retries left,
// send the task to the end of the channel's buffer.
func doUpload(tasks []taskConfig, channelSize int, retryCount int,
	isStaging, dryRun bool) (int, error) {
	// TODO(b/197010274): remove skeleton code.
	return 0, nil
}

// validate checks the values of the required flags and returns an error they
// aren't populated. Since multiple flags are required, the error message may
// include multiple error statements.
func (b *uploadDebugSymbols) validate() error {
	errStr := ""
	if b.gsPath == "" {
		errStr = "error: -gs-path value is required.\n"
	}
	if strings.HasPrefix(b.gsPath, "gs://") {
		errStr = fmt.Sprint(errStr, "error: -gs-path must point to a google storage location. E.g. gs://some-bucket/debug.tgz")
	}
	if strings.HasSuffix(b.gsPath, "debug.tgz") {
		errStr = fmt.Sprint(errStr, "error: -gs-path must point to a debug.tgz file.")
	}
	if b.workerCount <= 0 {
		errStr = fmt.Sprint(errStr, "error: -worker-count value must be greater than zero.\n")
	}
	if b.retryCount < 0 {
		errStr = fmt.Sprint(errStr, "error: -retry-count value may not be negative.\n")
	}

	if errStr != "" {
		return fmt.Errorf(errStr)
	}
	return nil
}

// Run is the function to be called by the CLI execution.
// TODO(b/197010274): Move business logic into a separate function so Run() can be tested fully.
func (b *uploadDebugSymbols) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	// Generate authenticated http client.
	ctx := context.Background()
	authOpts, err := b.authFlags.Options()
	if err != nil {
		log.Fatal(err)
	}
	client, err := generateClient(ctx, authOpts)
	if err != nil {
		log.Fatal(err)
	}
	// Create local dir and file for tarball to live in.
	workDir, err := ioutil.TempDir("", "tarball")
	if err != nil {
		log.Fatal(err)
	}
	symbolDir, err := ioutil.TempDir(workDir, "symbols")
	if err != nil {
		log.Fatal(err)
	}

	tgzPath := filepath.Join(workDir, "debug.tgz")
	tarbalPath := filepath.Join(workDir, "debug.tar")
	defer os.RemoveAll(workDir)

	err = downloadTgz(client, b.gsPath, tgzPath)
	if err != nil {
		log.Fatal(err)
	}

	err = unzipTgz(tgzPath, tarbalPath)
	if err != nil {
		log.Fatal(err)
	}

	symbolFiles, err := unpackTarball(tarbalPath, symbolDir)
	if err != nil {
		log.Fatal(err)
	}

	tasks := generateConfigs(symbolFiles, b.dryRun, b.isStaging)
	if err != nil {
		log.Fatal(err)
	}

	retcode, err := doUpload(tasks, b.channelSize, b.retryCount, b.isStaging, b.dryRun)

	if err != nil {
		log.Fatal(err)
	}
	// TODO(b/197010274): remove skeleton code.
	// Return:
	// 		0: Success, all symbols uploaded.
	// 		1: Failure, more failures occurred than retries were allotted
	return retcode
}
