# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import logging
from datetime import datetime
from datetime import timedelta

from google.appengine.ext import ndb

from model.code_coverage import PresubmitCoverageData
from services import bigquery_helper

_PAGE_SIZE = 100


def ExportPerClCoverageMetrics():
  """Reads presubmit coverage data from Datastore and exports
  it to a Bigquery table."""
  query = PresubmitCoverageData.query(
      PresubmitCoverageData.update_timestamp >= datetime.now() -
      timedelta(days=7))
  total_rows = 0
  bigquery_rows = []
  more = True
  cursor = None
  while more:
    results, cursor, more = query.fetch_page(_PAGE_SIZE, start_cursor=cursor)
    for result in results:
      bigquery_rows.append(_CreateBigqueryRow(result))
      total_rows += 1
  logging.info('Total patchsets processed = %d', total_rows)
  bigquery_helper.ReportRowsToBigquery(bigquery_rows, 'findit-for-me',
                                       'code_coverage_summaries',
                                       'per_cl_coverage')


def _CreateBigqueryRow(coverage_data):
  """Create a bigquery row for per cl coverage.

    Returns a dict whose keys are column names and values are column values
    corresponding to the schema of the bigquery table.

    Args:
    coverage_data (PresubmitCoverageData): The PresubmitCoverageData fetched
  from Datastore
    """
  incremental_coverage = []
  for coverage_percentage in coverage_data.incremental_percentages:
    incremental_coverage.append({
        'total_lines': coverage_percentage.total_lines,
        'covered_lines': coverage_percentage.covered_lines,
        'path': coverage_percentage.path
    })

  return {
      'cl_number': coverage_data.cl_patchset.change,
      'cl_patchset': coverage_data.cl_patchset.patchset,
      'server_host': coverage_data.cl_patchset.server_host,
      'incremental_coverage': incremental_coverage
  }
