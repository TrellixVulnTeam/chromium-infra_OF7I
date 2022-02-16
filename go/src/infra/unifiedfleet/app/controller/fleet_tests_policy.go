// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc/codes"

	api "infra/unifiedfleet/api/v1/rpc"

	grpcStatus "google.golang.org/grpc/status"
)

const (
	// LUCI Auth group which is used to verify if a service account has permissions to run public Chromium tests in ChromeOS lab
	PublicUsersToChromeOSAuthGroup = "public-chromium-in-chromeos-builders"
)

// InvalidBoardError is the error raised when a private board is specified for a public test
type InvalidBoardError struct {
	Board string
}

func (e *InvalidBoardError) Error() string {
	return fmt.Sprintf("Cannnot run public tests on a private board : %s", e.Board)
}

// InvalidModelError is the error raised when a private model is specified for a public test
type InvalidModelError struct {
	Model string
}

func (e *InvalidModelError) Error() string {
	return fmt.Sprintf("Cannnot run public tests on a private model : %s", e.Model)
}

// InvalidImageError is the error raised when an invalid image is specified for a public test
type InvalidImageError struct {
	Image string
}

func (e *InvalidImageError) Error() string {
	return fmt.Sprintf("Cannnot run public tests on an image which is not allowlisted : %s", e.Image)
}

// InvalidTestError is the error raised when an invalid image is specified for a public test
type InvalidTestError struct {
	TestName string
}

func (e *InvalidTestError) Error() string {
	return fmt.Sprintf("Public user cannnot run the not allowlisted test : %s", e.TestName)
}

func IsValidTest(ctx context.Context, req *api.CheckFleetTestsPolicyRequest) error {
	logging.Infof(ctx, "Request to check from crosfleet: %s", req)
	logging.Infof(ctx, "Service account being validated: %s", auth.CurrentIdentity(ctx).Email())
	isMemberInPublicGroup, err := auth.IsMember(ctx, PublicUsersToChromeOSAuthGroup)
	if err != nil {
		// Ignoring error for now till we validate the service account membership check is correct
		logging.Errorf(ctx, "Request to check public chrome auth group membership failed: %s", err)
		return nil
	}

	if !isMemberInPublicGroup {
		return nil
	}

	// Validate if the board and model are public
	if req.Board == "" {
		return grpcStatus.Errorf(codes.InvalidArgument, "Invalid input - Board cannot be empty for public tests.")
	}
	if !contains(getValidPublicBoards(), req.Board) {
		return &InvalidBoardError{Board: req.Board}
	}
	if req.Model == "" {
		return grpcStatus.Errorf(codes.InvalidArgument, "Invalid input - Model cannot be empty for public tests.")
	}
	if !contains(getValidPublictModels(), req.Model) {
		return &InvalidModelError{Model: req.Model}
	}

	// Validate Test Name
	if req.TestName == "" {
		return grpcStatus.Errorf(codes.InvalidArgument, "Invalid input - Test name cannot be empty for public tests.")
	}
	if !contains(getValidPublicTestNames(), req.TestName) {
		return &InvalidTestError{TestName: req.TestName}
	}

	// Validate Image
	if req.Image == "" {
		return grpcStatus.Errorf(codes.InvalidArgument, "Invalid input - Image cannot be empty for public tests.")
	}
	if !contains(getValidPublicImages(), req.Image) {
		return &InvalidImageError{Image: req.Image}
	}

	return nil
}

func getValidPublicTestNames() []string {
	return []string{"tast.lacros"}
}

func getValidPublicBoards() []string {
	return []string{"eve", "kevin"}
}

func getValidPublictModels() []string {
	return []string{"eve", "kevin"}
}

func getValidPublicImages() []string {
	return []string{"R100-14495.0.0-rc1"}
}

func contains(listItems []string, name string) bool {
	for _, item := range listItems {
		if name == item {
			return true
		}
	}
	return false
}
