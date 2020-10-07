#!/bin/bash
# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Install the Python "distutilscross" package with patches.
#
# This script consumes:
# - ARCHIVE_PATH is the path to the "distutilscross" archive file.
# - distutilscross-py3.patch exists in the current directory.

# Load our installation utility functions.
. /install-util.sh

if [ -z "${ARCHIVE_PATH}" ]; then
  echo "ERROR: Missing required environment."
  exit 1
fi

ROOT=${PWD}

# Unpack our archive and enter its base directory (whatever it is named).
tar -xzf ${ARCHIVE_PATH}
cd $(get_archive_dir ${ARCHIVE_PATH})

patch -p1 < ../distutilscross-py3.patch

python3 setup.py install
