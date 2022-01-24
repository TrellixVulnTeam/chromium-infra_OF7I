#!/bin/bash
# Copyright 2022 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"

if [[ $PLATFORM == windows* ]]; then
    go build -o "${PREFIX}/esbuild.exe" ./cmd/esbuild
else
    go build -o "${PREFIX}/esbuild" ./cmd/esbuild
fi
