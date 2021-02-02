# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import logging
from datetime import datetime
from datetime import timedelta

from google.appengine.ext import ndb

from libs import time_util
from model.code_coverage import PresubmitCoverageData
from model.test_location import TestLocation
from services import bigquery_helper
from services import test_tag_util

_PAGE_SIZE = 100

# Time period for which coverage report is to fetched and processed
_NUM_REPORT_DAYS = 2


def ExportPerClCoverage():
  """Exports per CL coverage metrics to Bigquery.

  Reads presubmit coverage data from Datastore, add few other dimensions to it
  and exports it to a Bigquery table.

  """
  query = PresubmitCoverageData.query(
      PresubmitCoverageData.update_timestamp >= datetime.now() -
      timedelta(days=_NUM_REPORT_DAYS))
  dir_to_component = test_tag_util.GetChromiumDirectoryToComponentMapping()
  dir_to_team = test_tag_util.GetChromiumDirectoryToTeamMapping()
  total_rows = 0
  more = True
  cursor = None
  while more:
    results, cursor, more = query.fetch_page(_PAGE_SIZE, start_cursor=cursor)
    for result in results:
      bqrow = _CreateBigqueryRow(result, dir_to_component, dir_to_team)
      if bqrow:
        bigquery_helper.ReportRowsToBigquery([bqrow], 'findit-for-me',
                                             'code_coverage_summaries',
                                             'per_cl_coverage')
        total_rows += 1
  logging.info('Total patchsets processed = %d', total_rows)


def _CreateBigqueryRow(coverage_data, dir_to_component, dir_to_team):
  """Create a bigquery row for per cl coverage.

  Returns a dict whose keys are column names and values are column values
  corresponding to the schema of the bigquery table.

  Args:
    coverage_data (PresubmitCoverageData): The PresubmitCoverageData fetched
        from Datastore
    dir_to_component (dict): Mapping from directory to  component
    dir_to_team (dict): Mapping from directory to team
  """
  if not coverage_data.incremental_percentages:
    return None
  coverage = []

  for inc_coverage in coverage_data.incremental_percentages:
    # ignore the leading double slash(//)
    path = inc_coverage.path[2:]
    test_location = TestLocation(file_path=path)
    absolute_coverage = [
        x for x in coverage_data.absolute_percentages
        if x.path == inc_coverage.path
    ][0]
    coverage.append({
        'total_lines_incremental':
            inc_coverage.total_lines,
        'covered_lines_incremental':
            inc_coverage.covered_lines,
        'total_lines_absolute':
            absolute_coverage.total_lines,
        'covered_lines_absolute':
            absolute_coverage.covered_lines,
        'path':
            path,
        'directory':
            test_tag_util.GetTestDirectoryFromLocation(test_location),
        'component':
            test_tag_util.GetTestComponentFromLocation(test_location,
                                                       dir_to_component),
        'team':
            test_tag_util.GetTestTeamFromLocation(test_location, dir_to_team),
    })
  return {
      'cl_number': coverage_data.cl_patchset.change,
      'cl_patchset': coverage_data.cl_patchset.patchset,
      'server_host': coverage_data.cl_patchset.server_host,
      'coverage': coverage,
      'insert_timestamp': time_util.GetUTCNow().isoformat()
  }
