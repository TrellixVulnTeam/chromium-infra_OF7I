#!/bin/bash
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"

# TODO(iannucci): Remove this (and the patch to enable this) once the fleet
# is using GLIBC 2.25 or higher. Currently the bulk of the fleet runs on
# Ubuntu 16.04, which as of this comment, uses GLIBC 2.23.
#
# This ALSO affects OS X on 10.11 and under when compiling with a newer version
# of XCode, EVEN if MACOSX_DEPLOYMENT_TARGET is 10.10.
#
# OpenSSL links against getentropy as a weak symbol... but unfortunately
# when we compile executables such as `git` and `python` against this static
# OpenSSL lib, the 'weakness' of this symbol is destroyed, and the linker
# immediately resolves it. On linux-amd64 this is not a problem, because we
# use the 'manylinux1' based docker containers, which have very old libc.
#
# However there's no manylinux equivalent for arm, and the Dockcross
# containers currently use a linux version which has a modern enough version
# of glibc to resolve getentropy, causing problems at runtime for
# linux-arm64 bots.
#
# When getentropy is not available, OpenSSL falls back to getrandom.
ARGS="-DNO_GETENTROPY=1"

case $_3PP_PLATFORM in
  mac-amd64)
    TARGET=darwin64-x86_64-cc
    ;;
  linux-*)
    TARGET="linux-${CROSS_TRIPLE%%-*}"
    ;;
  *)
    echo IDKWTF
    exit 1
    ;;
esac

perl Configure -lpthread --prefix="$PREFIX" no-shared $ARGS "$TARGET"

make -j "$(nproc)"
make install_sw
