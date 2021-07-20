# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api
from recipe_engine.recipe_api import Property

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

COPYPE = 'Copy-PE.ps1'
ADDFILE = 'Add-FileToDiskImage.ps1'

# Format strings for use in mount cmdline options
MOUNT_CMD = 'Mount-WindowsImage'
MOUNT_IMG_FILE = '-ImagePath "{}"'
MOUNT_INDEX = '-Index {}'
MOUNT_DIR = '-Path "{}"'
MOUNT_LOG_PATH = '-LogPath "{}"'
MOUNT_LOG_LEVEL = '-LogLevel {}'

# Format strings for use in unmount cmdline options
UNMOUNT_CMD = 'Dismount-WindowsImage'
UNMOUNT_DIR = '-Path "{}"'
UNMOUNT_LOG_PATH = '-LogPath "{}"'
UNMOUNT_LOG_LEVEL = '-LogLevel {}'
UNMOUNT_SAVE = '-Save'  # Save changes to the wim
UNMOUNT_DISCARD = '-Discard'  # Discard changes to the wim

# Format strings for use in Set-ItemProperty cmdline options
SETIP_CMD = 'Set-ItemProperty'
SETIP_PATH = '-Path {}'
SETIP_NAME = '-Name {}'
SETIP_VALUE = '-Value {}'


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
          self.init_win_pe_image(
              wib.Arch.Name(wpec.winpe_arch).replace('ARCH_', '').lower(),
              self._workdir)
          try:
            for action in wpec.offline_customization:
              with self.m.step.nest('performing action ' + action.name):
                self.perform_winpe_action(action)
          except Exception as e:
            # Unmount the image and discard changes on failure
            self.deinit_win_pe_image(save=False)
            raise e
          else:
            self.deinit_win_pe_image()

  def perform_winpe_action(self, action):
    """Performs the given action"""
    for f in action.files:
      self.execute_script(self._addfile, '-DiskImage', str(self._workdir),
                          '-SourceFile', f.src, '-ImageDestinationPath', f.dst)

  def init_win_pe_image(self, arch, dest, index=1):
    """Calls Copy-PE to create WinPE media folder for arch"""
    with self.m.step.nest('generate windows image folder for ' + arch + ' in ' +
                          str(dest)):
      # gen a winpe arch dir for the given arch
      self.execute_script(self._copype, '-WinPeArch', arch, '-Destination',
                          str(dest))
      # Mount the boot.wim to mount dir for modification
      self.mount_win_wim(
          dest.join('mount'), dest.join('media', 'sources', 'boot.wim'), index,
          self.m.path['cleanup'].join(MOUNT_CMD))

  def deinit_win_pe_image(self, save=True):
    """Unmounts the wim pe image and saves changes to it"""
    self.unmount_win_wim(
        self._workdir.join('mount'),
        self.m.path['cleanup'].join(MOUNT_CMD),
        save=save)

  def execute_script(self, command, *args):
    """Executes the windows powershell script"""
    res = self.m.step(
        'Exec powershell',
        self.gen_ps_script_cmd(command, *args),
        stdout=self.m.json.output())
    step_res = res.stdout
    if step_res:
      if not step_res['Success']:
        res.presentation.logs['ErrorInfo'] = step_res['ErrorInfo']['Message']
        raise self.m.step.InfraFailure('{} run failed: {}'.format(
            command, step_res['ErrorInfo']['Message']))

  def gen_ps_script_cmd(self, command, *args):
    """Generate the powershell command."""
    # Invoke the powershell command
    cmd = ['powershell', '-Command']
    # Append script and args.
    invoker = str(command) + ' ' + ' '.join(args)
    cmd.append(invoker)
    return cmd

  def mount_win_wim(self, mnt_dir, image, index, logs,
                    log_level='WarningsInfo'):
    """Mount the wim to a dir for modification"""
    with self.m.step.nest('Mount wim for modification') as r:
      # Check if the logs directory exists and the permissions for the dir.
      self.m.file.ensure_directory('Check logs folder', logs)
      # Change the permissions for the logs dir. Without this Mount cmd fails
      # with COMException
      chmod = [
          SETIP_PATH.format(logs),
          SETIP_NAME.format('Attributes'),
          SETIP_VALUE.format('Normal')
      ]
      self.m.step('chmod {}'.format(logs),
                  self.gen_ps_script_cmd(SETIP_CMD, *chmod))

      # Args for the mount cmd
      args = [
          MOUNT_IMG_FILE.format(image),
          MOUNT_INDEX.format(index),
          MOUNT_DIR.format(mnt_dir),
          MOUNT_LOG_PATH.format(logs.join('mount.log')),  # include logs
          MOUNT_LOG_LEVEL.format(log_level)
      ]
      self.m.step('Mount wim to {}'.format(mnt_dir),
                  self.gen_ps_script_cmd(MOUNT_CMD, *args))

      # Read the logs and update the step with mount cmd logs
      c = self.m.file.read_raw(
          'Reading {}'.format(logs),
          logs.join('mount.log'),
          test_data='dism mount logs')
      r.presentation.logs[MOUNT_CMD] = c

  def unmount_win_wim(self, mnt_dir, logs, log_level='WarningsInfo', save=True):
    """Unmount the win pe wim from the given directory"""
    with self.m.step.nest('Unmount wim') as r:
      # Args for the unmount cmd
      args = [
          UNMOUNT_DIR.format(mnt_dir),
          UNMOUNT_LOG_PATH.format(logs.join('unmount.log')),
          UNMOUNT_LOG_LEVEL.format(log_level)
      ]
      # Save/Discard the changes to the wim
      if save:
        args.append(UNMOUNT_SAVE)
      else:
        # TODO(anushruth): add coverage for this
        args.append(UNMOUNT_DISCARD)  # pragma: no cover
      self.m.step('Unmount wim at {}'.format(mnt_dir),
                  self.gen_ps_script_cmd(UNMOUNT_CMD, *args))

      # Read the logs and update the step with mount cmd logs
      c = self.m.file.read_raw(
          'Reading {}'.format(logs),
          logs.join('unmount.log'),
          test_data='dism unmount logs')
      r.presentation.logs[UNMOUNT_CMD] = c
