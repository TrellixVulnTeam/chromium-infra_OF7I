#!/bin/bash
# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
cd "$(dirname "$0")"

if [ $# != 1 ]; then
    echo "usage: $ $0 <version>"
    echo "e.g. $ $0 1.24.1"
    exit 1
fi

version="${1}"

find . | grep -F '/' | grep -F -v './update.sh' | grep -F -v 'README.monorail' | \
    sort -r | xargs rm -r
curl -sL https://files.pythonhosted.org/packages/b3/17/1e567ff78c83854e16b98694411fe6e08c3426af866ad11397cddceb80d3/redis-"${version}".tar.gz | \
    tar xvz --strip-components 2 redis-"${version}"/redis

