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
from model.code_coverage import SummaryCoverageData
from services.code_coverage import referenced_coverage
from services import bigquery_helper


class ReferencedCoverageTest(WaterfallTestCase):

  @mock.patch.object(
      referenced_coverage,
      '_GetAllowedBuilders',
      return_value=['linux-code-coverage'])
  @mock.patch.object(GitilesRepository, 'GetSource')
  def testFileModifiedSinceReferenceCommit_FileCoverageGetsCreated(
      self, mock_file_content, *_):
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
        path='//a/myfile.cc',
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
        path='//a/myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(
        entity.data, {
            'path': '//a/myfile.cc',
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
  def testFileModifiedSinceReferenceCommit_DirSummaryGetsCreated(
      self, mock_file_content, *_):
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
        path='//a/myfile.cc',
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

    entity1 = SummaryCoverageData.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
        data_type='dirs',
        path='//a/',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    entity2 = SummaryCoverageData.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
        data_type='dirs',
        path='//',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(
        entity1.data, {
            'dirs': [],
            'path':
                '//a/',
            'summaries': [{
                'covered': 2,
                'total': 2,
                'name': 'line'
            }],
            'files': [{
                'path': '//a/myfile.cc',
                'name': 'myfile.cc',
                'summaries': [{
                    'covered': 2,
                    'total': 2,
                    'name': 'line'
                }]
            }]
        })
    self.assertEqual(
        entity2.data, {
            'dirs': [{
                'path': '//a/',
                'name': 'a/',
                'summaries': [{
                    'covered': 2,
                    'total': 2,
                    'name': 'line'
                }]
            }],
            'path': '//',
            'summaries': [{
                'covered': 2,
                'total': 2,
                'name': 'line'
            }],
            'files': []
        })

  @mock.patch.object(
      referenced_coverage,
      '_GetAllowedBuilders',
      return_value=['linux-code-coverage'])
  @mock.patch.object(GitilesRepository, 'GetSource')
  def testFileModifiedSinceReferenceCommit_PostsubmitReportGetsCreated(
      self, mock_file_content, *_):
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
        path='//a/myfile.cc',
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

    entity = PostsubmitReport.Get(
        server_host='chromium.googlesource.com',
        project='chromium/src',
        ref='refs/heads/main',
        revision='rev',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(entity.summary_metrics, [{
        'covered': 2,
        'total': 2,
        'name': 'line'
    }])

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
        path='//a/myfile.cc',
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
        path='//a/myfile.cc',
        bucket='ci',
        builder='linux-code-coverage',
        modifier_id=123)
    self.assertEqual(entity, None)
