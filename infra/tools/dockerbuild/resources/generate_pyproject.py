# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file
"""Generate and overwrite the pyproject.toml
This script is used for generating pyproject.toml to specify build system
requirements (PEP 518). It can't be done by patching the source code because
PEP 508 only allow specify dependencies by url, which means we must determine
the absolute path during runtime for 'file://'.

--remotes will generate requirements which will be fetched from pip repository
--locals will generate requirements which will be installed from local path
"""

import argparse
import os
import pathlib

__PYPROJECT_TOML = """
[build-system]
requires = [ {} ]
"""

if __name__ == "__main__":
  parser = argparse.ArgumentParser()
  # TODO(fancl): We should remove '--remotes' eventually and having all
  # dependencies available locally
  parser.add_argument('--remotes', nargs='*', default=[])
  parser.add_argument('--locals', nargs='*', default=[])
  args = parser.parse_args()

  deps = args.remotes
  for dep in args.locals:
    name, path = dep.split('@')
    path = os.path.join(os.path.abspath(path))
    deps.append('@'.join((name, pathlib.Path(path).as_uri())))

  with open('pyproject.toml', 'w') as f:
    f.write(__PYPROJECT_TOML.format(','.join('\'{}\''.format(d) for d in deps)))