# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

from datetime import datetime
import mock
import webapp2

from gae_libs.handlers.base_handler import BaseHandler
from model.code_coverage import DependencyRepository
from model.code_coverage import PostsubmitReport
from handlers.code_coverage import update_postsubmit_report
from waterfall.test.wf_testcase import WaterfallTestCase


def _CreateSampleManifest():
  """Returns a sample manifest for testing purpose.

  Note: only use this method if the exact values don't matter.
  """
  return [
      DependencyRepository(
          path='//',
          server_host='chromium.googlesource.com',
          project='chromium/src.git',
          revision='ccccc')
  ]


def _CreateSampleCoverageSummaryMetric():
  """Returns a sample coverage summary metric for testing purpose.

  Note: only use this method if the exact values don't matter.
  """
  return [{
      'covered': 1,
      'total': 2,
      'name': 'region'
  }, {
      'covered': 1,
      'total': 2,
      'name': 'function'
  }, {
      'covered': 1,
      'total': 2,
      'name': 'line'
  }]


class UpdatePostsubmitReportTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/postsubmit-report/update',
       update_postsubmit_report.UpdatePostsubmitReport),
  ],
                                       debug=True)

  def setUp(self):
    super(UpdatePostsubmitReportTest, self).setUp()
    self.UpdateUnitTestConfigSettings(
        'code_coverage_settings', {
            'postsubmit_platform_info_map': {
                'chromium': {
                    'linux': {
                        'bucket': 'coverage',
                        'builder': 'linux-code-coverage',
                        'coverage_tool': 'clang',
                        'ui_name': 'Linux (C/C++)',
                    },
                },
            },
        })

  def tearDown(self):
    self.UpdateUnitTestConfigSettings('code_coverage_settings', {})
    super(UpdatePostsubmitReportTest, self).tearDown()

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testPostsubmitReportUpdated_AllTest(self, *_):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    manifest = _CreateSampleManifest()
    server_host = 'chromium.googlesource.com'
    project = 'chromium/src'
    luci_project = 'chromium'
    platform = 'linux'
    ref = 'refs/heads/main'
    revision = '99999'
    coverage_config = self.GetUnitTestConfigSettings(
    ).code_coverage_settings.get('postsubmit_platform_info_map',
                                 {}).get(luci_project, {})[platform]
    bucket = coverage_config['bucket']
    builder = coverage_config['builder']
    report = PostsubmitReport.Create(
        server_host=server_host,
        project=project,
        ref=ref,
        revision=revision,
        bucket=bucket,
        builder=builder,
        commit_timestamp=datetime(2018, 1, 1),
        manifest=manifest,
        summary_metrics=_CreateSampleCoverageSummaryMetric(),
        build_id=123456789,
        visible=False)
    report.put()

    request_url = (
        '/coverage/task/postsubmit-report/update?host=%s&luci_project=%s'
        '&platform=%s&project=%s&ref=%s&revision=%s&visible=%s') % (
            server_host, luci_project, platform, project, ref, revision, True)
    response = self.test_app.post(request_url)

    self.assertEqual(200, response.status_int)
    updated = PostsubmitReport.Get(server_host, project, ref, revision, bucket,
                                   builder)
    self.assertTrue(updated.visible)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testPostsubmitReportUpdated_UnitTest(self, *_):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    manifest = _CreateSampleManifest()
    server_host = 'chromium.googlesource.com'
    project = 'chromium/src'
    luci_project = 'chromium'
    platform = 'linux'
    ref = 'refs/heads/main'
    revision = '99999'
    coverage_config = self.GetUnitTestConfigSettings(
    ).code_coverage_settings.get('postsubmit_platform_info_map',
                                 {}).get(luci_project, {})[platform]
    bucket = coverage_config['bucket']
    builder = coverage_config['builder'] + '_unit'
    report = PostsubmitReport.Create(
        server_host=server_host,
        project=project,
        ref=ref,
        revision=revision,
        bucket=bucket,
        builder=builder,
        commit_timestamp=datetime(2018, 1, 1),
        manifest=manifest,
        summary_metrics=_CreateSampleCoverageSummaryMetric(),
        build_id=123456789,
        visible=False)
    report.put()

    request_url = (
        '/coverage/task/postsubmit-report/update?host=%s&luci_project=%s'
        '&platform=%s&project=%s&ref=%s&revision=%s'
        '&test_suite_type=%s&visible=%s') % (server_host, luci_project,
                                             platform, project, ref, revision,
                                             'unit', True)
    response = self.test_app.post(request_url)

    self.assertEqual(200, response.status_int)
    updated = PostsubmitReport.Get(server_host, project, ref, revision, bucket,
                                   builder)
    self.assertTrue(updated.visible)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testPostsubmitReportUpdated_WithModifier(self, *_):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    manifest = _CreateSampleManifest()
    server_host = 'chromium.googlesource.com'
    project = 'chromium/src'
    luci_project = 'chromium'
    platform = 'linux'
    ref = 'refs/heads/main'
    revision = '99999'
    coverage_config = self.GetUnitTestConfigSettings(
    ).code_coverage_settings.get('postsubmit_platform_info_map',
                                 {}).get(luci_project, {})[platform]
    bucket = coverage_config['bucket']
    builder = coverage_config['builder']
    modifier_id = 123
    report = PostsubmitReport.Create(
        server_host=server_host,
        project=project,
        ref=ref,
        revision=revision,
        bucket=bucket,
        builder=builder,
        commit_timestamp=datetime(2018, 1, 1),
        manifest=manifest,
        summary_metrics=_CreateSampleCoverageSummaryMetric(),
        build_id=123456789,
        visible=False,
        modifier_id=123)
    report.put()

    request_url = (
        '/coverage/task/postsubmit-report/update?host=%s&luci_project=%s'
        '&platform=%s&project=%s&ref=%s&revision=%s'
        '&modifier_id=%d&visible=%s') % (server_host, luci_project, platform,
                                         project, ref, revision, modifier_id,
                                         True)
    response = self.test_app.post(request_url)

    self.assertEqual(200, response.status_int)
    updated = PostsubmitReport.Get(server_host, project, ref, revision, bucket,
                                   builder, modifier_id)
    self.assertTrue(updated.visible)
