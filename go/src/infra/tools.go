// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build tools

package main

import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/smartystreets/goconvey"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/stringer"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"

	_ "go.chromium.org/luci/gae/tools/proto-gae"
	_ "go.chromium.org/luci/grpc/cmd/cproto"
	_ "go.chromium.org/luci/grpc/cmd/svcdec"
	_ "go.chromium.org/luci/grpc/cmd/svcmux"
	_ "go.chromium.org/luci/tools/cmd/assets"

	_ "infra/cmd/bqexport"

	// Used exclusively to build a CIPD package out of it.
	_ "go.skia.org/infra/gold-client/cmd/goldctl" // noinstall
)
