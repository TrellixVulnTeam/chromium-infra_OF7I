#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"

export CXXFLAGS+=" -fPIC -std=c++11"
./autogen.sh

./configure --enable-static --disable-shared \
  --prefix="$PREFIX" \
  --host="$CROSS_TRIPLE"
make -j $(nproc)
if [[ $_3PP_PLATFORM == $_3PP_TOOL_PLATFORM ]]; then
  make check
fi
make install -j $(nproc)
