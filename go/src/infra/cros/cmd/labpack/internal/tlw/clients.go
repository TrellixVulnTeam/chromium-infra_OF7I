// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tlw

import (
	"context"
	"net/http"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cros/cmd/labpack/internal/site"
	steps "infra/cros/cmd/labpack/internal/steps"
	"infra/cros/recovery"
	"infra/cros/recovery/tlw"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// NewAccess creates TLW Access for recovery engine.
func NewAccess(ctx context.Context, in *steps.LabpackInput) (tlw.Access, error) {
	hc, err := httpClient(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "create tlw access: create http client").Err()
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    in.InventoryService,
		Options: site.UFSPRPCOptions,
	})
	csac := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       hc,
			Host:    in.AdminService,
			Options: site.DefaultPRPCOptions,
		},
	)
	// TODO(otabek@): Replace with access to F20 services.
	access, err := recovery.NewLocalTLWAccess(ic, csac)
	return access, errors.Annotate(err, "create tlw access").Err()
}

// httpClient returns an HTTP client with authentication set up.
func httpClient(ctx context.Context) (*http.Client, error) {
	o := auth.Options{
		Method: auth.LUCIContextMethod,
		Scopes: []string{
			auth.OAuthScopeEmail,
			"https://www.googleapis.com/auth/cloud-platform",
		},
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, o)
	c, err := a.Client()
	return c, errors.Annotate(err, "create http client").Err()
}
