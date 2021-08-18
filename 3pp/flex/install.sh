#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"

./autogen.sh

if [[ $_3PP_TOOL_PLATFORM != $_3PP_PLATFORM ]]; then
  # Bootstrap doesn't work correctly for our cross-compiles, but it's
  # not necessary.
  EXTRA_OPTS="--disable-bootstrap"
fi

./configure --enable-static --disable-shared \
  --prefix="$PREFIX" \
  --host="$CROSS_TRIPLE" $EXTRA_OPTS
make -j $(nproc)
make install -j $(nproc)
