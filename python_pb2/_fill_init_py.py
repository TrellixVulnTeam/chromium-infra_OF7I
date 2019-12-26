# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import sys

# Ensure __init__.py file in each directory with _pb2 files.
# See Makefile nearby.
for dirpath, dirnames, filenames in os.walk(sys.argv[1]):
  if not dirnames and not filenames:
    continue
  if '__init__.py' in filenames:
    continue
  with open(os.path.join(dirpath, '__init__.py'), 'w'):
    pass
