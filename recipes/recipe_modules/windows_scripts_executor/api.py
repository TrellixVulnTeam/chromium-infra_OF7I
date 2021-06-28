# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api
from recipe_engine.recipe_api import Property

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

COPYPE = 'Copy-PE.ps1'
ADDFILE = 'Add-FileToDiskImage.ps1'


class WindowsPSExecutorAPI(recipe_api.RecipeApi):
  """API for using Windows PowerShell scripts."""

  def __init__(self, *args, **kwargs):
    super(WindowsPSExecutorAPI, self).__init__(*args, **kwargs)
    self._scripts = self.resource('WindowsPowerShell\Scripts')
    self._copype = self._scripts.join(COPYPE)
    self._addfile = self._scripts.join(ADDFILE)
    self._workdir = ''

  def execute_wib_config(self, config):
    """Executes the windows image builder user config."""
    # Using a directory in cache for working
    self._workdir = self.m.path['cache'].join('WinPEImage')
    with self.m.step.nest('execute config ' + config.name):
      wpec = config.offline_winpe_customization
      if wpec:
        with self.m.step.nest('offline winpe customization ' + wpec.name):
          # TODO(anushruth): Change this to wpec.arch before submission.
          self.init_win_pe_image('amd64', self._workdir)
          for action in wpec.offline_customization:
            with self.m.step.nest('performing action ' + action.name):
              self.perform_winpe_action(action)

  def perform_winpe_action(self, action):
    """Performs the given action"""
    for f in action.files:
      self.execute_script(self._addfile, '-DiskImage', str(self._workdir),
                          '-SourceFile', f.src, '-ImageDestinationPath', f.dst)

  def init_win_pe_image(self, arch, dest):
    """Calls Copy-PE to create WinPE media folder for arch"""
    with self.m.step.nest('generate windows image folder for ' + arch + ' in ' +
                          str(dest)):
      self.execute_script(self._copype, '-WinPeArch', arch, '-Destination',
                          str(dest))

  def execute_script(self, command, *args):
    """Executes the windows powershell script"""
    res = self.m.step(
        'Exec powershell',
        self.gen_ps_script_cmd(command, *args),
        stdout=self.m.json.output())
    step_res = res.stdout
    if step_res:
      res.presentation.logs['command'] = step_res['Command']
      if not step_res['Success']:
        res.presentation.logs['ErrorInfo'] = step_res['ErrorInfo']['Message']

  def gen_ps_script_cmd(self, command, *args):
    """Generate the powershell command."""
    # Invoke the powershell command
    cmd = ['powershell', '-Command']
    # Append script and args.
    invoker = str(command) + ' ' + ' '.join(args)
    cmd.append(invoker)
    return cmd
