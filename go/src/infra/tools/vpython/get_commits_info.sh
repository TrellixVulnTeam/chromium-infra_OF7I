#!/bin/bash -e

# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

COMMIT_RANGE_START=$1
COMMIT_RANGE_END=$2

if [ -z "$1" ] || [ -z "$2" ]; then
  echo "USAGE: get_commits_info.sh <START_COMMIT> <END_COMMIT>"
  exit 1
fi

COMMIT_RANGE=$(git --no-pager log --pretty=format:"%H" ${COMMIT_RANGE_START}..${COMMIT_RANGE_END} go/src/infra/tools/vpython/)

DEPS_TEMP=$(mktemp)
get_luci_go_commit() {
  git show $1:DEPS > ${DEPS_TEMP}
  gclient getdep --deps-file=${DEPS_TEMP} --revision='infra/go/src/go.chromium.org/luci'
}

get_luci_vpython_log() {
  LUCI_GO_COMMIT_START=$(get_luci_go_commit $1)
  LUCI_GO_COMMIT_END=$(get_luci_go_commit $2)
  (cd go/src/go.chromium.org/luci && git --no-pager log --pretty=oneline ${LUCI_GO_COMMIT_START}..${LUCI_GO_COMMIT_END} vpython)
}

COMMIT_PREVIOUS=${COMMIT_RANGE_END}
for COMMIT_CURRENT in ${COMMIT_RANGE}; do
  get_luci_vpython_log ${COMMIT_CURRENT} ${COMMIT_PREVIOUS}
  git --no-pager show -s --pretty=oneline ${COMMIT_CURRENT}
  COMMIT_PREVIOUS=${COMMIT_CURRENT}
done
get_luci_vpython_log ${COMMIT_RANGE_START} ${COMMIT_PREVIOUS}