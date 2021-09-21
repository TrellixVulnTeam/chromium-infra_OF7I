#!/bin/bash
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"
DEPS_PREFIX="$2"

DEST=$PREFIX/virtualenv-$_3PP_VERSION
mkdir $DEST
cp -R * $DEST  # Excludes ".git"

# Install our newer wheels.
rm -f $DEST/virtualenv_support/*.whl
cp -f $DEPS_PREFIX/*.whl $DEST/virtualenv_support/
