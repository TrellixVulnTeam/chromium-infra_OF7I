// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc/metadata"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	commonFlags "infra/cmd/mallet/internal/cmd/cmdlib"
	"infra/cmd/mallet/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/cros/cmd/labpack/logger"
	kclient "infra/cros/karte/client"
	"infra/cros/recovery"
	"infra/cros/recovery/karte"
	"infra/cros/recovery/logger/metrics"
	"infra/cros/recovery/tasknames"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
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
		c.Flags.StringVar(&c.karteServer, "karte-server", "", "Use karte metric to record the action.")

		c.Flags.StringVar(&c.devJumpHost, "dev-jump-host", "", "Jump host for SSH (Dev-only feature).")
		c.Flags.StringVar(&c.logRoot, "log-root", "", "Path to the custom json config file.")
		c.Flags.BoolVar(&c.generateLogFiles, "generate-log-files", false, "Generate log files. Default is no.")

		c.Flags.BoolVar(&c.onlyVerify, "only-verify", false, "Block recovery actions and run only verifiers. Default is no.")
		c.Flags.BoolVar(&c.updateInventory, "update-inv", false, "Update UFS at the end execution. Default is no.")
		c.Flags.BoolVar(&c.deployTask, "deploy", false, "Trigger deploy task. Default is no.")
		c.Flags.BoolVar(&c.showSteps, "steps", false, "Show generated steps. Default is no.")
		return c
	},
}

type localRecoveryRun struct {
	subcommands.CommandRunBase
	commonFlags.CommonFlags
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	devJumpHost      string
	logRoot          string
	configFile       string
	karteServer      string
	onlyVerify       bool
	updateInventory  bool
	deployTask       bool
	showSteps        bool
	generateLogFiles bool
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
	tn := tasknames.Recovery
	if c.deployTask {
		tn = tasknames.Deploy
	}
	ctx := cli.GetContext(a, c, env)

	// React to user cancel.
	ctx, cancel := context.WithCancel(ctx)
	cs := make(chan os.Signal, 1)
	signal.Notify(cs, os.Interrupt)
	defer func() {
		signal.Stop(cs)
		cancel()
	}()
	go func() {
		select {
		case <-cs:
			cancel()
		case <-ctx.Done():
		}
	}()
	logRoot, err := c.getLogRoot()
	if err != nil {
		return errors.Annotate(err, "local recovery").Err()
	}
	// Created logger and log files.
	logger, err := c.createLogger(ctx, logRoot)
	if err != nil {
		return errors.Annotate(err, "local recovery: create logger").Err()
	}
	defer logger.Close()
	ctx = setupContextNamespace(ctx, ufsUtil.OSNamespace)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "local recovery: create http client").Err()
	}
	e := c.envFlags.Env()
	logger.Debugf("Init TLW with inventory: %s and csa: %s sources", e.UFSService, e.AdminService)
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
	defer access.Close(ctx)

	var metrics metrics.Metrics
	if c.karteServer != "" {
		var err error
		authOptions, err := c.authFlags.Options()
		if err != nil {
			return errors.Annotate(err, "create action").Err()
		}
		if c.karteServer == "dev" {
			metrics, err = karte.NewMetrics(ctx, kclient.DevConfig(authOptions))
		} else if c.karteServer == "prod" {
			metrics, err = karte.NewMetrics(ctx, kclient.ProdConfig(authOptions))
		} else if c.karteServer == "local" {
			metrics, err = karte.NewMetrics(ctx, kclient.LocalConfig(authOptions))
		} else {
			metrics, err = karte.NewMetrics(ctx, kclient.EmptyConfig())
		}
		if err == nil {
			logger.Infof("internal run: metrics client successfully created.")
		} else {
			return errors.Annotate(err, "ineer run: failed to instantiate karte client of server: %q", c.karteServer).Err()
		}
	}

	in := &recovery.RunArgs{
		UnitName:              unit,
		Access:                access,
		Logger:                logger,
		EnableRecovery:        !c.onlyVerify,
		EnableUpdateInventory: c.updateInventory,
		ShowSteps:             c.showSteps,
		Metrics:               metrics,
		TaskName:              tn,
		LogRoot:               logRoot,
		DevJumpHost:           c.devJumpHost,
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
	logger.Infof("Task on %q has completed successfully", unit)
	return nil
}

func (c *localRecoveryRun) getLogRoot() (string, error) {
	// TODO(gregorynisbet): Clean up the logs, include the current timestamp, or generally
	// do something smarter than just setting the logs to "./logs".
	logRoot := c.logRoot
	if logRoot == "" {
		logRoot = "./logs"
	}
	logRoot, err := filepath.Abs(logRoot)
	if err != nil {
		return logRoot, errors.Annotate(err, "get log root").Err()
	}
	err = os.MkdirAll(logRoot, 755)
	return logRoot, errors.Annotate(err, "get log root").Err()
}

// setupContextNamespace sets namespace to the context for UFS client.
func setupContextNamespace(ctx context.Context, namespace string) context.Context {
	md := metadata.Pairs(ufsUtil.Namespace, namespace)
	return metadata.NewOutgoingContext(ctx, md)
}

// recoveryLogger represents local recovery logger implementation.
//
// Logger created to extend logger for indentation.
type recoveryLogger struct {
	log logger.Logger
	// Logger indentation for messages.
	indentation int32
}

// Create custom logger config with custom formatter.
func (c *localRecoveryRun) createLogger(ctx context.Context, logDir string) (*recoveryLogger, error) {
	const callDepth = 3
	stdLevel := logging.Info
	if c.Verbose() {
		stdLevel = logging.Debug
	}
	l := &recoveryLogger{}
	_, log, err := logger.NewLogger(ctx, callDepth, logDir, stdLevel, logger.DefaultFormat, c.generateLogFiles)
	l.log = log
	return l, errors.Annotate(err, "create logger").Err()
}

// Close log resources.
func (l *recoveryLogger) Close() {
	l.log.Close()
}

// Debugf log message at Debug level.
func (l *recoveryLogger) Debugf(format string, args ...interface{}) {
	l.log.Debugf(l.indentString(format), args...)
}

// Infof is like Debugf, but logs at Info level.
func (l *recoveryLogger) Infof(format string, args ...interface{}) {
	l.log.Infof(l.indentString(format), args...)
}

// Warningf is like Debugf, but logs at Warning level.
func (l *recoveryLogger) Warningf(format string, args ...interface{}) {
	l.log.Warningf(l.indentString(format), args...)
}

// Errorf is like Debug, but logs at Error level.
func (l *recoveryLogger) Errorf(format string, args ...interface{}) {
	l.log.Errorf(l.indentString(format), args...)
}

// Indent increment indentation for logger.
func (l *recoveryLogger) Indent() {
	atomic.AddInt32(&l.indentation, 1)
}

// Dedent decrement indentation for logger.
func (l *recoveryLogger) Dedent() {
	atomic.AddInt32(&l.indentation, -1)
}

// Apply indent to the string.
func (l *recoveryLogger) indentString(v string) string {
	i := atomic.LoadInt32(&l.indentation)
	if i <= 0 {
		return v
	}
	indent := strings.Repeat("  ", int(i))
	return indent + v
}
