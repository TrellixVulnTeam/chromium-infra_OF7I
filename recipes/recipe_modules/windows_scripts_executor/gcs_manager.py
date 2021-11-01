# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from . import helper


class GCSManager:
  """
    GCSManager is used to download required artifacts from google cloud storage
    and generate a pinned config for the downloaded artifacts.
  """

  def __init__(self, step, gsutil, path, mfile, cache):
    self._step = step
    self._gsutil = gsutil
    self._cache = cache
    self._path = path
    self._file = mfile
    self._pending_uploads = {}

  def download_packages(self, name, config):
    """ download_packages downloads all the gcs refs """
    with self._step.nest(name):
      # Download all the artifacts
      helper.iter_src(config, self.download_package)

  def download_package(self, src):
    """ download_package downloads a given src to disk if it is a gcs_src."""
    if src.WhichOneof('src') == 'gcs_src':
      pkg = src.gcs_src
      local_source = self._cache.join(pkg.bucket, pkg.source)
      if not self._path.exists(local_source):
        self._gsutil.download(pkg.bucket, pkg.source, local_source)

  def get_local_src(self, gcs_src):
    """ get_local_src returns the path to the source on disk"""
    return self._cache.join(gcs_src.bucket,
                            helper.conv_to_win_path(gcs_src.source))

  def record_upload(self, gcs_src, local_src=''):
    """ record_upload records an upload request for file in local_src."""
    if local_src == '':  # pragma: no cover
      self._pending_uploads[self.get_local_src(gcs_src)] = gcs_src
    else:
      self._pending_uploads[local_src] = gcs_src

  def upload_packages(self):
    """ upload_packages uploads all the packages that were recorded by
        record_upload """
    with self._step.nest('Upload all pending gcs artifacts'):
      for pkg_path, pkg in self._pending_uploads.items():
        if self._path.exists(pkg_path):
          self._gsutil.upload(pkg_path, pkg.bucket, pkg.source)
          # Copy the file to local cache and avoid having to download it again
          local_cache = self.get_local_src(pkg)
          if not self._path.exists(local_cache):
            self._file.copy('cp {} to cache'.format(pkg_path), pkg_path,
                            local_cache)
          del self._pending_uploads[pkg_path]
