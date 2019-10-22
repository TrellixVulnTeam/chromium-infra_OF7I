# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import urllib2


def get_mstone(branch, raise_exception=True):
  milestones_json = urllib2.urlopen(
      'https://chromiumdash.appspot.com/fetch_milestones?num=5')
  recent_milestones = {m['chromium_branch']: m['milestone']
                       for m in json.load(milestones_json)}
  if not recent_milestones and raise_exception:
    raise Exception('Failed to fetch milestone data')
  return recent_milestones.get(str(branch), None)
