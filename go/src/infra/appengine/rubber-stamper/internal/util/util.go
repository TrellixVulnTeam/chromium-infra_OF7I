// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"context"
	"fmt"

	"cloud.google.com/go/errorreporting"
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

// SetupErrorReporting sets up an ErrorReporting client in the context.
func SetupErrorReporting(ctx context.Context) (context.Context, error) {
	// Get the app's appId.
	signer := auth.GetSigner(ctx)
	if signer == nil {
		return nil, errors.New("failed to get the Signer instance representing the service")
	}
	info, err := signer.ServiceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get service info: %v", err)
	}
	errorClient, err := errorreporting.NewClient(ctx, info.AppID, errorreporting.Config{
		ServiceName: "default",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create error reporting client: %v", err)
	}

	return context.WithValue(ctx, &errorReportingClientCtxKey, errorClient), nil
}

// GetErrorReportingClient gets the ErrorReporting client from the context.
// Returns nil when there's no ErrorReporting client.
func GetErrorReportingClient(ctx context.Context) *errorreporting.Client {
	client, _ := ctx.Value(&errorReportingClientCtxKey).(*errorreporting.Client)
	return client
}
