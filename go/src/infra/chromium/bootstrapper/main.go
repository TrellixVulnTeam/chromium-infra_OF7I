// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	stderrors "errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"infra/chromium/bootstrapper/bootstrap"
	"infra/chromium/bootstrapper/cas"
	"infra/chromium/bootstrapper/cipd"
	"infra/chromium/bootstrapper/gerrit"
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
	"google.golang.org/protobuf/types/known/structpb"
)

func getBuild(ctx context.Context, input io.Reader) (*buildbucketpb.Build, error) {
	logging.Infof(ctx, "reading build input")
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, errors.Annotate(err, "failed to read build input").Err()
	}
	logging.Infof(ctx, "unmarshalling build input")
	build := &buildbucketpb.Build{}
	if err = proto.Unmarshal(data, build); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshall build").Err()
	}
	return build, nil
}

type options struct {
	propertiesOptional bool
}

func (o options) performBootstrap(ctx context.Context, input io.Reader, exeRoot, buildOutputPath string) (*exec.Cmd, error) {
	build, err := getBuild(ctx, input)
	if err != nil {
		return nil, err
	}

	logging.Infof(ctx, "creating bootstrap input")
	inputOpts := bootstrap.InputOptions{PropertiesOptional: o.propertiesOptional}
	bootstrapInput, err := inputOpts.NewInput(build)
	if err != nil {
		return nil, err
	}

	var recipeInput []byte
	var cmd []string

	// Introduce a new block to shadow the ctx variable so that the outer
	// value can't be used accidentally
	{
		group, ctx := errgroup.WithContext(ctx)

		exeCh := make(chan *bootstrap.BootstrappedExe, 1)

		// Get the arguments for the command
		group.Go(func() error {
			logging.Infof(ctx, "creating CIPD client")
			cipdClient, err := cipd.NewClient(ctx, exeRoot)
			if err != nil {
				return err
			}

			bootstrapper := bootstrap.NewExeBootstrapper(cipdClient, cas.NewClient(ctx, exeRoot))

			logging.Infof(ctx, "determining bootstrapped executable")
			exe, err := bootstrapper.GetBootstrappedExeInfo(ctx, bootstrapInput)
			if err != nil {
				return err
			}
			exeCh <- exe

			logging.Infof(ctx, "setting up bootstrapped executable")
			cmd, err = bootstrapper.DeployExe(ctx, exe)
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
			bootstrapper := bootstrap.NewBuildBootstrapper(gitiles.NewClient(ctx), gerrit.NewClient(ctx))

			logging.Infof(ctx, "getting bootstrapped config")
			config, err := bootstrapper.GetBootstrapConfig(ctx, bootstrapInput)
			if err != nil {
				return err
			}

			var exe *bootstrap.BootstrappedExe
			select {
			case exe = <-exeCh:
			case <-ctx.Done():
				return ctx.Err()
			}

			logging.Infof(ctx, "updating build")
			err = config.UpdateBuild(build, exe)
			if err != nil {
				return err
			}

			logging.Infof(ctx, "marshalling bootstrapped build input")
			recipeInput, err = proto.Marshal(build)
			return errors.Annotate(err, "failed to marshall bootstrapped build input: <%s>", build).Err()
		})

		if err := group.Wait(); err != nil {
			return nil, err
		}
	}

	cmdCtx := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	cmdCtx.Stdin = bytes.NewBuffer(recipeInput)
	return cmdCtx, nil
}

func execute(ctx context.Context) error {
	outputPath := flag.String("output", "", "Path to write the final build.proto state to.")
	propertiesOptional := flag.Bool("properties-optional", false, "Whether missing $bootstrap/properties property should be allowed")
	flag.Parse()

	opts := options{propertiesOptional: *propertiesOptional}
	cmd, err := opts.performBootstrap(ctx, os.Stdin, "exe", *outputPath)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logging.Infof(ctx, "executing %s", cmd.Args)
	err = cmd.Run()
	// An ExitError indicates that we were able to bootstrap the executable
	// and that it failed, as opposed to being unable to launch the
	// bootstrapped executable
	var exitErr *exec.ExitError
	if !stderrors.As(err, &exitErr) {
		return bootstrap.ExeFailure.Apply(err)
	}
	return err
}

type logdogBuildStreamFactory func(ctx context.Context) (streamclient.DatagramStream, error)

func newBuildStream(ctx context.Context) (streamclient.DatagramStream, error) {
	logdog, err := logdogbootstrap.Get()
	if err != nil {
		return nil, err
	}
	return logdog.Client.NewDatagramStream(
		ctx,
		luciexe.BuildProtoStreamSuffix,
		streamclient.WithContentType(luciexe.BuildProtoContentType),
	)
}

func reportBootstrapFailure(ctx context.Context, bootstrapErr error, buildStreamFactory logdogBuildStreamFactory) (err error) {
	// If the error has the ExeFailure tag, then that indicates that we were
	// able to bootstrap the executable and that it failed. In that case, it
	// should have populated the build proto with steps and a result, so we
	// should not modify the build proto.
	if bootstrap.ExeFailure.In(bootstrapErr) {
		return nil
	}

	stream, err := buildStreamFactory(ctx)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := stream.Close()
		if closeErr != nil {
			if err != nil {
				logging.Errorf(ctx, closeErr.Error())
			} else {
				err = closeErr
			}
		}
	}()

	build := &buildbucketpb.Build{}
	build.SummaryMarkdown = fmt.Sprintf("<pre>%s</pre>", bootstrapErr.Error())
	build.Status = buildbucketpb.Status_INFRA_FAILURE
	if bootstrap.PatchRejected.In(bootstrapErr) {
		build.Output = &buildbucketpb.Build_Output{
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"failure_type": structpb.NewStringValue("PATCH_FAILURE"),
				},
			},
		}
	}

	outputData, err := proto.Marshal(build)
	if err != nil {
		err = errors.Annotate(err, "failed to marshal output build.proto").Err()
		return
	}
	err = stream.WriteDatagram(outputData)
	if err != nil {
		err = errors.Annotate(err, "failed to write modified build").Err()
		return
	}
	return
}

func main() {
	ctx := context.Background()
	ctx = gologger.StdConfig.Use(ctx)

	if err := execute(ctx); err != nil {
		logging.Errorf(ctx, err.Error())
		reportErr := reportBootstrapFailure(ctx, err, newBuildStream)
		if reportErr != nil {
			logging.Errorf(ctx, errors.Annotate(err, "failed to report bootstrap failure").Err().Error())
		}
		os.Exit(1)
	}
}
