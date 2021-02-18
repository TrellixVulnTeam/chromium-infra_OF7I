// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package asset

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// UpdateAssetCmd Update asset by given name.
var UpdateAssetCmd = &subcommands.Command{
	UsageLine: "asset [Options...]",
	ShortDesc: "Update a asset(Chromebook, Servo, Labstation)",
	LongDesc:  cmdhelp.UpdateAssetLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateAsset{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.AssetFileText)

		c.Flags.StringVar(&c.name, "name", "", "Asset tag of the asset")
		c.Flags.StringVar(&c.location, "location", "", "location of the asset in barcode format. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.zone, "zone", "", "Zone that the asset is in. "+cmdhelp.ZoneFilterHelpText)
		c.Flags.StringVar(&c.aisle, "aisle", "", "Aisle that the asset is in. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.row, "row", "", "Row that the asset is in. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.rack, "rack", "", "Rack name that the asset is in. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.racknumber, "racknumber", "", "Rack number that the asset is in. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.position, "position", "", "Position that the asset is in. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.shelf, "shelf", "", "Shelf that the asset is in. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.barcode, "barcode", "", "barcode of the asset. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.assetType, "type", "", "Type of asset. "+cmdhelp.AssetTypesHelpText)
		c.Flags.StringVar(&c.model, "model", "", "model of the asset. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.costcenter, "costcenter", "", "Cost center of the asset. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.gcn, "gcn", "", "Google code name of the asset. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.target, "target", "", "Build target of the asset. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.board, "board", "", "Reference board of the asset. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.mac, "mac", "", "Mac address of the asset. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.phase, "phase", "", "Phase of the asset. "+cmdhelp.ClearFieldHelpText)
		return c
	},
}

type updateAsset struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	name       string
	location   string
	zone       string
	aisle      string
	row        string
	shelf      string
	rack       string
	racknumber string
	position   string
	barcode    string
	assetType  string
	model      string
	costcenter string
	gcn        string
	target     string
	board      string
	mac        string
	phase      string
}

func (c *updateAsset) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateAsset) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UFS service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	var asset ufspb.Asset
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &asset); err != nil {
			return err
		}
		asset.Realm = ufsUtil.ToUFSRealm(asset.GetLocation().GetZone().String())
	} else {
		if err := c.parseArgs(&asset); err != nil {
			return err
		}
	}
	if err := utils.PrintExistingAsset(ctx, ic, asset.Name); err != nil {
		return err
	}
	asset.Name = ufsUtil.AddPrefix(ufsUtil.AssetCollection, asset.Name)
	res, err := ic.UpdateAsset(ctx, &ufsAPI.UpdateAssetRequest{
		Asset: &asset,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"location":   "location",
			"zone":       "location.zone",
			"aisle":      "location.aisle",
			"row":        "location.row",
			"shelf":      "location.shelf",
			"rack":       "location.rack",
			"racknumber": "location.rack_number",
			"position":   "location.position",
			"barcode":    "location.barcode_name",
			"type":       "type",
			"model":      "model",
			"costcenter": "info.cost_center",
			"gcn":        "info.google_code_name",
			"target":     "info.build_target",
			"board":      "info.reference_board",
			"mac":        "info.ethernet_mac_address",
			"phase":      "info.phase",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The asset after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully updated the asset %s\n", res.Name)
	return nil
}

func (c *updateAsset) parseArgs(asset *ufspb.Asset) error {
	var err error
	asset.Name = c.name
	asset.Info = &ufspb.AssetInfo{}
	if c.location == utils.ClearFieldValue || c.location == "" {
		asset.Location = &ufspb.Location{}
	} else {
		asset.Location, err = utils.GetLocation(c.location)
		if err != nil {
			return err
		}
		if asset.Location.Rack == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Invalid input, rack required but not found in location %s", c.location)
		}
		if asset.Location.Zone == ufspb.Zone_ZONE_UNSPECIFIED {
			return cmdlib.NewQuietUsageError(c.Flags, "Invalid zone in location %s", c.location)
		}
	}
	if c.zone == utils.ClearFieldValue {
		asset.GetLocation().Zone = ufsUtil.ToUFSZone("")
	} else {
		asset.GetLocation().Zone = ufsUtil.ToUFSZone(c.zone)
	}
	if c.aisle == utils.ClearFieldValue {
		asset.GetLocation().Aisle = ""
	} else {
		asset.GetLocation().Aisle = c.aisle
	}
	if c.row == utils.ClearFieldValue {
		asset.GetLocation().Row = ""
	} else {
		asset.GetLocation().Row = c.row
	}
	if c.shelf == utils.ClearFieldValue {
		asset.GetLocation().Shelf = ""
	} else {
		asset.GetLocation().Shelf = c.shelf
	}
	if c.rack == utils.ClearFieldValue {
		asset.GetLocation().Rack = ""
	} else {
		asset.GetLocation().Rack = c.rack
	}
	if c.racknumber == utils.ClearFieldValue {
		asset.GetLocation().RackNumber = ""
	} else {
		asset.GetLocation().RackNumber = c.racknumber
	}
	if c.position == utils.ClearFieldValue {
		asset.GetLocation().Position = ""
	} else {
		asset.GetLocation().Position = c.position
	}
	if c.barcode == utils.ClearFieldValue {
		asset.GetLocation().BarcodeName = ""
	} else {
		asset.GetLocation().BarcodeName = c.barcode
	}
	if c.assetType == utils.ClearFieldValue {
		asset.Type = ufsUtil.ToAssetType("")
	} else {
		asset.Type = ufsUtil.ToAssetType(c.assetType)
	}
	if c.model == utils.ClearFieldValue {
		asset.Model = ""
	} else {
		asset.Model = c.model
	}
	if c.costcenter == utils.ClearFieldValue {
		asset.Info.CostCenter = ""
	} else {
		asset.Info.CostCenter = c.costcenter
	}
	if c.gcn == utils.ClearFieldValue {
		asset.Info.GoogleCodeName = ""
	} else {
		asset.Info.GoogleCodeName = c.gcn
	}
	if c.target == utils.ClearFieldValue {
		asset.Info.BuildTarget = ""
	} else {
		asset.Info.BuildTarget = c.target
	}
	if c.board == utils.ClearFieldValue {
		asset.Info.ReferenceBoard = ""
	} else {
		asset.Info.ReferenceBoard = c.board
	}
	if c.mac == utils.ClearFieldValue {
		asset.Info.EthernetMacAddress = ""
	} else {
		asset.Info.EthernetMacAddress = c.mac
	}
	if c.phase == utils.ClearFieldValue {
		asset.Info.Phase = ""
	} else {
		asset.Info.Phase = c.phase
	}
	asset.Realm = ufsUtil.ToUFSRealm(asset.GetLocation().GetZone().String())
	return nil
}

func (c *updateAsset) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.name != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-name' cannot be specified at the same time.")
		}
		if c.location != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-location' cannot be specified at the same time.")
		}
		if c.zone != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-zone' cannot be specified at the same time.")
		}
		if c.aisle != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-aisle' cannot be specified at the same time.")
		}
		if c.row != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-row' cannot be specified at the same time.")
		}
		if c.shelf != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-shelf' cannot be specified at the same time.")
		}
		if c.rack != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-rack' cannot be specified at the same time.")
		}
		if c.racknumber != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-racknumber' cannot be specified at the same time.")
		}
		if c.position != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-position' cannot be specified at the same time.")
		}
		if c.barcode != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-barcode' cannot be specified at the same time.")
		}
		if c.assetType != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-type' cannot be specified at the same time.")
		}
		if c.model != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-model' cannot be specified at the same time.")
		}
		if c.costcenter != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-costcenter' cannot be specified at the same time.")
		}
		if c.gcn != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-gcn' cannot be specified at the same time.")
		}
		if c.target != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-target' cannot be specified at the same time.")
		}
		if c.board != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-board' cannot be specified at the same time.")
		}
		if c.mac != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-mac' cannot be specified at the same time.")
		}
		if c.phase != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-phase' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" {
		if c.name == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if c.location == "" && c.zone == "" && c.aisle == "" && c.row == "" && c.shelf == "" &&
			c.rack == "" && c.racknumber == "" && c.position == "" && c.barcode == "" && c.assetType == "" &&
			c.model == "" && c.costcenter == "" && c.gcn == "" && c.target == "" &&
			c.board == "" && c.mac == "" && c.phase == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
		if c.zone != "" && !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zone)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid zone name, please check help info for '-zone'.", c.zone)
		}
		if c.assetType != "" && !ufsUtil.IsAssetType(c.assetType) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid asset type, please check help info for '-type'.", c.assetType)
		}
	}
	return nil
}
