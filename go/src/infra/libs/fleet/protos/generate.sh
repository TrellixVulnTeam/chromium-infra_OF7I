#!/bin/bash -e
#
# Copyright 2020 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
#
# Runs protoc over the protos in this repo to produce generated proto code.
# protoc and protoc-gen-go are already installed in chops go env.

# Clean up existing generated files in path go/.
find go -name '*.pb.go' -exec rm '{}' \;

# Generate go file for files in path src/.
protoc -Isrc --go_out=paths=source_relative:go src/*.proto
