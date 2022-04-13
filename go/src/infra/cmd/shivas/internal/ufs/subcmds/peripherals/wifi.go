// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package peripherals

import (
	"context"
	"fmt"
	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	rpc "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/util"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"
)

var (
	AddPeripheralWifiCmd     = wifiCmd(actionAdd)
	ReplacePeripheralWifiCmd = wifiCmd(actionReplace)
	DeletePeripheralWifiCmd  = wifiCmd(actionDelete)
)

var csvHeaderMap = map[string]bool{
	"dut":           true,
	"router":        true,
	"wifi_features": true,
}

// wifiCmd creates command for adding, removing, or completely replacing wifi features or routers on a DUT.
func wifiCmd(mode action) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "peripheral-wifi -dut {DUT name} -wifi-feature {wifi feature} -router {hostname:h1,build_target:b1,model:m1,feature:f1} [-router {hostname:hn,...}...]",
		ShortDesc: "Manage wifi router connections to a DUT",
		LongDesc:  cmdhelp.ManagePeripheralWifiLongDesc,
		CommandRun: func() subcommands.CommandRun {
			c := manageWifiCmd{mode: mode}
			c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
			c.envFlags.Register(&c.Flags)
			c.commonFlags.Register(&c.Flags)

			c.Flags.StringVar(&c.dutName, "dut", "", "DUT name to update")
			c.Flags.Var(flag.StringSlice(&c.wifiFeatures), "wifi-feature", "wifi feature of the testbed, can be specified multiple times")
			c.Flags.Var(utils.CSVStringList(&c.routers), "router", "comma separated router info. require hostname:h1")
			c.Flags.StringVar(&c.wifiFile, "f", "", "File path to csv or json file. Note: Can only use in replace action. json file replaces the whole wifi proto, csv file replace multiple duts")

			return &c
		},
	}
}

// manageWifiCmd supports adding, replacing, or deleting Wifi features or routers.
// TODO (b/227504173): Add json file, csv file input options.
type manageWifiCmd struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	dutName         string
	wifiFeatures    []string
	wifiFeaturesMap map[string]map[lab.Wifi_Feature]bool
	routers         [][]string
	routersMap      map[string]map[string]*lab.WifiRouter // set of WifiRouter

	wifiFile            string
	wifiJsonFileWifiObj *lab.Wifi
	isCSVUpdate         bool
	isJsonUpdate        bool
	mode                action
}

// Run executed the wifi management subcommand. It cleans up passed flags and validates them.
func (c *manageWifiCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.run(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// run implements the core logic for Run.
func (c *manageWifiCmd) run(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.cleanAndValidateFlags(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, util.OSNamespace)

	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UFS service %s\n", e.UnifiedFleetService)
	}

	client := rpc.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	if c.isCSVUpdate {
		// only running in replacing mode. checked in cleanAndValidateFlag
		for dutName := range c.routersMap {
			err = c.runSingleDut(ctx, client, dutName)
			if err != nil {
				return errors.Annotate(err, "update csv error %s", dutName).Err()
			}
		}
	} else {
		return c.runSingleDut(ctx, client, c.dutName)
	}

	return nil
}

func (c *manageWifiCmd) runSingleDut(ctx context.Context, client rpc.FleetClient, dutName string) error {
	lse, err := client.GetMachineLSE(ctx, &rpc.GetMachineLSERequest{
		Name: util.AddPrefix(util.MachineLSECollection, dutName),
	})
	if err != nil {
		return err
	}
	if err := utils.IsDUT(lse); err != nil {
		return errors.Annotate(err, "not a dut").Err()
	}

	var (
		peripherals = lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
		currentWifi = peripherals.GetWifi()
	)
	nw, err := c.runWifiAction(currentWifi, dutName)
	if err != nil {
		return err
	}
	if c.commonFlags.Verbose() {
		fmt.Println("New Wifi", nw)
	}

	peripherals.Wifi = nw
	// TODO(b/226024082): Currently field masks are implemented in a limited way. Subsequent update
	// on UFS could add field masks for wifi and then they could be included here.
	_, err = client.UpdateMachineLSE(ctx, &rpc.UpdateMachineLSERequest{MachineLSE: lse})
	return err
}

// runWifiAction returns a new list of wifi feature and routers based on the action specified in c and current list.
// note: runWifiAction currently only modifies wifi.Features and wifi.WifiRouters
func (c *manageWifiCmd) runWifiAction(current *lab.Wifi, dutName string) (*lab.Wifi, error) {
	switch c.mode {
	case actionAdd:
		return c.addWifi(current, dutName)
	case actionReplace:
		if c.commonFlags.Verbose() {
			fmt.Println("Replacing", current)
		}
		return c.replaceWifi(current, dutName)
	case actionDelete:
		return c.deleteWifi(current, dutName)
	default:
		return nil, errors.Reason("unknown action %d", c.mode).Err()
	}
}

// replaceWifi replaces routers and/or wifi features with specified routers and/or wifi features.
func (c *manageWifiCmd) replaceWifi(current *lab.Wifi, dutName string) (*lab.Wifi, error) {
	if len(c.wifiFeaturesMap[dutName]) != 0 {
		current.Features = make([]lab.Wifi_Feature, 0)
		for feature := range c.wifiFeaturesMap[dutName] {
			current.Features = append(current.Features, feature)
		}
	}
	if len(c.routersMap[dutName]) != 0 {
		current.WifiRouters = make([]*lab.WifiRouter, 0)
		for hostname := range c.routersMap[dutName] {
			current.WifiRouters = append(current.WifiRouters, c.routersMap[dutName][hostname])
		}
	}
	return current, nil
}

// addWifi takes the current wifi returns the same wifi with wifi features and routers specified in c added.
// It returns an error if a duplicate is specified.
func (c *manageWifiCmd) addWifi(current *lab.Wifi, dutName string) (*lab.Wifi, error) {
	for _, router := range current.GetWifiRouters() {
		if _, ok := c.routersMap[dutName][router.GetHostname()]; ok {
			return nil, errors.Reason("wifi router %s already exists", router.GetHostname()).Err()
		}
	}
	for _, feature := range current.GetFeatures() {
		if c.wifiFeaturesMap[dutName][feature] {
			return nil, errors.Reason("wifi feature %s already exists", feature).Err()
		}
	}
	for hostname := range c.routersMap[dutName] {
		current.WifiRouters = append(current.WifiRouters, c.routersMap[dutName][hostname])
	}
	for feature := range c.wifiFeaturesMap[dutName] {
		current.Features = append(current.Features, feature)
	}
	return current, nil
}

// deleteWifi returns a wifi by removing those wifi feature, routers specified in c from current.
// It returns an error if a non-existent wifi feature or router is attempted to be removed.
func (c *manageWifiCmd) deleteWifi(current *lab.Wifi, dutName string) (*lab.Wifi, error) {
	currentFeaturesMap := make(map[lab.Wifi_Feature]bool)
	for _, feature := range current.GetFeatures() {
		currentFeaturesMap[feature] = true
	}
	currentRoutersMap := make(map[string]*lab.WifiRouter)
	for _, router := range current.GetWifiRouters() {
		currentRoutersMap[router.GetHostname()] = router
	}
	for feature := range c.wifiFeaturesMap[dutName] {
		if _, ok := currentFeaturesMap[feature]; !ok {
			return nil, errors.Reason("wifi feature %s does not exist", feature).Err()
		}
		delete(currentFeaturesMap, feature)
	}
	for hostname := range c.routersMap[dutName] {
		if _, ok := currentRoutersMap[hostname]; !ok {
			return nil, errors.Reason("wifi router %s does not exist", hostname).Err()
		}
		delete(currentRoutersMap, hostname)
	}
	current.Features = make([]lab.Wifi_Feature, 0, len(currentFeaturesMap))
	for feature := range currentFeaturesMap {
		current.Features = append(current.Features, feature)
	}
	current.WifiRouters = make([]*lab.WifiRouter, 0, len(currentRoutersMap))
	for hostname := range currentRoutersMap {
		current.WifiRouters = append(current.WifiRouters, currentRoutersMap[hostname])
	}
	return current, nil
}

const (
	errDuplicateDut           = "duplicate dut specified"
	errDuplicateBuildTarget   = "duplicate build_target specified"
	errDuplicateModel         = "duplicate model specified"
	errDuplicateRouterFeature = "duplicate router feature specified"
	errDuplicateWifiFeature   = "duplicate wifi feature specified"
	errInvalidRouterFeature   = "invalid router feature"
	errInvalidWifiFeature     = "invalid wifi feature"
	errNoRouterAndNoFeature   = "at least one -router or one -wifi-feature is required"
)

// cleanAndValidateFlags returns an error with the result of all validations. It strips whitespaces
// around hostnames and removes empty ones.
func (c *manageWifiCmd) cleanAndValidateFlags() error {
	var errStrs []string
	if c.routersMap == nil {
		c.routersMap = map[string]map[string]*lab.WifiRouter{}
	}
	if c.wifiFeaturesMap == nil {
		c.wifiFeaturesMap = map[string]map[lab.Wifi_Feature]bool{}
	}
	if len(c.wifiFile) != 0 {
		if utils.IsCSVFile(c.wifiFile) {
			c.isCSVUpdate = true
			records, err := utils.ParseMCSVFile(c.wifiFile)
			if err != nil {
				return errors.Annotate(err, "parsing CSV file error").Err()
			}
			for i, rec := range records {
				if i == 0 {
					if len(rec) == 0 {
						return errors.Annotate(err, "header should not be empty").Err()
					}
					for _, key := range rec {
						if csvHeaderMap[key] != true {
							return errors.Reason("invalid header %q", key).Err()
						}
					}
					continue
				}
				dut, routers, wifiFeatures := parseWifiCSVRow(records[0], rec)
				err := c.validateSingleDut(dut, routers, wifiFeatures)
				if err != nil {
					return errors.Annotate(err, "invalid input row number %d", i).Err()
				}
			}
			return nil

		} else {
			c.isJsonUpdate = true
			if len(c.dutName) == 0 {
				errStrs = append(errStrs, errDUTMissing)
			}
			if c.wifiJsonFileWifiObj == nil {
				c.wifiJsonFileWifiObj = &lab.Wifi{}
			}
			c.routersMap[c.dutName] = map[string]*lab.WifiRouter{}
			c.wifiFeaturesMap[c.dutName] = map[lab.Wifi_Feature]bool{}
			if err := utils.ParseJSONFile(c.wifiFile, c.wifiJsonFileWifiObj); err != nil {
				return errors.Annotate(err, "json parse error").Err()
			}
			for _, wifiFeature := range c.wifiJsonFileWifiObj.Features {
				c.wifiFeaturesMap[c.dutName][wifiFeature] = true
			}
			for _, router := range c.wifiJsonFileWifiObj.WifiRouters {
				if hostname := strings.TrimSpace(router.Hostname); hostname == "" {
					return errors.Reason("invalid router hostname. %s", router).Err()
				} else {
					c.routersMap[c.dutName][hostname] = router
				}
			}
			return nil
		}
	} else {
		err := c.validateSingleDut(c.dutName, c.routers, c.wifiFeatures)
		if err != nil {
			return err
		}
		if len(errStrs) == 0 {
			return nil
		}
		return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf("Wrong usage!!\n%s", strings.Join(errStrs, "\n")))
	}
}

func parseWifiCSVRow(header []string, row []string) (dut string, routers [][]string, wifiFeatures []string) {
	for i, headerKey := range header {
		rowField := strings.ToLower(strings.TrimSpace(row[i]))
		switch headerKey {
		case "dut":
			dut = rowField
		case "wifi_features":
			if rowField != "" {
				wifiFeatures = strings.Split(rowField, ";")
			}
		case "router":
			if rowField != "" {
				routers = append(routers, strings.Split(rowField, ";"))
			}
		default:
			fmt.Println("shouldn't be in here since header fields are checked. add cases that is in csvHeaderMap")
		}
	}
	return dut, routers, wifiFeatures
}

// validateSingleDut validate input for single dut. Use for multiple row update and single cli input
func (c *manageWifiCmd) validateSingleDut(dutName string, routersInput [][]string, wifiFeaturesInput []string) error {
	var errStrs []string
	if len(dutName) == 0 {
		errStrs = append(errStrs, errDUTMissing)
	}
	if _, ok := c.routersMap[dutName]; ok {
		return errors.Reason("%s: %s", errDuplicateDut, dutName).Err()
	}
	if _, ok := c.wifiFeaturesMap[dutName]; ok {
		return errors.Reason("%s: %s", errDuplicateDut, dutName).Err()
	}
	c.routersMap[dutName] = map[string]*lab.WifiRouter{}
	c.wifiFeaturesMap[dutName] = map[lab.Wifi_Feature]bool{}
	for _, routerCSV := range routersInput {
		newRouter := &lab.WifiRouter{}
		newRouterFeaturesMap := make(map[lab.WifiRouter_Feature]bool)
		for _, keyValStr := range routerCSV {
			keyValList := strings.Split(keyValStr, ":")
			if len(keyValList) != 2 {
				errStrs = append(errStrs, fmt.Sprintf("Invalid key:val for router tag %q", keyValList))
			}
			key := strings.ToLower(strings.TrimSpace(keyValList[0]))
			val := strings.ToLower(strings.TrimSpace(keyValList[1]))
			switch key {
			case "hostname":
				if newRouter.GetHostname() != "" {
					errStrs = append(errStrs, errDuplicateHostname)
				}
				newRouter.Hostname = val
			case "model":
				if newRouter.GetModel() != "" {
					errStrs = append(errStrs, errDuplicateModel)
				}
				newRouter.Model = val
			case "build_target":
				if newRouter.GetBuildTarget() != "" {
					errStrs = append(errStrs, errDuplicateBuildTarget)
				}
				newRouter.BuildTarget = val
			case "feature":
				val = strings.ToUpper(val)
				if fInt, ok := lab.WifiRouter_Feature_value[val]; !ok {
					errStrs = append(errStrs, fmt.Sprintf("%s: %q", errInvalidRouterFeature, val))
				} else {
					if newRouterFeaturesMap[lab.WifiRouter_Feature(fInt)] {
						errStrs = append(errStrs, errDuplicateRouterFeature)
					}
					newRouterFeaturesMap[lab.WifiRouter_Feature(fInt)] = true
				}
			default:
				errStrs = append(errStrs, fmt.Sprintf("unsupported router key: %q", key))
			}
		}
		if newRouter.GetHostname() == "" {
			errStrs = append(errStrs, errEmptyHostname)
		} else {
			for feature := range newRouterFeaturesMap {
				newRouter.Features = append(newRouter.Features, feature)
			}
			if _, ok := c.routersMap[dutName][newRouter.GetHostname()]; ok {
				errStrs = append(errStrs, errDuplicateHostname)
			}
			c.routersMap[dutName][newRouter.GetHostname()] = newRouter
		}
	}

	for _, feature := range wifiFeaturesInput {
		feature = strings.ToUpper(strings.TrimSpace(feature))
		if fInt, ok := lab.Wifi_Feature_value[feature]; !ok {
			errStrs = append(errStrs, fmt.Sprintf("%s: %q", errInvalidWifiFeature, feature))
		} else {
			if c.wifiFeaturesMap[dutName][lab.Wifi_Feature(fInt)] {
				errStrs = append(errStrs, errDuplicateWifiFeature)
			}
			c.wifiFeaturesMap[dutName][lab.Wifi_Feature(fInt)] = true
		}
	}

	if len(c.routersMap[dutName]) == 0 && len(c.wifiFeaturesMap[dutName]) == 0 {
		errStrs = append(errStrs, errNoRouterAndNoFeature)
	}

	if len(errStrs) == 0 {
		return nil
	}

	return cmdlib.NewQuietUsageError(c.Flags, fmt.Sprintf("Wrong usage!!\n%s", strings.Join(errStrs, "\n")))
}
