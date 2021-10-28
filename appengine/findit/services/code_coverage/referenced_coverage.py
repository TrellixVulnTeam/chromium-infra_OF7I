# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from collections import defaultdict
import json
import logging
import difflib
import Queue
from threading import Thread

from google.appengine.ext import ndb

from common.findit_http_client import FinditHttpClient
from libs import time_util
from libs.gitiles.gitiles_repository import GitilesRepository
from model.code_coverage import CoverageReportModifier
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from model.code_coverage import SummaryCoverageData
from services import bigquery_helper
from services.code_coverage import aggregation_util
from services.code_coverage import code_coverage_util
from services.code_coverage import diff_util

_PAGE_SIZE = 100

_CHROMIUM_SERVER_HOST = 'chromium.googlesource.com'
_CHROMIUM_GERRIT_HOST = 'chromium-review.googlesource.com'
_CHROMIUM_PROJECT = 'chromium/src'
_CHROMIUM_REPO = GitilesRepository(
    FinditHttpClient(
        # Don't log 404 as it is expected
        # e.g. a file can missing at a parent commit.
        no_error_logging_statuses=[404]),
    'https://%s/%s.git' % (_CHROMIUM_SERVER_HOST, _CHROMIUM_PROJECT))
_EXPONENTIAL_BACKOFF_LIMIT_SECONDS = 2048


def _GetModifiedLinesSinceCommit(latest_lines, commit_lines):
  if not commit_lines:
    return []
  diff_lines = [
      x for x in difflib.unified_diff(latest_lines, commit_lines, lineterm='')
  ]
  unchanged_lines = (
      diff_util.generate_line_number_mapping(diff_lines, latest_lines,
                                             commit_lines).keys())
  modified_lines = [
      x for x in range(1,
                       len(latest_lines) + 1) if x not in unchanged_lines
  ]
  return modified_lines


def _GetReferencedFileCoverage(file_coverage, modified_lines):
  """Returns line coverage metrics for interesting lines in a file.

  Args:
    file_coverage (FileCoverageData): File coverage report from latest full
                                      codebase run.
    modified_lines (set): Set of lines modified since the reference commit.

  Returns:
    A dict containing coverage info dropped for all lines except
    modified_lines. Returns None if there are no lines with coverage info.
  """

  total = 0
  covered = 0
  # add a dummy range to simplify logic
  modified_line_ranges = [{'first': -1, 'last': -1}]
  for line_range in file_coverage.data['lines']:
    for line_num in range(line_range['first'], line_range['last'] + 1):
      if line_num in modified_lines:
        total += 1
        if line_num == modified_line_ranges[-1]['last'] + 1 and line_range[
            'count'] == modified_line_ranges[-1]['count']:
          # Append to the last interesting line range if line numbers are
          # continuous and they share the same execution count
          modified_line_ranges[-1]['last'] += 1
        else:
          # Line range gets broken by an unmodified line
          # or new line range with a different execution count is encountered
          modified_line_ranges.append({
              'first': line_num,
              'last': line_num,
              'count': line_range['count']
          })
        if line_range['count'] != 0:
          covered += 1
  if total > 0:
    data = {
        'path': file_coverage.path,
        'lines': modified_line_ranges[1:],
        'summaries': [{
            'name': 'line',
            'total': total,
            'covered': covered
        }],
        'revision': file_coverage.gitiles_commit.revision
    }
    return data


def _FetchFileContentAtCommit(file_path, revision, file_content_queue):
  """Fetches lines in a file at the specified revision.

  Args:
    file_path (string): chromium/src relative path to file whose content is to
      be fetched. Must start with '//'.
    revision (string): commit hash of the revision.
    file_content_queue (Queue): Queue which holds the output.
  """
  assert file_path.startswith('//'), 'All file path should start with "//".'
  content, status = _CHROMIUM_REPO.GetSourceAndStatus(file_path[2:], revision)
  wait_sec = 1
  # short term qps exceeded, retry with exponential backoff
  while status == 429 and wait_sec < _EXPONENTIAL_BACKOFF_LIMIT_SECONDS:
    wait_sec *= 2
    time.sleep(wait_sec)
    content, status = _CHROMIUM_REPO.GetSourceAndStatus(file_path[2:], revision)
  if wait_sec >= _EXPONENTIAL_BACKOFF_LIMIT_SECONDS:
    logging.warning(
        "Couldn't fetch content for %s at revision %s due to exceeding qps",
        file_path, revision)
  file_content_queue.put((revision, content.split('\n') if content else []))


def _FlushEntities(entities, total, last=False):
  """Creates datastore entities in a batched manner"""
  if len(entities) < 100 and not (last and entities):
    return entities, total

  ndb.put_multi(entities)
  total += len(entities)
  logging.info('Dumped %d coverage data entities', total)

  return [], total


def _ExportFileCoverage(files_coverage, postsubmit_report, modifier_id):
  """Exports file coverage entities to datastore.

  Args:
    files_coverage(dict): Mapping from file path to corresponding
                                coverage data.
    postsubmit_report(PostsubmitReport): Full codebase report for which
                                referenced entities are being exported.
    modifier_id(int): Id of the CoverageReportModifier corresponding to the
                      reference commit.
  """
  entities = []
  total = 0
  logging.info("Dumping file coverage")
  for i in range(0, len(files_coverage)):
    data = files_coverage[i]
    entity = FileCoverageData.Create(
        postsubmit_report.gitiles_commit.server_host,
        postsubmit_report.gitiles_commit.project,
        postsubmit_report.gitiles_commit.ref,
        postsubmit_report.gitiles_commit.revision, data['path'],
        postsubmit_report.bucket, postsubmit_report.builder, data, modifier_id)
    entities.append(entity)
    entities, total = _FlushEntities(entities, total)
  _FlushEntities(entities, total, last=True)


def _ExportDirSummaryCoverage(directories_coverage, postsubmit_report,
                              modifier_id):
  """Exports directory summary coverage entities to datastore.

  Args:
    directories_coverage(dict): Mapping from directory path to corresponding
                                coverage data.
    postsubmit_report(PostsubmitReport): Full codebase report for which
                                referenced entities are being exported.
    modifier_id(int): Id of the CoverageReportModifier corresponding to the
                      reference commit.
  """
  entities = []
  total = 0
  logging.info("Dumping directory coverage")
  for path, data in directories_coverage.items():
    entity = SummaryCoverageData.Create(
        postsubmit_report.gitiles_commit.server_host,
        postsubmit_report.gitiles_commit.project,
        postsubmit_report.gitiles_commit.ref,
        postsubmit_report.gitiles_commit.revision, 'dirs', path,
        postsubmit_report.bucket, postsubmit_report.builder, data, modifier_id)
    entities.append(entity)
    entities, total = _FlushEntities(entities, total)
  _FlushEntities(entities, total, last=True)


def _CreateReferencedCoverage(modifier_id, postsubmit_report):
  """Creates coverage entities referenced against a past commit.

  Args:
    modifier_id(int): id of the CoverageReportModifier
        object corresponding to the reference commit
    postsubmit_report(PostsubmitReport): Full codebase report which acts as
                        input to the algorithm.
  """

  reference_commit = CoverageReportModifier.Get(modifier_id).reference_commit
  # Fetch file coverage entities corresponding to the full codebase report
  query = FileCoverageData.query(
      FileCoverageData.gitiles_commit.server_host ==
      postsubmit_report.gitiles_commit.server_host,
      FileCoverageData.gitiles_commit.project ==
      postsubmit_report.gitiles_commit.project,
      FileCoverageData.gitiles_commit.ref ==
      postsubmit_report.gitiles_commit.ref,
      FileCoverageData.gitiles_commit.revision ==
      postsubmit_report.gitiles_commit.revision,
      FileCoverageData.bucket == postsubmit_report.bucket,
      FileCoverageData.builder == postsubmit_report.builder,
      FileCoverageData.modifier_id == 0)
  more = True
  cursor = None
  referenced_file_coverage = []
  file_content_queue = Queue.Queue()
  while more:
    results, cursor, more = query.fetch_page(_PAGE_SIZE, start_cursor=cursor)
    for file_coverage in results:
      content_at_latest_thread = Thread(
          target=_FetchFileContentAtCommit,
          args=(file_coverage.path, file_coverage.gitiles_commit.revision,
                file_content_queue))
      content_at_reference_commit_thread = Thread(
          target=_FetchFileContentAtCommit,
          args=(file_coverage.path, reference_commit, file_content_queue))
      content_at_latest_thread.start()
      content_at_reference_commit_thread.start()
      content_at_latest_thread.join()
      content_at_reference_commit_thread.join()

      # Consume content from all threads
      contents = defaultdict(list)
      while not file_content_queue.empty():
        # It's correct to do block=False as all threads have been joined before.
        k, v = file_content_queue.get(block=False)
        contents[k] = v
      assert contents[file_coverage.gitiles_commit.revision], (
          "File Content not found for path %s at commit %s" %
          (file_coverage.path, file_coverage.gitiles_commit.revision))

      referenced_coverage = _GetReferencedFileCoverage(
          file_coverage,
          _GetModifiedLinesSinceCommit(
              contents[file_coverage.gitiles_commit.revision],
              contents[reference_commit]))
      if referenced_coverage:
        referenced_file_coverage.append(referenced_coverage)

  referenced_directory_coverage, _ = (
      aggregation_util.get_aggregated_coverage_data_from_files(
          referenced_file_coverage))
  _ExportFileCoverage(referenced_file_coverage, postsubmit_report, modifier_id)
  _ExportDirSummaryCoverage(referenced_directory_coverage, postsubmit_report,
                            modifier_id)
  # Create a top level PostsubmitReport entity with visible = True
  if referenced_directory_coverage:
    referenced_report = PostsubmitReport.Create(
        server_host=postsubmit_report.gitiles_commit.server_host,
        project=postsubmit_report.gitiles_commit.project,
        ref=postsubmit_report.gitiles_commit.ref,
        revision=postsubmit_report.gitiles_commit.revision,
        bucket=postsubmit_report.bucket,
        builder=postsubmit_report.builder,
        commit_timestamp=postsubmit_report.commit_timestamp,
        manifest=postsubmit_report.manifest,
        summary_metrics=referenced_directory_coverage['//']['summaries'],
        build_id=postsubmit_report.build_id,
        visible=True,
        modifier_id=modifier_id)
    referenced_report.put()


def CreateReferencedCoverage(modifier_id, builder):
  # NDB caches each result in the in-context cache while accessing.
  # This is problematic as due to the size of the result set,
  # cache grows beyond the memory quota. Turn this off to prevent oom errors.
  #
  # Read more at:
  # https://cloud.google.com/appengine/docs/standard/python/ndb/cache#incontext
  # https://github.com/googlecloudplatform/datastore-ndb-python/issues/156#issuecomment-110869490
  context = ndb.get_context()
  context.set_cache_policy(False)
  # Fetch latest full codebase coverage report for the builder
  query = PostsubmitReport.query(
      PostsubmitReport.gitiles_commit.server_host == _CHROMIUM_SERVER_HOST,
      PostsubmitReport.gitiles_commit.project == _CHROMIUM_PROJECT,
      PostsubmitReport.bucket == 'ci', PostsubmitReport.builder == builder,
      PostsubmitReport.visible == True, PostsubmitReport.modifier_id ==
      0).order(-PostsubmitReport.commit_timestamp)
  report = query.fetch(limit=1)[0]
  _CreateReferencedCoverage(modifier_id, report)
