// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package f20

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"

	"infra/cmd/mallet/internal/site"
	"infra/cmdsupport/cmdlib"
)

// Tlw subcommand: testing tlw.
var Tlw = &subcommands.Command{
	UsageLine: "tlw",
	ShortDesc: "Verify LTW action on local run",
	LongDesc:  "Verify LTW action on local run",
	CommandRun: func() subcommands.CommandRun {
		c := &tlwCheckRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.Flags.StringVar(&c.tlwPath, "tlw-path", "127.0.0.1:7151", "The TLW access point.")
		return c
	},
}

type tlwCheckRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	tlwPath string
}

func (c *tlwCheckRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *tlwCheckRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)

	var unit string
	if len(args) == 0 {
		return errors.New("Repair target is not specified")
	} else {
		unit = args[0]
	}

	// Only for local testing
	conn, err := grpc.Dial(c.tlwPath, grpc.WithInsecure())
	if err != nil {
		return err
	}
	cl := tls.NewWiringClient(conn)

	// Open port for the DUT
	reqAdd := &tls.OpenDutPortRequest{
		Name: unit,
		Port: int32(22),
	}
	add, err := cl.OpenDutPort(ctx, reqAdd)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "OpenDutPort %s: %s\n", unit, add)

	// open port
	reqOp := &tls.ExposePortToDutRequest{
		DutName:            unit,
		LocalPort:          22,
		RequireRemoteProxy: false,
	}
	p, err := cl.ExposePortToDut(ctx, reqOp)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "ExposePortToDut %s: %s\n", unit, p)
	// Example how to connect to the port.
	// connStr := fmt.Sprintf("%s:%s", p.ExposedAddress, p.ExposedPort)
	// conn2, err := grpc.Dial(connStr, grpc.WithInsecure())
	// if err != nil {
	// 	return err
	// }

	// get DUT info
	req := &tls.GetDutRequest{Name: unit}
	op, err := cl.GetDut(ctx, req)
	if err != nil {
		return err
	}

	fmt.Fprintf(a.GetOut(), "GetDut %s: %s\n", unit, op)
	return nil
}
