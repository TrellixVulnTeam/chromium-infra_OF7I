# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
from datetime import datetime
from datetime import timedelta
from libs import time_util

from google.appengine.ext import ndb
from libs import time_util
from waterfall.test.wf_testcase import WaterfallTestCase
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from services.code_coverage import files_with_low_coverage
from services import bigquery_helper

_DEFAULT_LUCI_PROJECT = 'chromium'


class FilesLowAbsCoverageTest(WaterfallTestCase):

  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  def testExportPerClCoverageMetrics_shouldSelectLatestReport(
      self, mocked_report_rows, *_):

    old_postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/master',
        revision='old_rev',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2021, 1, 6),
        manifest=[],
        summary_metrics={},
        build_id=1000,
        visible=True)
    old_postsubmit_report.put()
    new_postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/master',
        revision='new_rev',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2021, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    new_postsubmit_report.put()

    file_coverage_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/master',
        revision='new_rev',
        path='//path/to/file.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={
            'path': '//path/to/file.cc',
            'summaries': [{
                'name': 'line',
                'total': 100,
                'covered': 50
            }]
        })
    file_coverage_data.put()

    files_with_low_coverage.ExportFilesWithLowCoverage()

    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'new_rev',
        'path': 'path/to/file.cc',
        'total_lines': 100,
        'covered_lines': 50,
        'insert_timestamp': '2020-09-21T00:00:00',
        'builder': 'linux-code-coverage'
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'files_with_low_coverage')

  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  def testExportPerClCoverageMetrics_shouldFilterOutHighCoverageFiles(
      self, mocked_report_rows, *_):

    postsubmit_report = PostsubmitReport.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/master',
        revision='rev',
        bucket='ci',
        builder='linux-code-coverage',
        commit_timestamp=datetime(2021, 1, 7),
        manifest=[],
        summary_metrics={},
        build_id=2000,
        visible=True)
    postsubmit_report.put()

    low_coverage_file_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/master',
        revision='rev',
        path='//path/to/file_low.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={
            'path': '//path/to/file_low.cc',
            'summaries': [{
                'name': 'line',
                'total': 100,
                'covered': 50
            }]
        })
    low_coverage_file_data.put()

    high_coverage_file_data = FileCoverageData.Create(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/master',
        revision='rev',
        path='//path/to/file_high.cc',
        bucket='ci',
        builder='linux-code-coverage',
        data={
            'path': '//path/to/file_high.cc',
            'summaries': [{
                'name': 'line',
                'total': 100,
                'covered': 75
            }]
        })
    high_coverage_file_data.put()

    files_with_low_coverage.ExportFilesWithLowCoverage()

    expected_bq_rows = [{
        'project': 'chromium/src',
        'revision': 'rev',
        'path': 'path/to/file_low.cc',
        'total_lines': 100,
        'covered_lines': 50,
        'insert_timestamp': '2020-09-21T00:00:00',
        'builder': 'linux-code-coverage'
    }]
    mocked_report_rows.assert_called_with(expected_bq_rows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'files_with_low_coverage')
