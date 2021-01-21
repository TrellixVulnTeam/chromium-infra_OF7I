# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import sys

mode = sys.argv[1]
assert mode in ('ref', 'repo')

d = json.load(sys.stdin)  # swarming task def
build_input = d['buildbucket']['bbagent_args']['build']['input']
cl_info = build_input['gerrit_changes'][0]
if mode == 'ref':
  print 'refs/changes/%d/%s/%s' % (int(cl_info['change']) % 100,
                                   cl_info['change'], cl_info['patchset'])
elif mode == 'repo':
  # Get the git host url from the code review url
  gerrit_host = cl_info['host']
  if gerrit_host == 'chromium-review.googlesource.com':
    git_host = 'chromium.googlesource.com'
  elif gerrit_host == 'chrome-internal-review.googlesource.com':
    git_host = 'chrome-internal.googlesource.com'
  else:
    print >> sys.stderr, 'Unknown gerrit host: %s' % gerrit_host
    sys.exit(1)

  git_repo = cl_info['project']
  print 'https://%s/%s' % (git_host, git_repo)
