// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

const deploy = "deploy"
const adminRepair = "admin_repair"
const adminReset = "admin_reset"

func TestUpdatesInventory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		task     string
		expected bool
	}{
		{deploy, true},
		{adminRepair, true},
		{adminReset, false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.task, func(t *testing.T) {
			t.Parallel()
			a := &args{}
			a.taskName = tc.task
			output := updatesInventory(a)
			if output != tc.expected {
				t.Errorf("Input task was %s - check was incorrect, got: %t, expected: %t", tc.task, output, tc.expected)
			}
		})
	}
}

func TestGetTaskName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		task     string
		expected string
	}{
		{adminRepair, repairTaskName},
		{deploy, deployTaskName},
		{adminReset, ""},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.task, func(t *testing.T) {
			t.Parallel()
			a := &args{}
			a.taskName = tc.task
			output := getTaskName(a)
			if output != tc.expected {
				t.Errorf("Input task was %s - taskName was incorrect, got: %s, expected: %s", tc.task, output, tc.expected)
			}
		})
	}
}

func TestIsDeployTask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		task     string
		expected bool
	}{
		{deploy, true},
		{adminRepair, false},
		{adminReset, false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.task, func(t *testing.T) {
			t.Parallel()
			a := &args{}
			a.taskName = tc.task
			output := isDeployTask(a)
			if output != tc.expected {
				t.Errorf("Input task was %s - check was incorrect, got: %t, expected: %t", tc.task, output, tc.expected)
			}
		})
	}
}

func TestIsRepairTask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		task     string
		expected bool
	}{
		{deploy, false},
		{adminRepair, true},
		{adminReset, false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.task, func(t *testing.T) {
			t.Parallel()
			a := &args{}
			a.taskName = tc.task
			output := isRepairTask(a)
			if output != tc.expected {
				t.Errorf("Input task was %s - check was incorrect, got: %t, expected: %t", tc.task, output, tc.expected)
			}
		})
	}
}
