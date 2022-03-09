// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/*
The labpack program allows to run repair tasks for ChromeOS devices in the lab.
For more information please read go/paris-.
Managed by Chrome Fleet Software (go/chrome-fleet-software).
*/
package main

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	luciauth "go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/luciexe/build"

	"infra/cros/cmd/labpack/internal/site"
	steps "infra/cros/cmd/labpack/internal/steps"
	"infra/cros/cmd/labpack/internal/tlw"
	kclient "infra/cros/karte/client"
	"infra/cros/recovery"
	"infra/cros/recovery/karte"
	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
	"infra/cros/recovery/tasknames"
)

// LuciexeProtocolPassthru should always be set to false in checked-in code.
// Only set it to true for development purposes.
const LuciexeProtocolPassthru = false

//
// DescribeMyDirectoryAndEnvironment controls whether labpack should write information
// about where it was run (cwd), what files are near it, and the contents of the environment.
const DescribeMyDirectoryAndEnvironment = true

// DescriptionCommand describes the environment where labpack was run. It must write all of its output to stdout.
const DescriptionCommand = `( echo BEGIN; echo PWD; pwd ; echo FIND; find . ; echo ENV; env; echo END )`

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", filepath.Base(os.Args[0])))
	log.Printf("Running version: %s", site.VersionNumber)
	log.Printf("Running in buildbucket mode")

	input := &steps.LabpackInput{}
	var writeOutputProps func(*steps.LabpackResponse)
	var mergeOutputProps func(*steps.LabpackResponse)
	if LuciexeProtocolPassthru {
		log.Printf("Bypassing luciexe.")
		panic("Bypassing luciexe not yet supported.")
	} else {
		build.Main(input, &writeOutputProps, &mergeOutputProps,
			func(ctx context.Context, args []string, state *build.State) error {

				cwd, err := os.Getwd()
				if err != nil {
					return errors.Annotate(err, "internal run: cannot get current working directory!").Err()
				}
				logRoot := filepath.Join(cwd, "logs")

				// TODO(otabek@): Add custom logger.
				lg := logger.NewLogger()

				// Right after instantiating the logger, but inside build.Main's callback,
				// make sure that we log what our environment looks like.
				if DescribeMyDirectoryAndEnvironment {
					describeEnvironment(os.Stderr)
					// Describe the contents of the directory once on the way out too.
					// We will use this information to decide what to persist.
					defer describeEnvironment(os.Stderr)
				}

				// Set the log (via the Go standard library's log package) to Stderr, since we know that stderr is collected
				// for the process as a whole. This will also indirectly influence lg.
				log.SetOutput(os.Stderr)

				// TODO(gregorynisbet): Remove canary.
				// Write a labpack canary file with contents that are unique.
				err = ioutil.WriteFile(
					filepath.Join(logRoot, "_labpack_canary"),
					[]byte("46182c32-2c9d-4abd-a029-54a71c90261e"),
					0b110_110_110,
				)
				if err == nil {
					lg.Info("successfully wrote canary file")
				} else {
					lg.Error("failed to write canary file: %s", err)
				}

				res := &steps.LabpackResponse{Success: true}
				err = internalRun(ctx, input, state, lg, logRoot)
				if err != nil {
					res.Success = false
					res.FailReason = err.Error()
					lg.Debug("Finished with err: %s", err)
				}
				writeOutputProps(res)

				// TODO(gregorynisbet): Extract to function.

				// Actually persist the logs
				swarmingTaskID := state.Infra().GetSwarming().GetTaskId()
				if swarmingTaskID == "" {
					// Failed to get the swarming task, log and move on.
					lg.Error("Swarming task is empty!")
				} else {
					// Just out of an abundance of caution, note that the following block is safe
					// even if the command /usr/bin/timeout does not exist (or timeout does not exist
					// at that path). Any errors that we encounter when running commands are logged and
					// don't interfere with the overall task.

					// upload.Upload can potentially loop for a long time. Set a timeout of 30s.
					//
					// Also, protect ourselves from this task by running it as a subprocess.
					// See b:223230183 for details.
					cmd := exec.Command(
						"/usr/bin/timeout",
						"30s",
						"paris-uploader",
						"upload",
						"-local",
						".",
						"-gs",
						// TODO(gregorynisbet): Allow this parameter to be overridden from outside.
						fmt.Sprintf("gs://chromeos-autotest-results/swarming-%s", swarmingTaskID),
					)
					// Forward both these streams to our stderr.
					cmd.Stdout = os.Stderr
					cmd.Stderr = os.Stderr

					if rErr := cmd.Run(); rErr == nil {
						lg.Info("call to paris-offloader was successful")
					} else {
						lg.Error("error when calling paris-offloader: %s", rErr)
					}
				}

				// if err is nil then will marked as SUCCESS
				return err
			},
		)
	}
	log.Printf("Exited successfully")
}

// internalRun main entry point to execution received request.
func internalRun(ctx context.Context, in *steps.LabpackInput, state *build.State, lg logger.Logger, logRoot string) (err error) {
	// Catching the panic here as luciexe just set a step as fail and but not exit execution.
	defer func() {
		if r := recover(); r != nil {
			lg.Debug("Received panic!")
			err = errors.Reason("panic: %s", r).Err()
		}
	}()
	if err = printInputs(ctx, in); err != nil {
		lg.Debug("Internal run: failed to marshal proto. Error: %s", err)
		return err
	}
	access, err := tlw.NewAccess(ctx, in)
	if err != nil {
		return errors.Annotate(err, "internal run").Err()
	}
	defer access.Close(ctx)

	task := tasknames.Recovery
	if t, ok := supportedTasks[in.TaskName]; ok {
		task = t
	}
	var metrics metrics.Metrics
	if !in.GetNoMetrics() {
		var err error
		metrics, err = karte.NewMetrics(ctx, kclient.ProdConfig(luciauth.Options{}))
		if err == nil {
			lg.Info("internal run: metrics client successfully created.")
		} else {
			// TODO(gregorynisbet): Make this error end the current function.
			lg.Error("internal run: failed to instantiate karte client: %s", err)
		}
	}
	cr, err := getConfiguration(in.GetConfiguration(), lg)
	if err != nil {
		return errors.Annotate(err, "internal run").Err()
	}

	runArgs := &recovery.RunArgs{
		UnitName:              in.GetUnitName(),
		TaskName:              task,
		Access:                access,
		Logger:                lg,
		ShowSteps:             !in.GetNoStepper(),
		Metrics:               metrics,
		EnableRecovery:        in.GetEnableRecovery(),
		EnableUpdateInventory: in.GetUpdateInventory(),
		ConfigReader:          cr,
		SwarmingTaskID:        state.Infra().GetSwarming().GetTaskId(),
		BuildbucketID:         state.Infra().GetBackend().GetTask().GetId().GetId(),
		LogRoot:               logRoot,
	}
	lg.Debug("Labpack: started recovery engine.")
	if err := recovery.Run(ctx, runArgs); err != nil {
		return errors.Annotate(err, "internal run").Err()
	}
	return nil
}

// getConfiguration read base64 configuration from input and create reader for recovery-engine.
// If configuration is empty then we will pass nil so recovery-engine will use default configuration.
func getConfiguration(config string, lg logger.Logger) (io.Reader, error) {
	if config == "" {
		lg.Debug("Labpack: received empty configuration.")
		return nil, nil
	}
	dc, err := b64.StdEncoding.DecodeString(config)
	if err != nil {
		return nil, errors.Annotate(err, "get configuration: decode configuration").Err()
	}
	lg.Debug("Received configuration:\n%s", string(dc))
	return bytes.NewReader(dc), nil
}

// Mapping of all supported tasks.
var supportedTasks = map[string]tasknames.TaskName{
	string(tasknames.Deploy):   tasknames.Deploy,
	string(tasknames.Recovery): tasknames.Recovery,
}

// printInputs prints input params.
func printInputs(ctx context.Context, input *steps.LabpackInput) (err error) {
	step, ctx := build.StartStep(ctx, "Input params")
	defer func() { step.End(err) }()
	req := step.Log("input proto")
	marsh := jsonpb.Marshaler{Indent: "  "}
	if err = marsh.Marshal(req, input); err != nil {
		return errors.Annotate(err, "failed to marshal proto").Err()
	}
	return nil
}

// describeEnvironment describes the environment where labpack is being run.
// TODO(gregorynisbet): Remove this thing.
func describeEnvironment(stderr io.Writer) error {
	command := exec.Command("/bin/sh", "-c", DescriptionCommand)
	// DescriptionCommand writes its contents to stdout, so wire it up to stderr.
	command.Stdout = stderr
	err := command.Run()
	return errors.Annotate(err, "describe environment").Err()
}
