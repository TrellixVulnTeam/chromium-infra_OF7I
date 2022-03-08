// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package linux

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.chromium.org/luci/common/errors"
)

type runnerResponse struct {
	output string
	err    error
}

var IsPathExistTests = []struct {
	testName string
	runnerResponse
	expectedErr error
}{
	{
		"Path Exist, no error",
		runnerResponse{"", nil},
		nil,
	},
	{
		"Path Not Exist, path exist error",
		runnerResponse{"", errors.Reason("runner: path not exist").Err()},
		errors.Reason("path exist: runner: path not exist").Err(),
	},
}

func TestIsPathExist(t *testing.T) {
	t.Parallel()
	for _, tt := range IsPathExistTests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			run := func(ctx context.Context, timeout time.Duration, cmd string, args ...string) (string, error) {
				if strings.HasPrefix(cmd, "test -e") {
					return tt.runnerResponse.output, tt.runnerResponse.err
				}
				return "", errors.Reason("runner: cmd not recognized").Err()
			}
			actualErr := IsPathExist(ctx, run, "test_path")
			if actualErr != nil && tt.expectedErr != nil {
				if !strings.Contains(actualErr.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
				}
			}
			if (actualErr == nil && tt.expectedErr != nil) || (actualErr != nil && tt.expectedErr == nil) {
				t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
			}
		})
	}
}

var PathHasEnoughValueTests = []struct {
	testName       string
	mapOfRunner    map[string]runnerResponse
	typeOfSpace    SpaceType
	minSpaceNeeded float64
	expectedErr    error
}{
	{
		"Path Exist, Enough Disk Storage, no error",
		map[string]runnerResponse{
			"test -e": {"", nil},
			"df -P":   {"/xxx/yyy/root         828753   164335      622245      21% /", nil},
		},
		SpaceTypeDisk,
		0.6,
		nil,
	},
	{
		"Path Exist, Enough Inode Storage, no error",
		map[string]runnerResponse{
			"test -e": {"", nil},
			"df -Pi":  {"/xxx/yyy/root         828753   164335      622245      21% /", nil},
		},
		SpaceTypeInode,
		100,
		nil,
	},
	{
		"Path Not Exist, Enough Disk Storage, path exist error",
		map[string]runnerResponse{
			"test -e": {"", errors.Reason("runner: path not exist").Err()},
			"df -P":   {"/xxx/yyy/root         828753   164335      622245      21% /", nil},
		},
		SpaceTypeDisk,
		0.6,
		errors.Reason("path exist").Err(),
	},
	{
		"Path Exist, Not Enough Disk Storage, no enough disk space error",
		map[string]runnerResponse{
			"test -e": {"", nil},
			"df -P":   {"/xxx/yyy/root         828753   164335      622245      21% /", nil},
		},
		SpaceTypeDisk,
		9999,
		errors.Reason("Not enough free disk").Err(),
	},
}

func TestPathHasEnoughValue(t *testing.T) {
	t.Parallel()
	for _, tt := range PathHasEnoughValueTests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			run := func(ctx context.Context, timeout time.Duration, cmd string, args ...string) (string, error) {
				for testCmd, res := range tt.mapOfRunner {
					if strings.HasPrefix(cmd, testCmd) {
						return res.output, res.err
					}
				}
				return "", errors.Reason("runner: cmd not recognized").Err()
			}
			actualErr := PathHasEnoughValue(ctx, run, "test_dut", "test_path", tt.typeOfSpace, tt.minSpaceNeeded)
			if actualErr != nil && tt.expectedErr != nil {
				if !strings.Contains(actualErr.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
				}
			}
			if (actualErr == nil && tt.expectedErr != nil) || (actualErr != nil && tt.expectedErr == nil) {
				t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
			}
		})
	}
}

var PathOccupiedSpacePercentageTests = []struct {
	testName           string
	mapOfRunner        map[string]runnerResponse
	expectedPercentage float64
	expectedErr        error
}{
	{
		"Path Exist, 21 percent occupied space, no error",
		map[string]runnerResponse{
			"test -e": {"", nil},
			"df":      {"/xxx/yyy/root         828753   164335      622245      21% /", nil},
		},
		21,
		nil,
	},
	{
		"Path Exist, runner error",
		map[string]runnerResponse{
			"test -e": {"", nil},
			"df":      {"/xxx/yyy/root         828753   164335      622245      21% /", errors.Reason("runner: df command not found").Err()},
		},
		-1,
		errors.Reason("path occupied space percentage: runner: df command not found").Err(),
	},
	{
		"Path Exist, convert float error",
		map[string]runnerResponse{
			"test -e": {"", nil},
			"df":      {"/xxx/yyy/root         828753   164335      622245      test% /", nil},
		},
		-1,
		errors.Reason("path occupied space percentage: strconv.ParseFloat").Err(),
	},
}

func TestPathOccupiedSpacePercentage(t *testing.T) {
	t.Parallel()
	for _, tt := range PathOccupiedSpacePercentageTests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			run := func(ctx context.Context, timeout time.Duration, cmd string, args ...string) (string, error) {
				for testCmd, res := range tt.mapOfRunner {
					if strings.HasPrefix(cmd, testCmd) {
						return res.output, res.err
					}
				}
				return "", errors.Reason("runner: cmd not recognized").Err()
			}
			actualPercentage, actualErr := PathOccupiedSpacePercentage(ctx, run, "test_path")
			if actualErr != nil && tt.expectedErr != nil {
				if !strings.Contains(actualErr.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
				}
			}
			if (actualErr == nil && tt.expectedErr != nil) || (actualErr != nil && tt.expectedErr == nil) {
				t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
			}
			if actualPercentage != tt.expectedPercentage {
				t.Errorf("Expected percentage: %v, but got: %v", tt.expectedPercentage, actualPercentage)
			}
		})
	}
}
