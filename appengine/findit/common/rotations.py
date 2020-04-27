# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Utilities to get the sheriff(s) on duty.

Currently only supports the Chrome Build Sheriff rotation."""

import json

from common.findit_http_client import FinditHttpClient
from libs import time_util
from model.wf_config import FinditConfig

_ROTATIONS_URL = 'https://rota-ng.appspot.com/legacy/sheriff.json'

_HTTP_CLIENT = FinditHttpClient()


def current_sheriffs():
  status_code, content, _headers = _HTTP_CLIENT.Get(_ROTATIONS_URL)
  if status_code == 200:
    content = json.loads(content)
    if 'emails' not in content:
      raise Exception('Malformed sheriff.json at %s' % _ROTATIONS_URL)
    return content['emails']
  else:
    raise Exception('Could not retrieve sheriff list from %s, got code %d' %
                    (_ROTATIONS_URL, status_code))
