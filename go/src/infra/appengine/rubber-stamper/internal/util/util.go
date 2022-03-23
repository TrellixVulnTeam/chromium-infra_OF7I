// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/auth"
)

var errorReportingClientCtxKey = "infra/appengine/rubber-stamper/internal/util/ErrorReportingClient"

// GetServiceAccountName returns the app's service account name.
func GetServiceAccountName(ctx context.Context) (string, error) {
	signer := auth.GetSigner(ctx)
	if signer == nil {
		return "", errors.New("failed to get the Signer instance representing the service")
	}
	info, err := signer.ServiceInfo(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get service info: %v", err)
	}
	return info.ServiceAccountName, nil
}
