// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package asset

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/genproto/protobuf/field_mask"

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
		c.Flags.BoolVar(&c.scan, "scan", false, "Update asset location using barcode scanner")

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
		c.Flags.StringVar(&c.board, "board", "", "board/build target of the asset. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.reference, "reference", "", "Reference board of the asset. "+cmdhelp.ClearFieldHelpText)
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
	scan         bool

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
	board      string
	reference  string
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
	if c.scan {
		return c.scanAndUpdateLocation(ctx, ic, a.GetOut(), os.Stdin)
	}
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
	existingAsset, err := utils.PrintExistingAsset(ctx, ic, asset.Name)
	if err != nil {
		return err
	}
	// Check HWID for non-partial update
	// TODO(anushruth): Check for file type when implementing mcsv support.
	if c.newSpecsFile != "" && existingAsset.GetInfo().GetHwid() != asset.GetInfo().GetHwid() {
		newHWID := asset.GetInfo().GetHwid()
		if newHWID == "" {
			return fmt.Errorf("users cannot update hwid to empty string manually")
		}
		prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
		if prompt != nil && !prompt(fmt.Sprintf("HWID can only be used by Fleet Admins. Are you sure you want to modify the HWID to %s?", newHWID)) {
			return nil
		}
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
			"board":      "info.build_target",
			"reference":  "info.reference_board",
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
	// Update location can be done either using location flag or parametric
	// location flags (zone, row, rack ...)
	if c.location == utils.ClearFieldValue {
		// Reset the location flag
		asset.Location = &ufspb.Location{}
	} else if c.location != "" {
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
	// If the location flag is not given, attempt to update location using
	// parametric location flags (zone, row, rack ...)
	if c.location == "" {
		// Assign empty location to asset
		asset.Location = &ufspb.Location{}
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
	if c.board == utils.ClearFieldValue {
		asset.Info.BuildTarget = ""
	} else {
		asset.Info.BuildTarget = c.board
	}
	if c.reference == utils.ClearFieldValue {
		asset.Info.ReferenceBoard = ""
	} else {
		asset.Info.ReferenceBoard = c.reference
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
	if c.newSpecsFile != "" || c.scan {
		errFmtStr := ""
		if c.newSpecsFile != "" {
			errFmtStr = "Wrong usage!!\nThe JSON input file is already specified. '%s' cannot be specified at the same time."
		}
		if c.scan {
			errFmtStr = "Wrong usage!!\n '-scan' is already specified. '%s' cannot be specified at the same time."
		}
		if c.name != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-name"))
		}
		if c.location != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-location"))
		}
		if c.zone != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-zone"))
		}
		if c.aisle != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-aisle"))
		}
		if c.row != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-row"))
		}
		if c.shelf != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-shelf"))
		}
		if c.rack != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-rack"))
		}
		if c.racknumber != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-racknumber"))
		}
		if c.position != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-position"))
		}
		if c.barcode != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-barcode"))
		}
		if c.assetType != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-barcode"))
		}
		if c.model != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-model"))
		}
		if c.costcenter != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-costcenter"))
		}
		if c.gcn != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-gcn"))
		}
		if c.board != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-board"))
		}
		if c.reference != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-reference"))
		}
		if c.mac != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-mac"))
		}
		if c.phase != "" {
			return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-phase"))
		}
	}
	if c.newSpecsFile == "" && !c.scan {
		if c.name == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if c.location == "" && c.zone == "" && c.aisle == "" && c.row == "" && c.shelf == "" &&
			c.rack == "" && c.racknumber == "" && c.position == "" && c.barcode == "" && c.assetType == "" &&
			c.model == "" && c.costcenter == "" && c.gcn == "" && c.board == "" &&
			c.reference == "" && c.mac == "" && c.phase == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
		if c.zone != "" && !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zone)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid zone name, please check help info for '-zone'.", c.zone)
		}
		if c.assetType != "" && !ufsUtil.IsAssetType(c.assetType) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid asset type, please check help info for '-type'.", c.assetType)
		}
		if c.location != "" {
			errFmtStr := "Wrong usage!!\n '-location' is already specified. '%s' cannot be specified at the same time."
			if c.zone != "" {
				return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-zone"))
			}
			if c.aisle != "" {
				return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-aisle"))
			}
			if c.row != "" {
				return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-row"))
			}
			if c.shelf != "" {
				return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-shelf"))
			}
			if c.rack != "" {
				return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-rack"))
			}
			if c.racknumber != "" {
				return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-racknumber"))
			}
			if c.position != "" {
				return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-position"))
			}
			if c.barcode != "" {
				return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf(errFmtStr, "-barcode"))
			}
		}
	}
	return nil
}

// scanAndUpdateLocation loops until quit, collects the new location and asset tags and updates the asset location
func (c *updateAsset) scanAndUpdateLocation(ctx context.Context, ic ufsAPI.FleetClient, w io.Writer, r io.Reader) error {
	var location *ufspb.Location
	scanner := bufio.NewScanner(r)

	// Attempt to get location
	location, err := deriveLocation(ctx, ic, c.location, c.rack, c.shelf, c.position)
	if err != nil {
		if c.commonFlags.Verbose() {
			fmt.Fprintf(w, "Cannot determine location from inputs. Need to scan the location.\n%v\n", err)
		}
	}

	fmt.Fprintf(w, "Connect the barcode scanner to your device.\n")
	prompt(w, location.GetRack())
	for scanner.Scan() {
		token := scanner.Text()
		if token == "" {
			prompt(w, location.GetRack())
			continue
		}
		// Attempt to update location
		if utils.IsLocation(token) {
			l, err := utils.GetLocation(token)
			if err != nil || l.GetRack() == "" {
				fmt.Fprintf(w, "Cannot determine rack for the location %s. %s\n", token, err.Error())
				continue
			}
			location = l
			prompt(w, location.GetRack())
			continue
		}

		if location != nil {
			// Create and add asset
			asset := &ufspb.Asset{
				Name:     ufsUtil.AddPrefix(ufsUtil.AssetCollection, token),
				Location: location,
			}

			_, err := ic.UpdateAsset(ctx, &ufsAPI.UpdateAssetRequest{
				Asset: asset,
				UpdateMask: &field_mask.FieldMask{
					Paths: []string{"location"},
				},
			})

			if err != nil {
				fmt.Fprintf(w, "Failed to update asset %s location to UFS. %s \n", token, err.Error())
			} else {
				fmt.Fprintf(w, "Updated asset %s location to UFS \n", token)
			}
		}
		prompt(w, location.GetRack())
	}
	return nil
}
