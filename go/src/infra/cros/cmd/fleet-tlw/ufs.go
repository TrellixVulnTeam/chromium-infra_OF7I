// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"

	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"
)

// getUFSDeviceLicenses looks up the licenses of a DUT based on the name
// specified.
func getUFSDeviceLicenses(ctx context.Context, ufsClient ufsapi.FleetClient, hostname string) ([]*tls.License, error) {
	lse, err := ufsClient.GetMachineLSE(ctx, &ufsapi.GetMachineLSERequest{
		Name: ufsutil.AddPrefix(ufsutil.MachineLSECollection, hostname),
	})
	if err != nil {
		logging.Infof(ctx, "GetUFSDeviceLicense: error when querying MachineLSE for %s", hostname)
		return nil, errors.Reason("no MachineLSE found for hostname %s", hostname).Tag(grpcutil.NotFoundTag).Err()
	}
	licenses := lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetLicenses()
	return convertLicenses(licenses), nil
}

// convertLicenses converts UFS Licenses to TLS Licenses proto format.
func convertLicenses(licenses []*lab.License) []*tls.License {
	var tlsLicenses []*tls.License
	for _, l := range licenses {
		tlsLicenses = append(tlsLicenses, &tls.License{
			Name: l.GetIdentifier(),
			Type: convertLicenseType(l.GetType()),
		})
	}
	return tlsLicenses
}

// convertLicenseType converts UFS LicenseType to TLS
// License_Type proto format.
func convertLicenseType(lt lab.LicenseType) tls.License_Type {
	switch lt {
	case lab.LicenseType_LICENSE_TYPE_WINDOWS_10_PRO:
		return tls.License_WINDOWS_10_PRO
	case lab.LicenseType_LICENSE_TYPE_MS_OFFICE_STANDARD:
		return tls.License_MS_OFFICE_STANDARD
	default:
		return tls.License_TYPE_UNSPECIFIED
	}
}
