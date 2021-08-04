# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
from datetime import datetime

from google.appengine.ext import ndb

from libs import time_util
from gae_libs.gitiles.cached_gitiles_repository import CachedGitilesRepository
from waterfall.test.wf_testcase import WaterfallTestCase
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
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

  # This test tests the case where feature commits adds a new file and the file
  # has not changed after that.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(
      feature_coverage,
      '_GetWatchedFeatureHashtags',
      return_value=['my_feature'])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(CachedGitilesRepository, 'GetSource')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_AddsNewFile_FileStaysIntact(self, mock_merged_changes,
                                                   mock_file_content,
                                                   mocked_report_rows, *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
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
        revision='rev',
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
    content_at_parent_commit = ''
    content_at_feature_commit = 'line1\nline2\nline3'
    latest_content = 'line1\nline2\nline3'
    mock_file_content.side_effect = [
        latest_content, content_at_feature_commit, content_at_parent_commit
    ]

    feature_coverage.ExportFeatureCoverage()

    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'rev',
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'path': 'myfile.cc',
        'total_lines':
            2,  # Two interesting lines are instrumented(line2, line3)
        'covered_lines': 1,  # One interesting lines is covered(line3)
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')

  # This test tests the case where feature commit adds new lines to an existing
  # file and the file has not changed after that.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(
      feature_coverage,
      '_GetWatchedFeatureHashtags',
      return_value=['my_feature'])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(CachedGitilesRepository, 'GetSource')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_ModifiesExistingFile_FileStaysIntact(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
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
        revision='rev',
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
    content_at_parent_commit = 'line1'
    content_at_feature_commit = 'line1\nline2'
    latest_content = 'line1\nline2'
    mock_file_content.side_effect = [
        latest_content, content_at_feature_commit, content_at_parent_commit
    ]

    feature_coverage.ExportFeatureCoverage()

    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'rev',
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'path': 'myfile.cc',
        'total_lines': 1,  # One interesting line is instrumented(line2)
        'covered_lines': 1,  # One interesting line is covered(line2)
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
  @mock.patch.object(
      feature_coverage,
      '_GetWatchedFeatureHashtags',
      return_value=['my_feature'])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(CachedGitilesRepository, 'GetSource')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_ModifiesExistingFile_FileGetsModfiedOutsideFeature(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
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
        revision='rev',
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
    content_at_parent_commit = 'line1'
    content_at_feature_commit = 'line1\nline2\nline3'
    latest_content = 'line1\nline2\nline3 modified'
    mock_file_content.side_effect = [
        latest_content, content_at_feature_commit, content_at_parent_commit
    ]

    feature_coverage.ExportFeatureCoverage()

    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'rev',
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'path': 'myfile.cc',
        'total_lines': 1,  # One interesting line is instrumented(line2)
        'covered_lines': 1,  # One interesting line is covered(line2)
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')

  # This test tests the case where a file is modified by a feature commit, but
  # later got deleted/moved outside feature boundaries
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(
      feature_coverage,
      '_GetWatchedFeatureHashtags',
      return_value=['my_feature'])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(CachedGitilesRepository, 'GetSource')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testSingleCommit_FileGotDeleted(self, mock_merged_changes,
                                      mock_file_content, mocked_report_rows,
                                      *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
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
    content_at_parent_commit = 'line1'
    content_at_feature_commit = 'line1\nline2'
    latest_content = ''
    mock_file_content.side_effect = [
        latest_content, content_at_feature_commit, content_at_parent_commit
    ]

    feature_coverage.ExportFeatureCoverage()
    self.assertFalse(mocked_report_rows.called)

  # This test tests the overlap between two feature commits i.e.
  # it tests the case where feature commit adds new lines to an existing file,
  # but a part of those modifications gets overwritten by another feature commit
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(
      feature_coverage,
      '_GetWatchedFeatureHashtags',
      return_value=['my_feature'])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(CachedGitilesRepository, 'GetSource')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testMultipleCommits_ModifiesExistingFile_SecondCommitPartiallyOverlaps(
      self, mock_merged_changes, mock_file_content, mocked_report_rows, *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
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
        revision='rev',
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
    content_at_parent_commit1 = 'line1'
    content_at_feature_commit1 = 'line1\nline2\nline3'
    content_at_feature_commit2 = 'line1\nline2\nline3 modified'
    latest_content = 'line1\nline2\nline3 modified'

    mock_file_content.side_effect = [
        latest_content, content_at_feature_commit1, content_at_parent_commit1,
        latest_content, content_at_feature_commit2, content_at_feature_commit1
    ]

    feature_coverage.ExportFeatureCoverage()

    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'rev',
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'path': 'myfile.cc',
        'total_lines': 2,  # Two interesting lines are instrumented
        # (line2 and line3 modified)
        'covered_lines': 2,  # Two interesting lines are covered
        # (line2 and line3 modified)
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
  @mock.patch.object(
      feature_coverage,
      '_GetWatchedFeatureHashtags',
      return_value=['my_feature'])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(CachedGitilesRepository, 'GetSource')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testMultipleCommits_FileGotDeleted(self, mock_merged_changes,
                                         mock_file_content, mocked_report_rows,
                                         *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
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

    latest_content = ''
    content_at_feature_commit1 = 'line1\nline2'
    content_at_parent_commit1 = 'line1'
    content_at_feature_commit2 = 'line1modified\nline2modified'

    mock_file_content.side_effect = [
        latest_content, content_at_feature_commit1, content_at_parent_commit1,
        latest_content, content_at_feature_commit2, content_at_feature_commit1
    ]

    feature_coverage.ExportFeatureCoverage()
    self.assertFalse(mocked_report_rows.called)

  # This test tests the case where the file under consideration is not supported
  # by coverage tooling e.g. xml, proto etc.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(
      feature_coverage,
      '_GetWatchedFeatureHashtags',
      return_value=['my_feature'])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testUnsupportedFileType_NoRowsCreated(self, mock_merged_changes,
                                            mocked_report_rows, *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
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

    feature_coverage.ExportFeatureCoverage()
    self.assertFalse(mocked_report_rows.called)

  # This test tests the case where the file under consideration is not present
  # in the latest coverage report.
  @mock.patch.object(
      feature_coverage,
      '_GetAllowedBuilders',
      return_value={'linux-code-coverage': ['.cc']})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(
      feature_coverage,
      '_GetWatchedFeatureHashtags',
      return_value=['my_feature'])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  @mock.patch.object(CachedGitilesRepository, 'GetSource')
  @mock.patch.object(code_coverage_util, 'FetchMergedChangesWithHashtag')
  def testMissinCoverageFileType_EmptyRowsCreated(self, mock_merged_changes,
                                                  mock_file_content,
                                                  mocked_report_rows, *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
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
    latest_content = 'line1'
    content_at_feature_commit = 'line1'
    content_at_parent_commit = ''
    mock_file_content.side_effect = [
        latest_content, content_at_feature_commit, content_at_parent_commit
    ]

    feature_coverage.ExportFeatureCoverage()

    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'rev',
        'builder': 'linux-code-coverage',
        'gerrit_hashtag': 'my_feature',
        'path': 'myfile.cc',
        'total_lines': None,
        'covered_lines': None,
        'commit_timestamp': '2020-01-07T00:00:00',
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'feature_coverage')
