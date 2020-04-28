# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This empty vpython spec prevents the test_bad_cwd.py script from ever
# having a vpython spec, even if this repo eventually adds one.
# [VPYTHON:BEGIN]
#
# [VPYTHON:END]

try:
  import six
  raise Exception('Script A imported \'six\', but has an empty venv!')
except ImportError:
  pass

import os
import subprocess
import sys

MY_DIR = os.path.dirname(os.path.abspath(__file__))

os.chdir(os.path.join(MY_DIR, "bad_cwd"))

print "I'm in script A"
sys.stdout.flush()

VPYTHON = os.getenv('_VPYTHON_MAIN_TEST_BINARY')
if not VPYTHON: # so we can run the test by hand
  VPYTHON = 'vpython.bat' if sys.platform.startswith('win') else 'vpython'
else:
  # so the the test binary behaves like vpython.exe
  os.putenv('_VPYTHON_MAIN_TEST_PASSTHROUGH', '1')

# Prior to the fix for crbug.com/1074636, this would fail because vpython
# wouldn't properly isolate itself from the current working directory during the
# creation of the virtualenv.

sys.stdout.write("vpython (expect SUCCESS): ")
sys.stdout.flush()
subprocess.call([VPYTHON, "math.py"])
