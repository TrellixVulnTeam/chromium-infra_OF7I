package cmd

import (
	"context"

	"infra/cros/cmd/prototype-rts/internal/rtd"
	"infra/cros/cmd/prototype-rts/internal/service"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
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
	// Run the services and retrieve the actual ports they start on.
	_, progressSinkPort, err := service.LaunchProgressSink(ctx, int32(inv.progressSinkPort))
	if err != nil {
		return errors.Annotate(err, "progress sink").Err()
	}
	_, tlsCommonPort, err := service.LaunchTLSCommon(ctx, int32(inv.tlsCommonPort))
	if err != nil {
		return errors.Annotate(err, "tls common").Err()
	}
	// TODO: Also launch TLW or prototype-tlw
	logging.Infof(ctx, "Services are running. Invoking RTD.")

	o := rtd.Orchestrator{}
	if err := o.StartRTDContainer(ctx, inv.imageURI); err != nil {
		return errors.Annotate(err, "failed StartRTDContainer").Err()
	}
	if err := o.Invoke(ctx, int32(progressSinkPort), int32(tlsCommonPort), inv.rtdCommand); err != nil {
		return errors.Annotate(err, "failed Invoke").Err()
	}
	if err := o.StopRTDContainer(ctx); err != nil {
		return errors.Annotate(err, "failed StopRTDContainer").Err()
	}
	return nil
}
