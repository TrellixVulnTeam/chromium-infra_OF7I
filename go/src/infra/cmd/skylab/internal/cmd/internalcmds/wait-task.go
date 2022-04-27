// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package internalcmds

import (
	"context"
	"fmt"
	"infra/cmd/skylab/internal/bb"
	"infra/cmd/skylab/internal/site"
	"io"
	"strconv"
	"time"

	"github.com/maruel/subcommands"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmdsupport/cmdlib"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_tool"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
)

// WaitTask subcommand: wait for a task to finish.
var WaitTask = &subcommands.Command{
	UsageLine: "wait-task [FLAGS...] TASK_ID",
	ShortDesc: "wait for a task to complete",
	LongDesc: `Wait for the tasks with the given Buildbucket build id to complete, and summarize its results.
Only tasks created via the skylab tool are supported`,
	CommandRun: func() subcommands.CommandRun {
		c := &waitTaskRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.timeoutMins, "timeout-mins", -1, "The maxinum number of minutes to wait for the task to finish. Default: no timeout.")
		// TODO: Delete this flag entirely.
		// There should be no users of this flag now, but remove in own CL for
		// easy revert.
		var unused bool
		c.Flags.BoolVar(&unused, "bb", true, "Deprecated. Has no effect.")
		return c
	},
}

type waitTaskRun struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    skycmdlib.EnvFlags
	timeoutMins int
}

func (c *waitTaskRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *waitTaskRun) innerRun(a subcommands.Application, env subcommands.Env) error {
	result, err := c.innerRunBuildbucket(a, env)
	if err != nil {
		return err
	}
	printJSONResults(a.GetOut(), result)
	return nil
}

func (c *waitTaskRun) innerRunBuildbucket(a subcommands.Application, env subcommands.Env) (*skylab_tool.WaitTaskResult, error) {
	taskID, err := parseBBTaskID(c.Flags.Arg(0))
	if err != nil {
		return nil, cmdlib.NewUsageError(c.Flags, err.Error())
	}

	ctx := cli.GetContext(a, c, env)
	ctx, cancel := cmdlib.MaybeWithTimeout(ctx, c.timeoutMins)
	defer cancel(context.Canceled)

	bClient, err := bb.NewClient(ctx, c.envFlags.Env().CTPBuilderInfo, c.authFlags)
	if err != nil {
		return nil, err
	}

	build, err := bClient.WaitForBuild(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return responseToTaskResult(bClient, build)
}

func parseBBTaskID(arg string) (int64, error) {
	if arg == "" {
		return -1, errors.Reason("missing buildbucket task id").Err()
	}
	ID, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return -1, errors.Reason("malformed buildbucket id: %s", err).Err()
	}
	return ID, nil
}

func responseToTaskResult(bClient *bb.Client, build *bb.Build) (*skylab_tool.WaitTaskResult, error) {
	buildID := build.ID
	u := bClient.BuildURL(buildID)
	// TODO(pprabhu) Add verdict to WaitTaskResult_Task and deprecate Failure /
	// Success fields.
	// Currently, we merely leave both fields unset when no definite verdict can
	// be returned.
	tr := &skylab_tool.WaitTaskResult_Task{
		Name:          "Test Platform Invocation",
		TaskRunUrl:    u,
		TaskRunId:     fmt.Sprintf("%d", buildID),
		TaskRequestId: fmt.Sprintf("%d", buildID),
		Failure:       isBuildFailed(build),
		Success:       isBuildPassed(build),
	}
	rs := build.Responses.GetTaggedResponses()
	if rs == nil {
		return nil, errors.Reason("missing tagged_responses").Err()
	}

	// CTP runs created using the Skylab tool have a single response tagged "default".
	r, ok := rs["default"]
	if !ok {
		return nil, errors.Reason("missing 'default' response in %+v", rs).Err()
	}

	var childResults []*skylab_tool.WaitTaskResult_Task
	for _, child := range r.GetTaskResults() {
		verdict := child.GetState().GetVerdict()
		failure := verdict == test_platform.TaskState_VERDICT_FAILED
		success := verdict == test_platform.TaskState_VERDICT_PASSED
		childResult := &skylab_tool.WaitTaskResult_Task{
			Name:        child.Name,
			TaskLogsUrl: child.LogUrl,
			TaskRunUrl:  child.TaskUrl,
			// Note: TaskRunID is deprecated and excluded here.
			Failure: failure,
			Success: success,
		}
		childResults = append(childResults, childResult)
	}
	return &skylab_tool.WaitTaskResult{
		ChildResults: childResults,
		Result:       tr,
		// Note: Stdout it not set.
	}, nil
}

func isBuildFailed(build *bb.Build) bool {
	return build.Response != nil && build.Response.GetState().GetVerdict() == test_platform.TaskState_VERDICT_FAILED
}

func isBuildPassed(build *bb.Build) bool {
	return build.Response != nil && build.Response.GetState().GetVerdict() == test_platform.TaskState_VERDICT_PASSED
}

func sleepOrCancel(ctx context.Context, duration time.Duration) error {
	sleepTimer := time.NewTimer(duration)
	select {
	case <-sleepTimer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func printJSONResults(w io.Writer, m *skylab_tool.WaitTaskResult) {
	err := cmdlib.JSONPBMarshaller.Marshal(w, m)
	if err != nil {
		panic(err)
	}
}
