# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
from datetime import datetime

from google.appengine.ext import ndb

from libs.gitiles.gitiles_repository import GitilesRepository
from libs import time_util
from waterfall.test.wf_testcase import WaterfallTestCase
from model.code_coverage import CoverageReportModifier
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from services.code_coverage import referenced_coverage
from services import bigquery_helper


class ReferencedCoverageTest(WaterfallTestCase):

  @mock.patch.object(
      referenced_coverage,
      '_GetAllowedBuilders',
      return_value=['linux-code-coverage'])
  @mock.patch.object(GitilesRepository, 'GetSource')
  def testFileModifiedSinceReferenceCommit(self, mock_file_content, *_):
    CoverageReportModifier(reference_commit='past_commit', id=123).put()
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
            'count': 10,
            'first': 1,
            'last': 5
        }]})
    file_coverage_data.put()
    content_at_feature_commit = 'line1\nline2\nline3'
    latest_content = 'line1\nline2\nline3\nline4\nline5'
    mock_file_content.side_effect = [latest_content, content_at_feature_commit]

    referenced_coverage.CreateReferencedCoverage()

    entity = FileCoverageData.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(
        entity.data, {
            'path': '//myfile.cc',
            'lines': [{
                'first': 4,
                'last': 5,
                'count': 10
            }],
            'summaries': [{
                'name': 'line',
                'total': 2,
                'covered': 2
            }],
            'revision': 'rev'
        })

  @mock.patch.object(
      referenced_coverage,
      '_GetAllowedBuilders',
      return_value=['linux-code-coverage'])
  @mock.patch.object(GitilesRepository, 'GetSource')
  def testFileUnmodifiedSinceReferenceCommit(self, mock_file_content, *_):
    CoverageReportModifier(reference_commit='past_commit', id=123).put()
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
            'count': 10,
            'first': 1,
            'last': 5
        }]})
    file_coverage_data.put()
    content_at_feature_commit = 'line1\nline2\nline3\nline4\nline5'
    latest_content = 'line1\nline2\nline3\nline4\nline5'
    mock_file_content.side_effect = [latest_content, content_at_feature_commit]

    referenced_coverage.CreateReferencedCoverage()

    entity = FileCoverageData.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
        path='//myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(entity, None)
