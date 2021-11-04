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

  def __init__(self, step, gitiles, f, path, cache):
    """ __init__ copies few module objects and cache dir path into class vars
        Args:
          step: module object for recipe_engine/step
          gitiles: module object for depot_tools/gitiles
          f: module object for recipe_engine/file
          path: module object for recipe_engine/path
          cache: path to cache file dir. Files from git will be saved here
    """
    # step is an instance of recipe_engine/step instance
    self._step = step
    # gitiles is depot tools module instance
    self._git = gitiles
    # f is recipe_modules/file instance
    self._file = f
    # path instance
    self._path = path
    # cache will be used to download the artifacts to
    self._cache = cache
    self._downloads = {}
    self._pkg_record = []

  def record_package(self, src):
    """ record all the git pkgs into a list
        Args:
          src: sources.Src proto object representing git_src
    """
    if src and src.WhichOneof('src') == 'git_src':
      self._pkg_record.append(src)

  def pin_packages(self):
    """ pin_package replaces a volatile ref to deterministic ref in git src"""
    for src in self._pkg_record:
      gen_key = self.get_local_src(src)
      # check of we pinned the source already
      if gen_key not in self._downloads.keys():
        pkg = src.git_src
        commits, _ = self._git.log(pkg.repo, pkg.ref + '/' + pkg.src)
        # pin the file to the latest available commit
        pkg.ref = commits[0]['commit']
        # record both keys for download
        self._downloads[gen_key] = src
        self._downloads[self.get_local_src(src)] = src

  def download_packages(self):
    """ download_package downloads a given src to disk if it is a git_src."""
    for path, src in self._downloads.items():
      local_path = self.get_local_src(src)
      if local_path == path and not self._path.exists(local_path):
        # only download pinned configs
        pkg = src.git_src
        http_page = '/'.join([pkg.repo, '+', pkg.ref, pkg.src])
        f = self._git.download_file(pkg.repo, pkg.src, branch=pkg.ref)
        # ensure that the dir enclosing file exists
        self._file.ensure_directory('Create dir',
                                    '\\'.join(str(local_path).split('\\')[:-1]))
        self._file.write_raw('Write {} to disk'.format(http_page), local_path,
                             f)

  def get_local_src(self, source):
    """ get_local_src returns the location of the downloaded package in
        disk
        Args:
          source: sources.Src object representing git_src
    """
    f_path = self._cache.join(source.git_src.ref,
                              helper.conv_to_win_path(source.git_src.src))
    return f_path
