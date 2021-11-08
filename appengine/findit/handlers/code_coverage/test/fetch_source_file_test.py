# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

from datetime import datetime
import mock
import webapp2

from gae_libs.handlers.base_handler import BaseHandler
from handlers.code_coverage import fetch_source_file
from handlers.code_coverage import utils
from libs.gitiles.gitiles_repository import GitilesRepository
from model.code_coverage import DependencyRepository
from model.code_coverage import PostsubmitReport
from waterfall.test.wf_testcase import WaterfallTestCase


def _CreateSamplePostsubmitReport(manifest=None,
                                  builder='linux-code-coverage',
                                  modifier_id=0):
  """Returns a sample PostsubmitReport for testing purpose.

  Note: only use this method if the exact values don't matter.
  """
  manifest = manifest or _CreateSampleManifest()
  return PostsubmitReport.Create(
      server_host='chromium.googlesource.com',
      project='chromium/src',
      ref='refs/heads/main',
      revision='aaaaa',
      bucket='coverage',
      builder=builder,
      commit_timestamp=datetime(2018, 1, 1),
      manifest=manifest,
      summary_metrics=_CreateSampleCoverageSummaryMetric(),
      build_id=123456789,
      modifier_id=modifier_id,
      visible=True)


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


class FetchSourceFileTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/fetch-source-file', fetch_source_file.FetchSourceFile),
  ],
                                       debug=True)

  def setUp(self):
    super(FetchSourceFileTest, self).setUp()
    self.UpdateUnitTestConfigSettings(
        'code_coverage_settings', {
            'allowed_gitiles_configs': {
                'chromium.googlesource.com': {
                    'chromium/src': ['refs/heads/main',]
                }
            },
        })

  def tearDown(self):
    self.UpdateUnitTestConfigSettings('code_coverage_settings', {})
    super(FetchSourceFileTest, self).tearDown()

  @mock.patch.object(utils, 'WriteFileContentToGs')
  @mock.patch.object(GitilesRepository, 'GetSource', return_value='test')
  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testFetchSourceFile(self, mocked_is_request_from_appself,
                          mocked_gitiles_get_source, mocked_write_to_gs):
    path = '//v8/src/dir/file.cc'
    revision = 'bbbbb'

    manifest = [
        DependencyRepository(
            path='//v8/',
            server_host='chromium.googlesource.com',
            project='v8/v8.git',
            revision='zzzzz')
    ]
    report = _CreateSamplePostsubmitReport(manifest=manifest)
    report.put()

    request_url = '/coverage/task/fetch-source-file'
    params = {
        'report_key': report.key.urlsafe(),
        'path': path,
        'revision': revision
    }
    response = self.test_app.post(request_url, params=params)
    self.assertEqual(200, response.status_int)
    mocked_is_request_from_appself.assert_called()

    # Gitiles should fetch the revision of last_updated_revision instead of
    # root_repo_revision and the path should be relative to //v8/.
    mocked_gitiles_get_source.assert_called_with('src/dir/file.cc', 'bbbbb')
    mocked_write_to_gs.assert_called_with(
        ('/source-files-for-coverage/chromium.googlesource.com/v8/v8.git/'
         'src/dir/file.cc/bbbbb'), 'test')
