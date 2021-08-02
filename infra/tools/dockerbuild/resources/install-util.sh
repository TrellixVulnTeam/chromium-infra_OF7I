# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Generic utility functions used by other installation scripts.

set -e
set -x
set -o pipefail

# Ensure that our mandatory environment variables are set:
#
# - LOCAL_PREFIX is the prefix to use for host installation.
# - CROSS_TRIPLE is the cross-compile host triple.
if [ -z "${LOCAL_PREFIX}" -o -z "${CROSS_TRIPLE}" ]; then
  echo "ERROR: Missing required environment."
  exit 1
fi

# Dockcross doesn't set OBJCOPY, so set it here to an appropriate binary.
# Unfortunately, the location is not the same across all of our images.
_OBJCOPY_LOCATIONS=(
  ${CROSS_ROOT}/bin/${CROSS_TRIPLE}-objcopy
  ${CROSS_ROOT}/bin/objcopy
  ${CROSS_ROOT}/objcopy
)
for location in ${_OBJCOPY_LOCATIONS[@]}; do
  if [ -x ${location} ]; then
    OBJCOPY=${location}
    break
  fi
done
export OBJCOPY

# Augment our PATH to include our local prefix's "bin" directory.
export PATH=${LOCAL_PREFIX}/bin:${PATH}

# Snapshot the cross-compile environment so we can toggle between them.
CROSS_AS=$AS
CROSS_AR=$AR
CROSS_CC=$CC
CROSS_CPP=$CPP
CROSS_CXX=$CXX
CROSS_LD=$LD
CROSS_OBJCOPY=$OBJCOPY

# Create and augment our CFLAGS and LDFLAGS.
CROSS_CFLAGS="$CFLAGS"
CROSS_LDFLAGS="$LDFLAGS"
CROSS_PYTHONPATH="$PYTHONPATH"
CROSS_CMAKE_TOOLCHAIN_FILE="$CMAKE_TOOLCHAIN_FILE"

CROSS_C_INCLUDE_PATH="$C_INCLUDE_PATH"
CROSS_CPLUS_INCLUDE_PATH="$CPLUS_INCLUDE_PATH"
CROSS_LD_LIBRARY_PATH="$LD_LIBRARY_PATH"
CROSS_LIBRARY_PATH="$LIBRARY_PATH"
CROSS_PKG_CONFIG_PATH="$PKG_CONFIG_PATH"

toggle_host() {
  AS=
  AR=
  CC=
  CPP=
  CXX=
  LD=
  OBJCOPY=

  CFLAGS=
  # Some tools ignore pkg-config flags, so set this explicitly to make sure
  # libffi and any other libraries here are locatable.
  LDFLAGS=-L${LOCAL_PREFIX}/lib64
  PYTHONPATH=
  CMAKE_TOOLCHAIN_FILE=

  C_INCLUDE_PATH=
  CPLUS_INCLUDE_PATH=
  LD_LIBRARY_PATH=
  LIBRARY_PATH=
  PKG_CONFIG_PATH=${LOCAL_PREFIX}/lib/pkgconfig
}

toggle_cross() {
  AS=${CROSS_AS}
  AR=${CROSS_AR}
  CC=${CROSS_CC}
  CPP=${CROSS_CPP}
  CXX=${CROSS_CXX}
  LD=${CROSS_LD}
  OBJCOPY=${CROSS_OBJCOPY}

  CFLAGS="${CROSS_CFLAGS}"
  LDFLAGS="${CROSS_LDFLAGS}"
  PYTHONPATH="${CROSS_PYTHONPATH}"
  CMAKE_TOOLCHAIN_FILE="${CROSS_CMAKE_TOOLCHAIN_FILE}"

  C_INCLUDE_PATH="${CROSS_C_INCLUDE_PATH}"
  CPLUS_INCLUDE_PATH="${CROSS_CPLUS_INCLUDE_PATH}"
  LD_LIBRARY_PATH="${CROSS_LD_LIBRARY_PATH}"
  LIBRARY_PATH="${CROSS_LIBRARY_PATH}"
  PKG_CONFIG_PATH="${CROSS_PKG_CONFIG_PATH}"
}

abspath() {
  local P=$1; shift
  echo $(readlink -f ${P})
}

get_archive_dir() {
  local ARCHIVE_PATH=$1; shift
  echo $(tar tf ${ARCHIVE_PATH} | head -n1 | sed  's#\(.*\)/.*#\1#')
}

if [ ! `which nproc` ]; then
  nproc() {
    grep processor < /proc/cpuinfo | wc -l
  }
fi
