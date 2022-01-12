# -*- coding: utf-8 -*-
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from datetime import datetime
import json
import mock
import webapp2

from handlers.code_coverage import serve_coverage
from handlers.code_coverage import utils
from model.code_coverage import CoveragePercentage
from model.code_coverage import CoverageReportModifier
from model.code_coverage import DependencyRepository
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from model.code_coverage import PresubmitCoverageData
from model.code_coverage import SummaryCoverageData
from services.code_coverage import code_coverage_util
from waterfall.test.wf_testcase import WaterfallTestCase


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


def _CreateSampleDirectoryCoverageData(builder='linux-code-coverage',
                                       modifier_id=0):
  """Returns a sample directory SummaryCoverageData for testing purpose.

  Note: only use this method if the exact values don't matter.
  """
  return SummaryCoverageData.Create(
      server_host='chromium.googlesource.com',
      project='chromium/src',
      ref='refs/heads/main',
      revision='aaaaa',
      data_type='dirs',
      path='//dir/',
      bucket='coverage',
      builder=builder,
      modifier_id=modifier_id,
      data={
          'dirs': [],
          'path':
              '//dir/',
          'summaries':
              _CreateSampleCoverageSummaryMetric(),
          'files': [{
              'path': '//dir/test.cc',
              'name': 'test.cc',
              'summaries': _CreateSampleCoverageSummaryMetric()
          }]
      })


def _CreateSampleFileCoverageData(builder='linux-code-coverage', modifier_id=0):
  """Returns a sample FileCoverageData for testing purpose.

  Note: only use this method if the exact values don't matter.
  """
  return FileCoverageData.Create(
      server_host='chromium.googlesource.com',
      project='chromium/src',
      ref='refs/heads/main',
      revision='aaaaa',
      path='//dir/test.cc',
      bucket='coverage',
      builder=builder,
      modifier_id=modifier_id,
      data={
          'path': '//dir/test.cc',
          'revision': 'bbbbb',
          'lines': [{
              'count': 100,
              'last': 2,
              'first': 1
          }],
          'timestamp': '140000',
          'uncovered_blocks': [{
              'line': 1,
              'ranges': [{
                  'first': 1,
                  'last': 2
              }]
          }]
      })


def _CreateSampleComponentCoverageData(builder='linux-code-coverage'):
  """Returns a sample component SummaryCoverageData for testing purpose.

  Note: only use this method if the exact values don't matter.
  """
  return SummaryCoverageData.Create(
      server_host='chromium.googlesource.com',
      project='chromium/src',
      ref='refs/heads/main',
      revision='aaaaa',
      data_type='components',
      path='Component>Test',
      bucket='coverage',
      builder=builder,
      data={
          'dirs': [{
              'path': '//dir/',
              'name': 'dir/',
              'summaries': _CreateSampleCoverageSummaryMetric()
          }],
          'path': 'Component>Test',
          'summaries': _CreateSampleCoverageSummaryMetric()
      })


class ServeCodeCoverageDataTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication(
      [('/coverage/api/coverage-data', serve_coverage.ServeCodeCoverageData),
       ('/coverage/p/.*', serve_coverage.ServeCodeCoverageData)],
      debug=True)

  def setUp(self):
    super(ServeCodeCoverageDataTest, self).setUp()
    self.UpdateUnitTestConfigSettings(
        'code_coverage_settings', {
            'serve_presubmit_coverage_data': True,
            'allowed_gitiles_configs': {
                'chromium.googlesource.com': {
                    'chromium/src': ['refs/heads/main',]
                }
            },
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
            'default_postsubmit_report_config': {
                'chromium': {
                    'host': 'chromium.googlesource.com',
                    'project': 'chromium/src',
                    'ref': 'refs/heads/main',
                    'platform': 'linux',
                },
            },
        })

  def tearDown(self):
    self.UpdateUnitTestConfigSettings('code_coverage_settings', {})
    super(ServeCodeCoverageDataTest, self).tearDown()

  def testServeCLPatchsetLinesData(self):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    patchset = 4
    data = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 1,
            'last': 2,
        }],
    }]
    PresubmitCoverageData.Create(
        server_host=host, change=change, patchset=patchset, data=data).put()

    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=%d&concise=1') % (host, project, change, patchset)
    response = self.test_app.get(request_url)

    expected_response_body = json.dumps({
        'data': {
            'files': [{
                'path':
                    'dir/test.cc',
                'lines': [{
                    'count': 100,
                    'line': 1,
                }, {
                    'count': 100,
                    'line': 2,
                }]
            }]
        },
    })
    self.assertEqual(expected_response_body, response.body)

  def testServeCLPatchsetLinesDataInvalidPatchset(self):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=NaN&concise=1') % (host, project, change)
    with self.assertRaisesRegexp(Exception, r'.*400.*'):
      self.test_app.get(request_url)

  @mock.patch.object(code_coverage_util, 'GetEquivalentPatchsets')
  def testServeCLPatchLinesDataNoEquivalentPatchsets(self,
                                                     mock_get_equivalent_ps):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    patchset = 4
    mock_get_equivalent_ps.return_value = []
    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=%d&concise=1') % (host, project, change, patchset)
    response = self.test_app.get(request_url, expect_errors=True)
    self.assertEqual(404, response.status_int)

  @mock.patch.object(code_coverage_util, 'GetEquivalentPatchsets')
  def testServeCLPatchLinesDataEquivalentPatchsetsHaveNoData(
      self, mock_get_equivalent_ps):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    patchset_src = 3
    patchset_dest = 4
    mock_get_equivalent_ps.return_value = [patchset_src]
    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=%d&concise=1') % (host, project, change,
                                                patchset_dest)
    response = self.test_app.get(request_url, expect_errors=True)
    self.assertEqual(404, response.status_int)

  @mock.patch.object(code_coverage_util,
                     'RebasePresubmitCoverageDataBetweenPatchsets')
  @mock.patch.object(code_coverage_util, 'GetEquivalentPatchsets')
  def testServeCLPatchLinesDataEquivalentPatchsetsMissingData(
      self, mock_get_equivalent_ps, mock_rebase_data):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    patchset_src = 3
    # 4 is based on 3, used to test that 5 would choose 3 instead of 4.
    patchset_mid = 4
    patchset_dest = 5
    data = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 1,
            'last': 2,
        }],
    }]
    PresubmitCoverageData.Create(
        server_host=host, change=change, patchset=patchset_src,
        data=data).put()
    mid_data = PresubmitCoverageData.Create(
        server_host=host, change=change, patchset=patchset_mid, data=data)
    mid_data.based_on = patchset_src
    mid_data.put()

    mock_get_equivalent_ps.return_value = [patchset_src, patchset_mid]
    mock_rebase_data.side_effect = (
        code_coverage_util.MissingChangeDataException(''))

    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=%d&concise=1') % (host, project, change,
                                                patchset_dest)
    response = self.test_app.get(request_url, expect_errors=True)
    self.assertEqual(404, response.status_int)

    mock_rebase_data.side_effect = RuntimeError('Some unknown http code')
    response = self.test_app.get(request_url, expect_errors=True)
    self.assertEqual(500, response.status_int)

  @mock.patch.object(code_coverage_util,
                     'RebasePresubmitCoverageDataBetweenPatchsets')
  @mock.patch.object(code_coverage_util, 'GetEquivalentPatchsets')
  def testServeCLPatchLinesDataEquivalentPatchsets(self, mock_get_equivalent_ps,
                                                   mock_rebase_data):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    patchset_src = 3
    # 4 is based on 3, used to test that 5 would choose 3 instead of 4.
    patchset_mid = 4
    patchset_dest = 5
    data = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 1,
            'last': 2,
        }],
    }]
    data_unit = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 1,
            'last': 3,
        }],
    }]
    PresubmitCoverageData.Create(
        server_host=host,
        change=change,
        patchset=patchset_src,
        data=data,
        data_unit=data_unit).put()
    mid_data = PresubmitCoverageData.Create(
        server_host=host,
        change=change,
        patchset=patchset_mid,
        data=data,
        data_unit=data_unit)
    mid_data.based_on = patchset_src
    mid_data.put()

    rebased_coverage_data = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 2,
            'last': 3,
        }],
    }]

    rebased_coverage_data_unit = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 2,
            'last': 4,
        }],
    }]

    mock_get_equivalent_ps.return_value = [patchset_src, patchset_mid]
    mock_rebase_data.side_effect = [
        rebased_coverage_data, rebased_coverage_data_unit
    ]

    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=%d&concise=1') % (host, project, change,
                                                patchset_dest)
    response = self.test_app.get(request_url)

    expected_response_body = json.dumps({
        'data': {
            'files': [{
                'path':
                    'dir/test.cc',
                'lines': [{
                    'count': 100,
                    'line': 2,
                }, {
                    'count': 100,
                    'line': 3,
                }]
            }]
        },
    })
    self.assertEqual(expected_response_body, response.body)
    src_entity = PresubmitCoverageData.Get(host, change, patchset_src)
    dest_entity = PresubmitCoverageData.Get(host, change, patchset_dest)
    self.assertEqual(patchset_src, dest_entity.based_on)
    self.assertEqual(src_entity.absolute_percentages,
                     dest_entity.absolute_percentages)
    self.assertEqual(src_entity.incremental_percentages,
                     dest_entity.incremental_percentages)
    self.assertEqual(src_entity.absolute_percentages_unit,
                     dest_entity.absolute_percentages_unit)
    self.assertEqual(src_entity.incremental_percentages_unit,
                     dest_entity.incremental_percentages_unit)
    self.assertEqual(rebased_coverage_data, dest_entity.data)
    self.assertEqual(rebased_coverage_data_unit, dest_entity.data_unit)

  def testServeCLPatchPercentagesData(self):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    patchset = 4
    data = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 1,
            'last': 2,
        }],
    }]
    entity = PresubmitCoverageData.Create(
        server_host=host, change=change, patchset=patchset, data=data)
    entity.absolute_percentages = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=2, covered_lines=1)
    ]
    entity.incremental_percentages = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=1, covered_lines=1)
    ]
    entity.absolute_percentages_unit = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=2, covered_lines=1)
    ]
    entity.incremental_percentages_unit = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=1, covered_lines=1)
    ]
    entity.put()

    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=%d&type=percentages&concise=1') % (
                       host, project, change, patchset)
    response = self.test_app.get(request_url)

    expected_response_body = json.dumps({
        'data': {
            'files': [{
                "path": "dir/test.cc",
                "absolute_coverage": {
                    "covered": 1,
                    "total": 2,
                },
                "incremental_coverage": {
                    "covered": 1,
                    "total": 1,
                },
                "absolute_unit_tests_coverage": {
                    "covered": 1,
                    "total": 2,
                },
                "incremental_unit_tests_coverage": {
                    "covered": 1,
                    "total": 1,
                },
            }]
        },
    })
    self.assertEqual(expected_response_body, response.body)

  @mock.patch.object(code_coverage_util, 'GetEquivalentPatchsets')
  def testServeCLPatchPercentagesDataEquivalentPatchsets(
      self, mock_get_equivalent_ps):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    patchset_src = 3
    patchset_dest = 4
    mock_get_equivalent_ps.return_value = [patchset_src]
    data = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 1,
            'last': 2,
        }],
    }]
    entity = PresubmitCoverageData.Create(
        server_host=host, change=change, patchset=patchset_src, data=data)
    entity.absolute_percentages = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=2, covered_lines=1)
    ]
    entity.incremental_percentages = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=1, covered_lines=1)
    ]
    entity.absolute_percentages_unit = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=2, covered_lines=1)
    ]
    entity.incremental_percentages_unit = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=1, covered_lines=1)
    ]
    entity.put()

    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=%d&type=percentages&concise=1') % (
                       host, project, change, patchset_dest)
    response = self.test_app.get(request_url)

    expected_response_body = json.dumps({
        'data': {
            'files': [{
                "path": "dir/test.cc",
                "absolute_coverage": {
                    "covered": 1,
                    "total": 2,
                },
                "incremental_coverage": {
                    "covered": 1,
                    "total": 1,
                },
                "absolute_unit_tests_coverage": {
                    "covered": 1,
                    "total": 2,
                },
                "incremental_unit_tests_coverage": {
                    "covered": 1,
                    "total": 1,
                },
            }]
        },
    })
    self.assertEqual(expected_response_body, response.body)

  def testServeCLPatchPercentagesDataJustUnitTestCoverage(self):
    host = 'chromium-review.googlesource.com'
    project = 'chromium/src'
    change = 138000
    patchset = 4
    data = [{
        'path': '//dir/test.cc',
        'lines': [{
            'count': 100,
            'first': 1,
            'last': 2,
        }],
    }]
    entity = PresubmitCoverageData.Create(
        server_host=host, change=change, patchset=patchset, data=data)
    entity.absolute_percentages_unit = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=2, covered_lines=1)
    ]
    entity.incremental_percentages_unit = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=1, covered_lines=1)
    ]
    entity.put()

    request_url = ('/coverage/api/coverage-data?host=%s&project=%s&change=%d'
                   '&patchset=%d&type=percentages&concise=1') % (
                       host, project, change, patchset)
    response = self.test_app.get(request_url)

    expected_response_body = json.dumps({
        'data': {
            'files': [{
                "path": "dir/test.cc",
                "absolute_coverage": None,
                "incremental_coverage": None,
                "absolute_unit_tests_coverage": {
                    "covered": 1,
                    "total": 2,
                },
                "incremental_unit_tests_coverage": {
                    "covered": 1,
                    "total": 1,
                },
            }]
        },
    })
    self.assertEqual(expected_response_body, response.body)

  def testServeFullRepoProjectView(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport()
    report.put()

    request_url = ('/coverage/p/chromium?host=%s&project=%s&ref=%s&platform=%s'
                   '&list_reports=true') % (host, project, ref, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  def testServeFullRepoProjectView_WithModifier(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport(modifier_id=123)
    report.put()

    request_url = ('/coverage/p/chromium?host=%s&project=%s&ref=%s&platform=%s'
                   '&list_reports=true&modifier_id=%d') % (host, project, ref,
                                                           platform, 123)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  def testServeFullRepoProjectViewDefaultReportConfig(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    report = _CreateSamplePostsubmitReport()
    report.put()

    response = self.test_app.get('/coverage/p/chromium?&list_reports=true')
    self.assertEqual(200, response.status_int)

  def testServeFullRepoDirectoryView(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    revision = 'aaaaa'
    path = '//dir/'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport()
    report.put()

    dir_coverage_data = _CreateSampleDirectoryCoverageData()
    dir_coverage_data.put()

    request_url = (
        '/coverage/p/chromium/dir?host=%s&project=%s&ref=%s&revision=%s'
        '&path=%s&platform=%s') % (host, project, ref, revision, path, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  def testServeFullRepoDirectoryView_WithModifierAndRevision(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    revision = 'aaaaa'
    path = '//dir/'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport(modifier_id=123)
    report.put()

    dir_coverage_data = _CreateSampleDirectoryCoverageData(modifier_id=123)
    dir_coverage_data.put()

    request_url = (
        '/coverage/p/chromium/dir?host=%s&project=%s&ref=%s&revision=%s'
        '&path=%s&platform=%s&modifier_id=%d') % (host, project, ref, revision,
                                                  path, platform, 123)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  def testServeFullRepoDirectoryView_WithModifier_WithoutRevision(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    path = '//dir/'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport(modifier_id=123)
    report.put()

    dir_coverage_data = _CreateSampleDirectoryCoverageData(modifier_id=123)
    dir_coverage_data.put()

    request_url = ('/coverage/p/chromium/dir?host=%s&project=%s&ref=%s'
                   '&path=%s&platform=%s&modifier_id=%d') % (
                       host, project, ref, path, platform, 123)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  def testServeFullRepoComponentView(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    revision = 'aaaaa'
    path = 'Component>Test'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport()
    report.put()

    component_coverage_data = _CreateSampleComponentCoverageData()
    component_coverage_data.put()

    request_url = ('/coverage/p/chromium/component?host=%s&project=%s&ref=%s'
                   '&revision=%s&path=%s&platform=%s') % (
                       host, project, ref, revision, path, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  def testServeFullRepo_UnitTestsOnly(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    revision = 'aaaaa'
    path = '//dir/'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport(builder='linux-code-coverage_unit')
    report.put()

    dir_coverage_data = _CreateSampleDirectoryCoverageData(
        builder='linux-code-coverage_unit')
    dir_coverage_data.put()

    request_url = (
        '/coverage/p/chromium/dir?host=%s&project=%s&ref=%s&revision=%s'
        '&path=%s&platform=%s&test_suite_type=unit') % (
            host, project, ref, revision, path, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  @mock.patch.object(utils, 'GetFileContentFromGs')
  def testServeFullRepoFileView(self, mock_get_file_from_gs):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    mock_get_file_from_gs.return_value = 'line one/nline two'

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    revision = 'aaaaa'
    path = '//dir/test.cc'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport()
    report.put()

    file_coverage_data = _CreateSampleFileCoverageData()
    file_coverage_data.put()

    request_url = ('/coverage/p/chromium/file?host=%s&project=%s&ref=%s'
                   '&revision=%s&path=%s&platform=%s') % (
                       host, project, ref, revision, path, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)
    mock_get_file_from_gs.assert_called_with(
        '/source-files-for-coverage/chromium.googlesource.com/chromium/'
        'src.git/dir/test.cc/bbbbb')

  @mock.patch.object(utils, 'GetFileContentFromGs')
  def testServeFullRepoFileView_WithModifierAndRevision(self,
                                                        mock_get_file_from_gs):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    mock_get_file_from_gs.return_value = 'line one/nline two'

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    revision = 'aaaaa'
    path = '//dir/test.cc'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport(modifier_id=123)
    report.put()

    file_coverage_data = _CreateSampleFileCoverageData(modifier_id=123)
    file_coverage_data.put()

    request_url = ('/coverage/p/chromium/file?host=%s&project=%s&ref=%s'
                   '&revision=%s&path=%s&platform=%s&modifier_id=%d') % (
                       host, project, ref, revision, path, platform, 123)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)
    mock_get_file_from_gs.assert_called_with(
        '/source-files-for-coverage/chromium.googlesource.com/chromium/'
        'src.git/dir/test.cc/bbbbb')

  @mock.patch.object(utils, 'GetFileContentFromGs')
  def testServeFullRepoFileView_WithModifier_WithoutRevision(
      self, mock_get_file_from_gs):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    mock_get_file_from_gs.return_value = 'line one/nline two'

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    path = '//dir/test.cc'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport(modifier_id=123)
    report.put()

    file_coverage_data = _CreateSampleFileCoverageData(modifier_id=123)
    file_coverage_data.put()

    request_url = ('/coverage/p/chromium/file?host=%s&project=%s&ref=%s'
                   '&path=%s&platform=%s&modifier_id=%d') % (
                       host, project, ref, path, platform, 123)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)
    mock_get_file_from_gs.assert_called_with(
        '/source-files-for-coverage/chromium.googlesource.com/chromium/'
        'src.git/dir/test.cc/bbbbb')

  def testServeFullRepoReferencedReport_RedirectsWithModifier(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    CoverageReportModifier(
        reference_commit='past_commit',
        reference_commit_timestamp=datetime(2020, 12, 31),
        id=123).put()
    request_url = '/coverage/p/chromium/referenced2021'
    response = self.test_app.get(request_url)
    self.assertEqual(302, response.status_int)
    self.assertIn('/coverage/p/chromium?modifier_id=123',
                  response.headers.get('Location', ''))

  @mock.patch.object(utils, 'GetFileContentFromGs')
  def testServeFullRepoFileViewWithNonAsciiChars(self, mock_get_file_from_gs):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    mock_get_file_from_gs.return_value = 'line one\n═══════════╪'
    report = _CreateSamplePostsubmitReport()
    report.put()

    file_coverage_data = _CreateSampleFileCoverageData()
    file_coverage_data.put()

    request_url = ('/coverage/p/chromium/file?host=%s&project=%s&ref=%s'
                   '&revision=%s&path=%s&platform=%s') % (
                       'chromium.googlesource.com', 'chromium/src',
                       'refs/heads/main', 'aaaaa', '//dir/test.cc', 'linux')
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)


class SplitLineIntoRegionsTest(WaterfallTestCase):

  def testRejoinSplitRegions(self):
    line = 'the quick brown fox jumped over the lazy dog'
    blocks = [{
        'first': 4,
        'last': 10,
    }, {
        'first': 20,
        'last': 23,
    }, {
        'first': 42,
        'last': 43,
    }]
    regions = serve_coverage._SplitLineIntoRegions(line, blocks)
    reconstructed_line = ''.join(region['text'] for region in regions)
    self.assertEqual(line, reconstructed_line)

  def testRegionsCorrectlySplit(self):
    line = 'onetwothreefourfivesixseven'
    blocks = [{
        'first': 4,
        'last': 6,
    }, {
        'first': 12,
        'last': 15,
    }, {
        'first': 20,
        'last': 22,
    }]
    regions = serve_coverage._SplitLineIntoRegions(line, blocks)

    self.assertEqual('one', regions[0]['text'])
    self.assertEqual('two', regions[1]['text'])
    self.assertEqual('three', regions[2]['text'])
    self.assertEqual('four', regions[3]['text'])
    self.assertEqual('five', regions[4]['text'])
    self.assertEqual('six', regions[5]['text'])
    self.assertEqual('seven', regions[6]['text'])

    # Regions should alternate between covered and uncovered.
    self.assertTrue(regions[0]['is_covered'])
    self.assertTrue(regions[2]['is_covered'])
    self.assertTrue(regions[4]['is_covered'])
    self.assertTrue(regions[6]['is_covered'])
    self.assertFalse(regions[1]['is_covered'])
    self.assertFalse(regions[3]['is_covered'])
    self.assertFalse(regions[5]['is_covered'])

  def testPrefixUncovered(self):
    line = 'NOCOVcov'
    blocks = [{'first': 1, 'last': 5}]
    regions = serve_coverage._SplitLineIntoRegions(line, blocks)
    self.assertEqual('NOCOV', regions[0]['text'])
    self.assertEqual('cov', regions[1]['text'])
    self.assertFalse(regions[0]['is_covered'])
    self.assertTrue(regions[1]['is_covered'])

  def testSuffixUncovered(self):
    line = 'covNOCOV'
    blocks = [{'first': 4, 'last': 8}]
    regions = serve_coverage._SplitLineIntoRegions(line, blocks)
    self.assertEqual('cov', regions[0]['text'])
    self.assertEqual('NOCOV', regions[1]['text'])
    self.assertTrue(regions[0]['is_covered'])
    self.assertFalse(regions[1]['is_covered'])
