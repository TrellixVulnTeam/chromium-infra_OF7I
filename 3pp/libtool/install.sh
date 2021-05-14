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
#
# Similarly, the path to 'sed' (even when specified as a tool: dependency)
# is not a constant, as it depends on the package being built. Search $PATH
# for sed as well.
GREP=grep SED=sed ./configure --enable-static --disable-shared \
  --prefix "$PREFIX" \
  --host "$CROSS_TRIPLE"
make install -j $(nproc)
