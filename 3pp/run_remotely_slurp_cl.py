# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import print_function

import json
import sys
import six.moves.urllib

# pylint: disable=line-too-long

d = json.load(sys.stdin)
if not d['issue_url']:
  print("Failed to get Gerrit CL associated with this repo.", file=sys.stderr)
  print(
      "Ensure that you've run `git cl upload` before using run_remotely.sh",
      file=sys.stderr)
  sys.exit(1)

# Print the final URL
print(six.moves.urllib.requests.urlopen(d['issue_url']).geturl())
