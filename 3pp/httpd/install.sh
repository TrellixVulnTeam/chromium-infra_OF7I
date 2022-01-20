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
    --enable-access-compat=shared \
    --enable-actions=shared \
    --enable-alias=shared \
    --enable-asis=shared \
    --enable-authz-core=shared \
    --enable-authz-host=shared \
    --enable-autoindex=shared \
    --enable-cgi=shared \
    --enable-env=shared \
    --enable-headers=shared \
    --enable-imagemap=shared \
    --enable-include=shared \
    --enable-log-config=shared \
    --enable-mime=shared \
    --enable-modules=none \
    --enable-negotiation=shared \
    --enable-rewrite=shared \
    --enable-ssl=shared \
    --enable-unixd=shared \
    --libexecdir="$PREFIX/libexec/apache2" \
    --with-apr-util="$DEPS_PREFIX" \
    --with-apr="$DEPS_PREFIX" \
    --with-mpm=prefork \
    --with-pcre="$DEPS_PREFIX" \
    --with-ssl="$DEPS_PREFIX"

make "-j$(nproc)"
make install
