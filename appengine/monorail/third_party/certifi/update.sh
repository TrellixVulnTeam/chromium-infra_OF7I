#!/bin/bash
# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
cd "$(dirname "$0")"

if [ $# != 1 ]; then
    echo "usage: $ $0 <version>"
    echo "e.g. $ $0 2018.11.29"
    exit 1
fi

version="${1}"

find . | grep -F '/' | grep -F -v './update.sh' | grep -F -v 'README.monorail' | \
    sort -r | xargs -r rm -r
curl -sL URL: https://github.com/certifi/python-certifi/archive/"${version}".tar.gz | \
    tar xvz --strip-components 2
