#!/bin/bash
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"

# Don't have libtool hardcode the path to 'grep', since it varies across
# some of our docker images. Instead, just trust grep from $PATH.
GREP=grep ./configure --enable-static --disable-shared \
  --prefix "$PREFIX" \
  --host "$CROSS_TRIPLE"
make install -j $(nproc)
