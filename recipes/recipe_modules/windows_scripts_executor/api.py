# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api
from recipe_engine.recipe_api import Property

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
    self._cipd_packages = ''

  def execute_wib_config(self, config):
    """Executes the windows image builder user config."""
    # Using a directory in cache for working
    self._workdir = self.m.path['cache'].join('WinPEImage')
    self._cipd_packages = self.m.path['cache'].join('CIPDPkgs')
    if config.arch == wib.Arch.ARCH_UNSPECIFIED:
      raise self.m.step.StepFailure('Missing arch in config')
    with self.m.step.nest('execute config ' + config.name):
      wpec = config.offline_winpe_customization
      if wpec:
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
    if f.WhichOneof('src') == 'cipd_src':
      res = self.cipd_ensure(f.cipd_src.package, f.cipd_src.refs,
                             f.cipd_src.platform)
      src = res['path']
    elif f.WhichOneof('src') == 'local_src':
      src = f.local_src
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

  def cipd_ensure(self, package, refs, platform, name=''):
    """ Downloads the given package and returns path to the files
        contained within."""
    ensure_file = self.m.cipd.EnsureFile()
    ensure_file.add_package(str(package) + '/' + str(platform), str(refs))
    if name == '':
      name = 'Downloading {}:{}'.format(package, refs)
    # download the package to dir indexed by refs
    dwload_loc = self._cipd_packages.join(refs, conv_to_win_path(package),
                                          platform)
    # Download the installer using cipd
    res = self.m.cipd.ensure(dwload_loc, ensure_file, name=name)
    # Append abs file path to res before returning
    res['path'] = dwload_loc.join('*')
    # Return the path to where the file currently exists
    return res


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


def conv_to_win_path(path):
  """ Converts unix paths to windows ones."""
  return '\\'.join(path.split('/'))
