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

// AddAssetCmd add an asset to database
var AddAssetCmd = &subcommands.Command{
	UsageLine: "asset [options...]",
	ShortDesc: "Add an asset(Chromebook, Servo, Labstation)",
	LongDesc:  cmdhelp.AddAssetLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addAsset{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.BoolVar(&c.interactive, "i", false, "Interactive mode")
		c.Flags.BoolVar(&c.scan, "scan", false, "scan location followed by asset using a barcode scanner")
		c.Flags.StringVar(&c.newSpecsFile, "f", "", "JSON file containing input")
		c.Flags.StringVar(&c.name, "name", "", "Asset tag of the asset")
		c.Flags.StringVar(&c.location, "location", "", "location of the asset in barcode format")
		c.Flags.StringVar(&c.zone, "zone", "", "Zone that the asset is in. "+cmdhelp.ZoneFilterHelpText)
		c.Flags.StringVar(&c.aisle, "aisle", "", "Aisle that the asset is in")
		c.Flags.StringVar(&c.row, "row", "", "Row that the asset is in")
		c.Flags.StringVar(&c.rack, "rack", "", "Rack that the asset is in")
		c.Flags.StringVar(&c.position, "position", "", "Position that the asset is in")
		c.Flags.StringVar(&c.assetType, "type", "", "Type of asset. "+cmdhelp.AssetTypesHelpText)
		return c
	},
}

type addAsset struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	interactive  bool
	newSpecsFile string
	scan         bool

	name      string
	location  string
	zone      string
	aisle     string
	row       string
	shelf     string
	rack      string
	position  string
	assetType string
	model     string
}

func (c *addAsset) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addAsset) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var createAssetRequest ufsAPI.CreateAssetRequest

	if !c.interactive && c.newSpecsFile == "" && !c.scan {
		rawAsset, err := c.parseArgs()
		if err != nil {
			return err
		}
		createAssetRequest.Asset = rawAsset
	} else if c.newSpecsFile != "" {
		if err := utils.ParseJSONFile(c.newSpecsFile, &createAssetRequest); err != nil {
			return err
		}
		ufsZone := createAssetRequest.GetAsset().GetLocation().GetZone()
		createAssetRequest.GetAsset().Realm = ufsUtil.ToUFSRealm(ufsZone.String())
	} else if c.interactive {
		fmt.Printf("Not implemented")
		return nil
	} else if c.scan {
		fmt.Printf("Not implemented")
		return nil
	}

	prefix, err := ufsUtil.GetResourcePrefix(createAssetRequest.Asset)
	if err != nil {
		return err
	}
	createAssetRequest.Asset.Name = ufsUtil.AddPrefix(prefix, createAssetRequest.Asset.Name)

	res, err := ic.CreateAsset(ctx, &createAssetRequest)
	if err != nil {
		return err
	}
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully added the asset: ", res.GetName())
	return nil
}

func (c *addAsset) validateArgs() error {
	if c.newSpecsFile == "" && !c.scan && !c.interactive {
		// Validate the raw inputs
		if c.name == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need asset name to create an asset")
		}
		if c.location == "" && c.zone == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need zone to create an asset")
		}
		if c.location == "" && c.rack == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need rack to create an asset")
		}
		if c.location == "" && !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zone)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Invalid zone %s", c.zone)
		}
		if c.assetType == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Missing asset type")
		} else if !ufsUtil.IsAssetType(c.assetType) {
			return cmdlib.NewQuietUsageError(c.Flags, "Invalid asset type %s", c.assetType)
		}
	}
	return nil
}

func (c *addAsset) parseArgs() (*ufspb.Asset, error) {
	asset := &ufspb.Asset{
		Name:  c.name,
		Type:  ufsUtil.ToAssetType(c.assetType),
		Model: c.model,
	}

	var err error

	if c.location != "" {
		asset.Location, err = utils.GetLocation(c.location)
		if err != nil {
			return nil, err
		}
		if asset.Location.Rack == "" {
			return nil, cmdlib.NewQuietUsageError(c.Flags, "Invalid input, rack required but not found in location %s", c.location)
		}
		if asset.Location.Zone == ufspb.Zone_ZONE_UNSPECIFIED {
			return nil, cmdlib.NewQuietUsageError(c.Flags, "Invalid zone in location %s", c.location)
		}
	} else {
		asset.Location = &ufspb.Location{}
		asset.Location.Aisle = c.aisle
		asset.Location.Row = c.row
		asset.Location.Rack = c.rack
		asset.Location.Shelf = c.shelf
		asset.Location.Position = c.position
		asset.Location.Zone = ufsUtil.ToUFSZone(c.zone)
	}
	asset.Realm = ufsUtil.ToUFSRealm(asset.GetLocation().GetZone().String())
	return asset, nil
}
