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
build_input = properties['$recipe_engine/buildbucket']['build']['input']
cl_info = build_input['gerritChanges'][0]
if mode == 'ref':
  print('refs/changes/%d/%s/%s' % (
    int(cl_info['change'])%100, cl_info['change'], cl_info['patchset']))
elif mode == 'repo':
  # Get the git host url from the code review url
  # chromium-review.googlesource.com -> chromium.googlesource.com
  # chrome-internal-review.googlesource.com -> chrome-internal.googlesource.com
  host = cl_info['host'].replace('-review', '')
  repo = cl_info['project']
  print('https://%s/%s' % (host, repo))
