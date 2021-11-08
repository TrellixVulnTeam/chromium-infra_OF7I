# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

from datetime import datetime
import mock
import webapp2

from gae_libs.handlers.base_handler import BaseHandler
from handlers.code_coverage import process_coverage
from handlers.code_coverage import utils
from model.code_coverage import CoveragePercentage
from model.code_coverage import DependencyRepository
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from model.code_coverage import PresubmitCoverageData
from model.code_coverage import SummaryCoverageData
from services.code_coverage import code_coverage_util
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


class ProcessCodeCoverageDataTest(WaterfallTestCase):
  app_module = webapp2.WSGIApplication([
      ('/coverage/task/process-data/.*',
       process_coverage.ProcessCodeCoverageData),
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

  def testPermissionInProcessCodeCoverageData(self):
    self.mock_current_user(user_email='test@google.com', is_admin=True)
    response = self.test_app.post(
        '/coverage/task/process-data/123?format=json', status=401)
    self.assertEqual(('Either not log in yet or no permission. '
                      'Please log in with your @google.com account.'),
                     response.json_body.get('error_message'))

  @mock.patch.object(code_coverage_util, 'CalculateIncrementalPercentages')
  @mock.patch.object(code_coverage_util, 'CalculateAbsolutePercentages')
  @mock.patch.object(process_coverage, '_GetValidatedData')
  @mock.patch.object(process_coverage, 'GetV2Build')
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
  @mock.patch.object(process_coverage, '_GetValidatedData')
  @mock.patch.object(process_coverage, 'GetV2Build')
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
  @mock.patch.object(process_coverage, '_GetValidatedData')
  @mock.patch.object(process_coverage, 'GetV2Build')
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

  @mock.patch.object(process_coverage.ProcessCodeCoverageData,
                     '_FetchAndSaveFileIfNecessary')
  @mock.patch.object(process_coverage, '_RetrieveChromeManifest')
  @mock.patch.object(process_coverage.CachedGitilesRepository, 'GetChangeLog')
  @mock.patch.object(process_coverage, '_GetValidatedData')
  @mock.patch.object(process_coverage, 'GetV2Build')
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
