# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os

from . import source
from . import builder
from . import util

from .build_types import Spec
from .builder import Builder


class InfraPackage(Builder):
  def __init__(self, name):
    """Wheel builder for pure Python wheels built from the current local repo's
       packages folder.
    """
    self._resolved_version = None
    super(InfraPackage, self).__init__(
        Spec(
            name,
            None,
            universal=True,
            pyversions=['py2'],
            default=True,
            patch_version=None,
        ))

  def version_fn(self, system):
    if self._resolved_version is None:
      pkg_path = os.path.join(
          os.path.dirname(__file__), '..', '..', '..', 'packages',
          self._spec.name)
      _, self._resolved_version = util.check_run(
          system,
          None,
          '.',
          ['python', 'setup.py', '--version'],
          cwd=pkg_path,
      )
    return self._resolved_version

  def build_fn(self, system, wheel):
    path = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'packages',
        self._spec.name)
    src = source.local_directory(self._spec.name, wheel.spec.version, path)
    return builder.BuildPackageFromSource(system, wheel, src)
