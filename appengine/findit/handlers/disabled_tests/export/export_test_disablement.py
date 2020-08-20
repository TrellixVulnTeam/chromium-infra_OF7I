# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from google.appengine.api import taskqueue

from common import constants
from gae_libs.handlers.base_handler import BaseHandler
from gae_libs.handlers.base_handler import Permission
from services.disabled_tests import detect_disabled_tests


class ExportTestDisablementCronJob(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandleGet(self):
    # Cron jobs run independently of each other. Therefore, there is no
    # guarantee that they will run either sequentially or simultaneously.
    #
    # Running disabled test export tasks concurrently doesn't bring much
    # benefits, so use task queue to enforce that at most one detection task
    # can be executed at any time.
    taskqueue.add(
        method='GET',
        queue_name=constants.DISABLED_TEST_EXPORT_QUEUE,
        target=constants.DISABLED_TEST_BACKEND,
        url='/disabled-tests/export/task/export-test-disablement')
    return {'return_code': 200}


class DisabledTestExport(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandleGet(self):
    detect_disabled_tests.StoreDisabledTestsInBigQuery()
    return {'return_code': 200}
