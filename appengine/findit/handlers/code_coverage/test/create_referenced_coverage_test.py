# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
import webapp2

from gae_libs.handlers.base_handler import BaseHandler
from handlers.code_coverage import create_referenced_coverage
from model.code_coverage import CoveragePercentage
from model.code_coverage import CoverageReportModifier
from services.code_coverage import referenced_coverage
from waterfall.test.wf_testcase import WaterfallTestCase


class CreateReferencedCoverageMetricsCronTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/cron/referenced-coverage',
       create_referenced_coverage.CreateReferencedCoverageMetricsCron),
  ],
                                       debug=True)

  @mock.patch.object(
      create_referenced_coverage.CreateReferencedCoverageMetricsCron,
      '_GetSourceBuilders',
      return_value=['linux-code-coverage', 'linux-code-coverage_unit'])
  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testTaskAddedToQueue(self, mocked_is_request_from_appself, _):
    CoverageReportModifier(reference_commit='c1', id=456).put()
    response = self.test_app.get('/coverage/cron/referenced-coverage')
    self.assertEqual(200, response.status_int)
    tasks = self.taskqueue_stub.get_filtered_tasks(
        queue_names='referenced-coverage-queue')
    self.assertEqual(2, len(tasks))
    self.assertTrue(mocked_is_request_from_appself.called)


class CreateReferencedCoverageMetricsTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/referenced-coverage',
       create_referenced_coverage.CreateReferencedCoverageMetrics),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  @mock.patch.object(referenced_coverage, 'CreateReferencedCoverage')
  def testReferencedCoverageLogicInvoked(self, mock_detect, _):
    url = ('/coverage/task/referenced-coverage'
           '?modifier_id=123&builder=linux-code-coverage')
    response = self.test_app.get(url, status=200)
    self.assertEqual(1, mock_detect.call_count)
    self.assertEqual(200, response.status_int)
