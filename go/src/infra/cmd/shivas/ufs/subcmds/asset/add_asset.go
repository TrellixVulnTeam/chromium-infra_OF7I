package asset

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.AddAssetFileText)
		c.Flags.StringVar(&c.name, "name", "", "Asset tag of the asset")
		c.Flags.StringVar(&c.location, "location", "", "location of the asset in barcode format")
		c.Flags.StringVar(&c.zone, "zone", "", "Zone that the asset is in. "+cmdhelp.ZoneFilterHelpText)
		c.Flags.StringVar(&c.aisle, "aisle", "", "Aisle that the asset is in")
		c.Flags.StringVar(&c.row, "row", "", "Row that the asset is in")
		c.Flags.StringVar(&c.rack, "rack", "", "Rack that the asset is in")
		c.Flags.StringVar(&c.position, "position", "", "Position that the asset is in")
		c.Flags.StringVar(&c.model, "model", "", "model of the asset")
		c.Flags.StringVar(&c.board, "board", "", "board/build target of the device")
		c.Flags.StringVar(&c.assetType, "type", "", "Type of asset. "+cmdhelp.AssetTypesHelpText)
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
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
	board     string
	tags      string
}

var mcsvFields = []string{
	"name",
	"zone",
	"rack",
	"model",
	"board",
	"assettype",
	"tags",
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
	var reqs []*ufsAPI.CreateAssetRequest
	if !c.interactive && c.newSpecsFile == "" && !c.scan {
		rawAsset, err := c.parseArgs()
		if err != nil {
			return err
		}
		createAssetRequest.Asset = rawAsset
		createAssetRequest.Asset.Name = ufsUtil.AddPrefix(ufsUtil.AssetCollection, createAssetRequest.Asset.Name)
	} else if c.newSpecsFile != "" {
		if utils.IsCSVFile(c.newSpecsFile) {
			reqs, err = c.parseMCSV()
			if err != nil {
				return err
			}
		} else {
			if err := utils.ParseJSONFile(c.newSpecsFile, &createAssetRequest); err != nil {
				return err
			}
			ufsZone := createAssetRequest.GetAsset().GetLocation().GetZone()
			createAssetRequest.GetAsset().Realm = ufsUtil.ToUFSRealm(ufsZone.String())
			createAssetRequest.Asset.Name = ufsUtil.AddPrefix(ufsUtil.AssetCollection, createAssetRequest.Asset.Name)
		}
	} else if c.interactive {
		fmt.Printf("Not implemented")
		return nil
	} else if c.scan {
		return c.scanAndAddAsset(ctx, ic, a.GetOut(), os.Stdin)
	}

	if len(reqs) == 0 {
		reqs = append(reqs, &createAssetRequest)
	}
	for _, r := range reqs {
		if !ufsUtil.ValidateTags(r.Asset.Tags) {
			fmt.Printf("Failed to add asset %s. Tags field contains invalidate characters.\n", r.Asset.GetName())
			continue
		}
		res, err := ic.CreateAsset(ctx, r)
		if err != nil {
			fmt.Printf("Failed to add asset %s. %s\n", r.Asset.GetName(), err)
			continue
		}
		res.Name = ufsUtil.RemovePrefix(res.Name)
		utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
		fmt.Println("Successfully added the asset: ", res.GetName())
	}
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
	if c.scan {
		if c.location == "" && c.rack == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Need location or rack to start scan")
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
		Info: &ufspb.AssetInfo{
			Model:       c.model,
			BuildTarget: c.board,
		},
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
	asset.Tags = utils.GetStringSlice(c.tags)
	return asset, nil
}

// parseMCSV parses the MCSV file and returns rack requests
func (c *addAsset) parseMCSV() ([]*ufsAPI.CreateAssetRequest, error) {
	records, err := utils.ParseMCSVFile(c.newSpecsFile)
	if err != nil {
		return nil, err
	}
	var reqs []*ufsAPI.CreateAssetRequest
	for i, rec := range records {
		// if i is 1, determine whether this is a header
		if i == 0 && utils.LooksLikeHeader(rec) {
			if err := utils.ValidateSameStringArray(mcsvFields, rec); err != nil {
				return nil, err
			}
			continue
		}
		req := &ufsAPI.CreateAssetRequest{
			Asset: &ufspb.Asset{
				Location: &ufspb.Location{},
				Info:     &ufspb.AssetInfo{},
			},
		}
		for i := range mcsvFields {
			name := mcsvFields[i]
			value := rec[i]
			switch name {
			case "name":
				req.Asset.Name = ufsUtil.AddPrefix(ufsUtil.AssetCollection, value)
			case "zone":
				if !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(value)) {
					return nil, fmt.Errorf("Error in line %d.\n%s is not a valid zone name. %s", i, value, cmdhelp.ZoneFilterHelpText)
				}
				ufsZone := ufsUtil.ToUFSZone(value)
				req.Asset.Location.Zone = ufsZone
				req.Asset.Realm = ufsUtil.ToUFSRealm(ufsZone.String())
			case "rack":
				if value == "" {
					return nil, fmt.Errorf("Error in line %d.\nNeed rack to create as asset. %s", i, name)
				}
				req.Asset.Location.Rack = value
			case "model":
				req.Asset.Model = value
				req.Asset.Info.Model = value
			case "board":
				req.Asset.Info.BuildTarget = value
			case "assettype":
				if !ufsUtil.IsAssetType(value) {
					return nil, fmt.Errorf("Error in line %d.\n%s is not a valid asset type. %s", i, value, cmdhelp.AssetTypesHelpText)
				}
				req.Asset.Type = ufsUtil.ToAssetType(value)
			case "tags":
				req.Asset.Tags = strings.Fields(value)
			default:
				return nil, fmt.Errorf("Error in line %d.\nUnknown field: %s", i, name)
			}
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

// scanAndAddAsset loops around and collects assets to be added to UFS. Useful for use with barcode scanner.
func (c *addAsset) scanAndAddAsset(ctx context.Context, ic ufsAPI.FleetClient, w io.Writer, r io.Reader) error {
	var location *ufspb.Location
	scanner := bufio.NewScanner(r)

	// Attempt to get location
	location, err := deriveLocation(ctx, ic, c.location, c.rack, c.zone, c.aisle, c.row, c.position)
	if err != nil {
		return err
	}
	// Throw an error if we don't know what rack we are putting the assets in. UFS will not accept this input.
	if location.GetRack() == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Cannot determine the rack from inputs. Try using -rack to explicity assign rack")
	}

	// Asset type set for this session
	assetType := ufsUtil.ToAssetType(c.assetType)

	fmt.Fprintf(w, "Connect the barcode scanner to your device.\n")
	prompt(w, location.GetRack())
	for scanner.Scan() {
		token := scanner.Text()
		if token == "" {
			prompt(w, location.GetRack())
			continue
		}
		// Attempt to update location mid scan
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

		// Create and add asset
		asset := &ufspb.Asset{
			Name:     ufsUtil.AddPrefix(ufsUtil.AssetCollection, token),
			Location: location,
			Type:     assetType,
		}

		_, err := ic.CreateAsset(ctx, &ufsAPI.CreateAssetRequest{
			Asset: asset,
		})

		if err != nil {
			fmt.Fprintf(w, "Failed to add asset %s to UFS. %s \n", token, err.Error())
		} else {
			fmt.Fprintf(w, "Added asset %s to UFS \n", token)
		}
		prompt(w, location.GetRack())
	}
	return nil
}

// deriveLocation Attempts to get location from all the possible inputs to the command.
func deriveLocation(ctx context.Context, ic ufsAPI.FleetClient, location, rack, zone, aisle, row, position string) (*ufspb.Location, error) {
	var loc *ufspb.Location
	// Attempt to decode location string if given.
	if location != "" {
		var err error
		loc, err = utils.GetLocation(location)
		if err != nil {
			return nil, err
		}
		return loc, nil
	} else if rack != "" {
		// If rack is input, check it's existence and get location from UFS.
		rack, err := ic.GetRack(ctx, &ufsAPI.GetRackRequest{
			Name: ufsUtil.AddPrefix(ufsUtil.RackCollection, rack),
		})
		if err != nil {
			return nil, err
		}
		loc = rack.GetLocation()
		// Update position if given or clear the field.
		loc.Position = position
		return loc, nil
	}
	// Attempt to generate location using raw inputs
	if loc == nil {
		loc = &ufspb.Location{}
	}
	loc.Rack = rack
	loc.Zone = ufsUtil.ToUFSZone(zone)
	loc.Aisle = aisle
	loc.Position = position
	loc.Row = row
	return loc, nil
}

// prompt prints the current rack for the scan. To be used by scanAndAddAsset
func prompt(w io.Writer, rack string) {
	fmt.Fprintf(w, "\nScan asset to be placed in %s: ", rack)
}
