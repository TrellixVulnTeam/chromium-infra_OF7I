// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime/debug"

	"infra/chromium/bootstrapper/bootstrap"
	"infra/chromium/bootstrapper/cipd"
	"infra/chromium/bootstrapper/gitiles"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	logdogbootstrap "go.chromium.org/luci/logdog/client/butlerlib/bootstrap"
	"go.chromium.org/luci/logdog/client/butlerlib/streamclient"
	"go.chromium.org/luci/luciexe"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

func getBuild(ctx context.Context, input io.Reader) (*buildbucketpb.Build, error) {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, errors.Annotate(err, "failed to read build input").Err()
	}
	build := &buildbucketpb.Build{}
	if err = proto.Unmarshal(data, build); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshall build").Err()
	}
	return build, nil
}

func performBootstrap(ctx context.Context, input io.Reader, cipdRoot, buildOutputPath string) (*exec.Cmd, error) {
	build, err := getBuild(ctx, input)
	if err != nil {
		return nil, err
	}

	bootstrapper, err := bootstrap.NewBootstrapper(build)
	if err != nil {
		return nil, err
	}

	var recipeInput []byte
	var cmd []string
	group, ectx := errgroup.WithContext(ctx)

	// Get the arguments for the command
	group.Go(func() error {
		recipeClient, err := cipd.NewClient(ectx, cipdRoot)
		if err != nil {
			return err
		}

		cmd, err = bootstrapper.SetupExe(ectx, recipeClient)
		if err != nil {
			return err
		}

		if buildOutputPath != "" {
			cmd = append(cmd, "--output", buildOutputPath)
		}

		return nil
	})

	// Get the input for the command
	group.Go(func() error {
		properties, err := bootstrapper.ComputeBootstrappedProperties(ectx, gitiles.NewClient(ectx))
		if err != nil {
			return err
		}

		build.Input.Properties = properties
		recipeInput, err = proto.Marshal(build)
		return errors.Annotate(err, "failed to marshall bootstrapped build input: <%s>", build).Err()
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	cmdCtx := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	cmdCtx.Stdin = bytes.NewBuffer(recipeInput)
	return cmdCtx, nil
}

func execute(ctx context.Context) error {
	outputPath := flag.String("output", "", "Path to write the final build.proto state to.")
	flag.Parse()

	cmd, err := performBootstrap(ctx, os.Stdin, "cipd", *outputPath)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func reportBootstrapFailure(ctx context.Context, summary string) error {
	bootstrap, err := logdogbootstrap.Get()
	if err != nil {
		return err
	}
	stream, err := bootstrap.Client.NewDatagramStream(
		ctx,
		luciexe.BuildProtoStreamSuffix,
		streamclient.WithContentType(luciexe.BuildProtoContentType),
	)
	if err != nil {
		return err
	}
	build := &buildbucketpb.Build{}
	build.SummaryMarkdown = fmt.Sprintf("<pre>%s</pre>", summary)
	build.Status = buildbucketpb.Status_INFRA_FAILURE
	outputData, err := proto.Marshal(build)
	if err != nil {
		return errors.Annotate(err, "failed to marshal output build.proto").Err()
	}
	return stream.WriteDatagram(outputData)
}

func main() {
	ctx := context.Background()
	ctx = gologger.StdConfig.Use(ctx)

	defer func() {
		if r := recover(); r != nil {
			err := reportBootstrapFailure(ctx, fmt.Sprintf("encountered panic: %s: %s", r, debug.Stack()))
			if err != nil {
				logging.Errorf(ctx, errors.Annotate(err, "failed to report bootstrap panic").Err().Error())
			}
			os.Exit(1)
		}
	}()

	if err := execute(ctx); err != nil {
		logging.Errorf(ctx, err.Error())
		// An ExitError indicates that we were able to bootstrap the
		// executable and that it failed, so it should have populated
		// the build proto with steps and a result
		if _, ok := err.(*exec.ExitError); !ok {
			err := reportBootstrapFailure(ctx, err.Error())
			if err != nil {
				logging.Errorf(ctx, errors.Annotate(err, "failed to report bootstrap failure").Err().Error())
			}
		}
		os.Exit(1)
	}
}
