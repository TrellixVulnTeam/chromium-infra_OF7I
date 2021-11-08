# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from google.appengine.api import taskqueue

from common import constants
from gae_libs.handlers.base_handler import BaseHandler, Permission
from services.code_coverage import files_absolute_coverage


class ExportFilesAbsoluteCoverageMetricsCron(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandleGet(self):
    # Cron jobs run independently of each other. Therefore, there is no
    # guarantee that they will run either sequentially or simultaneously.
    #
    # Executing this job concurrently doesn't bring much
    # benefits, so use task queue to enforce that at most one  task
    # can be executed at any time.
    taskqueue.add(
        method='GET',
        queue_name=constants.FILES_ABSOLUTE_COVERAGE_QUEUE,
        target=constants.CODE_COVERAGE_BACKEND,
        url='/coverage/task/files-absolute-coverage')
    return {'return_code': 200}


class ExportFilesAbsoluteCoverageMetrics(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandleGet(self):
    files_absolute_coverage.ExportFilesAbsoluteCoverage()
    return {'return_code': 200}
