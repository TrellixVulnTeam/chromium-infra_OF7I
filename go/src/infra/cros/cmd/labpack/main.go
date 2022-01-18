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
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/auth"
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

// DescriptionCommand describes the environment where labpack was run. It must write all of its output to stderr
// and none of its output to stdout for hygiene purposes.
const DescriptionCommand = `( echo BEGIN; echo PWD; pwd ; echo FIND; find . ; echo ENV; env; echo END ) 1>&2`

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", filepath.Base(os.Args[0])))
	log.Printf("Running version: %s", site.VersionNumber)
	log.Printf("Running in build-bucket mode")

	input := &steps.LabpackInput{}
	var writeOutputProps func(*steps.LabpackResponse)
	var mergeOutputProps func(*steps.LabpackResponse)
	if LuciexeProtocolPassthru {
		log.Printf("Bypassing luciexe.")
		panic("Bypassing luciexe not yet supported.")
	} else {
		build.Main(input, &writeOutputProps, &mergeOutputProps,
			func(ctx context.Context, args []string, state *build.State) error {
				// TODO(otabek@): Add custom logger.
				lg := logger.NewLogger()

				// Right after instantiating the logger, but inside build.Main's callback,
				// make sure that we log what our environment looks like.
				if DescribeMyDirectoryAndEnvironment {
					describeEnvironment(os.Stderr)
				}

				// Set the log (via the Go standard library's log package) to Stderr, since we know that stderr is collected
				// for the process as a whole. This will also indirectly influence lg.
				log.SetOutput(os.Stderr)

				res := &steps.LabpackResponse{Success: true}
				err := internalRun(ctx, input, state, lg)
				if err != nil {
					res.Success = false
					res.FailReason = err.Error()
					lg.Debug("Finished with err: %s", err)
				}
				writeOutputProps(res)
				// if err is nil then will marked as SUCCESS
				return err
			},
		)
	}
	log.Printf("Exited successfully")
}

// internalRun main entry point to execution received request.
func internalRun(ctx context.Context, in *steps.LabpackInput, state *build.State, lg logger.Logger) (err error) {
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
	defer access.Close()

	task := tasknames.Recovery
	if t, ok := supportedTasks[in.TaskName]; ok {
		task = t
	}
	var metrics metrics.Metrics
	if !in.GetNoMetrics() {
		var err error
		metrics, err = karte.NewMetrics(ctx, kclient.DevConfig(auth.Options{}))
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
	command.Stderr = stderr
	err := command.Run()
	return errors.Annotate(err, "describe environment").Err()
}
