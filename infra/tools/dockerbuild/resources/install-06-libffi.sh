#!/bin/bash
# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Build host libffi, needed for python3.
#
# This script consumes:
# - ARCHIVE_PATH is the path to the libffi archive file.
#
# This script expects to be called in a host build environment.

# Load our installation utility functions.
. /install-util.sh

if [ -z "${ARCHIVE_PATH}" ] ; then
  echo "ERROR: Missing required environment."
  exit 1
fi

# Unpack our archive and enter its base directory (whatever it is named).
tar -xzf "${ARCHIVE_PATH}"
cd "$(get_archive_dir "${ARCHIVE_PATH}")" || exit 1

##
# Build Host libffi
##

toggle_host

./autogen.sh
./configure \
  --prefix="${LOCAL_PREFIX}" \
  --disable-shared
make -j"$(nproc)"
make install
