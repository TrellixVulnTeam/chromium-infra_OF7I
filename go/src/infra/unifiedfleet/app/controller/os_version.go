// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	ufsds "infra/unifiedfleet/app/model/datastore"
)

// ImportOSes inserts chrome os_version to datastore.
func ImportOSes(ctx context.Context, oses []*ufspb.OSVersion, pageSize int) (*ufsds.OpResults, error) {
	deleteNonExistingOSes(ctx, oses, pageSize)
	logging.Debugf(ctx, "Importing %d os versions", len(oses))
	return configuration.ImportOses(ctx, oses)
}

// ListOSes lists the chrome os_version
func ListOSes(ctx context.Context, pageSize int32, pageToken string, filter string, keysOnly bool) ([]*ufspb.OSVersion, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, configuration.GetOSVersionIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing os versions").Err()
		}
	}
	return configuration.ListOSes(ctx, pageSize, pageToken, filterMap, keysOnly)
}

func deleteNonExistingOSes(ctx context.Context, oses []*ufspb.OSVersion, pageSize int) (*ufsds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range oses {
		resMap[r.GetValue()] = true
	}
	resp, err := configuration.GetAllOSes(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.OSVersion)
		if _, ok := resMap[s.GetValue()]; !ok {
			toDelete = append(toDelete, s.GetValue())
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing oses", len(toDelete))
	return deleteByPage(ctx, toDelete, pageSize, configuration.DeleteOSes), nil
}
