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

  def __init__(self, step, cipd, cache):
    # cipd module (from depot_tools) instance
    self._cipd = cipd
    # step module (from recipe_engine) instance
    self._step = step
    # cache dir to be used to download the packages to
    self._cache = cache
    self._ensure_file = self._cipd.EnsureFile()
    # dict to avoid multiple downloads of the same package
    self._packages = {}

  def pin_packages(self, name, config):
    """ pin_packages pins the cipd packages to unique refs. This will make the
        src unique and allow for deterministic builds."""
    with self._step.nest(name):
      # pin all the cipd srcs
      helper.iter_src(config, self.pin_src)

  def pin_src(self, src):
    """ pin_src changes any volatile cipd refs like 'latest', 'prod', etc,.
        to absolute instances."""
    if src.WhichOneof('src') == 'cipd_src':
      cipd_s = src.cipd_src
      desc = self._cipd.describe('/'.join([cipd_s.package, cipd_s.platform]),
                                 cipd_s.refs)
      # update the refs with the corresponding instance id
      cipd_s.refs = desc.pin.instance_id

  def download_packages(self, name, config):
    """ download_packages collects all cipd refs from config and downloads the
        packages."""
    # Update the ensure file with the packages required to be downloaded
    helper.iter_src(config, self.add_src_to_ensurefile)
    if len(self._packages) > 0:
      # Download all the packages collected
      self._cipd.ensure(self._cache, self._ensure_file, name=name)

  def add_src_to_ensurefile(self, src):
    """ add_src_to_ensurefile adds the given src to cipd ensurefile."""
    if src.WhichOneof('src') == 'cipd_src':
      cipd_s = src.cipd_src
      # Generate the location under _cache for downloading the package. The dir
      # structure includes refs to allow for using multiple instances of the
      # package
      loc = '/'.join([cipd_s.refs, cipd_s.package, cipd_s.platform])
      if loc not in self._packages.keys():
        # Add src to packages dict. This avoids repeats of src downloads
        self._packages[loc] = cipd_s
        # Generate the complete package name
        pname = '/'.join([cipd_s.package, cipd_s.platform])
        # Add the package to the ensure file
        self._ensure_file.add_package(str(pname), str(cipd_s.refs), str(loc))

  def get_local_src(self, src):
    """ get_local_src returns local path to given src file"""
    key = '/'.join([src.refs, src.package, src.platform])
    # return the deref
    return self._cache.join(helper.conv_to_win_path(key))
