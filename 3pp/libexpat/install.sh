#!/bin/bash
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"
DEPS_PREFIX=$2

cd ./expat

PATH=$DEPS_PREFIX/bin:$PATH ./buildconf.sh
./configure --prefix="$PREFIX" --enable-shared=no --host "$CROSS_TRIPLE"
make install
