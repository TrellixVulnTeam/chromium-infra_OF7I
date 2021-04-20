// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/autotest/hostinfo"
	inventoryclient "infra/libs/skylab/inventory/inventoryclient"
	ufsUtil "infra/unifiedfleet/app/util"
)

const defaultAutoservPath = "./server/autoserv"
const defaultControlDir = "/tr"
const filePermissions = 0o644

// LabpackCmd is a local repair job that writes its results to
// a user-specified directory organized under a control directory.
var LabpackCmd = &subcommands.Command{
	UsageLine: "labpack",
	ShortDesc: "Run labpack locally",
	LongDesc:  "Run labpack locally",
	CommandRun: func() subcommands.CommandRun {
		c := &repair{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.jobname, "jobname", "", "Local name for job")
		c.Flags.StringVar(&c.autoservDir, "autoservdir", "", "Path to autoserv")
		return c
	},
}

type repair struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	jobname     string
	autoservDir string
}

// Run prints errors and forwards control to innerRun.
func (c *repair) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// hostInfoStorePath gets the path that autoserv expects to be able to read
// a host info file from.
func hostInfoStorePath(controlDir string, hostname string) (string, error) {
	if controlDir == "" {
		return "", fmt.Errorf("controlDir cannot be empty")
	}
	if hostname == "" {
		return "", fmt.Errorf("hostname cannot be empty")
	}
	return fmt.Sprintf("%s/host_info_store/%s.store", controlDir, hostname), nil
}

func (c *repair) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}

	e := c.envFlags.Env()

	return doRepair(
		ctx,
		repairParams{
			hc:               hc,
			autoserv:         defaultAutoservPath,
			hostnames:        args,
			controlDir:       "/tr",
			adminService:     e.AdminService,
			inventoryService: e.InventoryService,
			autoservDir:      c.autoservDir,
			jobname:          c.jobname,
		},
	)
}

// repairParams is a parameter struct of all non-context parameters
// to a local repair job.
type repairParams struct {
	hc               *http.Client
	autoserv         string
	hostnames        []string
	controlDir       string
	adminService     string
	inventoryService string
	autoservDir      string
	jobname          string
}

// getHostname extracts the single hostname to work on from a params struct.
// In the future, when multiple simultaneous repairs are supported, it will go away.
func (r repairParams) getHostname() (string, error) {
	switch len(r.hostnames) {
	case 0:
		return "", fmt.Errorf("hostnames cannot be empty")
	case 1:
		return r.hostnames[0], nil
	}
	return "", fmt.Errorf("multiple hostnames not supported")
}

// validate a repairParams struct, perform shallow checks for reasonableness
// of params struct.
func (r repairParams) validate() error {
	if r.hc == nil {
		return fmt.Errorf("http client cannot be nil")
	}
	if r.autoserv == "" {
		return fmt.Errorf("autoserv cannot be empty")
	}
	if r.controlDir == "" {
		return fmt.Errorf("controlDir cannot be empty")
	}
	if r.adminService == "" {
		return fmt.Errorf("adminService cannot be empty")
	}
	if r.jobname == "" {
		return fmt.Errorf("jobname cannot be empty")
	}
	return nil
}

// doRepair kicks off a local repair task, writes its output to stderr,
// and synchronously waits for it to finish.
func doRepair(ctx context.Context, r repairParams) error {
	if err := r.validate(); err != nil {
		return err
	}

	hostname, err := r.getHostname()
	if err != nil {
		return err
	}

	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       r.hc,
			Host:    r.adminService,
			Options: site.DefaultPRPCOptions,
		},
	)

	invC := inventoryclient.NewInventoryClient(
		r.hc,
		r.inventoryService,
		nil,
	)

	g := hostinfo.NewGetter(invC, invWithSVClient)
	hiContents, err := g.GetContentsForHostname(ctx, hostname)
	if err != nil {
		return err
	}

	hiPath, err := hostInfoStorePath(r.controlDir, hostname)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(hiPath, []byte(hiContents), filePermissions)
	if err != nil {
		return err
	}

	args := []string{
		r.autoserv,
		"-s",
		"--host-info-subdir",
		fmt.Sprintf("%s/%s", r.controlDir, "host_info_store"),
		"-m",
		hostname,
		"--lab",
		"True",
		"--local-only-host-info",
		"True",
		"-R",
		"-r",
		fmt.Sprintf("%s/%s", r.controlDir, r.jobname),
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if r.autoservDir != "" {
		cmd.Dir = r.autoservDir
	}
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
