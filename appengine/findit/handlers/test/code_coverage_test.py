# -*- coding: utf-8 -*-
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from datetime import datetime
import json
import mock
import webapp2

from google.appengine.ext import ndb


from gae_libs.handlers.base_handler import BaseHandler
from gae_libs import token
from handlers import code_coverage
from libs.gitiles.gitiles_repository import GitilesRepository
from model.code_coverage import CoveragePercentage
from model.code_coverage import DependencyRepository
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from model.code_coverage import PresubmitCoverageData
from model.code_coverage import SummaryCoverageData
from services.code_coverage import code_coverage_util
from services.code_coverage import feature_coverage
from services.code_coverage import files_absolute_coverage
from services.code_coverage import per_cl_metrics
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


def _CreateSamplePostsubmitReport(manifest=None, builder='linux-code-coverage'):
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
      visible=True)


def _CreateSampleDirectoryCoverageData(builder='linux-code-coverage'):
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


def _CreateSampleRootComponentCoverageData(builder='linux-code-coverage'):
  """Returns a sample component SummaryCoverageData for >> for testing purpose.

  Note: only use this method if the exact values don't matter.
  """
  return SummaryCoverageData.Create(
      server_host='chromium.googlesource.com',
      project='chromium/src',
      ref='refs/heads/main',
      revision='aaaaa',
      data_type='components',
      path='>>',
      bucket='coverage',
      builder=builder,
      data={
          'dirs': [{
              'path': 'Component>Test',
              'name': 'Component>Test',
              'summaries': _CreateSampleCoverageSummaryMetric()
          }],
          'path': '>>'
      })


def _CreateSampleFileCoverageData(builder='linux-code-coverage'):
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


class FetchSourceFileTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/fetch-source-file', code_coverage.FetchSourceFile),
      ('/coverage/task/process-data/.*', code_coverage.ProcessCodeCoverageData),
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

  def testPermissionInProcessCodeCoverageData(self):
    self.mock_current_user(user_email='test@google.com', is_admin=True)
    response = self.test_app.post(
        '/coverage/task/process-data/123?format=json', status=401)
    self.assertEqual(('Either not log in yet or no permission. '
                      'Please log in with your @google.com account.'),
                     response.json_body.get('error_message'))

  @mock.patch.object(code_coverage, '_WriteFileContentToGs')
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


class ProcessCodeCoverageDataTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/process-data/.*', code_coverage.ProcessCodeCoverageData),
  ],
                                       debug=True)

  def setUp(self):
    super(ProcessCodeCoverageDataTest, self).setUp()
    self.UpdateUnitTestConfigSettings(
        'code_coverage_settings', {
            'allowed_builders': [
                'chromium/try/linux-rel',
                'chrome/coverage/linux-code-coverage',
            ]
        })

  def tearDown(self):
    self.UpdateUnitTestConfigSettings('code_coverage_settings', {})
    super(ProcessCodeCoverageDataTest, self).tearDown()

  @mock.patch.object(code_coverage_util, 'CalculateIncrementalPercentages')
  @mock.patch.object(code_coverage_util, 'CalculateAbsolutePercentages')
  @mock.patch.object(code_coverage, '_GetValidatedData')
  @mock.patch.object(code_coverage, 'GetV2Build')
  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testProcessCLPatchData(self, mocked_is_request_from_appself,
                             mocked_get_build, mocked_get_validated_data,
                             mocked_abs_percentages, mocked_inc_percentages):
    # Mock buildbucket v2 API.
    build = mock.Mock()
    build.builder.project = 'chromium'
    build.builder.bucket = 'try'
    build.builder.builder = 'linux-rel'
    build.output.properties.items.return_value = [
        ('coverage_is_presubmit', True),
        ('coverage_gs_bucket', 'code-coverage-data'),
        ('coverage_metadata_gs_paths', [
            'presubmit/chromium-review.googlesource.com/138000/4/try/'
            'linux-rel/123456789/metadata'
        ]), ('mimic_builder_names', ['linux-rel'])
    ]
    build.input.gerrit_changes = [
        mock.Mock(
            host='chromium-review.googlesource.com',
            project='chromium/src',
            change=138000,
            patchset=4)
    ]
    mocked_get_build.return_value = build

    # Mock get validated data from cloud storage.
    coverage_data = {
        'dirs': None,
        'files': [{
            'path':
                '//dir/test.cc',
            'lines': [{
                'count': 100,
                'first': 1,
                'last': 1,
            }, {
                'count': 0,
                'first': 2,
                'last': 2,
            }],
        }],
        'summaries': None,
        'components': None,
    }
    mocked_get_validated_data.return_value = coverage_data

    abs_percentages = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=2, covered_lines=1)
    ]
    mocked_abs_percentages.return_value = abs_percentages

    inc_percentages = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=1, covered_lines=1)
    ]
    mocked_inc_percentages.return_value = inc_percentages

    request_url = '/coverage/task/process-data/build/123456789'
    response = self.test_app.post(request_url)
    self.assertEqual(200, response.status_int)
    mocked_is_request_from_appself.assert_called()

    mocked_get_validated_data.assert_called_with(
        '/code-coverage-data/presubmit/chromium-review.googlesource.com/138000/'
        '4/try/linux-rel/123456789/metadata/all.json.gz')

    expected_entity = PresubmitCoverageData.Create(
        server_host='chromium-review.googlesource.com',
        change=138000,
        patchset=4,
        data=coverage_data['files'])
    expected_entity.absolute_percentages = abs_percentages
    expected_entity.incremental_percentages = inc_percentages
    expected_entity.insert_timestamp = datetime.now()
    expected_entity.update_timestamp = datetime.now()
    fetched_entities = PresubmitCoverageData.query().fetch()

    self.assertEqual(1, len(fetched_entities))
    self.assertEqual(expected_entity.cl_patchset,
                     fetched_entities[0].cl_patchset)
    self.assertEqual(expected_entity.data, fetched_entities[0].data)
    self.assertEqual(expected_entity.absolute_percentages,
                     fetched_entities[0].absolute_percentages)
    self.assertEqual(expected_entity.incremental_percentages,
                     fetched_entities[0].incremental_percentages)
    self.assertEqual(expected_entity.based_on, fetched_entities[0].based_on)

  @mock.patch.object(code_coverage_util, 'CalculateIncrementalPercentages')
  @mock.patch.object(code_coverage_util, 'CalculateAbsolutePercentages')
  @mock.patch.object(code_coverage, '_GetValidatedData')
  @mock.patch.object(code_coverage, 'GetV2Build')
  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testProcessCLPatchDataUnitTestBuilder(self,
                                            mocked_is_request_from_appself,
                                            mocked_get_build,
                                            mocked_get_validated_data,
                                            mocked_abs_percentages,
                                            mocked_inc_percentages):
    # Mock buildbucket v2 API.
    build = mock.Mock()
    build.builder.project = 'chromium'
    build.builder.bucket = 'try'
    build.builder.builder = 'linux-rel'
    build.output.properties.items.return_value = [
        ('coverage_is_presubmit', True),
        ('coverage_gs_bucket', 'code-coverage-data'),
        ('coverage_metadata_gs_paths', [
            'presubmit/chromium-review.googlesource.com/138000/4/try/'
            'linux-rel_unit/123456789/metadata'
        ]), ('mimic_builder_names', ['linux-rel_unit'])
    ]
    build.input.gerrit_changes = [
        mock.Mock(
            host='chromium-review.googlesource.com',
            project='chromium/src',
            change=138000,
            patchset=4)
    ]
    mocked_get_build.return_value = build

    # Mock get validated data from cloud storage.
    coverage_data = {
        'dirs': None,
        'files': [{
            'path':
                '//dir/test.cc',
            'lines': [{
                'count': 100,
                'first': 1,
                'last': 1,
            }, {
                'count': 0,
                'first': 2,
                'last': 2,
            }],
        }],
        'summaries': None,
        'components': None,
    }
    mocked_get_validated_data.return_value = coverage_data

    abs_percentages = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=2, covered_lines=1)
    ]
    mocked_abs_percentages.return_value = abs_percentages

    inc_percentages = [
        CoveragePercentage(
            path='//dir/test.cc', total_lines=1, covered_lines=1)
    ]
    mocked_inc_percentages.return_value = inc_percentages

    request_url = '/coverage/task/process-data/build/123456789'
    response = self.test_app.post(request_url)
    self.assertEqual(200, response.status_int)
    mocked_is_request_from_appself.assert_called()

    mocked_get_validated_data.assert_called_with(
        '/code-coverage-data/presubmit/chromium-review.googlesource.com/138000/'
        '4/try/linux-rel_unit/123456789/metadata/all.json.gz')

    expected_entity = PresubmitCoverageData.Create(
        server_host='chromium-review.googlesource.com',
        change=138000,
        patchset=4,
        data_unit=coverage_data['files'])
    expected_entity.absolute_percentages_unit = abs_percentages
    expected_entity.incremental_percentages_unit = inc_percentages
    expected_entity.insert_timestamp = datetime.now()
    expected_entity.update_timestamp = datetime.now()
    fetched_entities = PresubmitCoverageData.query().fetch()

    self.assertEqual(1, len(fetched_entities))
    self.assertEqual(expected_entity.cl_patchset,
                     fetched_entities[0].cl_patchset)
    self.assertEqual(expected_entity.data_unit, fetched_entities[0].data_unit)
    self.assertEqual(expected_entity.absolute_percentages_unit,
                     fetched_entities[0].absolute_percentages_unit)
    self.assertEqual(expected_entity.incremental_percentages_unit,
                     fetched_entities[0].incremental_percentages_unit)
    self.assertEqual(expected_entity.based_on, fetched_entities[0].based_on)

  @mock.patch.object(code_coverage_util, 'CalculateIncrementalPercentages')
  @mock.patch.object(code_coverage_util, 'CalculateAbsolutePercentages')
  @mock.patch.object(code_coverage, '_GetValidatedData')
  @mock.patch.object(code_coverage, 'GetV2Build')
  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testProcessCLPatchDataMergingData(self, _, mocked_get_build,
                                        mocked_get_validated_data,
                                        mocked_abs_percentages,
                                        mocked_inc_percentages):
    # Mock buildbucket v2 API.
    build = mock.Mock()
    build.builder.project = 'chromium'
    build.builder.bucket = 'try'
    build.builder.builder = 'linux-rel'
    build.output.properties.items.return_value = [
        ('coverage_is_presubmit', True),
        ('coverage_gs_bucket', 'code-coverage-data'),
        ('coverage_metadata_gs_paths', [
            'presubmit/chromium-review.googlesource.com/138000/4/try/'
            'linux-rel/123456789/metadata'
        ]), ('mimic_builder_names', ['linux-rel'])
    ]
    build.input.gerrit_changes = [
        mock.Mock(
            host='chromium-review.googlesource.com',
            project='chromium/src',
            change=138000,
            patchset=4)
    ]
    mocked_get_build.return_value = build

    # Mock get validated data from cloud storage.
    coverage_data = {
        'dirs': None,
        'files': [{
            'path': '//dir/test.cc',
            'lines': [{
                'count': 100,
                'first': 1,
                'last': 1,
            }],
        }],
        'summaries': None,
        'components': None,
    }
    mocked_get_validated_data.return_value = coverage_data

    mocked_abs_percentages.return_value = []
    mocked_inc_percentages.return_value = []

    existing_entity = PresubmitCoverageData.Create(
        server_host='chromium-review.googlesource.com',
        change=138000,
        patchset=4,
        data=[{
            'path': '//dir/test.cc',
            'lines': [{
                'count': 100,
                'first': 2,
                'last': 2,
            }],
        }])
    existing_entity.put()

    request_url = '/coverage/task/process-data/build/123456789'
    response = self.test_app.post(request_url)
    self.assertEqual(200, response.status_int)

    expected_entity = PresubmitCoverageData.Create(
        server_host='chromium-review.googlesource.com',
        change=138000,
        patchset=4,
        data=[{
            'path': '//dir/test.cc',
            'lines': [{
                'count': 100,
                'first': 1,
                'last': 2,
            }],
        }])
    expected_entity.absolute_percentages = []
    expected_entity.incremental_percentages = []
    fetched_entities = PresubmitCoverageData.query().fetch()

    mocked_abs_percentages.assert_called_with(expected_entity.data)
    self.assertEqual(1, len(fetched_entities))
    self.assertEqual(expected_entity.cl_patchset,
                     fetched_entities[0].cl_patchset)
    self.assertEqual(expected_entity.data, fetched_entities[0].data)
    self.assertEqual(expected_entity.absolute_percentages,
                     fetched_entities[0].absolute_percentages)
    self.assertEqual(expected_entity.incremental_percentages,
                     fetched_entities[0].incremental_percentages)
    self.assertEqual(expected_entity.based_on, fetched_entities[0].based_on)


  @mock.patch.object(code_coverage.ProcessCodeCoverageData,
                     '_FetchAndSaveFileIfNecessary')
  @mock.patch.object(code_coverage, '_RetrieveChromeManifest')
  @mock.patch.object(code_coverage.CachedGitilesRepository, 'GetChangeLog')
  @mock.patch.object(code_coverage, '_GetValidatedData')
  @mock.patch.object(code_coverage, 'GetV2Build')
  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testProcessFullRepoData(self, mocked_is_request_from_appself,
                              mocked_get_build, mocked_get_validated_data,
                              mocked_get_change_log, mocked_retrieve_manifest,
                              mocked_fetch_file, *_):
    # Mock buildbucket v2 API.
    build = mock.Mock()
    build.builder.project = 'chrome'
    build.builder.bucket = 'coverage'
    build.builder.builder = 'linux-code-coverage'
    build.output.properties.items.return_value = [
        ('coverage_is_presubmit', False),
        ('coverage_gs_bucket', 'code-coverage-data'),
        ('coverage_metadata_gs_paths', [
            'postsubmit/chromium.googlesource.com/chromium/src/'
            'aaaaa/coverage/linux-code-coverage/123456789/metadata',
            'postsubmit/chromium.googlesource.com/chromium/src/'
            'aaaaa/coverage/linux-code-coverage_unit/123456789/metadata'
        ]),
        ('mimic_builder_names',
         ['linux-code-coverage', 'linux-code-coverage_unit'])
    ]
    build.input.gitiles_commit = mock.Mock(
        host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        id='aaaaa')
    mocked_get_build.return_value = build

    # Mock Gitiles API to get change log.
    change_log = mock.Mock()
    change_log.committer.time = datetime(2018, 1, 1)
    mocked_get_change_log.return_value = change_log

    # Mock retrieve manifest.
    manifest = _CreateSampleManifest()
    mocked_retrieve_manifest.return_value = manifest

    # Mock get validated data from cloud storage for both all.json and file
    # shard json.
    all_coverage_data = {
        'dirs': [{
            'path': '//dir/',
            'dirs': [],
            'files': [{
                'path': '//dir/test.cc',
                'name': 'test.cc',
                'summaries': _CreateSampleCoverageSummaryMetric()
            }],
            'summaries': _CreateSampleCoverageSummaryMetric()
        }],
        'file_shards': ['file_coverage/files1.json.gz'],
        'summaries':
            _CreateSampleCoverageSummaryMetric(),
        'components': [{
            'path': 'Component>Test',
            'dirs': [{
                'path': '//dir/',
                'name': 'dir/',
                'summaries': _CreateSampleCoverageSummaryMetric()
            }],
            'summaries': _CreateSampleCoverageSummaryMetric()
        }],
    }

    file_shard_coverage_data = {
        'files': [{
            'path':
                '//dir/test.cc',
            'revision':
                'bbbbb',
            'lines': [{
                'count': 100,
                'last': 2,
                'first': 1
            }],
            'timestamp':
                '140000',
            'uncovered_blocks': [{
                'line': 1,
                'ranges': [{
                    'first': 1,
                    'last': 2
                }]
            }]
        }]
    }

    mocked_get_validated_data.side_effect = [
        all_coverage_data, file_shard_coverage_data, all_coverage_data,
        file_shard_coverage_data
    ]

    request_url = '/coverage/task/process-data/build/123456789'
    response = self.test_app.post(request_url)
    self.assertEqual(200, response.status_int)
    mocked_is_request_from_appself.assert_called()

    fetched_reports = PostsubmitReport.query().fetch()
    self.assertEqual(2, len(fetched_reports))
    self.assertEqual(_CreateSamplePostsubmitReport(), fetched_reports[0])
    self.assertEqual(
        _CreateSamplePostsubmitReport(builder='linux-code-coverage_unit'),
        fetched_reports[1])
    mocked_fetch_file.assert_called_with(
        _CreateSamplePostsubmitReport(builder='linux-code-coverage_unit'),
        '//dir/test.cc', 'bbbbb')

    fetched_file_coverage_data = FileCoverageData.query().fetch()
    self.assertEqual(2, len(fetched_file_coverage_data))
    self.assertEqual(_CreateSampleFileCoverageData(),
                     fetched_file_coverage_data[0])
    self.assertEqual(
        _CreateSampleFileCoverageData(builder='linux-code-coverage_unit'),
        fetched_file_coverage_data[1])

    fetched_summary_coverage_data = SummaryCoverageData.query().fetch()
    self.assertListEqual([
        _CreateSampleRootComponentCoverageData(),
        _CreateSampleRootComponentCoverageData(
            builder='linux-code-coverage_unit'),
        _CreateSampleComponentCoverageData(),
        _CreateSampleComponentCoverageData(builder='linux-code-coverage_unit'),
        _CreateSampleDirectoryCoverageData(),
        _CreateSampleDirectoryCoverageData(builder='linux-code-coverage_unit')
    ], fetched_summary_coverage_data)

  @mock.patch.object(code_coverage.ProcessCodeCoverageData,
                     '_FetchAndSaveFileIfNecessary')
  @mock.patch.object(code_coverage, '_RetrieveChromeManifest')
  @mock.patch.object(code_coverage.CachedGitilesRepository, 'GetChangeLog')
  @mock.patch.object(code_coverage, '_IsFileAvailableInGs')
  @mock.patch.object(code_coverage, '_GetValidatedData')
  @mock.patch.object(code_coverage, 'GetV2Build')
  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testProcessFullRepoData_ReferencedCoverage(
      self, mocked_is_request_from_appself, mocked_get_build,
      mocked_get_validated_data, mocked_is_file_available_in_gs,
      mocked_get_change_log, mocked_retrieve_manifest, mocked_fetch_file, *_):
    # Mock buildbucket v2 API.
    build = mock.Mock()
    build.builder.project = 'chrome'
    build.builder.bucket = 'coverage'
    build.builder.builder = 'linux-code-coverage'
    build.output.properties.items.return_value = [
        ('coverage_is_presubmit', False),
        ('coverage_gs_bucket', 'code-coverage-data'),
        ('coverage_metadata_gs_paths', [
            'postsubmit/chromium.googlesource.com/chromium/src/'
            'aaaaa/coverage/linux-code-coverage/123456789/metadata',
        ]), ('mimic_builder_names', ['linux-code-coverage'])
    ]
    build.input.gitiles_commit = mock.Mock(
        host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        id='aaaaa')
    mocked_get_build.return_value = build

    # Mock Gitiles API to get change log.
    change_log = mock.Mock()
    change_log.committer.time = datetime(2018, 1, 1)
    mocked_get_change_log.return_value = change_log

    # Mock retrieve manifest.
    manifest = _CreateSampleManifest()
    mocked_retrieve_manifest.return_value = manifest

    # Mock get validated data from cloud storage for both all.json and file
    # shard json.
    all_coverage_data = {
        'dirs': [{
            'path': '//dir/',
            'dirs': [],
            'files': [{
                'path': '//dir/test.cc',
                'name': 'test.cc',
                'summaries': _CreateSampleCoverageSummaryMetric()
            }],
            'summaries': _CreateSampleCoverageSummaryMetric()
        }],
        'file_shards': ['file_coverage/files1.json.gz'],
        'summaries':
            _CreateSampleCoverageSummaryMetric(),
        'components': [{
            'path': 'Component>Test',
            'dirs': [{
                'path': '//dir/',
                'name': 'dir/',
                'summaries': _CreateSampleCoverageSummaryMetric()
            }],
            'summaries': _CreateSampleCoverageSummaryMetric()
        }],
    }

    file_shard_coverage_data = {
        'files': [{
            'path':
                '//dir/test.cc',
            'revision':
                'bbbbb',
            'lines': [{
                'count': 100,
                'last': 2,
                'first': 1
            }],
            'timestamp':
                '140000',
            'uncovered_blocks': [{
                'line': 1,
                'ranges': [{
                    'first': 1,
                    'last': 2
                }]
            }]
        }]
    }

    mocked_is_file_available_in_gs.return_value = True
    mocked_get_validated_data.side_effect = [
        all_coverage_data, file_shard_coverage_data, all_coverage_data,
        file_shard_coverage_data
    ]

    request_url = '/coverage/task/process-data/build/123456789'
    response = self.test_app.post(request_url)
    self.assertEqual(200, response.status_int)
    mocked_is_request_from_appself.assert_called()

    fetched_reports = PostsubmitReport.query().fetch()
    self.assertEqual(2, len(fetched_reports))
    self.assertEqual(_CreateSamplePostsubmitReport(), fetched_reports[0])
    self.assertEqual(
        _CreateSamplePostsubmitReport(builder='linux-code-coverage_referenced'),
        fetched_reports[1])
    mocked_fetch_file.assert_called_with(
        _CreateSamplePostsubmitReport(builder='linux-code-coverage_referenced'),
        '//dir/test.cc', 'bbbbb')

    fetched_file_coverage_data = FileCoverageData.query().fetch()
    self.assertEqual(2, len(fetched_file_coverage_data))
    self.assertEqual(_CreateSampleFileCoverageData(),
                     fetched_file_coverage_data[0])
    self.assertEqual(
        _CreateSampleFileCoverageData(builder='linux-code-coverage_referenced'),
        fetched_file_coverage_data[1])

    fetched_summary_coverage_data = SummaryCoverageData.query().fetch()
    self.assertListEqual([
        _CreateSampleRootComponentCoverageData(),
        _CreateSampleRootComponentCoverageData(
            builder='linux-code-coverage_referenced'),
        _CreateSampleComponentCoverageData(),
        _CreateSampleComponentCoverageData(
            builder='linux-code-coverage_referenced'),
        _CreateSampleDirectoryCoverageData(),
        _CreateSampleDirectoryCoverageData(
            builder='linux-code-coverage_referenced')
    ], fetched_summary_coverage_data)


class ServeCodeCoverageDataTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/api/coverage-data', code_coverage.ServeCodeCoverageData),
      ('.*/coverage', code_coverage.ServeCodeCoverageData),
      ('.*/coverage/component', code_coverage.ServeCodeCoverageData),
      ('.*/coverage/dir', code_coverage.ServeCodeCoverageData),
      ('.*/coverage/file', code_coverage.ServeCodeCoverageData),
  ],
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

  @mock.patch.object(code_coverage.code_coverage_util, 'GetEquivalentPatchsets')
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

  @mock.patch.object(code_coverage.code_coverage_util, 'GetEquivalentPatchsets')
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

  @mock.patch.object(code_coverage.code_coverage_util,
                     'RebasePresubmitCoverageDataBetweenPatchsets')
  @mock.patch.object(code_coverage.code_coverage_util, 'GetEquivalentPatchsets')
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

  @mock.patch.object(code_coverage.code_coverage_util,
                     'RebasePresubmitCoverageDataBetweenPatchsets')
  @mock.patch.object(code_coverage.code_coverage_util, 'GetEquivalentPatchsets')
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

  @mock.patch.object(code_coverage.code_coverage_util, 'GetEquivalentPatchsets')
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

    request_url = ('/p/chromium/coverage?host=%s&project=%s&ref=%s&platform=%s'
                   '&list_reports=true') % (host, project, ref, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  def testServeFullRepoProjectViewDefaultReportConfig(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    report = _CreateSamplePostsubmitReport()
    report.put()

    response = self.test_app.get('/p/chromium/coverage?&list_reports=true')
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
        '/p/chromium/coverage/dir?host=%s&project=%s&ref=%s&revision=%s'
        '&path=%s&platform=%s') % (host, project, ref, revision, path, platform)
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

    request_url = ('/p/chromium/coverage/component?host=%s&project=%s&ref=%s'
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
        '/p/chromium/coverage/dir?host=%s&project=%s&ref=%s&revision=%s'
        '&path=%s&platform=%s&test_suite_type=unit') % (
            host, project, ref, revision, path, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  def testServeFullRepo_ReferencedMode(self):
    self.mock_current_user(user_email='test@google.com', is_admin=False)

    host = 'chromium.googlesource.com'
    project = 'chromium/src'
    ref = 'refs/heads/main'
    revision = 'aaaaa'
    path = '//dir/'
    platform = 'linux'

    report = _CreateSamplePostsubmitReport(
        builder='linux-code-coverage_referenced')
    report.put()

    dir_coverage_data = _CreateSampleDirectoryCoverageData(
        builder='linux-code-coverage_referenced')
    dir_coverage_data.put()

    request_url = (
        '/p/chromium/coverage/dir?host=%s&project=%s&ref=%s&revision=%s'
        '&path=%s&platform=%s&reference_mode=true') % (host, project, ref,
                                                       revision, path, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)

  @mock.patch.object(code_coverage, '_GetFileContentFromGs')
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

    request_url = ('/p/chromium/coverage/file?host=%s&project=%s&ref=%s'
                   '&revision=%s&path=%s&platform=%s') % (
                       host, project, ref, revision, path, platform)
    response = self.test_app.get(request_url)
    self.assertEqual(200, response.status_int)
    mock_get_file_from_gs.assert_called_with(
        '/source-files-for-coverage/chromium.googlesource.com/chromium/'
        'src.git/dir/test.cc/bbbbb')

  @mock.patch.object(code_coverage, '_GetFileContentFromGs')
  def testServeFullRepoFileViewWithNonAsciiChars(self, mock_get_file_from_gs):
    self.mock_current_user(user_email='test@google.com', is_admin=False)
    mock_get_file_from_gs.return_value = 'line one\n═══════════╪'
    report = _CreateSamplePostsubmitReport()
    report.put()

    file_coverage_data = _CreateSampleFileCoverageData()
    file_coverage_data.put()

    request_url = ('/p/chromium/coverage/file?host=%s&project=%s&ref=%s'
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
    regions = code_coverage._SplitLineIntoRegions(line, blocks)
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
    regions = code_coverage._SplitLineIntoRegions(line, blocks)

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
    regions = code_coverage._SplitLineIntoRegions(line, blocks)
    self.assertEqual('NOCOV', regions[0]['text'])
    self.assertEqual('cov', regions[1]['text'])
    self.assertFalse(regions[0]['is_covered'])
    self.assertTrue(regions[1]['is_covered'])

  def testSuffixUncovered(self):
    line = 'covNOCOV'
    blocks = [{'first': 4, 'last': 8}]
    regions = code_coverage._SplitLineIntoRegions(line, blocks)
    self.assertEqual('cov', regions[0]['text'])
    self.assertEqual('NOCOV', regions[1]['text'])
    self.assertTrue(regions[0]['is_covered'])
    self.assertFalse(regions[1]['is_covered'])


class ExportPerClCoverageMetricsCronTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/cron/per-cl-coverage',
       code_coverage.ExportPerClCoverageMetricsCron),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testTaskAddedToQueue(self, mocked_is_request_from_appself):
    response = self.test_app.get('/coverage/cron/per-cl-coverage')
    self.assertEqual(200, response.status_int)
    response = self.test_app.get('/coverage/cron/per-cl-coverage')
    self.assertEqual(200, response.status_int)

    tasks = self.taskqueue_stub.get_filtered_tasks(
        queue_names='per-cl-coverage-metrics-queue')
    self.assertEqual(2, len(tasks))
    self.assertTrue(mocked_is_request_from_appself.called)


class ExportPerClCoverageMetricsTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/per-cl-coverage',
       code_coverage.ExportPerClCoverageMetrics),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  @mock.patch.object(per_cl_metrics, 'ExportPerClCoverage')
  def testPerClCoverageMetricsExported(self, mock_detect, _):
    response = self.test_app.get('/coverage/task/per-cl-coverage', status=200)
    self.assertEqual(1, mock_detect.call_count)
    self.assertEqual(200, response.status_int)


class ExportFilesAbsoluteCoverageMetricsCronTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/cron/files-absolute-coverage',
       code_coverage.ExportFilesAbsoluteCoverageMetricsCron),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testTaskAddedToQueue(self, mocked_is_request_from_appself):
    response = self.test_app.get('/coverage/cron/files-absolute-coverage')
    self.assertEqual(200, response.status_int)
    response = self.test_app.get('/coverage/cron/files-absolute-coverage')
    self.assertEqual(200, response.status_int)

    tasks = self.taskqueue_stub.get_filtered_tasks(
        queue_names='files-absolute-coverage-metrics-queue')
    self.assertEqual(2, len(tasks))
    self.assertTrue(mocked_is_request_from_appself.called)


class ExportFilesAbsoluteCoverageMetricsTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/files-absolute-coverage',
       code_coverage.ExportFilesAbsoluteCoverageMetrics),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  @mock.patch.object(files_absolute_coverage, 'ExportFilesAbsoluteCoverage')
  def testAbsoluteCoverageFilesExported(self, mock_detect, _):
    response = self.test_app.get(
        '/coverage/task/files-absolute-coverage', status=200)
    self.assertEqual(1, mock_detect.call_count)
    self.assertEqual(200, response.status_int)


class ExportFeatureCoverageMetricsCronTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/cron/feature-coverage',
       code_coverage.ExportFeatureCoverageMetricsCron),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  def testTaskAddedToQueue(self, mocked_is_request_from_appself):
    response = self.test_app.get('/coverage/cron/feature-coverage')
    self.assertEqual(200, response.status_int)
    response = self.test_app.get('/coverage/cron/feature-coverage')
    self.assertEqual(200, response.status_int)

    tasks = self.taskqueue_stub.get_filtered_tasks(
        queue_names='feature-coverage-metrics-queue')
    self.assertEqual(2, len(tasks))
    self.assertTrue(mocked_is_request_from_appself.called)


class ExportFeatureCoverageMetricsTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/feature-coverage',
       code_coverage.ExportFeatureCoverageMetrics),
  ],
                                       debug=True)

  @mock.patch.object(BaseHandler, 'IsRequestFromAppSelf', return_value=True)
  @mock.patch.object(feature_coverage, 'ExportFeatureCoverage')
  def testFeatureCoverageFilesExported(self, mock_detect, _):
    response = self.test_app.get('/coverage/task/feature-coverage', status=200)
    self.assertEqual(1, mock_detect.call_count)
    self.assertEqual(200, response.status_int)


class UpdatePostsubmitReportTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/postsubmit-report/update',
       code_coverage.UpdatePostsubmitReport),
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
  def testPostsubmitReportUpdated(self, *_):
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
