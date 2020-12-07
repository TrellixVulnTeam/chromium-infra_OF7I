// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/phosphorus"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/cmd/phosphorus/internal/autotest/atutil"
	"infra/libs/lro"
)

// Prejob subcommand: Run a prejob (e.g. provision) against a DUT.
var Prejob = &subcommands.Command{
	UsageLine: "prejob -input_json /path/to/input.json",
	ShortDesc: "Run a prejob against a DUT.",
	LongDesc: `Run a prejob against a DUT.

Provision the DUT via 'autoserv --provision' if desired provisionable labels
do not match the existing ones. Otherwise, reset the DUT via
'autosev --reset'`,
	CommandRun: func() subcommands.CommandRun {
		c := &prejobRun{}
		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.phosphorus.PrejobRequest")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to write JSON encoded test_platform.phosphorus.PrejobResponse to")
		return c
	},
}

type prejobRun struct {
	commonRun
}

func (c *prejobRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.validateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return 1
	}

	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, args, env); err != nil {
		logApplicationError(ctx, a, err)
		return 1
	}
	return 0
}

const (
	// TODO(pprabhu): Find a configurable way for drone to provide us the port
	// number.
	tlsPort = 7152
)

func (c *prejobRun) innerRun(ctx context.Context, args []string, env subcommands.Env) error {
	var r phosphorus.PrejobRequest
	if err := readJSONPb(c.inputPath, &r); err != nil {
		return err
	}
	if err := validatePrejobRequest(r); err != nil {
		return err
	}

	if d := google.TimeFromProto(r.Deadline); !d.IsZero() {
		var c context.CancelFunc
		log.Printf("Running with deadline %s (current time: %s)", d, time.Now().UTC())
		ctx, c = context.WithDeadline(ctx, d)
		defer c()
	}

	if r.UseTls {
		resp, err := runTLSProvision(ctx, r, tlsConfig{
			Port: tlsPort,
		})
		if err != nil {
			return err
		}
		return writeJSONPb(c.outputPath, resp)
	}

	resp, err := runPrejobLegacy(ctx, r)
	if err != nil {
		return err
	}
	return writeJSONPb(c.outputPath, resp)
}

// This function will be obsoleted once runTLSProvision is globally enabled.
func runPrejobLegacy(ctx context.Context, r phosphorus.PrejobRequest) (*phosphorus.PrejobResponse, error) {
	var ar *atutil.Result
	var err error
	if contains(r.ExistingProvisionableLabels, r.DesiredProvisionableLabels) {
		ar, err = runResetLegacy(ctx, r)
	} else {
		ar, err = runProvisionLegacy(ctx, r)
	}

	if err != nil {
		return nil, err
	}
	switch {
	case ar.Success():
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_SUCCEEDED}, nil
	case ar.RunResult.Aborted:
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_ABORTED}, nil
	default:
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_FAILED}, nil
	}
}

func validatePrejobRequest(r phosphorus.PrejobRequest) error {
	missingArgs := getCommonMissingArgs(r.Config)

	if r.DutHostname == "" {
		missingArgs = append(missingArgs, "DUT hostname")
	}

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}

// contains tests whether map y is contained within map x.
func contains(x, y map[string]string) bool {
	for k, v := range y {
		if x[k] != v {
			return false
		}
	}
	return true
}

// runProvisionLegacy provisions a single host. It is a wrapper around
// `autoserv --provision`. It cannot modify its point arguments.
//
// This function will be obsoleted once runTLSProvision is globally enabled.
func runProvisionLegacy(ctx context.Context, r phosphorus.PrejobRequest) (*atutil.Result, error) {
	j := getMainJob(r.Config)
	var labels []string
	for k, v := range r.DesiredProvisionableLabels {
		labels = append(labels, k+":"+v)
	}
	subDir := fmt.Sprintf("provision_%s", r.DutHostname)
	fullPath := filepath.Join(r.Config.Task.ResultsDir, subDir)
	p := &atutil.Provision{
		Host:       r.DutHostname,
		Labels:     labels,
		ResultsDir: fullPath,
	}
	ar, err := atutil.RunAutoserv(ctx, j, p, os.Stdout)
	if err != nil {
		return nil, errors.Annotate(err, "run provision").Err()
	}
	return ar, nil
}

// runResetLegacy resets a single host. It is a wrapper around
// `autoserv --reset`.
//
// This function will be obsoleted once runTLSProvision is globally enabled.
func runResetLegacy(ctx context.Context, r phosphorus.PrejobRequest) (*atutil.Result, error) {
	j := getMainJob(r.Config)
	subDir := fmt.Sprintf("prejob_%s", r.DutHostname)
	fullPath := filepath.Join(r.Config.Task.ResultsDir, subDir)
	a := &atutil.AdminTask{
		Host:       r.DutHostname,
		ResultsDir: fullPath,
		Type:       atutil.Reset,
	}
	ar, err := atutil.RunAutoserv(ctx, j, a, os.Stdout)
	if err != nil {
		return nil, errors.Annotate(err, "run reset").Err()
	}
	return ar, nil
}

type tlsConfig struct {
	Port int
}

// runTLSProvision provisions a DUT via the TLS API.
// See go/cros-tls go/cros-prover
//
// Errors returned from the actual provision operation are interpreted into
// the response. An error is returned for failure modes that can not be mapped
// to a response.
func runTLSProvision(ctx context.Context, r phosphorus.PrejobRequest, tc tlsConfig) (*phosphorus.PrejobResponse, error) {
	p, err := gsPathToImage(r.DesiredProvisionableLabels)
	if err != nil {
		return nil, errors.Annotate(err, "run TLS Provision").Err()
	}
	req := tls.ProvisionDutRequest{
		Name: r.DutHostname,
		Image: &tls.ProvisionDutRequest_ChromeOSImage{
			PathOneof: &tls.ProvisionDutRequest_ChromeOSImage_GsPathPrefix{
				GsPathPrefix: p,
			},
		},
	}

	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", tc.Port), grpc.WithInsecure())
	if err != nil {
		return nil, errors.Annotate(err, "run TLS Provision").Err()
	}
	defer conn.Close()

	c := tls.NewCommonClient(conn)

	op, err := c.ProvisionDut(ctx, &req)
	if err != nil {
		// Errors here indicate a failure in starting the operation, not failure
		// in the provision attempt.
		return nil, errors.Annotate(err, "run TLS Provision").Err()
	}

	op, err = lro.Wait(ctx, longrunning.NewOperationsClient(conn), op.GetName())
	if err != nil {
		// TODO(pprabhu) Cancel operation.
		// - Create 60 second headroom before deadline for cancellation.
		// - Cancel operation and wait up to deadline for cancellation to complete.
		// - Return multi-error with failure to cancel, if cancellation fails.
		s, isGRPCErr := status.FromError(err)
		if err == context.DeadlineExceeded || (isGRPCErr && s.Code() == codes.InvalidArgument) {
			return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_ABORTED}, nil
		}
		return nil, errors.Annotate(err, "run TLS Provision").Err()
	}
	if s := op.GetError(); s != nil {
		// Error here is a failure in the provision attempt.
		// TODO(pprabhu) Surface detailed errors up.
		// See https://docs.google.com/document/d/12w5pPntorUY1cgDHHxT3Nu6wdhVox288g5_BnyKCPOE/edit#heading=h.fj6zbs6kop08
		logging.Errorf(ctx, "Provision failed: (code: %s, message: %s, details: %s", s.Code, s.Message, s.Details)
		return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_FAILED}, nil
	}
	return &phosphorus.PrejobResponse{State: phosphorus.PrejobResponse_SUCCEEDED}, nil
}

const (
	osVersionKey         = "cros-version"
	gsImageArchivePrefix = "gs://chromeos-image-archive"
)

// This computation of the GS archive location from OS image version is a hack.
// Historical note: This computation used to happen inside autoserv before the
// introduction of a TLS service. The full path to the image location should be
// hoisted up to the clients of test platform. This hack is a step in the
// direction of hoisting the computation up through the stack.
func gsPathToImage(labels map[string]string) (string, error) {
	for k, v := range labels {
		if k == osVersionKey {
			return fmt.Sprintf("%s/%s", gsImageArchivePrefix, v), nil
		}
	}
	return "", errors.Reason("failed to determine GS location for image").Err()
}
