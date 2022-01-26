# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
from . import cipd_manager
from . import git_manager
from . import gcs_manager
from . import helper


class Source:
  """ Source handles all the pinning, downloading, and uploading artifacts to
      Git, GCS and CIPD repositories. See (git|gcs|cipd)_manager.py for
      implementation details on how each is handled.
  """

  def __init__(self, cache, step, path, m_file, raw_io, cipd, gsutil, gitiles,
               git, archive):
    """ __init__ generates the src managers (git, gcs and cipd) and stores them
        in class variables.
        Args:
          cache: path to dir that can be used to download artifacts
          step: ref to recipe_engine/step module object
          path: ref to recipe_engine/path module object
          m_file: ref to recipe_engine/file module object
          raw_io: ref to recipe_engine/raw_io module object
          cipd: ref to recipe_engine/cipd module object
          gsutil: ref to depot_tools/gsutil module object
          gitiles: ref to depot_tools/gitiles module object
          git: ref to depot_tools/git module object
          archive: ref to recipe_engine/archive object
    """
    # dir to store CIPD downloaded packages
    cipd_dir = cache.join('CIPDPkgs')
    # dir to store GIT downloaded packages
    git_dir = cache.join('GITPkgs')
    # dir to store GCS downloaded packages
    gcs_dir = cache.join('GCSPkgs')
    helper.ensure_dirs(m_file, [
        cipd_dir, git_dir, gcs_dir,
        gcs_dir.join('chrome-gce-images', 'WIB-WIM')
    ])
    self._cipd = cipd_manager.CIPDManager(step, cipd, path, cipd_dir)
    self._gcs = gcs_manager.GCSManager(step, gsutil, path, m_file, raw_io,
                                       archive, gcs_dir)
    self._git = git_manager.GITManager(step, gitiles, git, m_file, path,
                                       git_dir)
    self._step = step

  def pin(self, src):
    """ pin pins all the recorded packages to static refs """
    if src and src.WhichOneof('src') == 'git_src':
      src.git_src.CopyFrom(self._git.pin_package(src.git_src))
      return src
    if src and src.WhichOneof('src') == 'gcs_src':
      src.gcs_src.CopyFrom(self._gcs.pin_package(src.gcs_src))
      return src
    if src and src.WhichOneof('src') == 'cipd_src':
      src.cipd_src.CopyFrom(self._cipd.pin_package(src.cipd_src))
      return src

  def download(self, src):
    """ download downloads all the pinned packages to cache on disk """
    if src and src.WhichOneof('src') == 'git_src':
      return self._git.download_package(src.git_src)
    if src and src.WhichOneof('src') == 'gcs_src':
      return self._gcs.download_package(src.gcs_src)
    if src and src.WhichOneof('src') == 'cipd_src':
      return self._cipd.download_package(src.cipd_src)

  def get_local_src(self, src):
    """ get_local_src returns path on the disk that points to the given src ref
        Args:
          src: sources.Src proto object that is ref to a downloaded artifact
    """
    if src and src.WhichOneof('src') == 'gcs_src':
      return self._gcs.get_local_src(src.gcs_src)
    if src and src.WhichOneof('src') == 'git_src':
      return self._git.get_local_src(src.git_src)
    if src and src.WhichOneof('src') == 'cipd_src':
      return self._cipd.get_local_src(src.cipd_src)
    if src and src.WhichOneof('src') == 'local_src':  # pragma: no cover
      return src.local_src

  def get_url(self, src):
    """ get_url returns string containing an url referencing the given src
        Args:
          src: sources.Src proto object that contains ref to an artifact
    """
    if src and src.WhichOneof('src') == 'gcs_src':
      return self._gcs.get_gs_url(src.gcs_src)
    if src and src.WhichOneof('src') == 'cipd_src':
      return self._cipd.get_cipd_url(src.cipd_src)
    if src and src.WhichOneof('src') == 'git_src':  # pragma: no cover
      return self._cipd.get_gitiles_url(src.git_src)

  def upload_package(self, dest, source):
    """ upload_package uploads a given package to the given destination"""
    if dest.WhichOneof('dest') == 'gcs_src':
      self._gcs.upload_package(dest, source)
    if dest and dest.WhichOneof('dest') == 'cipd_src':
      self._cipd.upload_package(dest, source)

  def exists(self, src):
    """ exists Returns True if the given src exists
        Args:
          src: sources.Src proto object representing an artifact
    """
    # TODO(anushruth): add support for git and cipd
    if src and src.WhichOneof('src') == 'gcs_src':
      return self._gcs.exists(src.gcs_src)
    return False  # pragma: no cover
