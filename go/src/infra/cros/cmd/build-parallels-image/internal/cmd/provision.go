// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"time"

	"infra/libs/lro"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
)

const (
	// parallelsDLCID is the ID of the DLC containing Parallels.
	parallelsDLCID = "pita"
	// overallTimeout is the overall timeout to apply to the provision operation.
	overallTimeout = 20 * time.Minute
)

// Provision subcommand: Provision a DUT.
var Provision = &subcommands.Command{
	UsageLine: "provision -dut-name DUT_NAME -image-gs-path \"gs://chromeos-image-archive/eve-release/R86-13380.0.0\"",
	ShortDesc: "Provisions a DUT.",
	LongDesc:  `Provisions a DUT with the requested Chrome OS build.`,
	CommandRun: func() subcommands.CommandRun {
		c := &provisionRun{}
		c.Flags.StringVar(&c.dutName, "dut-name", "", "The resource name for the DUT")
		c.Flags.StringVar(&c.imageGSPath, "image-gs-path", "", "The Google Storage path (prefix) where images are located. For example, 'gs://chromeos-image-archive/eve-release/R86-13380.0.0'.")
		c.Flags.IntVar(&c.tlsPort, "tls-port", 7152, "The port on which to connect to TLS")
		return c
	},
}

type provisionRun struct {
	subcommands.CommandRunBase

	authFlags authcli.Flags

	tlsPort     int
	dutName     string
	imageGSPath string
}

func (c *provisionRun) validateArgs() error {
	if c.dutName == "" {
		return errors.New("-dut-name is not specified")
	}
	if c.imageGSPath == "" {
		return errors.New("-image-gs-path is not specified")
	}
	if c.tlsPort < 1 || c.tlsPort > 65535 {
		return fmt.Errorf("-tls-port is invalid, got %v, want 1-65535", c.tlsPort)
	}
	return nil
}

func (c *provisionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.validateArgs(); err != nil {
		errors.Log(ctx, err)
		c.Flags.Usage()
		return 1
	}

	provisionCtx, cancel := context.WithTimeout(ctx, overallTimeout)
	defer cancel()
	if err := runTLSProvision(provisionCtx, c.dutName, c.imageGSPath, c.tlsPort); err != nil {
		errors.Log(ctx, err)
		return 2
	}

	return 0
}

// runTLSProvision provisions a DUT via the TLS API.
// See go/cros-tls go/cros-prover
func runTLSProvision(ctx context.Context, dutName, imageGSPath string, tlsPort int) error {
	// Allocate 1 minute of the overall timeout to cancelling the provision
	// operation if something goes wrong. Keep the original context around
	// so that we can use it when cancelling.
	cancelCtx := ctx
	if deadline, ok := ctx.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline.Add(-1*time.Minute))
		defer cancel()
	}

	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", tlsPort), grpc.WithInsecure())
	if err != nil {
		return errors.Annotate(err, "dial TLS").Err()
	}
	defer conn.Close()

	cc := tls.NewCommonClient(conn)

	op, err := cc.ProvisionDut(ctx, &tls.ProvisionDutRequest{
		Name: dutName,
		Image: &tls.ProvisionDutRequest_ChromeOSImage{
			PathOneof: &tls.ProvisionDutRequest_ChromeOSImage_GsPathPrefix{
				GsPathPrefix: imageGSPath,
			},
		},
		DlcSpecs: []*tls.ProvisionDutRequest_DLCSpec{
			{Id: parallelsDLCID},
		},
	})
	if err != nil {
		// Errors here indicate a failure in starting the operation, not failure
		// in the provision attempt.
		return errors.Annotate(err, "start TLS Provision").Err()
	}

	oc := longrunning.NewOperationsClient(conn)
	op, err = lro.Wait(ctx, oc, op.GetName())
	if err != nil {
		werr := errors.Annotate(err, "run TLS Provision").Err()
		if _, err := oc.CancelOperation(cancelCtx, &longrunning.CancelOperationRequest{
			Name: op.GetName(),
		}); err != nil {
			return errors.NewMultiError(werr, errors.Annotate(err, "cancel TLS Provision").Err())
		}
		return werr
	}
	if s := op.GetError(); s != nil {
		// Error here is a failure in the provision attempt.
		return fmt.Errorf("TLS Provision failed: (code: %d, message: %s, details: %s)", s.Code, s.Message, s.Details)
	}
	return nil
}
