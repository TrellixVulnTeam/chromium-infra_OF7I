// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package lsedeployment

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetMachineLSEDeploymentCmd get machine lse deployment by given name.
var GetMachineLSEDeploymentCmd = &subcommands.Command{
	UsageLine: "host-deployment ...",
	ShortDesc: "Get deployment records by filters",
	LongDesc: `Get deployment records by filters.

Example:

shivas get host-deployment {serialNumber1} {serialNumber2}

shivas get host-deployment -host host1 -host host2 -deploymentID id1 -deploymentID id2

Gets the deployment records and prints the output in user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getMachineLSEDeployment{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.hosts), "host", "Name(s) of a hostname to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.deploymentIDs), "deploymentID", "Identifier(s) of a unique deployment ID to filter by. Can be specified multiple times.")
		return c
	},
}

type getMachineLSEDeployment struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags

	// Filters
	hosts         []string
	deploymentIDs []string

	pageSize int
	keysOnly bool
}

func (c *getMachineLSEDeployment) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getMachineLSEDeployment) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ns, err := c.envFlags.Namespace()
	if err != nil {
		return err
	}
	ctx = utils.SetupContext(ctx, ns)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
	full := utils.FullMode(c.outputFlags.Full())
	var res []proto.Message
	if len(args) > 0 {
		res = utils.ConcurrentGet(ctx, ic, args, c.getSingle)
	} else {
		res, err = utils.BatchList(ctx, ic, ListMachineLSEDeployments, c.formatFilters(), c.pageSize, c.keysOnly, full)
	}
	if err != nil {
		return err
	}
	return utils.PrintEntities(ctx, ic, res, utils.PrintMachineLSEDeploymentsJSON, printMachineLSEDeploymentFull, printMachineLSEDeploymentNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getMachineLSEDeployment) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetMachineLSEDeployment(ctx, &ufsAPI.GetMachineLSEDeploymentRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSEDeploymentCollection, name),
	})
}

func (c *getMachineLSEDeployment) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.HostFilterName, c.hosts)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.DeploymentIdentifierFilterName, c.deploymentIDs)...)
	return filters
}

// ListMachineLSEDeployments calls the list MachineLSEDeployments to get a list of host deployment records.
func ListMachineLSEDeployments(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListMachineLSEDeploymentsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListMachineLSEDeployments(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, len(res.GetMachineLseDeployments()))
	for i, dr := range res.GetMachineLseDeployments() {
		protos[i] = dr
	}
	return protos, res.GetNextPageToken(), nil
}

func printMachineLSEDeploymentFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	return printMachineLSEDeploymentNormal(msgs, tsv, false)
}

func printMachineLSEDeploymentNormal(msgs []proto.Message, tsv, keysOnly bool) error {
	if tsv {
		utils.PrintTSVMachineLSEDeployments(msgs, keysOnly)
		return nil
	}
	utils.PrintTableTitle(utils.MachineLSEDeploymentTitle, tsv, keysOnly)
	utils.PrintMachineLSEDeployments(msgs, keysOnly)
	return nil
}
