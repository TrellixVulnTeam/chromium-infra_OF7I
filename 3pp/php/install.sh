#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"
DEPS_PREFIX="$2"

./configure --prefix="$PREFIX" \
    --disable-cgi \
    --disable-cli \
    --with-apxs2="$DEPS_PREFIX/bin/apxs" \
    --with-zlib="$DEPS_PREFIX" \
    --without-iconv

make "-j$(nproc)"
make install
