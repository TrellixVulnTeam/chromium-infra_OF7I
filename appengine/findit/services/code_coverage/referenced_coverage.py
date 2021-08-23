# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import logging
import difflib

from google.appengine.ext import ndb

from common.findit_http_client import FinditHttpClient
from libs import time_util
from libs.gitiles.gitiles_repository import GitilesRepository
from model.code_coverage import CoverageReportModifier
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from services import bigquery_helper
from services.code_coverage import code_coverage_util
from services.code_coverage import diff_util

_PAGE_SIZE = 100

# List of builders for which coverage metrics to be exported.
# These should be ci builders.
_SOURCE_BUILDERS = [
    'linux-code-coverage'
    'win10-code-coverage',
    'android-code-coverage',
    'android-code-coverage-native',
    'ios-simulator-code-coverage',
    'linux-chromeos-code-coverage',
    'linux-code-coverage_unit',
    'win10-code-coverage_unit',
    'android-code-coverage_unit',
    'android-code-coverage-native_unit',
    'ios-simulator-code-coverage_unit',
    'linux-chromeos-code-coverage_unit',
]

_CHROMIUM_SERVER_HOST = 'chromium.googlesource.com'
_CHROMIUM_GERRIT_HOST = 'chromium-review.googlesource.com'
_CHROMIUM_PROJECT = 'chromium/src'
_CHROMIUM_REPO = GitilesRepository(
    FinditHttpClient(),
    'https://%s/%s.git' % (_CHROMIUM_SERVER_HOST, _CHROMIUM_PROJECT))


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


def _GetReferencedFileCoverage(file_coverage, modified_lines, modifier_id):
  """Returns line coverage metrics for interesting lines in a file.

  Args:
    file_coverage (FileCoverageData): File coverage report from latest full
                                      codebase run.
    modified_lines (set): Set of lines modified since the reference commit.
    modified_id (string): Id of the CoverageReportModifier corresponding to the
                          reference commit.

  Returns:
    A FileCoverageData object with coverage info dropped for all lines except
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
        if line_num == modified_line_ranges[-1]['last'] + 1:
          modified_line_ranges[-1]['last'] += 1
        else:
          # Line range gets broken by an unmodified line
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
    return FileCoverageData.Create(file_coverage.gitiles_commit.server_host,
                                   file_coverage.gitiles_commit.project,
                                   file_coverage.gitiles_commit.ref,
                                   file_coverage.gitiles_commit.revision,
                                   file_coverage.path, file_coverage.bucket,
                                   file_coverage.builder, data, modifier_id)


def _GetActiveReferenceCommits():
  """Returns commits against which coverage is to be generated.

  Returns value is a dict where key is reference commit and and value is the
  id of the corresponding CoverageReportModifier.
  """
  query = CoverageReportModifier.query(
      CoverageReportModifier.server_host == _CHROMIUM_SERVER_HOST,
      CoverageReportModifier.project == _CHROMIUM_PROJECT,
      CoverageReportModifier.is_active == True,
      CoverageReportModifier.reference_commit != None)
  commits = {}
  for x in query.fetch():
    commits[x.reference_commit] = x.key.id()
  return commits


def _GetFileContentAtCommit(file_path, revision):
  """Returns lines in a file at the specified revision.

  Args:
    file_path (string): chromium/src relative path to file whose content is to
      be fetched. Must start with '//'.
    revision (string): commit hash of the revision.

  Returns:
    A list of strings representing the lines in the file. If file is not found
    at the specified revision, an empty list is returned.
  """
  assert file_path.startswith('//'), 'All file path should start with "//".'
  content = _CHROMIUM_REPO.GetSource(file_path[2:], revision)
  return content.split('\n') if content else []


def _CreateReferencedCoverage(postsubmit_report):
  """Creates coverage entities referenced against a past commit.

  Args:
    postsubmit_report(PostsubmitReport): Full codebase report which acts as
                        input to the algorithm.
  """

  for reference_commit, modifier_id in _GetActiveReferenceCommits().items():
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
    while more:
      results, cursor, more = query.fetch_page(_PAGE_SIZE, start_cursor=cursor)
      referenced_entities = []
      for file_coverage in results:
        latest_lines = _GetFileContentAtCommit(
            file_coverage.path, file_coverage.gitiles_commit.revision)
        assert latest_lines

        reference_commit_lines = _GetFileContentAtCommit(
            file_coverage.path, reference_commit)
        modified_lines = _GetModifiedLinesSinceCommit(latest_lines,
                                                      reference_commit_lines)
        referenced_coverage = _GetReferencedFileCoverage(
            file_coverage, modified_lines, modifier_id)
        if referenced_coverage:
          referenced_entities.append(referenced_coverage)

      ndb.put_multi(referenced_entities)


def _GetAllowedBuilders():
  return _SOURCE_BUILDERS


def CreateReferencedCoverage():
  # NDB caches each result in the in-context cache while accessing.
  # This is problematic as due to the size of the result set,
  # cache grows beyond the memory quota. Turn this off to prevent oom errors.
  #
  # Read more at:
  # https://cloud.google.com/appengine/docs/standard/python/ndb/cache#incontext
  # https://github.com/googlecloudplatform/datastore-ndb-python/issues/156#issuecomment-110869490
  context = ndb.get_context()
  context.set_cache_policy(False)
  for builder in _GetAllowedBuilders():
    # Fetch latest full codebase coverage report for the builder
    query = PostsubmitReport.query(
        PostsubmitReport.gitiles_commit.server_host == _CHROMIUM_SERVER_HOST,
        PostsubmitReport.gitiles_commit.project == _CHROMIUM_PROJECT,
        PostsubmitReport.bucket == 'ci', PostsubmitReport.builder == builder,
        PostsubmitReport.visible == True).order(
            -PostsubmitReport.commit_timestamp)
    report = query.fetch(limit=1)[0]
    _CreateReferencedCoverage(report)
