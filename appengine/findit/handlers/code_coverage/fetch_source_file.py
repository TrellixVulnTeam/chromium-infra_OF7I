# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

import logging

from gae_libs.handlers.base_handler import BaseHandler, Permission
from handlers.code_coverage import utils
from model import entity_util


class FetchSourceFile(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandlePost(self):
    report_key = self.request.get('report_key')
    path = self.request.get('path')
    revision = self.request.get('revision')

    assert report_key, 'report_key is required'
    assert path, 'path is required'
    assert revision, 'revision is required'

    report = entity_util.GetEntityFromUrlsafeKey(report_key)
    assert report, ('Postsubmit report does not exist for urlsafe key' %
                    report_key)

    file_content = utils.GetFileContentFromGitiles(report.manifest, path,
                                                   revision)
    if not file_content:
      logging.error('Failed to get file from gitiles for %s@%s' %
                    (path, revision))
      return

    gs_path = utils.ComposeSourceFileGsPath(report.manifest, path, revision)
    utils.WriteFileContentToGs(gs_path, file_content)
