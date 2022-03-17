# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import re

# Names of active well-known experiments
BBAGENT_GET_BUILD = 'luci.buildbucket.bbagent_getbuild'
BBAGENT_DOWNLOAD_CIPD = 'luci.buildbucket.agent.cipd_installation'
CANARY = 'luci.buildbucket.canary_software'
NON_PROD = 'luci.non_production'
USE_BBAGENT = 'luci.buildbucket.use_bbagent'

_VALID_NAME = re.compile(r'^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$')


def check_invalid_name(exp_name, well_known_experiments):
  """Returns an error message string if this is an invalid expirement.

  Returns None if `exp_name` is a valid experiment name."""
  if not _VALID_NAME.match(exp_name):
    return 'does not match %r' % _VALID_NAME.pattern

  if exp_name.startswith('luci.') and exp_name not in well_known_experiments:
    return 'unknown experiment has reserved prefix "luci."'

  return
