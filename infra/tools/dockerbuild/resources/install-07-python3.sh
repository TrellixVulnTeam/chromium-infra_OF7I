#!/bin/bash
# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Build host Python3 installation. It is installed as an "alternate"
# Python interpreter, that is, it must be called as "python3".
#
# This script consumes:
# - ARCHIVE_PATH is the path to the Python archive file.
# - CROSS_CONFIG_SITE is the path to the "config.site" file to use for
#   cross-compiling.
#
# This script expects to be called in a host build environment.

# Load our installation utility functions.
. /install-util.sh

if \
  [ -z "${ARCHIVE_PATH}" ] || \
  [ -z "${CROSS_CONFIG_SITE}" ]; then
  echo "ERROR: Missing required environment."
  exit 1
fi

# Resolve CROSS_CONFIG_SITE to absolute path, since we will reference it after
# we chdir.
CROSS_CONFIG_SITE=$(abspath ${CROSS_CONFIG_SITE})

# Unpack our archive and enter its base directory (whatever it is named).
tar -xzf "${ARCHIVE_PATH}"
cd "$(get_archive_dir "${ARCHIVE_PATH}")" || exit 1

##
# Build Host Python3
##

toggle_host

./configure \
  --prefix="${LOCAL_PREFIX}"
make -j"$(nproc)"
make install

##
# Build Cross-compile Python3
##
toggle_cross

make clean

CONFIG_SITE=${CROSS_CONFIG_SITE} READELF=`which readelf` \
  ./configure \
  --prefix="${CROSS_PREFIX}" \
  --host=${CROSS_TRIPLE} \
  --build=$(gcc -dumpmachine) \
  --disable-ipv6
make -j$(nproc)

make install
