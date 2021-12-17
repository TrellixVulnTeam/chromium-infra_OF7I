# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from . import helper


GIT_URL = '{}/+/{}/{}'


class GITManager:
  """
    GITManager is used to download required artifacts from git and generate
    a pinned config for the downloaded artifacts. It uses gitiles to pin the
    sources to refs, but downloads them using git. This is done as image
    processor should be able to pin the sources without having to download
    them, as depending on the result of the pinning. We might not need to
    download them.
  """

  def __init__(self, step, gitiles, git, f, path, cache):
    """ __init__ copies few module objects and cache dir path into class vars
        Args:
          step: module object for recipe_engine/step
          git: module object for depot_tools/git
          gitiles: module object for depot_tools/gitiles
          f: module object for recipe_engine/file
          path: module object for recipe_engine/path
          cache: path to cache file dir. Files from git will be saved here
    """
    # step is an instance of recipe_engine/step instance
    self._step = step
    # gitiles is depot tools module instance. For use in pinning the sources
    self._gitiles = gitiles
    # git is depot tools module instance. For use in downloading the sources
    self._git = git
    # f is recipe_modules/file instance
    self._file = f
    # path instance
    self._path = path
    # cache will be used to download the artifacts to
    self._cache = cache
    self._pinned_srcs = {}
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
    """ pin_package replaces a volatile ref to deterministic ref in all
        git_src"""
    for src in self._pkg_record:
      # gen_key is the path for the unpinned src
      gen_key = self.get_local_src(src)
      pkg = src.git_src
      # check if we pinned the source already
      if gen_key not in self._pinned_srcs.keys():
        commits, _ = self._gitiles.log(pkg.repo, pkg.ref + '/' + pkg.src)
        # pin the file to the latest available commit
        pkg.ref = commits[0]['commit']
        # copy the pinned src to avoid redoing it
        self._pinned_srcs[gen_key] = src
        # copy the pinned src to download list/dict
        self._downloads[self.get_local_src(src)] = src
      else:
        # copy the ref to src
        pkg.ref = self._pinned_srcs[gen_key].git_src.ref

  def download_packages(self):
    """ download_package downloads all recorded git_src"""
    for _, src in self._downloads.items():
      g_src = src.git_src
      local_path = self._cache.join(g_src.ref)
      self._git.checkout(
          step_suffix=g_src.src,
          url=g_src.repo,
          dir_path=local_path,
          ref=g_src.ref,
          file_name=g_src.src)

  def get_gitiles_url(self, src):
    """ get_gitiles_url returns string representing an url for the given source
        Args:
          src: sources.Src object representing cipd_src object
    """
    if src and src.WhichOneof('src') == 'git_src':  # pragma: no cover
      return GIT_URL.format(src.git_src.repo, src.git_src.ref, src.git_src.src)

  def get_local_src(self, source):
    """ get_local_src returns the location of the downloaded package in
        disk
        Args:
          source: sources.Src object representing git_src
    """
    f_path = self._cache.join(source.git_src.ref,
                              helper.conv_to_win_path(source.git_src.src))
    return f_path
