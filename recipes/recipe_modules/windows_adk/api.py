# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import subprocess

from recipe_engine import recipe_api
from recipe_engine.recipe_api import Property

WIN_ADK_PACKAGE = 'infra/3pp/tools/win_adk/windows-amd64'


class WindowsADKApi(recipe_api.RecipeApi):
  """API for using Windows ADK distributed via CIPD."""

  def __init__(self, win_adk_refs, *args, **kwargs):
    super(WindowsADKApi, self).__init__(*args, **kwargs)

    self._win_adk_refs = win_adk_refs

  def ensure(self, install=True):
    """Ensure the presence of the Windows ADK."""
    with self.m.step.nest('ensure windows adk present'):
      with self.m.context(infra_steps=True):
        if install:
          self.ensure_win_adk(refs=self._win_adk_refs)

  # TODO(actodd): reconfigure 3pp builder to preserve the software name
  # TODO(actodd): Use ensure file for refs features
  def ensure_win_adk(self, refs):
    """Downloads & Installs the Windows ADK."""

    ensure_file = self.m.cipd.EnsureFile()
    ensure_file.add_package(WIN_ADK_PACKAGE, refs)
    cipd_dir = self.m.path['start_dir'].join('cipd', '3pp')
    self.m.cipd.ensure(cipd_dir, ensure_file)
