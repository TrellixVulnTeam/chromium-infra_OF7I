# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
from . import customization
from . import helper
from . import mount_wim
from . import unmount_wim
from . import regedit
from . import add_windows_package
from . import add_windows_driver

from PB.recipes.infra.windows_image_builder import (offline_winpe_customization
                                                    as winpe)
from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import sources

COPYPE = 'Copy-PE.ps1'
ADDFILE = 'Copy-Item'


class OfflineWinPECustomization(customization.Customization):
  """ WinPE based customization support """

  def __init__(self, cust, **kwargs):
    """ __init__ generates a ref for the given customization
        Args:
          cust: wib.Customization proto object representing
          wib.OfflineWinPECustomization
    """
    super(OfflineWinPECustomization, self).__init__(**kwargs)
    # ensure that the customization is of the correct type
    assert cust.WhichOneof('customization') == 'offline_winpe_customization'
    # generate a copy of customization
    self._customization = wib.Customization()
    self._customization.CopyFrom(cust)
    # use a custom work dir
    self._name = self._customization.offline_winpe_customization.name
    self._workdir = self._path['cleanup'].join(self._name, 'workdir')
    self._scratchpad = self._path['cleanup'].join(self._name, 'sp')
    self._canon_cust = None

    # record all the sources
    wpec = self._customization.offline_winpe_customization
    self._source.record_download(wpec.image_src)
    if wpec:
      for off_action in wpec.offline_customization:
        for action in off_action.actions:
          self._source.record_download(helper.get_src_from_action(action))

  def get_canonical_cfg(self):
    """ get_canonical_cfg returns canonical config after removing name and dest
        Example:
          Given a config
            Customization{
              offline_winpe_customization: OfflineWinPECustomization{
                name: "winpe_gce_vanilla"
                image_src: Src{...}
                image_dest: GCSSrc{...}
                offline_customization: [...]
              }
            }
          returns config
            Customization{
              offline_winpe_customization: OfflineWinPECustomization{
                name: ""
                image_src: Src{...}
                image_dest: GCSSrc{...}
                offline_customization: [...]
              }
            }
    """
    if not self._canon_cust:
      wpec = self._customization.offline_winpe_customization
      # Generate customization without any names or dest refs. This will make
      # customization deterministic to the generated image
      cust = wib.Customization(
          offline_winpe_customization=winpe.OfflineWinPECustomization(
              image_src=wpec.image_src,
              offline_customization=[
                  helper.get_build_offline_customization(c)
                  for c in wpec.offline_customization
              ],
          ),)
      self._canon_cust = cust
    return self._canon_cust

  def get_output(self):
    """ return the output of executing this config. Doesn't guarantee that the
        output exists"""
    if self._key:
      return sources.GCSSrc(
          bucket='chrome-gce-images',
          source='WIB-WIM/{}.wim'.format(self._key),
      )
    return None  # pragma: no cover

  def execute_customization(self):
    """ execute_customization initializes the winpe image, runs the given
        actions and repackages the image and uploads the result to GCS"""
    wpec = self._customization.offline_winpe_customization
    if wpec and len(wpec.offline_customization) > 0:
      with self._step.nest('offline winpe customization ' + wpec.name):
        src = self._source.get_local_src(wpec.image_src)
        if not src:
          src = self._workdir.join('media', 'sources', 'boot.wim')
        self.init_win_pe_image(self._arch, src, self._workdir)
        try:
          for action in wpec.offline_customization:
            self.perform_winpe_actions(action)
        except Exception:
          # Unmount the image and discard changes on failure
          self.deinit_win_pe_image(src, save=False)
          raise
        else:
          self.deinit_win_pe_image(src)

  def init_win_pe_image(self, arch, source, dest, index=1):
    """ init_win_pe_image initializes the source image (if given) by mounting
        it to dest
        Args:
          arch: string representing architecture of the image
          source: path to the wim that needs to be modified
          dest: path to the dir where image can be mounted (under mount dir)
          index: index of the image to be mounted
    """
    with self._step.nest('Init WinPE image modification ' + arch + ' in ' +
                         str(dest)):
      if not self._path.exists(source):
        # gen a winpe arch dir for the given arch
        self._powershell(
            'Gen WinPE media for {}'.format(arch),
            self._scripts.join('Copy-PE.ps1'),
            args=['-WinPeArch', arch, '-Destination',
                  str(dest)])
      # ensure that the destination exists
      dest = dest.join('mount')
      self._file.ensure_directory('Ensure mount point', dest)

      # Mount the boot.wim to mount dir for modification
      mount_wim.mount_win_wim(self._powershell, dest, source, index,
                              self._path['cleanup'])

  def deinit_win_pe_image(self, src, save=True):
    """ deinit_win_pe_image unmounts the winpe image and saves/discards changes
        to it
        Args:
          src: path to image that is currently mounted
          save: bool to determine if we need to save the changes to this image.
    """
    with self._step.nest('Deinit WinPE image modification'):
      if save:
        # copy the config used for building the image
        source = self._configs.join('{}.cfg'.format(self._key))
        self.execute_script('Add cfg {}'.format(source), 'Copy-Item', None,
                            '-Path', source, '-Recurse', '-Force',
                            '-Destination', self._workdir.join('mount'))
      unmount_wim.unmount_win_wim(
          self._powershell,
          self._workdir.join('mount'),
          self._scratchpad,
          save=save)
      if save:
        default_src = self.get_output()
        self._file.copy(
            name='Copy output to destination',
            source=src,
            # use the expected destination in the GCS cache
            dest=self._source.get_local_src(sources.Src(gcs_src=default_src)))
        # upload the output to default bucket for offline_winpe_customization
        self._source.record_upload(default_src)
        # upload to any custom destinations that might be given
        cust = self._customization.offline_winpe_customization
        custom_destination = cust.image_dest
        if custom_destination.bucket and custom_destination.source:
          self._source.record_upload(custom_destination,
                                     self._source.get_url(default_src))

  def perform_winpe_action(self, action):
    """ perform_winpe_action Performs the given action
        Args:
          action: actions.Action proto object that specifies an action to be
          performed
    """
    a = action.WhichOneof('action')
    if a == 'add_file':
      return self.add_file(action.add_file)

    if a == 'add_windows_package':
      src = self._source.get_local_src(action.add_windows_package.src)
      return self.add_windows_package(action.add_windows_package, src)

    if a == 'add_windows_driver':
      src = self._source.get_local_src(action.add_windows_driver.src)
      return self.add_windows_driver(action.add_windows_driver, src)

    if a == 'edit_offline_registry':
      return regedit.edit_offline_registry(self._powershell, self._scripts,
                                           action.edit_offline_registry,
                                           self._workdir.join('mount'))

  def perform_winpe_actions(self, offline_action):
    """ perform_winpe_actions Performs the given offline_action
        Args:
          offline_action: actions.OfflineAction proto object that needs to be
          executed
    """
    for a in offline_action.actions:
      self.perform_winpe_action(a)

  def add_windows_package(self, awp, src):
    """ add_windows_package runs Add-WindowsPackage command in powershell.
        https://docs.microsoft.com/en-us/powershell/module/dism/add-windowspackage?view=windowsserver2019-ps
        Args:
          awp: actions.AddWindowsPackage proto object
          src: Path to the package on bot disk
    """
    add_windows_package.install_package(self._powershell, awp, src,
                                        self._workdir.join('mount'),
                                        self._scratchpad)

  def add_file(self, af):
    """ add_file runs Copy-Item in Powershell to copy the given file to image.
        https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.management/copy-item?view=powershell-5.1
        Args:
          af: actions.AddFile proto object
    """
    src = self._source.get_local_src(af.src)
    if self._path.isdir(src):
      src.join('*')  # pragma: no cover
    self.execute_script('Add file {}'.format(src), ADDFILE, None, '-Path', src,
                        '-Recurse', '-Force', '-Destination',
                        self._workdir.join('mount', af.dst))

  def add_windows_driver(self, awd, src):
    """ add_windows_driver runs Add-WindowsDriver command in powershell.
        https://docs.microsoft.com/en-us/powershell/module/dism/add-windowsdriver?view=windowsserver2019-ps
        Args:
          awd: actions.AddWindowsDriver proto object
          src: Path to the driver on bot disk
    """
    add_windows_driver.install_driver(self._powershell, awd, src,
                                      self._workdir.join('mount'),
                                      self._scratchpad)
