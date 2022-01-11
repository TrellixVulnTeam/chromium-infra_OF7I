# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

import logging

from gae_libs.handlers.base_handler import BaseHandler, Permission

from handlers.code_coverage import utils
from model.code_coverage import PostsubmitReport


class UpdatePostsubmitReport(BaseHandler):
  PERMISSION_LEVEL = Permission.CORP_USER

  def HandlePost(self):
    luci_project = self.request.get('luci_project')
    platform = self.request.get('platform')
    platform_info_map = utils.GetPostsubmitPlatformInfoMap(luci_project)
    if platform not in platform_info_map:
      return BaseHandler.CreateError('Platform: %s is not supported' % platform,
                                     400)
    test_suite_type = self.request.get('test_suite_type', 'all')
    modifier_id = int(self.request.get('modifier_id', '0'))
    bucket = platform_info_map[platform]['bucket']

    builder = platform_info_map[platform]['builder']
    if test_suite_type == 'unit':
      builder += '_unit'

    project = self.request.get('project')
    host = self.request.get('host')
    ref = self.request.get('ref')
    revision = self.request.get('revision')
    visible = self.request.get('visible').lower() == 'true'

    logging.info("host = %s", host)
    logging.info("project = %s", project)
    logging.info("ref = %s", ref)
    logging.info("revision = %s", revision)
    logging.info("bucket = %s", bucket)
    logging.info("builder = %s", builder)
    logging.info("modifier_id = %d", modifier_id)

    report = PostsubmitReport.Get(
        server_host=host,
        project=project,
        ref=ref,
        revision=revision,
        bucket=bucket,
        builder=builder,
        modifier_id=modifier_id)

    if not report:
      return BaseHandler.CreateError('Report record not found', 404)

    # At present, we only update visibility
    report.visible = visible
    report.put()

    return {'return_code': 200}
