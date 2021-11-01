# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api
from recipe_engine.recipe_api import Property
from . import cipd_manager
from . import git_manager
from . import gcs_manager
# Windows command helpers
from . import add_windows_package
from . import mount_wim
from . import unmount_wim
from . import regedit

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import sources

COPYPE = 'Copy-PE.ps1'
ADDFILE = 'Copy-Item'

class WindowsPSExecutorAPI(recipe_api.RecipeApi):
  """API for using Windows PowerShell scripts."""

  def __init__(self, *args, **kwargs):
    super(WindowsPSExecutorAPI, self).__init__(*args, **kwargs)
    self._scripts = self.resource('WindowsPowerShell\Scripts')
    self._copype = self._scripts.join(COPYPE)
    self._workdir = ''
    self._cipd = None
    self._git = None
    self._gcs = None
    self._configs_dir = ''
    self.cfg_key = 'configuration_key'

  def module_init(self):
    """ module_init initializes all the dirs and sub modules required."""
    dirs = []
    # Use configs dir in cleanup to store all the pinned configs
    self._configs_dir = self.m.path['cleanup'].join('configs')
    dirs.append(self._configs_dir)
    # Using a dir in cache to download all cipd artifacts
    cipd_packages = self.m.path['cache'].join('CIPDPkgs')
    dirs.append(cipd_packages)
    # Using a dir in cache to download all git artifacts
    git_packages = self.m.path['cache'].join('GITPkgs')
    dirs.append(git_packages)
    # Using a dir in cache to download all the GCS artifacts
    gcs_packages = self.m.path['cache'].join('GCSPkgs')
    dirs.append(gcs_packages)
    # TODO(anushruth): move this to gcs_manager
    dirs.append(gcs_packages.join('chrome-gce-images', 'WIB-WIM'))
    # Initialize cipd downloader
    self._cipd = cipd_manager.CIPDManager(self.m.step, self.m.cipd,
                                          cipd_packages)
    # Initialize git downloader
    self._git = git_manager.GITManager(self.m.step, self.m.gitiles, self.m.file,
                                       git_packages)
    # Initialize the gcs downloader
    self._gcs = gcs_manager.GCSManager(self.m.step, self.m.gsutil, self.m.path,
                                       self.m.file, gcs_packages)
    self.ensure_dirs(dirs)

  def ensure_dirs(self, dirs):
    """ ensure_dirs ensures that the correct dirs are created on the bot"""
    with self.m.step.nest('Create the necessary dirs'):
      for d in dirs:
        self.m.file.ensure_directory('Ensure {}'.format(d), d)

  def pin_wib_config(self, config):
    """ pin_wib_config pins the given config to current refs."""
    if config.arch == wib.Arch.ARCH_UNSPECIFIED:
      raise self.m.step.StepFailure('Missing arch in config')

    # Pin all the cipd instance
    self._cipd.pin_packages('Pin all the cipd packages', config)
    # Pin all the windows artifacts from git
    self._git.pin_packages('Pin git artifacts to refs', config)

  def save_config_to_disk(self, config):
    """ save_config_to_disk writes the given config to disk, calculates a hash
        of the config and returns the hash. Configs are saved as two files,
        <image-name>.cfg and <config-hash>.cfg"""
    cfg_file = self._configs_dir.join('{}.cfg'.format(config.name))
    self.m.file.write_proto(
        'Write config {}'.format(cfg_file), cfg_file, config, codec='TEXTPB')
    # Calculate the checksum for the config for use as unique id for the image.
    self.cfg_key = self.m.file.file_hash(cfg_file)
    key_file = self._configs_dir.join('{}.cfg'.format(self.cfg_key))
    self.m.file.copy('Copy {} to {}'.format(cfg_file, key_file), cfg_file,
                     key_file)
    return self.cfg_key

  def execute_wib_config(self, config):
    """Executes the windows image builder user config."""
    # Using a directory in cache for working
    self._workdir = self.m.path['cache'].join('WinPEImage')
    arch = wib.Arch.Name(config.arch).replace('ARCH_', '').lower()
    with self.m.step.nest('execute config ' + config.name):
      for customization in config.customizations:
        cust_type = customization.WhichOneof('customization')
        if cust_type == 'offline_winpe_customization':
          self.execute_offline_winpe_customization(
              customization.offline_winpe_customization, arch)

  def download_wib_artifacts(self, config):
    # Download all the windows artifacts from cipd
    self._cipd.download_packages('Get all cipd artifacts', config)
    # Download all the windows artifacts from git
    self._git.download_packages('Get all git artifacts', config)
    # Download all the artifacts from cloud storage
    self._gcs.download_packages('Get all gcs artifacts', config)

  def upload_wib_artifacts(self):
    """ upload_wib_artifacts uploads pending artifacts"""
    self._gcs.upload_packages()

  def perform_winpe_actions(self, offline_action):
    """Performs the given offline_action"""
    for a in offline_action.actions:
      self.perform_winpe_action(a)

  def perform_winpe_action(self, action):
    """Performs the given action"""
    a = action.WhichOneof('action')
    if a == 'add_file':
      self.add_file(action.add_file)
    elif a == 'add_windows_package':
      src = self.get_local_src(action.add_windows_package.src)
      self.add_windows_package(action.add_windows_package, src)

    #TODO(actodd): Add tests and remove "no cover" tag
    elif a == 'edit_offline_registry':  # pragma: no cover
      regedit.edit_offline_registry(self.m.powershell, self._scripts,
                                    action.edit_offline_registry,
                                    self._workdir.join('mount'))

  def get_local_src(self, src):
    """ get_local_src returns path to the source given on disk"""
    if src.WhichOneof('src') == 'cipd_src':
      # Deref the cipd src and copy it to f
      return self._cipd.get_local_src(src.cipd_src)
    if src.WhichOneof('src') == 'git_src':
      return self._git.get_local_src(src.git_src)
    if src.WhichOneof('src') == 'local_src':  # pragma: no cover
      return src.local_src
    if src.WhichOneof('src') == 'gcs_src':
      return self._gcs.get_local_src(src.gcs_src)
    return ''

  def add_windows_package(self, f, src):
    add_windows_package.install_package(self.m.powershell, f, src,
                                        self._workdir.join('mount'),
                                        self.m.path['cleanup'])

  def add_file(self, f):
    src = self.get_local_src(f.src)
    if self.m.path.isdir(src):
      src.join('*')  # pragma: no cover
    self.execute_script('Add file {}'.format(src), ADDFILE, None, '-Path', src,
                        '-Recurse', '-Force', '-Destination',
                        self._workdir.join('mount', f.dst))

  def execute_offline_winpe_customization(self, wpec, arch):
    """ execute_offline_winpe_customization executes the config to build wim"""
    if wpec and len(wpec.offline_customization) > 0:
      with self.m.step.nest('offline winpe customization ' + wpec.name):
        src = self.get_local_src(wpec.image_src)
        if self.get_local_src(wpec.image_src) == '':
          src = self._workdir.join('media', 'sources', 'boot.wim')
        self.init_win_pe_image(arch, src, self._workdir)
        try:
          for action in wpec.offline_customization:
            self.perform_winpe_actions(action)
        except Exception:
          # Unmount the image and discard changes on failure
          self.deinit_win_pe_image(src, self._workdir, save=False)
          raise
        else:
          self.deinit_win_pe_image(src, self._workdir)

  def init_win_pe_image(self, arch, source, dest, index=1):
    """Calls Copy-PE to create WinPE media folder for arch"""
    with self.m.step.nest('Init WinPE image modification ' + arch + ' in ' +
                          str(dest)):
      if not self.m.path.exists(source):
        # gen a winpe arch dir for the given arch
        self.m.powershell(
            'Gen WinPE media for {}'.format(arch),
            self._copype,
            args=['-WinPeArch', arch, '-Destination',
                  str(dest)])
      # ensure that the destination exists
      dest = dest.join('mount')
      self.m.file.ensure_directory('Ensure mount point', dest)

      # Mount the boot.wim to mount dir for modification
      mount_wim.mount_win_wim(self.m.powershell, dest, source, index,
                              self.m.path['cleanup'])

  def deinit_win_pe_image(self, src, dest, save=True):
    """Unmounts the winpe image and saves/discards changes to it"""
    with self.m.step.nest('Deinit WinPE image modification'):
      if save:
        # copy the config file to the root of the wim
        source = self._configs_dir.join('{}.cfg'.format(self.cfg_key))
        self.execute_script('Add cfg {}'.format(source), ADDFILE, None, '-Path',
                            source, '-Recurse', '-Force', '-Destination',
                            self._workdir.join('mount'))
      unmount_wim.unmount_win_wim(
          self.m.powershell,
          dest.join('mount'),
          self.m.path['cleanup'],
          save=save)
      if save:
        # upload the output to default bucket for offline_winpe_customization
        self._gcs.record_upload(
            sources.GCSSrc(
                bucket='chrome-gce-images',
                source='WIB-WIM/{}.wim'.format(self.cfg_key),
            ), str(src))

  def execute_script(self, name, command, logs=None, *args):
    """Executes the windows powershell script"""
    step_res = self.m.powershell(name, command, logs=logs, args=list(args))
    return step_res
