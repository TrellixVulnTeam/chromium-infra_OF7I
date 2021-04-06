# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import re

# Names of well-known experiments
CANARY = 'luci.buildbucket.canary_software'
USE_BBAGENT = 'luci.buildbucket.use_bbagent'
NON_PROD = 'luci.non_production'
USE_REALMS = 'luci.use_realms'

# TODO(iannucci): remove this when uses of this experiment rename to a
# different experiment.
USE_RBE_CAS = 'luci.swarming.use_rbe_cas'

WELL_KNOWN = frozenset([
    CANARY,
    NON_PROD,
    USE_BBAGENT,
    USE_REALMS,
])

_VALID_NAME = re.compile(r'^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$')


def check_invalid_name(exp_name):
  """Returns an error message string if this is an invalid expirement.

  Returns None if `exp_name` is a valid experiment name."""
  # TODO(iannucci): Remove this when USE_RBE_CAS is removed.
  if exp_name == USE_RBE_CAS:
    return

  if not _VALID_NAME.match(exp_name):
    return 'does not match %r' % _VALID_NAME.pattern

  if exp_name.startswith('luci.') and exp_name not in WELL_KNOWN:
    return 'unknown experiment has reserved prefix "luci."'

  return
