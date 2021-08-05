#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"
DEPS_PREFIX="$2"

make PROTOBUF_CXXFLAGS="-I$DEPS_PREFIX/include" \
  PROTOBUF_LDFLAGS="-L$DEPS_PREFIX/lib -lprotobuf" \
  LIBNL_CXXFLAGS="-I$DEPS_PREFIX/include/libnl3" \
  LIBNL_LDFLAGS="-lnl-3 -lnl-route-3"

cp ./nsjail "$PREFIX"
