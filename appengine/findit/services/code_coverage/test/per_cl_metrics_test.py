# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock

from datetime import datetime
from datetime import timedelta
from parameterized import parameterized

from google.appengine.ext import ndb
from waterfall.test.wf_testcase import WaterfallTestCase
from model.code_coverage import PresubmitCoverageData
from model.code_coverage import CLPatchset
from services.code_coverage import per_cl_metrics
from services import bigquery_helper

_DEFAULT_LUCI_PROJECT = 'chromium'


class PerClMetricsTest(WaterfallTestCase):

  @parameterized.expand([(
      # multipleFiles_shouldAddCoverages
      {
          'datastore_entity':
              PresubmitCoverageData(
                  key=ndb.Key('PresubmitCoverageData', 'chromium$123$1'),
                  cl_patchset=CLPatchset(
                      change=123,
                      patchset=1,
                      server_host="http://coverageserver.com"),
                  update_timestamp=datetime.now() - timedelta(days=2),
                  incremental_percentages=[
                      {
                          "path": "//chrome/browser/abc",
                          "covered_lines": 100,
                          "total_lines": 100
                      },
                      {
                          "path": "//components/variations/xyz",
                          "covered_lines": 50,
                          "total_lines": 100
                      },
                  ],
                  data={}),
          'expected_bqrows': [{
              'cl_number':
                  123,
              'cl_patchset':
                  1,
              'server_host':
                  "http://coverageserver.com",
              'incremental_coverage': [{
                  'total_lines': 100,
                  'covered_lines': 100,
                  'path': '//chrome/browser/abc'
              }, {
                  'total_lines': 100,
                  'covered_lines': 50,
                  'path': '//components/variations/xyz'
              }]
          }]
      },)])
  @mock.patch.object(bigquery_helper, '_GetBigqueryClient')
  @mock.patch.object(bigquery_helper, 'ReportRowsToBigquery', return_value={})
  def testExportPerClCoverageMetrics(self, cases, mocked_report_rows, *_):
    cases['datastore_entity'].put()
    per_cl_metrics.ExportPerClCoverageMetrics()
    mocked_report_rows.assert_called_with(cases['expected_bqrows'],
                                          'findit-for-me',
                                          'code_coverage_summaries',
                                          'per_cl_coverage')
