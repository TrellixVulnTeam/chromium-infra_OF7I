#!/bin/bash

# Copyright 2021 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -o pipefail

msg() {
        1>&2 printf '%s\n' "$@"
}

die() {
        1>&2 printf '%s\n' "$@"
        exit 1
}

msg 'BEGIN discover services'
prpc show localhost:8800 || die 'failed to discover services'
msg 'END discover services'

msg 'BEGIN describe karte'
# Manually exercise the describe endpoint over HTTP.
(curl -X POST 'http://localhost:8800/prpc/discovery.Discovery/Describe' --output - | cat -v)|| die 'failed to discover services'
msg 'END describe karte'

# Create the data associated with a CreateAction RPC call.
cat 1>/tmp/karte.json <<EOF
{
        "action": {
                "kind": "aaaaa"
        }
}
EOF

msg 'BEGIN create action'
# Create an action using the CreateAction RPC endpoint.
(curl -X POST \
        -H "Content-Type: application/json" \
        --data-binary @/tmp/karte.json \
        'http://localhost:8800/prpc/chromeos.karte.Karte/CreateAction' --output - | cat -v) || die 'failed to curl karte prpc'
msg 'END create action'
