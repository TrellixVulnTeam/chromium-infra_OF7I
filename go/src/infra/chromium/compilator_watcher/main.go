// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Displays steps from the compilator to the chromium orchestrator

package main

import (
	"compress/zlib"
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"

	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/buildbucket/protoutil"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/luciexe/exe"
	"google.golang.org/protobuf/types/known/timestamppb"
	"infra/chromium/compilator_watcher/internal/bb"
)

// Set by the compilator.py recipe in tools/build
const swarmingTriggerPropsStepName = "swarming trigger properties"
const swarmingOutputPropKey = "swarming_trigger_properties"

const swarmingPhase = "getSwarmingTriggerProps"
const localTestPhase = "getLocalTests"

func main() {
	exe.Run(luciEXEMain, exe.WithZlibCompression(zlib.BestCompression))
}

// The exe.MainFn entry point for this luciexe binary.
func luciEXEMain(ctx context.Context, input *buildbucket_pb.Build, userArgs []string, send exe.BuildSender) (err error) {
	ctx = logging.SetLevel(ctx, logging.Info)

	defer func() {
		// processErr updates the returned err and input's SummaryMarkdown
		err = processErr(ctx, err, input, send)
		send()
	}()

	input.Status = buildbucket_pb.Status_STARTED
	input.StartTime = timestamppb.New(clock.Now(ctx))
	send()
	parsedArgs, err := parseArgs(userArgs)
	if err != nil {
		return err
	}

	compBuild, err := copySteps(ctx, input, parsedArgs, send)
	if err != nil {
		return err
	}

	err = copyOutputProperties(ctx, input, compBuild, parsedArgs, send)
	if err != nil {
		return err
	}

	if parsedArgs.phase == swarmingPhase && swarmingPropsInCompBuildOutput(compBuild) {
		input.Status = buildbucket_pb.Status_SUCCESS
	} else {
		input.Status = compBuild.GetStatus()
		input.SummaryMarkdown = compBuild.GetSummaryMarkdown()
	}
	input.EndTime = timestamppb.New(clock.Now(ctx))
	send()
	return nil
}

type cmdArgs struct {
	compilatorID                   int64
	phase                          string
	compPollingTimeoutSec          time.Duration
	compPollingIntervalSec         time.Duration
	maxConsecutiveGetBuildTimeouts int64
}

func parseArgs(args []string) (cmdArgs, error) {
	fs := flag.NewFlagSet("f1", flag.ContinueOnError)

	compBuildId := fs.String("compilator-id", "", "Buildbucket ID of triggered compilator")
	getSwarmingTriggerProps := fs.Bool("get-swarming-trigger-props", false, "Sub-build will report steps up to `swarming trigger properties`")
	getLocalTests := fs.Bool("get-local-tests", false, "Sub-build will report steps of local tests")
	compPollingTimeoutSec := fs.Int64(
		"compilator-polling-timeout-sec",
		7200,
		"Max number of seconds to poll compilator")

	compPollingIntervalSec := fs.Int64(
		"compilator-polling-interval-sec",
		10,
		"Number of seconds to wait between compilator polls")

	maxGetBuildTimeouts := fs.Int64(
		"max-consecutive-get-build-timeouts",
		3,
		"The maximum amount of consecutive timeouts allowed when running GetBuild for the compilator build")

	if err := fs.Parse(args); err != nil {
		return cmdArgs{}, err
	}

	errs := errors.NewMultiError()
	if *compBuildId == "" {
		errs = append(errs, errors.Reason("compilator-id is required").Err())
	}
	if *getSwarmingTriggerProps == *getLocalTests {
		errs = append(errs, errors.Reason(
			"Exactly one of -get-swarming-trigger-props or -get-local-tests is required").Err())
	}
	if errs.First() != nil {
		return cmdArgs{}, errs
	}

	phase := localTestPhase
	if *getSwarmingTriggerProps {
		phase = swarmingPhase
	}
	convertedCompBuildID, err := strconv.ParseInt(*compBuildId, 10, 64)
	if err != nil {
		return cmdArgs{}, err
	}

	return cmdArgs{
		compilatorID:                   convertedCompBuildID,
		phase:                          phase,
		compPollingTimeoutSec:          time.Duration(*compPollingTimeoutSec) * time.Second,
		compPollingIntervalSec:         time.Duration(*compPollingIntervalSec) * time.Second,
		maxConsecutiveGetBuildTimeouts: *maxGetBuildTimeouts,
	}, nil
}

var hideCompSteps = stringset.NewFromSlice(
	"setup_build", "report builders", "use rts: False", "use rts: True",
	"gclient config", "gerrit fetch current CL info",
	"gclient runhooks", "set_output_gitiles_commit", "read test spec",
	"get compile targets for scripts", "git diff to analyze patch",
	"create .code-coverage",
)

func getLatestBuildStepName(build *buildbucket_pb.Build) string {
	buildSteps := build.GetSteps()
	buildStepsLen := len(buildSteps)
	if buildStepsLen > 0 {
		return buildSteps[buildStepsLen-1].GetName()
	} else {
		return ""
	}
}

func updateFilteredSteps(
	luciBuild *buildbucket_pb.Build,
	compBuild *buildbucket_pb.Build,
	phase string) {
	if phase == swarmingPhase {
		luciBuild.Steps = getStepsUntilSwarmingTriggerProps(compBuild)
	} else {
		luciBuild.Steps = getStepsAfterSwarmingTriggerProps(compBuild)
	}
}

func getStepsUntilSwarmingTriggerProps(
	compBuild *buildbucket_pb.Build) []*buildbucket_pb.Step {
	var filteredSteps []*buildbucket_pb.Step
	for _, compBuildStep := range compBuild.GetSteps() {
		name := compBuildStep.GetName()

		if !hideCompSteps.Has(name) {
			filteredSteps = append(filteredSteps, compBuildStep)
		} else {
			// Only display hidden step if somethings wrong with it
			if compBuildStep.Status != buildbucket_pb.Status_STARTED && compBuildStep.Status != buildbucket_pb.Status_SUCCESS {
				filteredSteps = append(filteredSteps, compBuildStep)
			}
		}
		if name == swarmingTriggerPropsStepName {
			break
		}
	}
	return filteredSteps
}

func updateLastStep(luciBuild *buildbucket_pb.Build, compBuild *buildbucket_pb.Build) {
	// This function is called when the latest compBuild step name has not
	// changed but the copied step in luciBuild should still be updated in
	// case the step's status has changed.
	compBuildSteps := compBuild.GetSteps()
	latestCompStep := compBuildSteps[len(compBuildSteps)-1]

	luciBuildSteps := luciBuild.GetSteps()

	if !hideCompSteps.Has(latestCompStep.GetName()) {
		luciBuildSteps[len(luciBuildSteps)-1] = latestCompStep
	} else {
		// Only display hidden step if somethings wrong with it
		if protoutil.IsEnded(latestCompStep.Status) && latestCompStep.Status != buildbucket_pb.Status_SUCCESS {
			if len(luciBuildSteps) == 0 || luciBuildSteps[len(luciBuildSteps)-1].GetName() != latestCompStep.GetName() {
				luciBuild.Steps = append(luciBuild.Steps, latestCompStep)
			} else {
				luciBuildSteps[len(luciBuildSteps)-1] = latestCompStep
			}
		}
	}
}

func getStepsAfterSwarmingTriggerProps(
	compBuild *buildbucket_pb.Build) []*buildbucket_pb.Step {

	for i, step := range compBuild.GetSteps() {
		if step.GetName() == swarmingTriggerPropsStepName {
			return compBuild.GetSteps()[i+1:]
		}

	}
	return []*buildbucket_pb.Step{}
}

func processErr(ctx context.Context, err error, luciBuild *buildbucket_pb.Build, send exe.BuildSender) error {
	if err == nil {
		return nil
	}
	// We want the status to show CANCELED instead of INFRA_FAILURE so
	// the orchestrator can handle a CANCELED status differently.
	if errors.Unwrap(err) == context.Canceled {
		luciBuild.SummaryMarkdown = "compilator_watcher context was canceled. Probably due to the parent orchestrator build being canceled."
		luciBuild.Status = buildbucket_pb.Status_CANCELED
		// Returning an err would automatically set the build status to FAILURE
		// See runUserCode() in https://source.chromium.org/chromium/infra/infra/+/main:go/src/go.chromium.org/luci/luciexe/exe/exe.go
		return nil
	}
	// This enforces the triggered sub_build to have an INFRA_FAILURE
	// status
	err = exe.InfraErrorTag.Apply(err)

	summaryMarkdownFmt := "Error while running compilator_watcher: %s"
	if errors.Unwrap(err) == context.DeadlineExceeded {
		luciBuild.SummaryMarkdown = fmt.Sprintf(
			summaryMarkdownFmt, "Timeout waiting for compilator build")
	} else {
		luciBuild.SummaryMarkdown = fmt.Sprintf(
			summaryMarkdownFmt, err)
	}
	return err
}

func copySteps(ctx context.Context, luciBuild *buildbucket_pb.Build, parsedArgs cmdArgs, send exe.BuildSender) (*buildbucket_pb.Build, error) {
	// Poll the compilator build until it's complete or the swarming props
	// are found, while copying over filtered steps depending on the given
	// phase.
	// Return the compilator build from the most recent GetBuild call

	bClient, err := bb.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	cctx, cancel := clock.WithTimeout(ctx, parsedArgs.compPollingTimeoutSec)
	defer cancel()

	var latestCompBuildStepName = ""

	var timeoutCounts int64 = 0
	for {
		// Check for context err like a timeout or cancelation before
		// continuing with the for loop
		if cctx.Err() != nil {
			return nil, cctx.Err()
		}

		compBuild, err := bClient.GetBuild(cctx, parsedArgs.compilatorID)

		// Check that the err is from the GetBuild call, not the
		// timeout set for polling
		if err != nil {
			if grpcutil.Code(err) == codes.DeadlineExceeded {
				if timeoutCounts < parsedArgs.maxConsecutiveGetBuildTimeouts {
					timeoutCounts += 1
					continue
				}
			}
			return nil, err
		}
		// Reset counter
		timeoutCounts = 0

		switch maybeLatestCompStepName := getLatestBuildStepName(compBuild); {
		case maybeLatestCompStepName != latestCompBuildStepName:
			latestCompBuildStepName = maybeLatestCompStepName
			updateFilteredSteps(luciBuild, compBuild, parsedArgs.phase)
			send()
		case maybeLatestCompStepName != "":
			updateLastStep(luciBuild, compBuild)
			send()
		}

		if protoutil.IsEnded(compBuild.GetStatus()) || (parsedArgs.phase == swarmingPhase && swarmingPropsInCompBuildOutput(compBuild)) {
			return compBuild, nil
		}

		if tr := clock.Sleep(cctx, parsedArgs.compPollingIntervalSec); tr.Err != nil {
			return compBuild, tr.Err
		}
	}
}

func swarmingPropsInCompBuildOutput(compBuild *buildbucket_pb.Build) bool {
	_, ok := compBuild.GetOutput().GetProperties().GetFields()[swarmingOutputPropKey]
	return ok
}

func copyOutputProperties(ctx context.Context, luciBuild *buildbucket_pb.Build, compBuild *buildbucket_pb.Build, parsedArgs cmdArgs, send exe.BuildSender) error {
	err := exe.WriteProperties(
		luciBuild.Output.Properties,
		compBuild.GetOutput().GetProperties().AsMap())
	if err != nil {
		return err
	}

	send()
	return nil
}
