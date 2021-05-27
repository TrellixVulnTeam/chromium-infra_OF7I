// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// UpdateCachingService updates existing CachingService in datastore.
func UpdateCachingService(ctx context.Context, cs *ufspb.CachingService, mask *field_mask.FieldMask) (*ufspb.CachingService, error) {
	f := func(ctx context.Context) error {
		// Get old/existing CachingService for logging and partial update.
		oldCs, err := caching.GetCachingService(ctx, cs.GetName())
		if err != nil {
			return errors.Annotate(err, "UpdateCachingService - get CachingService %s failed", cs.GetName()).Err()
		}
		// Validate the input.
		if err := validateUpdateCachingService(ctx, oldCs, cs, mask); err != nil {
			return errors.Annotate(err, "UpdateCachingService - validation failed").Err()
		}
		// Copy for logging.
		oldCsCopy := oldCs
		// Partial update by field mask.
		if mask != nil && len(mask.Paths) > 0 {
			// Validate partial update field mask.
			if err := validateCachingServiceUpdateMask(ctx, cs, mask); err != nil {
				return err
			}
			// Clone oldCs for logging as the oldCs will be updated with new values.
			oldCsCopy = proto.Clone(oldCs).(*ufspb.CachingService)
			// Process the field mask to get updated values.
			cs, err = processCachingServiceUpdateMask(ctx, oldCs, cs, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateCachingService - processing update mask failed").Err()
			}
		}
		if _, err := caching.BatchUpdateCachingServices(ctx, []*ufspb.CachingService{cs}); err != nil {
			return errors.Annotate(err, "UpdateCachingService - unable to batch update CachingService %s", cs.Name).Err()
		}
		hc := getCachingServiceHistoryClient(cs)
		if err := hc.stUdt.updateStateHelper(ctx, cs.GetState()); err != nil {
			return err
		}
		hc.logCachingServiceChanges(oldCsCopy, cs)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateCachingService - failed to update CachingService %s in datastore", cs.Name).Err()
	}
	return cs, nil
}

// GetCachingService returns CachingService for the given id from datastore.
func GetCachingService(ctx context.Context, id string) (*ufspb.CachingService, error) {
	return caching.GetCachingService(ctx, id)
}

// DeleteCachingService deletes the given CachingService in datastore.
func DeleteCachingService(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		// Get the CachingService for logging.
		cs, err := caching.GetCachingService(ctx, id)
		if err != nil {
			return errors.Annotate(err, "DeleteCachingService - get CachingService %s failed", id).Err()
		}
		if err := caching.DeleteCachingService(ctx, id); err != nil {
			return errors.Annotate(err, "DeleteCachingService - unable to delete CachingService %s", id).Err()
		}
		hc := getCachingServiceHistoryClient(cs)
		hc.stUdt.deleteStateHelper(ctx)
		hc.logCachingServiceChanges(cs, nil)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return errors.Annotate(err, "DeleteCachingService - failed to delete CachingService %s in datastore", id).Err()
	}
	return nil
}

// ListCachingServices lists the CachingServices in datastore.
func ListCachingServices(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.CachingService, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, caching.GetCachingServiceIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing CachingServices").Err()
		}
	}
	filterMap = resetStateFilter(filterMap)
	return caching.ListCachingServices(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// processCachingServiceUpdateMask processes update field mask to get only specific update
// fields and return a complete CachingService object with updated and existing fields.
func processCachingServiceUpdateMask(ctx context.Context, oldCs *ufspb.CachingService, cs *ufspb.CachingService, mask *field_mask.FieldMask) (*ufspb.CachingService, error) {
	// Update the fields in the existing/old CachingService.
	for _, path := range mask.Paths {
		switch path {
		case "port":
			oldCs.Port = cs.GetPort()
		case "serving_subnet":
			oldCs.ServingSubnet = cs.GetServingSubnet()
		case "serving_subnets":
			oldCs.ServingSubnets = mergeTags(oldCs.GetServingSubnets(), cs.GetServingSubnets())
		case "serving_subnets.remove":
			oldSubnets := oldCs.GetServingSubnets()
			for _, s := range cs.GetServingSubnets() {
				oldSubnets = ufsUtil.RemoveStringEntry(oldSubnets, s)
			}
			oldCs.ServingSubnets = oldSubnets
		case "primary_node":
			oldCs.PrimaryNode = cs.GetPrimaryNode()
		case "secondary_node":
			oldCs.SecondaryNode = cs.GetSecondaryNode()
		case "state":
			oldCs.State = cs.GetState()
		case "description":
			oldCs.Description = cs.GetDescription()
		}
	}
	// Return existing/old CachingService with new updated values.
	return oldCs, nil
}

// validateCreateCachingService validates if a CachingService can be created.
//
// checks if the CachingService already exists.
func validateCreateCachingService(ctx context.Context, cs *ufspb.CachingService) error {
	// Check if CachingService already exists.
	return resourceAlreadyExists(ctx, []*Resource{GetCachingServiceResource(cs.Name)}, nil)
}

// validateUpdateCachingService validates if an exsting CachingService can be updated.
//
// checks if the CachingService does not exist.
func validateUpdateCachingService(ctx context.Context, oldCs *ufspb.CachingService, cs *ufspb.CachingService, mask *field_mask.FieldMask) error {
	// Check if resources does not exist.
	return ResourceExist(ctx, []*Resource{GetCachingServiceResource(cs.Name)}, nil)
}

// validateCachingServiceUpdateMask validates the update mask for CachingService partial update.
func validateCachingServiceUpdateMask(ctx context.Context, cs *ufspb.CachingService, mask *field_mask.FieldMask) error {
	if mask != nil {
		// Validate the give field mask.
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateCachingServiceUpdateMask - name cannot be updated, delete and create a CachingService instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateCachingServiceUpdateMask - update_time cannot be updated, it is a output only field")
			case "port":
			case "serving_subnet":
			case "serving_subnets":
			case "serving_subnets.remove":
			case "primary_node":
			case "secondary_node":
			case "state":
			case "description":
				// Valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateCachingServiceUpdateMask - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}

func getCachingServiceHistoryClient(m *ufspb.CachingService) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: ufsUtil.AddPrefix(ufsUtil.CachingServiceCollection, m.Name),
		},
	}
}
