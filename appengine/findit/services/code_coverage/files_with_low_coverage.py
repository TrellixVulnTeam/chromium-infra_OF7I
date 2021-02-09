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
    logging.info("Latest Revision: %s", latest_revision)
    commit_timestamp = report.commit_timestamp
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
      # NDB caches each result in the in-context cache while accessing.
      # This is problematic as due to the size of the result set,
      # cache grows beyond the memory quota. `use_cache = False` turns this off.
      #
      # Read more at:
      # https://cloud.google.com/appengine/docs/standard/python/ndb/cache#incontext
      # https://github.com/googlecloudplatform/datastore-ndb-python/issues/156#issuecomment-110869490
      results, cursor, more = query.fetch_page(
          _PAGE_SIZE,
          start_cursor=cursor,
          config=ndb.ContextOptions(use_cache=False))
      bq_rows = _CreateBigqueryRows(results, commit_timestamp)
      if bq_rows:
        bigquery_helper.ReportRowsToBigquery(bq_rows, 'findit-for-me',
                                             'code_coverage_summaries',
                                             'files_with_low_coverage')
        total_rows += len(bq_rows)
      logging.info('Total rows added so far = %d', total_rows)

    logging.info('Total rows added for builder %s = %d', builder, total_rows)


def _CreateBigqueryRows(file_coverage_results, commit_timestamp):
  """Create bigquery rows for files with low coverage.

  Returns a list of dict objects whose keys are column names and
  values are column values corresponding to the schema of the bigquery table.
  Each dict object corresponds to exactly one file below the coverage bar.

  Args:
    file_coverage_results (list): List of FileCoverageData for the latest
      full codebase report
    commit_timestamp (ndb.DateTimeProperty): Commit timestamp of the
      revision for which full codebase report was generated.
  """
  bq_rows = []
  for file_coverage_result in file_coverage_results:
    try:
      data = file_coverage_result.data
      for metric in data['summaries']:
        if metric['name'] == 'line':
          total_lines = metric['total']
          covered_lines = metric['covered']
          break
      if covered_lines / float(total_lines) < _MIN_ABS_COVERAGE_RATIO:
        bq_rows.append({
            'project': file_coverage_result.gitiles_commit.project,
            'revision': file_coverage_result.gitiles_commit.revision,
            'path': data['path'][2:],
            'total_lines': total_lines,
            'covered_lines': covered_lines,
            'commit_timestamp': commit_timestamp.isoformat(),
            'insert_timestamp': time_util.GetUTCNow().isoformat(),
            'builder': file_coverage_result.builder
        })
    except ZeroDivisionError:
      logging.warning("Encounted total_lines = 0 for file coverage %s",
                      file_coverage_result)
  return bq_rows
