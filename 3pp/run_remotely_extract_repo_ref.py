# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import sys

mode = sys.argv[1]
assert mode in ('ref', 'repo')

d = json.load(sys.stdin)  # swarming task def
cmdline = d['task_slices'][0]['properties']['command']
for i, arg in enumerate(cmdline):
  if arg == '-properties':
    properties = json.loads(cmdline[i+1])
    break
if mode == 'ref':
  build_input = properties['$recipe_engine/buildbucket']['build']['input']
  ref = build_input['gerritChanges'][0]
  print('refs/changes/%d/%s/%s' % (
    int(ref['change'])%100, ref['change'], ref['patchset']))
elif mode == 'repo':
  print(properties['repository'])
