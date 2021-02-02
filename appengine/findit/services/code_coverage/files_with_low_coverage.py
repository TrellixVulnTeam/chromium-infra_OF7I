# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import logging
from datetime import datetime
from datetime import timedelta

from google.appengine.ext import ndb

from libs import time_util
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from services import bigquery_helper

_PAGE_SIZE = 100

# List of builders for which coverage metrics to be exported.
# These should be ci builders.
_SOURCE_BUILDERS = ['linux-code-coverage']

# Coverage bar.
# Files below this coverage ratio will be considered 'low' coverage.
_MIN_ABS_COVERAGE_RATIO = 0.7


def ExportFilesWithLowCoverage():
  """Exports metrics for files with low coverage to Bigquery.

  Reads FileCoverageData for latest revision, keeps only those not meeting the
  coverage bar and exports them to a Bigquery table.
  """
  total_rows = 0
  project = "chromium/src"
  server_host = "chromium.googlesource.com"
  for builder in _SOURCE_BUILDERS:
    # Find latest revision
    query = PostsubmitReport.query(
        PostsubmitReport.gitiles_commit.server_host == server_host,
        PostsubmitReport.gitiles_commit.project == project,
        PostsubmitReport.bucket == "ci", PostsubmitReport.builder == builder,
        PostsubmitReport.visible == True).order(
            -PostsubmitReport.commit_timestamp)
    entities = query.fetch(limit=1)
    report = entities[0]
    latest_revision = report.gitiles_commit.revision
    # Process File Coverage reports for the latest revision
    query = FileCoverageData.query(
        FileCoverageData.gitiles_commit.server_host == server_host,
        FileCoverageData.gitiles_commit.project == project,
        FileCoverageData.gitiles_commit.ref == "refs/heads/master",
        FileCoverageData.gitiles_commit.revision == latest_revision,
        FileCoverageData.bucket == "ci", FileCoverageData.builder == builder)
    more = True
    cursor = None
    while more:
      results, cursor, more = query.fetch_page(_PAGE_SIZE, start_cursor=cursor)
      for result in results:
        bq_row = _CreateBigqueryRow(result)
        if bq_row:
          bigquery_helper.ReportRowsToBigquery([bq_row], 'findit-for-me',
                                               'code_coverage_summaries',
                                               'files_with_low_coverage')
          total_rows += 1
  logging.info('Total rows appended = %d', total_rows)


def _CreateBigqueryRow(file_coverage_data):
  """Create a bigquery row for a file with low coverage.

  Returns a dict whose keys are column names and values are column values
  corresponding to the schema of the bigquery table. Returns None if 
  file's coverage is above the bar.

  Args:
    file_coverage_data (FileCoverageData): Coverage report for the
      corresponding file.
  """
  data = file_coverage_data.data
  for metric in data['summaries']:
    if metric['name'] == 'line':
      total_lines = metric['total']
      covered_lines = metric['covered']
      break
  if covered_lines / float(total_lines) < _MIN_ABS_COVERAGE_RATIO:
    return {
        'project': file_coverage_data.gitiles_commit.project,
        'revision': file_coverage_data.gitiles_commit.revision,
        'path': data['path'][2:],
        'total_lines': total_lines,
        'covered_lines': covered_lines,
        'insert_timestamp': time_util.GetUTCNow().isoformat(),
        'builder': file_coverage_data.builder
    }
