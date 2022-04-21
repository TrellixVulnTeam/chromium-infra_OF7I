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
	"time"

	"github.com/golang/protobuf/jsonpb"
	luciauth "go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	lucigs "go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/luciexe/build"
	"google.golang.org/grpc/metadata"

	"infra/cros/cmd/labpack/internal/site"
	steps "infra/cros/cmd/labpack/internal/steps"
	"infra/cros/cmd/labpack/internal/tlw"
	kclient "infra/cros/karte/client"
	"infra/cros/recovery"
	"infra/cros/recovery/karte"
	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
	"infra/cros/recovery/tasknames"
	"infra/cros/recovery/upload"
	ufsUtil "infra/unifiedfleet/app/util"
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

type ResponseUpdater func(*steps.LabpackResponse)

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", filepath.Base(os.Args[0])))
	log.Printf("Running version: %s", site.VersionNumber)
	log.Printf("Running in buildbucket mode")

	input := &steps.LabpackInput{}
	var writeOutputProps ResponseUpdater
	var mergeOutputProps ResponseUpdater
	if LuciexeProtocolPassthru {
		log.Printf("Bypassing luciexe.")
		log.Fatalf("Bypassing luciexe not yet supported.")
	} else {
		build.Main(input, &writeOutputProps, &mergeOutputProps,
			func(ctx context.Context, args []string, state *build.State) error {
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

				err := mainRunInternal(ctx, input, state, lg, writeOutputProps)
				return errors.Annotate(err, "main").Err()
			},
		)
	}
	log.Printf("Labpack done!")
}

// mainRun runs function for BB and provide result.
func mainRunInternal(ctx context.Context, input *steps.LabpackInput, state *build.State, lg logger.Logger, writeOutputProps ResponseUpdater) error {
	// Result errors which specify the result of main run.
	var resultErrors []error
	// Run recovery lib and get response.
	// Set result as fail by default in case it fail to finish by some reason.
	res := &steps.LabpackResponse{
		Success:    false,
		FailReason: "Fail by unknown reason!",
	}
	defer func() {
		// Write result as last step.
		writeOutputProps(res)
	}()
	if err := internalRun(ctx, input, state, lg); err != nil {
		res.Success = false
		res.FailReason = err.Error()
		resultErrors = append(resultErrors, err)
	}
	if err := uploadLogs(ctx, state, lg); err != nil {
		res.Success = false
		if len(resultErrors) == 0 {
			// We should not override runerror reason as it more important.
			// If upload logs error is only exits then set it as reason.
			res.FailReason = err.Error()
		}
		resultErrors = append(resultErrors, err)
	}
	// if err is nil then will marked as SUCCESS
	if len(resultErrors) == 0 {
		// Reset reason and state as no errors detected.
		res.Success = true
		res.FailReason = ""
		return nil
	}
	return errors.Annotate(errors.MultiError(resultErrors), "run recovery").Err()
}

// Upload logs to google cloud.
// The function has to fail only if that is critical fro the process.
// TODO: Need collect metrics of success collected logs per run to Karte.
func uploadLogs(ctx context.Context, state *build.State, lg logger.Logger) (rErr error) {
	step, ctx := build.StartStep(ctx, "Upload logs")
	lg.Infof("Beginning to upload logs")
	defer func() { step.End(rErr) }()
	// Construct the client that we will need to push the logs first.
	// Eventually, we will make this error fatal. However, for right now, we will
	// just log whether we succeeded or failed to build the client.
	authenticator := luciauth.NewAuthenticator(ctx, luciauth.SilentLogin, luciauth.Options{})
	if authenticator != nil {
		lg.Infof("NewAuthenticator(...): successfully authed!")
	} else {
		return errors.Reason("NewAuthenticator(...): did not successfully auth!").Err()
	}
	email, err := authenticator.GetEmail()
	if err == nil {
		lg.Infof("Auth email is %q", email)
	} else {
		return errors.Annotate(err, "upload logs").Err()
	}

	rt, err := authenticator.Transport()
	if err == nil {
		lg.Infof("authenticator.Transport(): successfully authed!")
	} else {
		lg.Errorf("authenticator.Transport(...): error: %s", err)
	}
	client, err := lucigs.NewProdClient(ctx, rt)
	if err == nil {
		lg.Infof("Successfully created client")
	} else {
		lg.Errorf("Failed to create client: %s", err)
	}

	// Actually persist the logs
	swarmingTaskID := state.Infra().GetSwarming().GetTaskId()
	if swarmingTaskID == "" {
		lg.Errorf("Swarming task is empty! Skipping upload logs")
	} else {
		// upload.Upload can potentially run for a long time. Set a timeout of 30s.
		//
		// upload.Upload does respond to cancellation (which callFuncWithTimeout uses internally), but
		// the correct of this code does not and should not depend on this fact.
		//
		// callFuncWithTimeout synchronously calls a function with a timeout and then unconditionally hands control
		// back to its caller. The goroutine that's created in the background will not by itself keep the process alive.
		// TODO(gregorynisbet): Allow this parameter to be overridden from outside.
		// TODO(crbug/1311842): Switch this bucket back to chromeos-autotest-results.
		gsURL := fmt.Sprintf("gs://chrome-fleet-karte-autotest-results/swarming-%s", swarmingTaskID)
		lg.Infof("Swarming task %q is non-empty. Uploading to %q", swarmingTaskID, gsURL)
		status, err := callFuncWithTimeout(ctx, 5*time.Minute, func(ctx context.Context) error {
			lg.Infof("Beginning upload attempt. Starting five minute timeout.")
			lg.Infof("Writing upload marker.")
			// TODO(b:227489086): Remove this file.
			if wErr := ioutil.WriteFile("_labpack_upload_marker", []byte("ca85a1f7-0de3-43c5-90ff-2e00b1041007"), 0o666); wErr != nil {
				lg.Errorf("Failed to write upload marker file: %s", wErr)
			}

			lg.Infof("Calling upload.")
			return upload.Upload(ctx, client, &upload.Params{
				// TODO(gregorynisbet): Change this to the log root.
				SourceDir:         ".",
				GSURL:             gsURL,
				MaxConcurrentJobs: 10,
			})
		})
		lg.Infof("Upload log subtask status: %s", status)
		if err != nil {
			// TODO: Register error to Karte.
			lg.Errorf("Upload task error: %s", err)
		}
	}
	return nil
}

// internalRun main entry point to execution received request.
func internalRun(ctx context.Context, in *steps.LabpackInput, state *build.State, lg logger.Logger) (err error) {
	defer func() {
		// Catching the panic here as luciexe just set a step as fail and but not exit execution.
		if r := recover(); r != nil {
			lg.Debugf("Received panic!")
			err = errors.Reason("panic: %s", r).Err()
		}
	}()
	if err = printInputs(ctx, in); err != nil {
		lg.Debugf("Internal run: failed to marshal proto. Error: %s", err)
		return err
	}
	ctx = setupContextNamespace(ctx, ufsUtil.OSNamespace)
	access, err := tlw.NewAccess(ctx, in)
	if err != nil {
		return errors.Annotate(err, "internal run").Err()
	}
	defer access.Close(ctx)

	// TODO: Need to use custom plan as default.
	task := tasknames.Recovery
	if t, ok := supportedTasks[in.TaskName]; ok {
		task = t
	}
	var metrics metrics.Metrics
	if !in.GetNoMetrics() {
		var err error
		metrics, err = karte.NewMetrics(ctx, kclient.ProdConfig(luciauth.Options{}))
		if err == nil {
			lg.Infof("internal run: metrics client successfully created.")
		} else {
			// TODO(gregorynisbet): Make this error end the current function.
			lg.Errorf("internal run: failed to instantiate karte client: %s", err)
		}
	}
	cr, err := getConfiguration(in.GetConfiguration(), lg)
	if err != nil {
		return errors.Annotate(err, "internal run").Err()
	}

	// TODO(gregorynisbet): Consider falling back to a temporary directory
	//                      if we for some reason cannot get our working directory.
	logRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot get working dir: %s\n", logRoot)
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
	lg.Debugf("Labpack: started recovery engine.")
	if err := recovery.Run(ctx, runArgs); err != nil {
		lg.Debugf("Labpack: finished recovery run with error: %v", err)
		return errors.Annotate(err, "internal run").Err()
	}
	lg.Debugf("Labpack: finished recovery successful")
	return nil
}

// getConfiguration read base64 configuration from input and create reader for recovery-engine.
// If configuration is empty then we will pass nil so recovery-engine will use default configuration.
func getConfiguration(config string, lg logger.Logger) (io.Reader, error) {
	if config == "" {
		lg.Debugf("Labpack: received empty configuration.")
		return nil, nil
	}
	dc, err := b64.StdEncoding.DecodeString(config)
	if err != nil {
		return nil, errors.Annotate(err, "get configuration: decode configuration").Err()
	}
	lg.Debugf("Received configuration:\n%s", string(dc))
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

// setupContextNamespace sets namespace to the context for UFS client.
func setupContextNamespace(ctx context.Context, namespace string) context.Context {
	md := metadata.Pairs(ufsUtil.Namespace, namespace)
	return metadata.NewOutgoingContext(ctx, md)
}
