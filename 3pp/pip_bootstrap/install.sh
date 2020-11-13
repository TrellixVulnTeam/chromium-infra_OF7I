#!/bin/bash
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

# Rename downloaded wheels to original filenames. `pip_bootstrap.py` installer
# script looks for python wheel with `pip-*`, whenever pip_bootstrap is used
# for another package (e.g. cpython3), cipd should return these wheel files
# with formatted names.
mv raw_source_0.whl pip-source.whl
mv raw_source_1.whl setuptools-source.whl
mv raw_source_2.whl wheel-source.whl

# Copy all wheel files and bootstrap script
cp -a ./* "$SCRIPT_DIR/pip_bootstrap.py" "$PREFIX"
