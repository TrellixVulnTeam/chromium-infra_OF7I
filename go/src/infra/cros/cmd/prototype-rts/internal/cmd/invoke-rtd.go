package cmd

import (
	"fmt"
	"time"

	"infra/cros/cmd/prototype-rts/internal/rtd"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// InvokeRTD starts an RTD container and executes Invocations against it.
func InvokeRTD() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "invoke-rtd",
		ShortDesc: "Starts a new RTD container then invokes RTD commands against it.",
		LongDesc:  "Starts a new RTD container then invokes RTD commands against it. RTS services must already be running locally as a prerequisite.",
		CommandRun: func() subcommands.CommandRun {
			c := &invokeCmd{}
			c.InitRTSFlags()
			c.Flags.StringVar(&c.rtdCommand, "rtd-command", "", "The executable that will run the RTD, e.g. \"tast\"")
			c.Flags.StringVar(&c.imageURI, "image-uri", "", "URI for RTD image, e.g. gcr.io/chromeos-rtd-dev/sean-test")
			return c
		},
	}
}

type invokeCmd struct {
	flags

	rtdCommand string
	imageURI   string
}

func (inv *invokeCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, inv, env)
	if err := inv.innerRun(ctx); err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

func (inv *invokeCmd) innerRun(ctx context.Context) error {
	// Test that the requisite services are running locally.
	dialContext, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	if _, err := grpc.DialContext(dialContext, fmt.Sprintf(":%d", inv.progressSinkPort), grpc.WithInsecure(), grpc.WithBlock()); err != nil {
		return errors.Annotate(err, "failed to connect to ProgressSinkService on port %d", inv.progressSinkPort).Err()
	}
	if _, err := grpc.DialContext(dialContext, fmt.Sprintf(":%d", inv.tlsCommonPort), grpc.WithInsecure(), grpc.WithBlock()); err != nil {
		return errors.Annotate(err, "failed to connect to TlsCommonService on port %d", inv.tlsCommonPort).Err()
	}
	logging.Infof(ctx, "Validated that gRPC servers are running for ProgressSinkService and TlsService")

	orch := rtd.Orchestrator{}
	if err := orch.StartRTDContainer(ctx, inv.imageURI); err != nil {
		return errors.Annotate(err, "failed StartRTDContainer").Err()
	}
	if err := orch.Invoke(ctx, int32(inv.progressSinkPort), int32(inv.tlsCommonPort), inv.rtdCommand); err != nil {
		return errors.Annotate(err, "failed Invoke").Err()
	}
	if err := orch.StopRTDContainer(ctx); err != nil {
		return errors.Annotate(err, "failed StopRTDContainer").Err()
	}
	return nil
}
