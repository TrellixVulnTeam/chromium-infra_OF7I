#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

# Get latest release tag through github API
github_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

CMD="$1"

latest_tflint=$(github_latest_release terraform-linters/tflint)
latest_tflint_gcp=$(github_latest_release terraform-linters/tflint-ruleset-google)

if [[ "${CMD}" == "latest" ]]; then
   echo "${latest_tflint}_${latest_tflint_gcp}"
   exit 0
fi

if [[ "${CMD}" = "get_url" ]]; then
  echo "{\"url\" : ["
  echo "  \"https://github.com/terraform-linters/tflint/archive/refs/tags/${latest_tflint}.tar.gz\","
  echo "  \"https://github.com/terraform-linters/tflint-ruleset-google/archive/refs/tags/${latest_tflint_gcp}.tar.gz\""
  echo " ],"
  echo " \"ext\" : \".tar.gz\""
  echo "}"
  exit 0
fi
