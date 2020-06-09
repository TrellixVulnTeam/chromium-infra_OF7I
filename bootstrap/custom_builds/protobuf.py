# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import subprocess
import sys


# Should match protobuf lib version in deps.pyl.
PROTOC_VERSION = '3.12.1'


def Build(source_path, wheelhouse_path):
  # Need a protoc of the same version in PATH already. Use `go/env.py` to grab
  # it from CIPD.
  ver = ''
  try:
    ver = subprocess.check_output(['protoc', '--version']).strip()
  except OSError:
    pass
  if ver != 'libprotoc %s' % PROTOC_VERSION:
    raise ValueError('Need protoc v%s in PATH' % PROTOC_VERSION)

  # This uses protoc in PATH to compile *.proto.
  cwd = os.path.join(source_path, 'python')
  subprocess.check_call(
      [
          'python', 'setup.py', 'bdist_wheel', '--universal',
          '--dist-dir', wheelhouse_path,
      ],
      cwd=cwd)
