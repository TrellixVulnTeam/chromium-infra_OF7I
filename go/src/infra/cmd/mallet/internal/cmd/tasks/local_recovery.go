// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc/metadata"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	commonFlags "infra/cmd/mallet/internal/cmd/cmdlib"
	"infra/cmd/mallet/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/cros/recovery"
	rlogger "infra/cros/recovery/logger"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
)

// LocalRecovery subcommand: Running verify/recovery against the DUT from local environment.
var LocalRecovery = &subcommands.Command{
	UsageLine: "local-recovery UNIT_NAME",
	ShortDesc: "run recovery from local env.",
	LongDesc: `Run recovery against a DUT from local environment.

For now only running in testing mode.`,
	CommandRun: func() subcommands.CommandRun {
		c := &localRecoveryRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.CommonFlags.Register(&c.Flags)
		c.envFlags.Register(&c.Flags)
		// TODO(otabek@) Add more details with instruction how to get default config as example.
		c.Flags.StringVar(&c.configFile, "config", "", "Path to the custom json config file.")
		c.Flags.BoolVar(&c.onlyVerify, "only-verify", false, "Block recovery actions and run only verifiers.")
		c.Flags.BoolVar(&c.updateInventory, "update-inv", false, "Update UFS at the end execution.")
		return c
	},
}

type localRecoveryRun struct {
	subcommands.CommandRunBase
	commonFlags.CommonFlags
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	configFile      string
	onlyVerify      bool
	updateInventory bool
}

// Run initiates execution of local recovery.
func (c *localRecoveryRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// innerRun executes internal logic of control.
func (c *localRecoveryRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var unit string
	if len(args) > 1 {
		return errors.New("does not support more than one unit per request")
	} else if len(args) == 1 {
		unit = args[0]
	}
	if unit == "" {
		return errors.New("unit is not specified")
	}
	ctx := cli.GetContext(a, c, env)
	ctx, logger := createLogger(ctx)
	if c.Verbose() {
		ctx = logging.SetLevel(ctx, logging.Debug)
	}
	ctx = setupContextNamespace(ctx, ufsUtil.OSNamespace)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "local recovery: create http client").Err()
	}
	e := c.envFlags.Env()
	logger.Debug("Init TLW with inventory: %s and csa: %s sources", e.UFSService, e.AdminService)
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UFSService,
		Options: site.UFSPRPCOptions,
	})
	csac := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       hc,
			Host:    e.AdminService,
			Options: site.DefaultPRPCOptions,
		},
	)
	access, err := recovery.NewLocalTLWAccess(ic, csac)
	if err != nil {
		return errors.Annotate(err, "local recovery: create tlw access").Err()
	}
	defer access.Close()
	in := &recovery.RunArgs{
		UnitName:              unit,
		Access:                access,
		Logger:                logger,
		EnableRecovery:        c.onlyVerify,
		EnableUpdateInventory: c.updateInventory,
	}
	if c.configFile != "" {
		in.ConfigReader, err = os.Open(c.configFile)
		if err != nil {
			return errors.Annotate(err, "local recovery: open config file").Err()
		}
	}
	if err = recovery.Run(ctx, in); err != nil {
		return errors.Annotate(err, "local recovery").Err()
	}
	logger.Info("Task on %q has completed successfully", unit)
	return nil
}

// setupContextNamespace sets namespace to the context for UFS client.
func setupContextNamespace(ctx context.Context, namespace string) context.Context {
	md := metadata.Pairs(ufsUtil.Namespace, namespace)
	return metadata.NewOutgoingContext(ctx, md)
}

// StdFormat is formatting for logger.
const StdFormat = `[%{level:.1s}%{time:2006-01-02T15:04:05:00} ` +
	` %{shortfile:20s}] %{message}`

// Create custom logger config with custom formatter.
func createLogger(ctx context.Context) (context.Context, *recoveryLogger) {
	// Creating cutsom logger config to
	logconfig := &gologger.LoggerConfig{
		Out:    os.Stderr,
		Format: StdFormat,
	}
	ctx = logconfig.Use(ctx)
	return ctx, &recoveryLogger{
		log:       logging.Get(ctx),
		callDepth: 2,
	}
}

// recoveryLogger represents local recovery logger implementation.
type recoveryLogger struct {
	log logging.Logger
	// Logger indentation for messages.
	indentation int
	// callDepth sets desired stack depth (code line at which logging message is reported).
	callDepth int
}

// Debug log message at Debug level.
func (l *recoveryLogger) Debug(format string, args ...interface{}) {
	l.log.LogCall(logging.Debug, l.callDepth, l.indentString(format), args)
}

// Info is like Debug, but logs at Info level.
func (l *recoveryLogger) Info(format string, args ...interface{}) {
	l.log.LogCall(logging.Info, l.callDepth, l.indentString(format), args)
}

// Warning is like Debug, but logs at Warning level.
func (l *recoveryLogger) Warning(format string, args ...interface{}) {
	l.log.LogCall(logging.Warning, l.callDepth, l.indentString(format), args)
}

// Error is like Debug, but logs at Error level.
func (l *recoveryLogger) Error(format string, args ...interface{}) {
	l.log.LogCall(logging.Error, l.callDepth, l.indentString(format), args)
}

// IndentLogging increment indentation for logger.
func (l *recoveryLogger) IndentLogging() {
	l.indentation += 1
}

// DedentLogging decrement indentation for logger.
func (l *recoveryLogger) DedentLogging() {
	if l.indentation > 0 {
		l.indentation -= 1
	}
}

// Apply indent to the string.
func (l *recoveryLogger) indentString(v string) string {
	indent := rlogger.GetIndent(l.indentation, "  ")
	return indent + v
}
