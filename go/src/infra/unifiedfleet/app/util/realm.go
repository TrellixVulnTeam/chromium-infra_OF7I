// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"context"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/realms"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
)

// OldBrowserLabAdminRealm is the Old realm for browser lab
//
// If a client sends this realm, replace it with BrowserLabAdminRealm
const OldBrowserLabAdminRealm = "chromium:ufs/browser-admin"

// BrowserLabAdminRealm is the admin realm for browser lab.
const BrowserLabAdminRealm = "@internal:ufs/browser"

// AtlLabAdminRealm is the admin realm for atl lab.
const AtlLabAdminRealm = "@internal:ufs/os-atl"

// AcsLabAdminRealm is the admin realm for acs lab.
const AcsLabAdminRealm = "@internal:ufs/os-acs"

// SkipRealmsCheck flag to skip realms check
var SkipRealmsCheck = false

// UFS registered permissions in process registry
var (
	// ConfigurationsGet allows to get configuration resources.
	ConfigurationsGet = realms.RegisterPermission("ufs.configurations.get")
	// ConfigurationsList allows to list configuration resources.
	ConfigurationsList = realms.RegisterPermission("ufs.configurations.list")
	// ConfigurationsCreate allows to create configuration resources.
	ConfigurationsCreate = realms.RegisterPermission("ufs.configurations.create")
	// ConfigurationUpdate allows to update configuration resources.
	ConfigurationsUpdate = realms.RegisterPermission("ufs.configurations.update")
	// ConfigurationsDelete allows to delete configuration resources.
	ConfigurationsDelete = realms.RegisterPermission("ufs.configurations.delete")

	// RegistrationsGet allows to get registration resources.
	RegistrationsGet = realms.RegisterPermission("ufs.registrations.get")
	// RegistrationsList allows to list registration resources.
	RegistrationsList = realms.RegisterPermission("ufs.registrations.list")
	// RegistrationsCreate allows to create registration resources.
	RegistrationsCreate = realms.RegisterPermission("ufs.registrations.create")
	// RegistrationsUpdate allows to update registration resources.
	RegistrationsUpdate = realms.RegisterPermission("ufs.registrations.update")
	// RegistrationsDelete allows to delete registration resources.
	RegistrationsDelete = realms.RegisterPermission("ufs.registrations.delete")

	// InventoriesGet allows to get inventory resources.
	InventoriesGet = realms.RegisterPermission("ufs.inventories.get")
	// InventoriesList allows to list inventory resources.
	InventoriesList = realms.RegisterPermission("ufs.inventories.list")
	// InventoriesCreate allows to create inventory resources.
	InventoriesCreate = realms.RegisterPermission("ufs.inventories.create")
	// InventoriesUpdate allows to update inventory resources.
	InventoriesUpdate = realms.RegisterPermission("ufs.inventories.update")
	// InventoriesDelete allows to delete inventory resources.
	InventoriesDelete = realms.RegisterPermission("ufs.inventories.delete")

	// NetworksGet allows to get network resources.
	NetworksGet = realms.RegisterPermission("ufs.networks.get")
	// NetworksList allows to list network resources.
	NetworksList = realms.RegisterPermission("ufs.networks.list")
	// NetworksCreate allows to create network resources.
	NetworksCreate = realms.RegisterPermission("ufs.networks.create")
	// NetworksUpdate allows to update network resources.
	NetworksUpdate = realms.RegisterPermission("ufs.networks.update")
	// NetworksDelete allows to delete network resources.
	NetworksDelete = realms.RegisterPermission("ufs.networks.delete")

	// ResourcesImport allows to import resource resources.
	ResourcesImport = realms.RegisterPermission("ufs.resources.import")
)

// CurrentUser returns the current user
func CurrentUser(ctx context.Context) string {
	return auth.CurrentUser(ctx).Email
}

// hasPermission checks if the user has permission in the realm
func hasPermission(ctx context.Context, perm realms.Permission, realm string) (bool, error) {
	has, err := auth.HasPermission(ctx, perm, realm)
	if err != nil {
		logging.Errorf(ctx, "failed to check realm %q ACLs", err.Error())
		return false, status.Errorf(codes.PermissionDenied, "failed to check realm %q ACLs", err)
	}
	return has, nil
}

// CheckPermission checks if the user has permission in the realm
//
// return error if user doesnt have permission or unable to check permission in realm
// else returns nil
func CheckPermission(ctx context.Context, perm realms.Permission, realm string) error {
	if SkipRealmsCheck {
		logging.Infof(ctx, "Skipping Realms check")
		return nil
	}
	if realm == "" {
		logging.Infof(ctx, "No permission check for empty realm. Entity permission %s allowed for the user %s", perm, auth.CurrentIdentity(ctx))
		return nil
	}
	allow, err := hasPermission(ctx, perm, realm)
	if err != nil {
		return err
	}
	if !allow {
		logging.Errorf(ctx, "%s does not have permission %s in the realm %s", auth.CurrentIdentity(ctx), perm, realm)
		return status.Errorf(codes.PermissionDenied, "%s does not have permission %s in the realm %s", auth.CurrentIdentity(ctx), perm, realm)
	}
	logging.Infof(ctx, "%s has permission %s in the realm %s", auth.CurrentIdentity(ctx), perm, realm)
	return nil
}

// ToUFSRealm returns the realm name based on zone string.
func ToUFSRealm(zone string) string {
	ufsZone := ToUFSZone(zone)
	if ufsZone == ufspb.Zone_ZONE_UNSPECIFIED {
		return ""
	} else if IsInBrowserZone(ufsZone.String()) {
		return BrowserLabAdminRealm
	} else if ufsZone == ufspb.Zone_ZONE_CHROMEOS3 || ufsZone == ufspb.Zone_ZONE_CHROMEOS5 ||
		ufsZone == ufspb.Zone_ZONE_CHROMEOS7 || ufsZone == ufspb.Zone_ZONE_CHROMEOS15 {
		return AcsLabAdminRealm
	} else if ufsZone == ufspb.Zone_ZONE_SATLAB {
		// TODO(eshwar) : Create a new satlab realm and return it here.
		return ""
	}
	return AtlLabAdminRealm
}

// GetValidRealmName replaces the older Browser realm with newer realm
func GetValidRealmName(realm string) string {
	if realm != "" && realm == OldBrowserLabAdminRealm {
		return BrowserLabAdminRealm
	}
	return realm
}
