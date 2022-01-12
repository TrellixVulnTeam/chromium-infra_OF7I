# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import datetime

from google.appengine.api import taskqueue

from common import constants
from gae_libs.handlers.base_handler import BaseHandler, Permission
from handlers.code_coverage import utils
from services.code_coverage import referenced_coverage


class CreateReferencedCoverageMetricsCron(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def _GetSourceBuilders(self):
    """Returns CI builders for which coverage metrics are to be generated."""
    return [
        'linux-code-coverage', 'mac-code-coverage', 'win10-code-coverage',
        'android-code-coverage', 'android-code-coverage-native',
        'ios-simulator-code-coverage', 'linux-chromeos-code-coverage',
        'linux-code-coverage_unit', 'mac-code-coverage_unit',
        'win10-code-coverage_unit', 'android-code-coverage_unit',
        'android-code-coverage-native_unit', 'ios-simulator-code-coverage_unit',
        'linux-chromeos-code-coverage_unit'
    ]

  def HandleGet(self):
    for modifier_id in utils.GetActiveReferenceCommits(
        server_host='chromium.googlesource.com', project='chromium/src'):
      for builder in self._GetSourceBuilders():
        url = '/coverage/task/referenced-coverage?modifier_id=%d&builder=%s' % (
            modifier_id, builder)
        taskqueue.add(
            method='GET',
            name='%s-%s' %
            (builder, datetime.datetime.now().strftime('%d%m%Y-%H%M%S')),
            queue_name=constants.REFERENCED_COVERAGE_QUEUE,
            target=constants.CODE_COVERAGE_REFERENCED_COVERAGE_WORKER,
            url=url)
    return {'return_code': 200}


class CreateReferencedCoverageMetrics(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandleGet(self):
    modifier_id = int(self.request.get('modifier_id'))
    builder = self.request.get('builder')
    referenced_coverage.CreateReferencedCoverage(modifier_id, builder)
    return {'return_code': 200}
