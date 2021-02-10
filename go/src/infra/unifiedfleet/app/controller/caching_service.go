// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/caching"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateCachingService creates a new CachingService in datastore.
func CreateCachingService(ctx context.Context, cs *ufspb.CachingService) (*ufspb.CachingService, error) {
	f := func(ctx context.Context) error {
		if err := validateCreateCachingService(ctx, cs); err != nil {
			return errors.Annotate(err, "CreateCachingService - validation failed").Err()
		}
		if _, err := caching.BatchUpdateCachingServices(ctx, []*ufspb.CachingService{cs}); err != nil {
			return errors.Annotate(err, "CreateCachingService - unable to batch update CachingService %s", cs.Name).Err()
		}
		hc := getCachingServiceHistoryClient(cs)
		if err := hc.stUdt.updateStateHelper(ctx, cs.GetState()); err != nil {
			return err
		}
		hc.logCachingServiceChanges(nil, cs)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "CreateCachingService - unable to create CachingService %s", cs.Name).Err()
	}
	return cs, nil
}

// validateCreateCachingService validates if a CachingService can be created.
//
// checks if the CachingService already exists.
func validateCreateCachingService(ctx context.Context, cs *ufspb.CachingService) error {
	// Check if CachingService already exists.
	return resourceAlreadyExists(ctx, []*Resource{GetCachingServiceResource(cs.Name)}, nil)
}

func getCachingServiceHistoryClient(m *ufspb.CachingService) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: ufsUtil.AddPrefix(ufsUtil.CachingServiceCollection, m.Name),
		},
	}
}
