// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This package contains constants for recoverylib, including task names for example.
// For more information, see b:208688399.
package tasknames

// TaskName describes which flow/plans will be involved in the process.
type TaskName string

const (
	// Task used to run auto recovery/repair flow in the lab.
	Recovery TaskName = "recovery"
	// Task used to prepare device to be used in the lab.
	Deploy TaskName = "deploy"
	// Task used to execute custom plans.
	// Configuration has to be provided by the user.
	Custom TaskName = "custom"
)
