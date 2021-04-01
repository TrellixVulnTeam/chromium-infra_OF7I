// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package parallels

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"infra/cros/cmd/phosphorus/internal/cmd"
	"infra/libs/lro"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	bpipb "go.chromium.org/chromiumos/infra/proto/go/uprev/build_parallels_image"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
)

const (
	// parallelsDLCID is the ID of the DLC containing Parallels.
	parallelsDLCID = "pita"

	// tlsPort is the port to use for connecting to TLS.
	tlsPort = 7152
)

// Provision subcommand: Provision a DUT, including with Parallels DLC.
var Provision = &subcommands.Command{
	UsageLine: "build-parallels-image-provision -input_json /path/to/input.json",
	ShortDesc: "Provisions a DUT.",
	LongDesc:  `Provisions a DUT with the requested Chrome OS build and Parallels DLC. For use in build_parallels_image.`,
	CommandRun: func() subcommands.CommandRun {
		c := &provisionRun{}
		c.Flags.StringVar(&c.InputPath, "input_json", "", "Path that contains JSON encoded engprod.build_parallels_image.ProvisionRequest")
		return c
	},
}

type provisionRun struct {
	cmd.CommonRun
}

func (c *provisionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.ValidateArgs(); err != nil {
		errors.Log(ctx, err)
		c.Flags.Usage()
		return 1
	}

	if err := c.innerRun(ctx, env); err != nil {
		errors.Log(ctx, err)
		return 2
	}
	return 0
}

func (c *provisionRun) innerRun(ctx context.Context, env subcommands.Env) error {
	r := &bpipb.ProvisionRequest{}
	if err := cmd.ReadJSONPB(c.InputPath, r); err != nil {
		return err
	}
	if err := validateProvisionRequest(r); err != nil {
		return err
	}

	if r.Deadline.IsValid() {
		var c context.CancelFunc
		d := r.Deadline.AsTime()
		log.Printf("Running with deadline %s (current time: %s)", d, time.Now().UTC())
		ctx, c = context.WithDeadline(ctx, d)
		defer c()
	}

	return runTLSProvision(ctx, r.DutName, r.ImageGsPath)
}

func validateProvisionRequest(r *bpipb.ProvisionRequest) error {
	missingArgs := validateConfig(r.GetConfig())

	if r.DutName == "" {
		missingArgs = append(missingArgs, "DUT hostname")
	}
	if r.ImageGsPath == "" {
		missingArgs = append(missingArgs, "GS image path")
	}
	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}
	return nil
}

// runTLSProvision provisions a DUT via the TLS API.
// See go/cros-tls go/cros-prover
func runTLSProvision(ctx context.Context, dutName, imageGSPath string) error {
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
