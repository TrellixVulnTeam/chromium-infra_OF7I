# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from . import helper


class GITManager:
  """
    GITManager is used to download required artifacts from git and generate
    a pinned config for the downloaded artifacts.
  """

  def __init__(self, step, gitiles, f, cache):
    # step is an instance of recipe_engine/step instance
    self._step = step
    # gitiles is depot tools module instance
    self._git = gitiles
    # f is recipe_modules/file instance
    self._file = f
    # cache will be used to download the artifacts to
    self._cache = cache
    self._artifacts = {}

  def pin_packages(self, name, config):
    """ pin_packages records all the git_src contained in config, estimates the
        refs to pull the same artifact reliably.
    """
    with self._step.nest(name):
      helper.iter_src(config, self.pin_package)

  def pin_package(self, src):
    """ pin_package replaces a volatile ref to deterministic ref in git src"""
    if src.WhichOneof('src') == 'git_src':
      pkg = src.git_src
      commits, _ = self._git.log(pkg.repo, pkg.ref + '/' + pkg.src)
      # pin the file to the latest available commit
      pkg.ref = commits[0]['commit']

  def download_packages(self, name, config):
    """ download_packages collects all git refs from the config and downloads
        the packages"""
    with self._step.nest(name):
      # Download all the artifacts
      helper.iter_src(config, self.download_package)

  def download_package(self, src):
    """ download_package downloads a given src to disk if it is a git_src."""
    if src.WhichOneof('src') == 'git_src':
      pkg = src.git_src
      http_page = '/'.join([pkg.repo, '+', pkg.ref, pkg.src])
      # Avoid downloading the same file multiple times
      if http_page not in self._artifacts.keys():
        f = self._git.download_file(pkg.repo, pkg.src, branch=pkg.ref)
        f_path = self._cache.join(pkg.ref, helper.conv_to_win_path(pkg.src))
        # ensure that the dir enclosing file exists
        self._file.ensure_directory('Create dir',
                                    '\\'.join(str(f_path).split('\\')[:-1]))
        self._file.write_raw('Write {} to disk'.format(http_page), f_path, f)

  def get_local_src(self, pkg):
    """ get_local_src returns the location of the downloaded package in
        disk"""
    f_path = self._cache.join(pkg.ref, helper.conv_to_win_path(pkg.src))
    return f_path
