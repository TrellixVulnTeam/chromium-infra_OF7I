// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package config is used temporarily as a kill switch when we disable automatic grouping.
// It should be removed after we finish disabling automatic grouping.
// We want to use this because the checking happen in multiple code paths.
package config

// EnableAutoGrouping is a flag to indicate whether failures should be automatically
// grouped by step name.
var EnableAutoGrouping = true
