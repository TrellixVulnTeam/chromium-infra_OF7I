# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api
from recipe_engine.recipe_api import Property
from . import cipd_manager
from . import git_manager

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

COPYPE = 'Copy-PE.ps1'
ADDFILE = 'Copy-Item'

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

# Format strings for use in mount cmdline options
EDIT_OFFLINE_REG_CMD = 'Edit-OfflineRegistry'
EDIT_OFFLINE_REG_IMG_PATH = '-OfflineImagePath "{}"'
EDIT_OFFLINE_REG_HIVE_FILE = '-OfflineRegHiveFile "{}"'
EDIT_OFFLINE_REG_KEY_PATH = '-RegistryKeyPath "{}"'
EDIT_OFFLINE_REG_PROPERTY_NAME = '-PropertyName "{}"'
EDIT_OFFLINE_REG_PROPERTY_VALUE = '-PropertyValue "{}"'
EDIT_OFFLINE_REG_PROPERTY_TYPE = '-PropertyType "{}"'


class WindowsPSExecutorAPI(recipe_api.RecipeApi):
  """API for using Windows PowerShell scripts."""

  def __init__(self, *args, **kwargs):
    super(WindowsPSExecutorAPI, self).__init__(*args, **kwargs)
    self._scripts = self.resource('WindowsPowerShell\Scripts')
    self._copype = self._scripts.join(COPYPE)
    self._workdir = ''
    self._cipd = None
    self._git = None

  def pin_wib_config(self, config):
    """ pin_wib_config pins the given config to current refs."""
    if config.arch == wib.Arch.ARCH_UNSPECIFIED:
      raise self.m.step.StepFailure('Missing arch in config')

    # Using a dir in cache to download all cipd artifacts
    cipd_packages = self.m.path['cache'].join('CIPDPkgs')
    # Using a dir in cache to download all git artifacts
    git_packages = self.m.path['cache'].join('GITPkgs')

    # Initialize cipd downloader
    self._cipd = cipd_manager.CIPDManager(self.m.step, self.m.cipd,
                                          cipd_packages)
    # Initialize git downloader
    self._git = git_manager.GITManager(self.m.step, self.m.gitiles, self.m.file,
                                       git_packages)

    # Pin all the cipd instance
    self._cipd.pin_packages('Pin all the cipd packages', config)
    # Pin all the windows artifacts from git
    self._git.pin_packages('Pin git artifacts to refs', config)

  def execute_wib_config(self, config):
    """Executes the windows image builder user config."""
    # Using a directory in cache for working
    self._workdir = self.m.path['cache'].join('WinPEImage')

    with self.m.step.nest('execute config ' + config.name):
      wpec = config.offline_winpe_customization
      if wpec and len(wpec.offline_customization) > 0:
        with self.m.step.nest('offline winpe customization ' + wpec.name):
          self.init_win_pe_image(
              wib.Arch.Name(config.arch).replace('ARCH_', '').lower(),
              self._workdir)
          try:
            for action in wpec.offline_customization:
              self.perform_winpe_action(action)
          except Exception:
            # Unmount the image and discard changes on failure
            self.deinit_win_pe_image(save=False)
            raise
          else:
            self.deinit_win_pe_image()

  def download_wib_artifacts(self, config):
    # Download all the windows artifacts from cipd
    self._cipd.download_packages('Get all cipd artifacts', config)
    # Download all the windows artifacts from git
    self._git.download_packages('Get all git artifacts', config)

  def perform_winpe_action(self, action):
    """Performs the given action"""
    for a in action.actions:
      action = a.WhichOneof('action')
      if action == 'add_file':
        self.add_file(a.add_file)
      elif action == 'install_file':  # pragma: no cover
        raise self.m.step.InfraFailure('Pending Implementation')

      #TODO(actodd): Add tests and remove "no cover" tag
      elif action == 'edit_offline_registry':  # pragma: no cover
        self.edit_offline_registry(a.edit_offline_registry)

  def add_file(self, f):
    src = ''
    #TODO(anushruth): Replace cipd_src with local_src for all actions
    # after downloading the files.
    if f.src.WhichOneof('src') == 'cipd_src':
      # Deref the cipd src and copy it to f
      src = self._cipd.get_local_src(f.src.cipd_src)
      # Include everything in the subdir
      src = src.join('*')
    elif f.src.WhichOneof('src') == 'git_src':
      src = self._git.get_local_src(f.src.git_src)
    elif f.src.WhichOneof('src') == 'local_src':  # pragma: no cover
      src = f.src.local_src
    self.execute_script('Add file {}'.format(src), ADDFILE, None, '-Path', src,
                        '-Recurse', '-Force', '-Destination',
                        self._workdir.join('mount', f.dst))

  def edit_offline_registry(self, edit_offline_registry_action):
    action = edit_offline_registry_action

    args = [
      EDIT_OFFLINE_REG_IMG_PATH.format(self._workdir),
      EDIT_OFFLINE_REG_HIVE_FILE.format(action.reg_hive_file),
      EDIT_OFFLINE_REG_KEY_PATH.format(action.reg_key_path),
      EDIT_OFFLINE_REG_PROPERTY_NAME.format(action.property_name),
      EDIT_OFFLINE_REG_PROPERTY_VALUE.format(action.property_value),
      EDIT_OFFLINE_REG_PROPERTY_TYPE.format(action.property_type)
    ]

    reg_key_leaf = action.reg_key_path.split('\\')[-1]
    name = 'Edit Offline Registry Key {} and Property {}'.format(
      reg_key_leaf, action.property_name)

    self.m.powershell(name, EDIT_OFFLINE_REG_CMD, args=args)

  def init_win_pe_image(self, arch, dest, index=1):
    """Calls Copy-PE to create WinPE media folder for arch"""
    with self.m.step.nest('generate windows image folder for ' + arch + ' in ' +
                          str(dest)):
      # gen a winpe arch dir for the given arch
      self.m.powershell(
          'Gen WinPE media for {}'.format(arch),
          self._copype,
          args=['-WinPeArch', arch, '-Destination',
                str(dest)])
      # Mount the boot.wim to mount dir for modification
      self.mount_win_wim(
          dest.join('mount'), dest.join('media', 'sources', 'boot.wim'), index,
          self.m.path['cleanup'].join(MOUNT_CMD))

  def deinit_win_pe_image(self, save=True):
    """Unmounts the winpe image and saves/discards changes to it"""
    self.unmount_win_wim(
        self._workdir.join('mount'),
        self.m.path['cleanup'].join(MOUNT_CMD),
        save=save)

  def execute_script(self, name, command, logs=None, *args):
    """Executes the windows powershell script"""
    step_res = self.m.powershell(name, command, logs=logs, args=list(args))
    return step_res

  def mount_win_wim(self, mnt_dir, image, index, logs,
                    log_level='WarningsInfo'):
    """Mount the wim to a dir for modification"""
    # Args for the mount cmd
    args = [
        MOUNT_IMG_FILE.format(image),
        MOUNT_INDEX.format(index),
        MOUNT_DIR.format(mnt_dir),
        MOUNT_LOG_PATH.format(logs.join('mount.log')),  # include logs
        MOUNT_LOG_LEVEL.format(log_level)
    ]
    self.m.powershell(
        'Mount wim to {}'.format(mnt_dir), MOUNT_CMD, logs=[logs], args=args)

  def unmount_win_wim(self, mnt_dir, logs, log_level='WarningsInfo', save=True):
    """Unmount the winpe wim from the given directory"""
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
      args.append(UNMOUNT_DISCARD)
    self.m.powershell(
        'Unmount wim at {}'.format(mnt_dir),
        UNMOUNT_CMD,
        logs=[logs],
        args=args)
