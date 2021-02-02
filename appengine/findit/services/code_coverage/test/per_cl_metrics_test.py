# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
from datetime import datetime
from datetime import timedelta
from libs import time_util

from parameterized import parameterized

from google.appengine.ext import ndb
from libs import time_util
from waterfall.test.wf_testcase import WaterfallTestCase
from model.code_coverage import PresubmitCoverageData
from model.code_coverage import CLPatchset
from services.code_coverage import per_cl_metrics
from services import bigquery_helper
from services import test_tag_util

_DEFAULT_LUCI_PROJECT = 'chromium'


class PerClMetricsTest(WaterfallTestCase):

  @mock.patch.object(
      test_tag_util,
      'GetChromiumDirectoryToComponentMapping',
      return_value={"chrome/browser/": "my_component"})
  @mock.patch.object(
      test_tag_util,
      'GetChromiumDirectoryToTeamMapping',
      return_value={"chrome/browser/": "my_team"})
  @mock.patch.object(time_util, 'GetUTCNow', return_value=datetime(2020, 9, 21))
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  def testExportPerClCoverageMetrics(self, mocked_report_rows, *_):
    datastore_entity = PresubmitCoverageData(
        key=ndb.Key('PresubmitCoverageData', 'chromium$123$1'),
        cl_patchset=CLPatchset(
            change=123, patchset=1, server_host="http://coverageserver.com"),
        incremental_percentages=[
            {
                "path": "//chrome/browser/abc",
                "covered_lines": 100,
                "total_lines": 100
            },
            {
                "path": "//chrome/browser/xyz",
                "covered_lines": 50,
                "total_lines": 100
            },
        ],
        absolute_percentages=[
            {
                "path": "//chrome/browser/abc",
                "covered_lines": 1000,
                "total_lines": 1000
            },
            {
                "path": "//chrome/browser/xyz",
                "covered_lines": 500,
                "total_lines": 1000
            },
        ],
        data={})
    datastore_entity.put()
    per_cl_metrics.ExportPerClCoverage()

    expected_bqrows = [{
        'cl_number': 123,
        'cl_patchset': 1,
        'server_host': "http://coverageserver.com",
        'coverage': [{
            'total_lines_incremental': 100,
            'covered_lines_incremental': 100,
            'total_lines_absolute': 1000,
            'covered_lines_absolute': 1000,
            'path': 'chrome/browser/abc',
            'directory': 'chrome/browser/',
            'component': 'my_component',
            'team': 'my_team'
        }, {
            'total_lines_incremental': 100,
            'covered_lines_incremental': 50,
            'total_lines_absolute': 1000,
            'covered_lines_absolute': 500,
            'path': 'chrome/browser/xyz',
            'directory': 'chrome/browser/',
            'component': 'my_component',
            'team': 'my_team'
        }],
        'insert_timestamp': '2020-09-21T00:00:00',
    }]
    mocked_report_rows.assert_called_with(expected_bqrows, 'findit-for-me',
                                          'code_coverage_summaries',
                                          'per_cl_coverage')
