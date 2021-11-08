# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

import mock
import webapp2

from gae_libs.handlers.base_handler import BaseHandler
from handlers.code_coverage import export_feature_coverage
from model.code_coverage import CoverageReportModifier
from services.code_coverage import feature_coverage
from waterfall.test.wf_testcase import WaterfallTestCase


class ExportAllFeatureCoverageMetricsCronTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/cron/all-feature-coverage',
       export_feature_coverage.ExportAllFeatureCoverageMetricsCron),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testTaskAddedToQueue(self, mocked_is_request_from_appself):
    response = self.test_app.get('/coverage/cron/all-feature-coverage')
    self.assertEqual(200, response.status_int)
    response = self.test_app.get('/coverage/cron/all-feature-coverage')
    self.assertEqual(200, response.status_int)

    tasks = self.taskqueue_stub.get_filtered_tasks(
        queue_names='all-feature-coverage-queue')
    self.assertEqual(2, len(tasks))
    self.assertTrue(mocked_is_request_from_appself.called)


class ExportAllFeatureCoverageMetricsTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/all-feature-coverage',
       export_feature_coverage.ExportAllFeatureCoverageMetrics),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testFeatureCoverageFilesExported(self, mocked_is_request_from_appself):
    CoverageReportModifier(gerrit_hashtag='f1', id=123).put()
    CoverageReportModifier(gerrit_hashtag='f2', id=456).put()

    response = self.test_app.get('/coverage/task/all-feature-coverage')
    self.assertEqual(200, response.status_int)

    tasks = self.taskqueue_stub.get_filtered_tasks(
        queue_names='feature-coverage-queue')
    self.assertEqual(2, len(tasks))
    self.assertTrue(mocked_is_request_from_appself.called)


class ExportFeatureCoverageMetricsTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/feature-coverage.*',
       export_feature_coverage.ExportFeatureCoverageMetrics),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  @mock.patch.object(feature_coverage, 'ExportFeatureCoverage')
  def testFeatureCoverageLogicInvoked(self, mock_detect, _):
    CoverageReportModifier(gerrit_hashtag='f1', id=123).put()
    response = self.test_app.get(
        '/coverage/task/feature-coverage?modifier_id=123', status=200)
    self.assertEqual(1, mock_detect.call_count)
    self.assertEqual(200, response.status_int)
