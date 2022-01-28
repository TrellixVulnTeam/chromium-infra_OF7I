# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import sources
from . import helper


class GCSManager:
  """
    GCSManager is used to download required artifacts from google cloud storage
    and generate a pinned config for the downloaded artifacts. Also supports
    uploading artifacts to GCS.
  """

  def __init__(self, step, gsutil, path, mfile, raw_io, archive, cache):
    """ __init__ copies few module objects and cache dir path into class vars
        Args:
          step: module object for recipe_engine/step
          gsutil: module object for depot_tools/gsutil
          path: module object for recipe_engine/path
          mfile: module object for recipe_engine/file
          raw_io: module object for recipe_engine/raw_io
          cache: path to cache file dir. Files from gcs will be saved here
    """
    self._step = step
    self._gsutil = gsutil
    self._cache = cache
    self._path = path
    self._file = mfile
    self._raw_io = raw_io
    self._archive = archive
    # dict mapping unpinned url to pinned package
    self._pinned_srcs = {}
    # dict mapping downloaded package paths to package
    self._downloaded_srcs = {}

  def pin_package(self, gcs_src):
    """ pin_package takes a gcs_src and returns a pinned gcs_src
        Args:
          gcs_src: sources.GCSSrc type referring to a package
    """
    pkg_url = self.get_gs_url(gcs_src)
    if pkg_url in self._pinned_srcs:
      return self._pinned_srcs[pkg_url]  # pragma: no cover
    else:
      pin_url = self.get_orig(pkg_url)
      if pin_url:
        # This package was linked to another
        b, s = self.get_bucket_source(pin_url)
        gcs_src.bucket = b
        gcs_src.source = s
      self._pinned_srcs[pkg_url] = gcs_src
      return gcs_src

  def get_orig(self, url):
    """ get_orig goes through the metadata to determine original object and
        returns url for the original GCS object. See upload_packages
        Args:
          url: string representing url that describes a gcs object
    """
    res = self._gsutil.stat(
        url,
        name='stat {}'.format(url),
        stdout=self._raw_io.output(),
        ok_ret='any')
    ret_code = res.exc_result.retcode
    if ret_code == 0:
      text = res.stdout.decode('utf-8')
      # return the given url if not pinned
      orig_url = url
      for line in text.split('\n'):
        if 'orig:' in line:
          orig_url = line.replace('orig:', '').strip()
      return orig_url
    return ''

  def exists(self, src):
    """ exists returns True if the given ref exists on GCS
        Args:
          src: sources.Src proto object to check for existence
    """
    return not self.get_orig(self.get_gs_url(src)) == ''

  def download_package(self, gcs_src):
    """ download_package downloads the given package if required and returns
        local_path to the package.
        gcs_src: source.GCSSrc object referencing a package
    """
    local_path = self.get_local_src(gcs_src)
    if not local_path in self._downloaded_srcs:
      self._gsutil.download(
          gcs_src.bucket,
          gcs_src.source,
          local_path,
          name='download {}'.format(self.get_gs_url(gcs_src)))
      self._downloaded_srcs[local_path] = gcs_src
    return local_path

  def get_local_src(self, gcs_src):
    """ get_local_src returns the path to the source on disk
        Args:
          gcs_src: sources.GCSSrc proto object referencing an artifact in GCS
    """
    return self._cache.join(gcs_src.bucket,
                            helper.conv_to_win_path(gcs_src.source))

  def get_gs_url(self, gcs_src):
    """ get_gs_url returns the gcs url for the given gcs src
        Args:
          gcs_src: sources.GCSSrc proto object referencing an artifact in GCS
    """
    return 'gs://{}/{}'.format(gcs_src.bucket, gcs_src.source)

  def get_bucket_source(self, url):
    """ get_bucket_source returns bucket and source given gcs url
        Args:
          url: gcs url representing a file on GCS
    """
    bs = url.replace('gs://', '')
    tokens = bs.split('/')
    bucket = tokens[0]
    source = bs.replace(bucket + '/', '')
    return bucket, source

  def upload_package(self, dest, source):
    """ upload_package uploads the contents of source on disk to dest.
        Args:
          dest: dest.Dest proto object representing an upload location
          source: path to the package on disk to be uploaded
    """
    if self._path.exists(source):
      package = source
      if self._path.isdir(source):
        # package the dir to zip
        package = source.join('gcs.zip')
        self._archive.package(source).archive(
            'Package {} for upload'.format(source), package)
      self._gsutil.upload(
          package,
          dest.gcs_src.bucket,
          dest.gcs_src.source,
          metadata=dest.tags,
          name='upload {}'.format(self.get_gs_url(dest.gcs_src)))
      if self._path.isdir(source):
        self._file.remove('Delete gcs.zip after upload', package)
