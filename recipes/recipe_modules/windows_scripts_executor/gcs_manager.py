# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from . import helper


class GCSManager:
  """
    GCSManager is used to download required artifacts from google cloud storage
    and generate a pinned config for the downloaded artifacts. Also supports
    uploading artifacts to GCS.
  """

  def __init__(self, step, gsutil, path, mfile, raw_io, cache):
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
    self._pending_uploads = {}
    self._pending_downloads = {}
    self._pkg_record = []

  def record_package(self, src):
    """ record_upload records the given src into a list if it is a gcs_src
        Args:
          src: sources.Src is a proto object that refers to a gcs_src ref
    """
    if src and src.WhichOneof('src') == 'gcs_src':
      self._pkg_record.append(src)

  def pin_packages(self):
    """ pin_packages pins the given src to a proper reference by checking
        object metadata"""
    for src in self._pkg_record:
      url = self.get_orig(self.get_gs_url(src.gcs_src))
      if url:
        # found the original file. Pin to the correct src
        b, s = self.get_bucket_source(url)
        src.gcs_src.bucket = b
        src.gcs_src.source = s
        self._pending_downloads[url] = src
        self._pkg_record.remove(src)

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
      text = res.stdout
      # return the given url if not pinned
      orig_url = url
      for line in text.split('\n'):
        if 'orig:' in line:
          orig_url = line.replace('orig:', '').strip()
      return orig_url
    return ''

  def exists(self, gcs_src):
    """ exists returns True if the given ref exists on GCS
        Args:
          gcs_src: sources.GCSSrc proto object to check for existence
    """
    return not self.get_orig(self.get_gs_url(gcs_src)) == ''

  def download_packages(self):
    """ download_packages downloads all the gcs refs """
    for url, pkg in self._pending_downloads.items():
      src = pkg.gcs_src
      self._gsutil.download(
          src.bucket,
          src.source,
          self.get_local_src(pkg),
          name='download gs://{}/{}'.format(src.bucket, src.source))
      del self._pending_downloads[url]

  def get_local_src(self, src):
    """ get_local_src returns the path to the source on disk
        Args:
          src: sources.Src proto representing gcs_src ref
    """
    return self._cache.join(src.gcs_src.bucket,
                            helper.conv_to_win_path(src.gcs_src.source))

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

  def record_upload(self, up_dest, source):
    """ record_upload records the upload to be made on upload_packages
        Args:
          up_dest: dest.Dest proto object representing a file to be created on
                   GCS.
          source: local path for the file to be uploaded
    """
    if up_dest and up_dest.WhichOneof('dest') == 'gcs_src':
      if 'orig' not in up_dest.tags:
        # Add orig tag to self if not given.
        up_dest.tags['orig'] = self.get_gs_url(up_dest.gcs_src)
      if source in self._pending_uploads.keys():
        self._pending_uploads[source].append(up_dest)
      else:
        self._pending_uploads[source] = [up_dest]

  def upload_packages(self):
    """ upload_packages uploads all the packages that were recorded by
        record_upload """
    failed_uploads = {}
    for source, uploads in self._pending_uploads.items():
      # check if the file exists before uploading
      if self._path.exists(source):
        for upload in uploads:
          pkg = upload.gcs_src
          self._gsutil.upload(
              source,
              pkg.bucket,
              pkg.source,
              metadata=upload.tags,
              name='upload gs://{}/{}'.format(pkg.bucket, pkg.source))
      else:
        # cannot upload this as the file is currently not available
        failed_uploads[source] = uploads  # pragma: nocover
    # update the pending uploads map
    self._pending_uploads = failed_uploads
