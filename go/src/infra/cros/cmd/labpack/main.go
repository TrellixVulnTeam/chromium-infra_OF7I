// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/*
The labpack program allows to run repair tasks for ChromeOS devices in the lab.
For more information please read go/AdminRepair.
Managed by Chrome Fleet Software (go/chrome-fleet-software).
*/
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	steps "infra/cros/cmd/labpack/steps"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/build"
)

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", filepath.Base(os.Args[0])))
	log.Printf("Starting with args: %s", os.Args)

	input := &steps.LabpackInput{}
	if isLocalRun() {
		log.Printf("Running in local mode")
		ctx := context.Background()
		state, ctx, err := build.Start(ctx, nil)
		defer func() { state.End(err) }()
		err = internalRun(ctx, input, state)
		return
	}
	log.Printf("Running in build-bucket mode")

	/*
		Notes for the next step.
		The input looks like
		{
			"$chromeos/service_version": {
				"version": {
				"skylabTool": "2"
				}
			},
			"labpack": "length",
			"your_name": "Supper user 1234"
		}
		need proto to read it like:
		message ServiceVersion {
			message VersionInfo {
				string skylabTool = 1;
			}
			VersionInfo version = 1;
		}
		to read version information
		var crosServiceVersionReader func(context.Context) *ServiceVersion
		func init() {
			MakePropertyReader("$chromeos/service_version", &crosServiceVersionReader)
		}
		and then we can read version input as
		serviceVersion := crosServiceVersionReader(ctx)
	*/
	var writeOutputProps func(*steps.LabpackResponse)
	var mergeOutputProps func(*steps.LabpackResponse)

	build.Main(input, &writeOutputProps, &mergeOutputProps,
		func(ctx context.Context, args []string, state *build.State) error {
			// this log expected to go to stdout
			logging.Infof(ctx, "have input %v", input)
			err := internalRun(ctx, input, state)
			// actual build code here, build is already Start'd
			// input was parsed from build.Input.Properties
			writeOutputProps(&steps.LabpackResponse{
				State: "Ready",
			})
			return err // if err is nil then will mark the Build as SUCCESS
		},
	)

	log.Printf("Exited successfully")
}

func logInputs(ctx context.Context, input *steps.LabpackInput) (err error) {
	step, ctx := build.StartStep(ctx, "inputs")
	defer func() { step.End(err) }()
	req := step.Log("input proto")
	marsh := jsonpb.Marshaler{Indent: "  "}
	if err = marsh.Marshal(req, input); err != nil {
		return errors.Annotate(err, "failed to marshal proto").Err()
	}
	return nil
}

func runRepairStep(ctx context.Context, idx int, input *steps.LabpackInput, state *build.State) (err error) {
	stepName := fmt.Sprintf("Repair of %d", idx)
	step, ctx := build.StartStep(ctx, stepName)
	defer func() { step.End(err) }()

	if idx > 5 {
		return errors.Reason("Repair step fail %d", idx).Err()
	}
	return nil
}

func runVerifierStep(ctx context.Context, idx int, input *steps.LabpackInput, state *build.State) (err error) {
	stepName := fmt.Sprintf("Verify of %d", idx)
	step, ctx := build.StartStep(ctx, stepName)
	defer func() { step.End(err) }()
	hasRepair := idx%2 == 0
	if hasRepair {
		repairErr := runRepairStep(ctx, idx, input, state)
		if repairErr != nil {
			logging.Infof(ctx, "repair step fail %v", repairErr)
		}
	}
	return nil
}

func internalRun(ctx context.Context, input *steps.LabpackInput, state *build.State) (err error) {
	if err = logInputs(ctx, input); err != nil {
		return err
	}
	for i := 1; i < 10; i++ {
		runVerifierStep(ctx, i, input, state)
	}
	return nil
}

func isLocalRun() bool {
	for _, arg := range os.Args {
		if arg == "-local" {
			return true
		}
	}
	return false
}
