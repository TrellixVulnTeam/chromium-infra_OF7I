// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package api contains the fleet service API.
package api

//go:generate cproto -proto-path ../../../../libs/fleet/protos/src
//go:generate svcdec -type RegistrationServer
//go:generate svcdec -type ConfigurationServer
