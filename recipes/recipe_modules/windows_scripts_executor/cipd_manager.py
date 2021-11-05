# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from . import helper


class CIPDManager:
  """
  CIPDManager is used to parse through the configs and download all the
  cipd packages. It also modifies the config such that same packages will be
  downloaded next time the modified config is used.
  """

  def __init__(self, step, cipd, path, cache):
    """ __init__ copies common module objects and cache dir for downloading
        cipd artifacts to
        Args:
          step: module object for recipe_engine/step
          cipd: module object for recipe_engine/cipd
          path: module object for recipe_engine/path
          cache: path to cache dir. CIPD artifacts will be downloaded to this
          dir
    """
    # cipd module (from depot_tools) instance
    self._cipd = cipd
    # step module (from recipe_engine) instance
    self._step = step
    # path module to check if file exists
    self._path = path
    # cache dir to be used to download the packages to
    self._cache = cache
    # dict to avoid multiple downloads of the same package
    self._packages = {}
    self._pkg_record = []

  def record_package(self, src):
    """ record_package records the given src for download. If it is a cipd_src
        Args:
          src: sources.Src object representing cipd_src object.
    """
    if src and src.WhichOneof('src') == 'cipd_src':
      self._pkg_record.append(src)

  def pin_packages(self):
    """ pin recorded packages to an instance and update the source"""
    for src in self._pkg_record:
      cipd_s = src.cipd_src
      desc = self._cipd.describe('/'.join([cipd_s.package, cipd_s.platform]),
                                 cipd_s.refs)
      # update the refs with the corresponding instance id
      cipd_s.refs = desc.pin.instance_id
      # cipd expects unix path
      local_path = '/'.join([cipd_s.refs, cipd_s.package, cipd_s.platform])
      self._packages[local_path] = src

  def download_packages(self):
    """ download_packages downloads all the pinned packages"""
    if len(self._packages) > 0:
      e_file = self._cipd.EnsureFile()
      # Add packages that aren't localy available to ensure file
      for loc, package in self._packages.items():
        self.add_src_to_ensurefile(package, loc, e_file)

      # Download all the packages from CIPD
      self._cipd.ensure(self._cache, e_file, name='Download all packages')
      # Remove the listing from packages dict
      for loc, package in self._packages.items():
        if self._path.exists(self._cache.join(helper.conv_to_win_path(loc))):
          del self._packages[loc]

  def add_src_to_ensurefile(self, src, loc, ensure_file):
    """ add_src_to_ensurefile adds the given src to cipd ensurefile.
        Args:
          src: sources.Src object representing cipd_src object
          loc: path to download the cipd artifact to
          ensure_file: CIPD EnsureFile object. Used for downloading multiple
          instances in parallel
    """
    if src.WhichOneof('src') == 'cipd_src':
      cipd_s = src.cipd_src
      # Generate the complete package name
      pname = '/'.join([cipd_s.package, cipd_s.platform])
      # Add the package to the ensure file
      ensure_file.add_package(str(pname), str(cipd_s.refs), str(loc))

  def get_local_src(self, src):
    """ get_local_src returns local path to given src file
        Args:
          src: sources.Src object representing cipd_src object
    """
    key = '/'.join(
        [src.cipd_src.refs, src.cipd_src.package, src.cipd_src.platform])
    # return the deref
    return self._cache.join(helper.conv_to_win_path(key))
