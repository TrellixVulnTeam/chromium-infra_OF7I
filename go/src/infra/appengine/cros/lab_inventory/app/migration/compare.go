// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package migration

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/pmezard/go-difflib/difflib"
	authclient "go.chromium.org/luci/auth"
	gitilesapi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/server/auth"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/frontend"
	"infra/appengine/cros/lab_inventory/app/migration/internal/gitstore"
	"infra/libs/cros/lab_inventory/datastore"
	"infra/libs/skylab/inventory"
)

func newGitilesClient(c context.Context, host string) (gitiles.GitilesClient, error) {
	t, err := auth.GetRPCTransport(c, auth.AsSelf, auth.WithScopes(authclient.OAuthScopeEmail, gitilesapi.OAuthScope))
	if err != nil {
		return nil, errors.Annotate(err, "failed to get RPC transport").Err()
	}
	return gitilesapi.NewRESTClient(&http.Client{Transport: t}, host, true)
}

const (
	stagingEnv     = "ENVIRONMENT_STAGING"
	prodEnv        = "ENVIRONMENT_PROD"
	maxErrorLogged = 10
)

func getV1Duts(ctx context.Context) (stringset.Set, map[string]*inventory.DeviceUnderTest, error) {
	env := config.Get(ctx).Environment

	gitilesHost := config.Get(ctx).GetInventory().GetHost()
	client, err := newGitilesClient(ctx, gitilesHost)
	if err != nil {
		return nil, nil, errors.Annotate(err, "fail to create inventory v1 client").Err()
	}
	store := gitstore.NewInventoryStore(nil, client)
	if err := store.Refresh(ctx); err != nil {
		return nil, nil, errors.Annotate(err, "fail to refresh inventory v1 store").Err()
	}
	duts := store.Lab.GetDuts()
	hostnames := make([]string, 0, len(duts))
	dutMap := map[string]*inventory.DeviceUnderTest{}
	for _, dut := range duts {
		dutEnv := dut.GetCommon().GetEnvironment().String()
		// Ignore all DUTs of non-current environment.
		if env == prodEnv && dutEnv == stagingEnv {
			continue
		}
		if env == stagingEnv && dutEnv != stagingEnv {
			continue
		}
		name := dut.GetCommon().GetHostname()
		hostnames = append(hostnames, name)
		dutMap[name] = dut
	}
	return stringset.NewFromSlice(hostnames...), dutMap, nil
}

func getV2Duts(ctx context.Context) (stringset.Set, map[string]*inventory.DeviceUnderTest, error) {
	duts, err := datastore.GetAllDevices(ctx)
	if err != nil {
		return nil, nil, err
	}
	if l := len(duts.Failed()); l > 0 {
		logging.Warningf(ctx, "Failed to get %d devices from v2", l)
		for i, d := range duts.Failed() {
			if i > maxErrorLogged {
				logging.Warningf(ctx, "...")
				break
			}
			logging.Warningf(ctx, "%s: %s", d.Entity.Hostname, d.Err.Error())
		}
	}

	// Filter out all servo v3s.
	v2Duts := make([]datastore.DeviceOpResult, 0, len(duts.Passed()))
	for _, d := range duts.Passed() {
		if !strings.HasSuffix(d.Entity.Hostname, "-servo") {
			v2Duts = append(v2Duts, d)
		}
	}
	extendedData, failedDevices := frontend.GetExtendedDeviceData(ctx, v2Duts)
	if len(failedDevices) > 0 {
		logging.Warningf(ctx, "Failed to get extended data")
		for i, d := range failedDevices {
			if i > maxErrorLogged {
				logging.Warningf(ctx, "...")
				break
			}
			logging.Warningf(ctx, "%s: %s: %s", d.Id, d.Hostname, d.ErrorMsg)
		}
	}

	hostnames := make([]string, len(duts.Passed()))
	dutMap := map[string]*inventory.DeviceUnderTest{}
	for _, d := range extendedData {
		v1Dut, err := api.AdaptToV1DutSpec(d)
		if err != nil {
			logging.Warningf(ctx, "Adapter failure: %s", err.Error())
		}
		name := v1Dut.GetCommon().GetHostname()
		hostnames = append(hostnames, name)
		dutMap[name] = v1Dut
	}
	return stringset.NewFromSlice(hostnames...), dutMap, nil
}

// CompareInventory compares the inventory from v1 and v2 and log the
// difference.
func CompareInventory(ctx context.Context) error {
	logDifference := func(lhs, rhs stringset.Set, msg string) {
		if d := lhs.Difference(rhs); d.Len() > 0 {
			logging.Warningf(ctx, msg)
			d.Iter(func(name string) bool {
				logging.Warningf(ctx, "%#v", name)
				return true
			})
		} else {
			logging.Infof(ctx, "No result of %#v", msg)
		}
	}
	v1Duts, v1DutMap, err := getV1Duts(ctx)
	if err != nil {
		return err
	}
	v2Duts, v2DutMap, err := getV2Duts(ctx)
	if err != nil {
		return err
	}
	logDifference(v1Duts, v2Duts, "Devices only in v1")
	logDifference(v2Duts, v1Duts, "Devices only in v2")

	count := 0
	v1Duts.Intersect(v2Duts).Iter(func(name string) bool {
		d1 := v1DutMap[name]
		d2 := v2DutMap[name]
		filterOutKnownDifference(d1, d2)
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(proto.MarshalTextString(d1)),
			B:        difflib.SplitLines(proto.MarshalTextString(d2)),
			FromFile: "v1",
			ToFile:   "v2",
			Context:  0,
		}
		diffText, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			logging.Errorf(ctx, "failed to compare %#v: %s", name, err.Error())
			return true
		}
		if diffText != "" {
			if count > maxErrorLogged {
				logging.Warningf(ctx, "and more difference ...")
				return false // Break the iteration.
			}
			logging.Warningf(ctx, "%#v is different: \n%s", name, diffText)
			count++
		}
		return true
	})
	return nil
}

func filterOutKnownDifference(d1, d2 *inventory.DeviceUnderTest) {
	// Add other know difference here.
	d1.GetCommon().Environment = d2.GetCommon().Environment
}
