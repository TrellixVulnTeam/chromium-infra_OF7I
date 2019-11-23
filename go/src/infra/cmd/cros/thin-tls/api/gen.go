// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package api contains the thin-tls API.
// This API is not final or stable.
package api

//go:generate protoc service.proto --go_out=plugins=grpc:.
