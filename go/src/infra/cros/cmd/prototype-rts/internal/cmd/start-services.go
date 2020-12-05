package cmd

import (
	"infra/cros/cmd/prototype-rts/internal/service"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/logging"
)

// StartServices starts the RTS gRPC services.
func StartServices() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "start-services",
		ShortDesc: "Starts the RTS services needed for test invocation.",
		LongDesc:  "Starts the RTS services needed for test invocation.",
		CommandRun: func() subcommands.CommandRun {
			c := &servicesCommand{}
			c.InitRTSFlags()
			return c
		},
	}
}

type servicesCommand struct {
	flags
}

func (sc *servicesCommand) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, sc, env)
	_, _, err := service.LaunchProgressSink(ctx, int32(sc.progressSinkPort))
	if err != nil {
		logging.Errorf(ctx, "ProgressSink: %v", err)
		return 1
	}
	_, _, err = service.LaunchTLSCommon(ctx, int32(sc.tlsCommonPort))
	if err != nil {
		logging.Errorf(ctx, "TLSCommon: %v", err)
		return 1
	}
	// TODO: Also launch TLW or prototype-tlw

	// Wait forever
	// TODO: Handle SIGTERM/SIGINT and stop processes and exit.
	logging.Infof(ctx, "Services are running. You can run an Invocation now.")
	select {}
}
