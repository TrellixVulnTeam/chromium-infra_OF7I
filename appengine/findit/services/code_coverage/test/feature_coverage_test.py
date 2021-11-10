# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
from datetime import datetime
import time

from google.appengine.ext import ndb

from libs import time_util
from libs.gitiles.gitiles_repository import GitilesRepository
from waterfall.test.wf_testcase import WaterfallTestCase
from model.code_coverage import CoverageReportModifier
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from model.code_coverage import SummaryCoverageData
from services.code_coverage import code_coverage_util
from services.code_coverage import feature_coverage
from services import bigquery_helper

_DEFAULT_LUCI_PROJECT = 'chromium'


def _CreateMockMergedChange(commit, parent_commit, filepath):
  return {
      'current_revision': commit,
      'revisions': {
          commit: {
              'commit': {
                  'parents': [{
                      'commit': parent_commit
                  }]
              },
              'files': {
                  filepath: {
                      # content of this dict does not matter
                  }
              }
          }
      },
      '_number': 123
  }


class FeatureIncrementalCoverageTest(WaterfallTestCase):

  # This test tests whether modified coverage entities get created in datastore,
  # with coverage data compressed in the right format. The subsequent tests do
  # not test this scenario and only test the export to bigquery.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testParityBetweenBqAndDatastoreExport(self, mock_merged_changes,
                                            mock_file_content,
                                            mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={
            'lines': [{
                'count': 10,
                'first': 1,
                'last': 5
            }, {
                'count': 0,
                'first': 6,
                'last': 7
            }]
        })
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc')
    ]
    commit_to_content = {
        'p1': 'line3\nline7',
        'c1': 'line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8',
        'latest': 'line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'latest',
        'run_id': run_id,
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'modifier_id': 123,
        'path': 'myfile.cc',
        'total_lines':
            5,  # Five interesting lines are instrumented(line 1,2,4,5,6)
        'covered_lines': 4,  # Four interesting lines are covered(line 1,2,4,5)
        'interesting_lines': 6,  # Six interesting lines (line 1,2,4,5,6,8)
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')
    entity = FileCoverageData.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(
        entity.data, {
            'path': '//myfile.cc',
            'lines': [{
                'first': 1,
                'last': 2,
                'count': 10
            }, {
                'first': 4,
                'last': 5,
                'count': 10
            }, {
                'first': 6,
                'last': 6,
                'count': 0
            }],
            'summaries': [{
                'name': 'line',
                'total': 5,
                'covered': 4
            }],
            'revision': 'latest'
        })

  # This test tests the case where feature commits adds a new file and the file
  # has not changed after that.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_AddsNewFile_FileStaysIntact(self, mock_merged_changes,
                                                   mock_file_content,
                                                   mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={
            'lines': [
                # first line is uninstrumented
                # second line is not covered
                # third line is covered
                {
                    'count': 0,
                    'first': 2,
                    'last': 2
                },
                {
                    'count': 100,
                    'first': 3,
                    'last': 3
                }
            ]
        })
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc')
    ]
    commit_to_content = {
        'p1': '',
        'c1': 'line1\nline2\nline3',
        'latest': 'line1\nline2\nline3'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'latest',
        'run_id': run_id,
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'modifier_id': 123,
        'path': 'myfile.cc',
        'total_lines':
            2,  # Two interesting lines are instrumented(line2, line3)
        'covered_lines': 1,  # One interesting lines is covered(line3)
        'interesting_lines': 3,
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')

  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testDirSummaryCoveageGetsCreated(self, mock_merged_changes,
                                       mock_file_content, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//a/myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={'lines': [{
            'count': 1,
            'first': 1,
            'last': 2
        }]})
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'a/myfile.cc')
    ]
    commit_to_content = {
        'p1': '',
        'c1': 'line1\nline2',
        'latest': 'line1\nline2'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    entity1 = SummaryCoverageData.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        data_type='dirs',
        path='//a/',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    entity2 = SummaryCoverageData.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        data_type='dirs',
        path='//',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(
        entity1.data, {
            'dirs': [],
            'path':
                '//a/',
            'summaries': [{
                'covered': 2,
                'total': 2,
                'name': 'line'
            }],
            'files': [{
                'path': '//a/myfile.cc',
                'name': 'myfile.cc',
                'summaries': [{
                    'covered': 2,
                    'total': 2,
                    'name': 'line'
                }]
            }]
        })
    self.assertEqual(
        entity2.data, {
            'dirs': [{
                'path': '//a/',
                'name': 'a/',
                'summaries': [{
                    'covered': 2,
                    'total': 2,
                    'name': 'line'
                }]
            }],
            'path': '//',
            'summaries': [{
                'covered': 2,
                'total': 2,
                'name': 'line'
            }],
            'files': []
        })

  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testPostsubmitReportGetsCreated(self, mock_merged_changes,
                                      mock_file_content, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//a/myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={'lines': [{
            'count': 1,
            'first': 1,
            'last': 2
        }]})
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'a/myfile.cc')
    ]
    commit_to_content = {
        'p1': '',
        'c1': 'line1\nline2',
        'latest': 'line1\nline2'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    entity = PostsubmitReport.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(entity.summary_metrics, [{
        'covered': 2,
        'total': 2,
        'name': 'line'
    }])
  # This test tests the case where feature commit adds new lines to an existing
  # file and the file has not changed after that.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_ModifiesExistingFile_FileStaysIntact(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={'lines': [{
            'count': 100,
            'first': 1,
            'last': 2
        }]})
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc')
    ]
    commit_to_content = {
        'p1': 'line1',
        'c1': 'line1\nline2',
        'latest': 'line1\nline2'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'latest',
        'run_id': run_id,
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'modifier_id': 123,
        'path': 'myfile.cc',
        'total_lines': 1,  # One interesting line is instrumented(line2)
        'covered_lines': 1,  # One interesting line is covered(line2)
        'interesting_lines': 1,
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')

  # This test tests the case where feature commit adds new lines to an existing
  # file, but a part of those modifications got overwritten by a commit outside
  # feature boundaries, thus reducing the number of interesting lines.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_ModifiesExistingFile_GetsPartiallyModifiedOutsideFeature(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={'lines': [{
            'count': 100,
            'first': 1,
            'last': 3
        }]})
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc')
    ]
    commit_to_content = {
        'p1': 'line1',
        'c1': 'line1\nline2\nline3',
        'latest': 'line1\nline2\nline3 modified'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'latest',
        'run_id': run_id,
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'modifier_id': 123,
        'path': 'myfile.cc',
        'total_lines': 1,  # One interesting line is instrumented(line2)
        'covered_lines': 1,  # One interesting line is covered(line2)
        'interesting_lines': 1,
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')

  # This test tests the case where feature commit adds new lines to an existing
  # file, but ALL those modifications got overwritten by a commit outside
  # feature boundaries, thus leaving no interesting lines.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_ModifiesExistingFile_GetsOverwrittenOutsideFeature(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={'lines': [{
            'count': 100,
            'first': 1,
            'last': 3
        }]})
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc')
    ]
    commit_to_content = {
        'p1': 'line1',
        'c1': 'line1\nline2\nline3',
        'latest': 'line1\nline2 modified\nline3 modified'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = time.time()

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    self.assertFalse(mocked_report_rows.called)

  # This test tests the case where a file is modified by a feature commit, but
  # later got deleted/moved outside feature boundaries
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_FileGotDeleted(self, mock_merged_changes,
                                      mock_file_content, mocked_report_rows,
                                      *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc')
    ]
    commit_to_content = {'p1': 'line1', 'c1': 'line1\nline2', 'latest': ''}
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    self.assertFalse(mocked_report_rows.called)

  # This test tests the overlap between two feature commits i.e.
  # it tests the case where feature commit adds new lines to an existing file,
  # but a part of those modifications gets overwritten by another feature commit
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testMultipleCommits_ModifiesExistingFile_SecondCommitPartiallyOverlaps(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={'lines': [{
            'count': 100,
            'first': 1,
            'last': 3
        }]})
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc'),
        _CreateMockMergedChange('c2', 'c1', 'myfile.cc'),
    ]
    commit_to_content = {
        'p1': 'line1',
        'c1': 'line1\nline2\nline3',
        'c2': 'line1\nline2\nline3 modified',
        'latest': 'line1\nline2\nline3 modified'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = time.time()

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'latest',
        'run_id': run_id,
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'modifier_id': 123,
        'path': 'myfile.cc',
        'total_lines': 2,  # Two interesting lines are instrumented
        # (line2 and line3 modified)
        'covered_lines': 2,  # Two interesting lines are covered
        # (line2 and line3 modified)
        'interesting_lines': 2,
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')

  # This test tests the case where a file is modified by two feature commits,
  # but later got deleted/moved outside feature boundaries
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testMultipleCommits_FileGotDeleted(self, mock_merged_changes,
                                         mock_file_content, mocked_report_rows,
                                         *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc'),
        _CreateMockMergedChange('c2', 'c1', 'myfile.cc'),
    ]
    commit_to_content = {
        'p1': 'line1',
        'c1': 'line1\nline2',
        'c2': 'line1modified\nline2modified',
        'latest': ''
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    self.assertFalse(mocked_report_rows.called)

  # This test tests the case where the file under consideration is not supported
  # by coverage tooling e.g. xml, proto etc.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testUnsupportedFileType_NoRowsCreated(self, mock_merged_changes,
                                            mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.xml'),
    ]
    run_id = time.time()

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    self.assertFalse(mocked_report_rows.called)

  # This test tests the case where the file under consideration is not present
  # in the latest coverage report, but such a file has non zero interesting
  # lines.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testMissingCoverage_NonZeroInterestingLines_EmptyRowsCreated(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc'),
    ]
    commit_to_content = {'p1': '', 'c1': 'line1', 'latest': 'line1'}
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'latest',
        'run_id': run_id,
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'modifier_id': 123,
        'path': 'myfile.cc',
        'total_lines': None,
        'covered_lines': None,
        'interesting_lines': 1,
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')

  # This test tests the case where the file under consideration is not present
  # in the latest coverage report. And also such a file as zero interesting
  # lines. This could be because file was part of feature commit, but the
  # changes made in those commits were overridden later on
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testMissingCoverage_ZeroInterestingLines_NoRowsCreated(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc'),
    ]
    commit_to_content = {
        'p1': 'line1',
        'c1': 'line1 modified',
        'latest': 'line1'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    self.assertFalse(mocked_report_rows.called)

  # This test tests the case when qps to gitiles exceeds short term limit
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testExponentialBackoffWithGitiles(self, mock_merged_changes,
                                        mock_file_content, mocked_report_rows,
                                        *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={'lines': [{
            'count': 100,
            'first': 1,
            'last': 2
        }]})
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc')
    ]
    commit_to_content = {
        'p1': 'line1',
        'c1': 'line1\nline2',
        'latest': 'line1\nline2'
    }

    qps_exceeded_already = []

    def _getMockContent(_, revision):
      if revision not in qps_exceeded_already:
        qps_exceeded_already.append(revision)
        return '', 429
      else:
        return commit_to_content[revision], 200

    mock_file_content.side_effect = _getMockContent
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'latest',
        'run_id': run_id,
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'modifier_id': 123,
        'path': 'myfile.cc',
        'total_lines': 1,  # One interesting line is instrumented(line2)
        'covered_lines': 1,  # One interesting line is covered(line2)
        'interesting_lines': 1,
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')

  # This test tests the case when two builders with different supporting
  # file types generate PostsubmitReports at the same commit. In this case,
  # bigquery table should contain rows only for the one supporting the file
  # type
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={
          'linux-code-coverage': ['.cc'],
          'android-code-coverage': ['.java']
      })
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(GitilesRepository, 'GetSourceAndStatus')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testMultipleBuildersWithReportsAtSameRevision(self, mock_merged_changes,
                                                    mock_file_content,
                                                    mocked_report_rows, *_):
    CoverageReportModifier(gerrit_hashtag='my_feature', id=123).put()
    PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True).put()
    PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        bucket='ci',
        builder='android-code-coverage',
        commit_timestamp=datetime(2020, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2001,
        visible=True).put()
    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='latest',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={'lines': [{
            'count': 100,
            'first': 1,
            'last': 2
        }]})
    file_coverage_data.put()
    mock_merged_changes.return_value = [
        _CreateMockMergedChange('c1', 'p1', 'myfile.cc')
    ]
    commit_to_content = {
        'p1': 'line1',
        'c1': 'line1\nline2',
        'latest': 'line1\nline2'
    }
    mock_file_content.side_effect = (
        lambda path, revision: (commit_to_content[revision], 200))
    run_id = int(time.time())

    feature_coverage.ExportFeatureCoverage(123, run_id)

    mock_merged_changes.assert_called_with('chromium-review.googlesource.com',
                                           'chromium/src', 'my_feature')
    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'latest',
        'run_id': run_id,
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'modifier_id': 123,
        'path': 'myfile.cc',
        'total_lines': 1,  # One interesting line is instrumented(line2)
        'covered_lines': 1,  # One interesting line is covered(line2)
        'interesting_lines': 1,
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')
