// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"infra/cmdsupport/cmdlib"
	ufsutil "infra/unifiedfleet/app/util"
	"strings"

	"google.golang.org/grpc/metadata"

	"go.chromium.org/luci/auth/client/authcli"
	swarmingapi "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/api/option"
)

const swarmingAPISuffix = "_ah/api/swarming/v1/"

// newSwarmingService returns a new service to interact with the Swarming API.
func newSwarmingService(ctx context.Context, swarmingServicePath string, authFlags *authcli.Flags) (*swarmingapi.Service, error) {
	httpClient, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return nil, err
	}
	swarmingService, err := swarmingapi.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}
	swarmingService.BasePath = swarmingServicePath + swarmingAPISuffix
	return swarmingService, nil
}

// hostnameToBotID returns the bot ID for a given DUT hostname.
func hostnameToBotID(ctx context.Context, swarmingService *swarmingapi.Service, hostname string) (string, error) {
	hostnameDim := fmt.Sprintf("dut_name:%s", hostname)
	botsListReply, err := swarmingService.Bots.List().Context(ctx).Dimensions(hostnameDim).Do()
	if err != nil {
		return "", err
	}
	bots := botsListReply.Items
	if len(bots) == 0 {
		return "", errors.Reason(fmt.Sprintf("Invalid host %s: no associated Swarming bots found", hostname)).Err()
	}
	return bots[0].BotId, nil
}

// countBotsWithDims returns the number of Swarming bots satisfying the given
// Swarming dimensions.
func countBotsWithDims(ctx context.Context, s *swarmingapi.Service, dimsMap map[string]string) (*swarmingapi.SwarmingRpcsBotsCount, error) {
	var dims []string
	for key, val := range dimsMap {
		dims = append(dims, fmt.Sprintf("%s:%s", key, val))
	}
	return s.Bots.Count().Context(ctx).Dimensions(dims...).Do()
}

// correctedHostname checks the given hostname for common errors when entering a
// DUT hostname, and returns a corrected hostname.
func correctedHostname(hostname string) string {
	hostname = strings.TrimPrefix(hostname, "crossk-")
	hostname = strings.TrimSuffix(hostname, ".cros")
	return hostname
}

// contextWithOSNamespace adds an "os" namespace to the given context, which
// is required for API calls to UFS.
func contextWithOSNamespace(ctx context.Context) context.Context {
	osMetadata := metadata.Pairs(ufsutil.Namespace, ufsutil.OSNamespace)
	return metadata.NewOutgoingContext(ctx, osMetadata)
}
