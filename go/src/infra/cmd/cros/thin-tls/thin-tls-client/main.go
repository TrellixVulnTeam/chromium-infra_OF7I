// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command thin-tls-client is a client for thin-tls.
// This is only to be used for very basic experimentation.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"infra/cmd/cros/thin-tls/api"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging/gologger"
	"google.golang.org/grpc"
)

func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(getApplication(), nil))
}

func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "thin-tls-client",
		Title: "Experimental client for thin-tls",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			cmdShell,
		},
	}
}

type commandRunFunc func()

var cmdShell = &subcommands.Command{
	UsageLine: "shell COMMAND",
	ShortDesc: "test DutShell RPC",
	LongDesc: `Test DutShell RPC.

This RPC runs a shell command on the DUT.`,
	CommandRun: func() subcommands.CommandRun {
		var r cmdShellRun
		r.Flags.StringVar(&r.address, "address", "localhost:50051", "Service address")
		return &r
	},
}

type cmdShellRun struct {
	subcommands.CommandRunBase
	address string
}

func (r *cmdShellRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	var command string
	switch len(args) {
	case 0:
		return logFatalf(a, "missing command argument")
	case 1:
		command = args[0]
	default:
		command = strings.Join(args, " ")
	}

	conn, err := grpc.Dial(r.address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return logFatalf(a, "did not connect: %v", err)
	}
	defer conn.Close()
	c := api.NewTlsClient(conn)

	ctx := context.Background()
	stream, err := c.DutShell(ctx, &api.DutShellRequest{
		Command: command,
	})
	if err != nil {
		return logFatalf(a, "could not greet: %v", err)
	}
	var status int32
	var exited bool
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			if !exited {
				return logFatalf(a, "received EOF when not exited")
			}
			logf(a, "command exit status: %d", status)
			return int(status)
		}
		if exited {
			return logFatalf(a, "received message after exit: %v", resp)
		}
		if err != nil {
			return logFatalf(a, "stream recv error: %v", err)
		}
		_, _ = a.GetOut().Write(resp.GetStdout())
		_, _ = a.GetErr().Write(resp.GetStderr())
		status = resp.GetStatus()
		exited = resp.GetExited()
	}
}

func logf(a subcommands.Application, format string, v ...interface{}) {
	if format[len(format)-1] != '\n' {
		format = format + "\n"
	}
	_, _ = fmt.Fprintf(a.GetErr(), format, v...)
}

// logFatalf is intended to be used like log.Fatalf while respecting
// the subcommands package.
// Should be called and returned immediately:
//
//  return logFatalf(a, "blah")
func logFatalf(a subcommands.Application, format string, v ...interface{}) int {
	logf(a, format, v)
	return 1
}
