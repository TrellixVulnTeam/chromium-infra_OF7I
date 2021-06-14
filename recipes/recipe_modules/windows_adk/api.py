# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import subprocess

from recipe_engine import recipe_api
from recipe_engine.recipe_api import Property

WIN_ADK_PACKAGE = 'infra/3pp/tools/win_adk/windows-amd64'
WIN_ADK_WINPE_PACKAGE = 'infra/3pp/tools/win_adk_winpe/windows-amd64'


class WindowsADKApi(recipe_api.RecipeApi):
  """API for using Windows ADK distributed via CIPD."""

  def __init__(self, win_adk_refs, win_adk_winpe_refs, *args, **kwargs):
    super(WindowsADKApi, self).__init__(*args, **kwargs)

    self._win_adk_refs = win_adk_refs
    self._win_adk_winpe_refs = win_adk_winpe_refs

  def ensure(self, install=True):
    """Ensure the presence of the Windows ADK."""
    with self.m.step.nest('ensure windows adk present'):
      with self.m.context(infra_steps=True):
        if install:
          self.ensure_win_adk(refs=self._win_adk_refs)
    with self.m.step.nest('ensure win-pe add-on present'):
      with self.m.context(infra_steps=True):
        if install:
          self.ensure_win_adk_winpe(refs=self._win_adk_winpe_refs)

  # TODO(actodd): reconfigure 3pp builder to preserve the software name
  def ensure_win_adk(self, refs):
    """Downloads & Installs the Windows ADK."""

    ensure_file = self.m.cipd.EnsureFile()
    ensure_file.add_package(WIN_ADK_PACKAGE, refs)
    cipd_dir = self.m.path['start_dir'].join('cipd', '3pp')
    self.m.cipd.ensure(cipd_dir, ensure_file)

  # TODO(actodd): reconfigure 3pp builder to preserve the software name
  def ensure_win_adk_winpe(self, refs):
    """Ensures that the WinPE add-on is available."""
    ensure_file = self.m.cipd.EnsureFile()
    ensure_file.add_package(WIN_ADK_WINPE_PACKAGE, refs)
    cipd_dir = self.m.path['start_dir'].join('cipd', '3pp')
    self.m.cipd.ensure(cipd_dir, ensure_file)

  def execute_config(self, config):
    """Executes the user config."""
    with self.m.step.nest('execute config ' + config.name):
      wpec = config.offline_winpe_customization
      if wpec:
        with self.m.step.nest('offline winpe customization ' + wpec.name):
          for action in wpec.offline_customization:
            with self.m.step.nest('performing action ' + action.name):
              self.perform_winpe_action()

  def perform_winpe_action(self):
    """Executes the scripts to perform the required action."""
    # TODO(anushruth): Call scripts to perform the action.
    return
