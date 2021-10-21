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

  def __init__(self, step, gsutil, cache):
    self._step = step
    self._gsutil = gsutil
    self._cache = cache

  def download_packages(self, name, config):
    """ download_packages downloads all the gcs refs """
    with self._step.nest(name):
      # Download all the artifacts
      helper.iter_src(config, self.download_package)

  def download_package(self, src):
    """ download_package downloads a given src to disk if it is a gcs_src."""
    if src.WhichOneof('src') == 'gcs_src':
      pkg = src.gcs_src
      self._gsutil.download(pkg.bucket, pkg.source,
                            self._cache.join(pkg.bucket, pkg.source))
