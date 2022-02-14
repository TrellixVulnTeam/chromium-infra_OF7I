#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"
DEPS_PREFIX="$2"

# Support zstd compression.
sed -i 's/#ZSTD_SUPPORT = 1/ZSTD_SUPPORT = 1/g' squashfs-tools/Makefile
sed -i "s@INCLUDEDIR = -I.@INCLUDEDIR = -I. -I$DEPS_PREFIX/include@g" squashfs-tools/Makefile
sed -i "s@LIBS = -lpthread -lm@LIBS = -lpthread -lm -L$DEPS_PREFIX/lib@g" squashfs-tools/Makefile

(cd squashfs-tools && make -j $(nproc))

PREFIX="$1"

cp -R * "$PREFIX"
