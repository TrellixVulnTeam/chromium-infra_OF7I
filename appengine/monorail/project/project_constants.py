# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Some constants used for managing Monorail Projects."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import re

PROJECT_NAME_PATTERN = '[a-z0-9][-a-z0-9]*[a-z0-9]'

MAX_PROJECT_NAME_LENGTH = 63

# Pattern to match a valid project name.  Users of this pattern MUST use
# the re.VERBOSE flag or the whitespace and comments we be considered
# significant and the pattern will not work.  See "re" module documentation.
_RE_PROJECT_NAME_PATTERN_VERBOSE = r"""
  (?=[-a-z0-9]*[a-z][-a-z0-9]*)   # Lookahead to make sure there is at least
                                  # one letter in the whole name.
  [a-z0-9]                        # Start with a letter or digit.
  [-a-z0-9]*                      # Follow with any number of valid characters.
  [a-z0-9]                        # End with a letter or digit.
"""

# Compiled regexp to match the project name and nothing more before or after.
RE_PROJECT_NAME = re.compile(
    '^%s$' % _RE_PROJECT_NAME_PATTERN_VERBOSE, re.VERBOSE)
