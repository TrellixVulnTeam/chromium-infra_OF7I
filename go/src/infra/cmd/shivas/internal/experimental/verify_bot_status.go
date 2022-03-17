// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package experimental

import (
	"fmt"
	"strings"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth/client/authcli"
	swarmingAPI "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	"infra/cros/dutstate"

	"infra/cmd/shivas/internal/ufs/subcmds/host"
	"infra/cmd/shivas/internal/ufs/subcmds/vm"
	sw "infra/libs/skylab/swarming"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AuditDutsCmd contains audit-duts command specification
var VerifyBotStatusCmd = &subcommands.Command{
	UsageLine: "verify-bot-status",
	ShortDesc: "Verify bots' status between swarming and UFS",
	LongDesc: `Verify bots' status between swarming and UFS.
	./shivas verify-bot-status -swarming-host a -swarming-host b ...
	Compare all bots in a swarming instance with their status in UFS.`,
	CommandRun: func() subcommands.CommandRun {
		c := &verifyBotStatusRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.Var(flag.StringSlice(&c.swarmingHosts), "swarming-host", "Name(s) of the swarming hosts to query")
		return c
	},
}

type verifyBotStatusRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	swarmingHosts []string
}

var defaultSwarmingServices = []string{
	"chrome-swarming.appspot.com",
	"chromium-swarm.appspot.com",
	"chromium-swarm-dev.appspot.com",
	"omnibot-swarming-server.appspot.com",
}

const defaultNamespace = "browser"

// Run the verify-bot-status cmd
func (c *verifyBotStatusRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *verifyBotStatusRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) (err error) {
	ctx := cli.GetContext(a, c, env)
	ns, err := c.envFlags.Namespace()
	if err != nil {
		return err
	}

	if len(c.swarmingHosts) == 0 {
		fmt.Printf("Use default swarming services in %s namespace\n", defaultNamespace)
		c.swarmingHosts = defaultSwarmingServices
		ns = defaultNamespace
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
	hostToStateMap := make(map[string]ufspb.State)
	res, err := utils.BatchList(ctx, ic, host.ListHosts, make([]string, 0), 0, false, false)
	for _, r := range res {
		lse := r.(*ufspb.MachineLSE)
		hostToStateMap[ufsUtil.RemovePrefix(lse.Name)] = lse.ResourceState
	}
	resVMs, err := utils.BatchList(ctx, ic, vm.ListVMs, make([]string, 0), 0, false, false)
	for _, r := range resVMs {
		vm := r.(*ufspb.VM)
		hostToStateMap[ufsUtil.RemovePrefix(vm.Name)] = vm.ResourceState
	}

	for _, swarmingService := range c.swarmingHosts {
		fmt.Println("\nQuerying ", swarmingService)
		swarmingClient, err := sw.NewClient(hc, fmt.Sprintf("https://%s/", swarmingService))
		if err != nil {
			return errors.Annotate(err, "failed to create swarming client").Err()
		}
		// Only query golo bots
		dims := []*swarmingAPI.SwarmingRpcsStringPair{
			{
				Key:   "gce",
				Value: "0",
			},
		}
		bots, err := swarmingClient.GetBots(ctx, dims)
		if err != nil {
			return err
		}
		for _, b := range bots {
			if b.IsDead || b.Quarantined || excludeBot(b) {
				continue
			}
			if dutstate.ConvertFromUFSState(hostToStateMap[b.BotId]).String() != "ready" {
				fmt.Println(b.BotId, hostToStateMap[b.BotId])
			}
		}
	}

	return nil
}

var excludeBotIDSubstr = []string{"flutter-", "skia-"}

func excludeBot(b *swarmingAPI.SwarmingRpcsBotInfo) bool {
	for _, s := range excludeBotIDSubstr {
		if strings.HasPrefix(b.BotId, s) {
			return true
		}
	}
	var hasZone bool
	for _, d := range b.Dimensions {
		if d.Key == "os" && len(d.Value) > 0 {
			switch d.Value[0] {
			// Exclude android, ChromeOS, Fuchsia bots
			case "Android", "ChromeOS", "Fuchsia":
				return true
			}
		}
		// Exclude docker bots
		if d.Key == "inside_docker" && len(d.Value) > 0 {
			return true
		}
		if d.Key == "zone" {
			hasZone = true
			for _, v := range d.Value {
				if v == "us-atl" || v == "us-iad" || v == "us-mtv" {
					return false
				}
			}
			return true
		}
	}
	if !hasZone {
		return true
	}
	return false
}
