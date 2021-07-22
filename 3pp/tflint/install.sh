#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

mkdir tflint
tar -xvf raw_source_0.tar.gz -C tflint --strip-components=1

mkdir tflint_ruleset_google
tar -xvf raw_source_1.tar.gz -C tflint_ruleset_google --strip-components=1

PREFIX=$(realpath "$1")

# Build tflint binary
(
    cd tflint
    mkdir dist
    go build -o dist/tflint
    cp dist/tflint "${PREFIX}"/tflint.bin

    # Build wrapper around tflint to set the plugin directory to ours
    mkdir "${PREFIX}"/plugins
    cat <<EOF> "${PREFIX}"/tflint
#!/usr/bin/env bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

SCRIPT_DIR="\$( cd "\$( dirname "\${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
TFLINT_PLUGIN_DIR="\${SCRIPT_DIR}"/plugins "\${SCRIPT_DIR}"/tflint.bin "\$@"
EOF

    chmod a+x "${PREFIX}"/tflint
)

# Download and build rulesets to bundle with the package
(
    cd tflint_ruleset_google
    go build
    mv tflint-ruleset-google "${PREFIX}"/plugins
)
