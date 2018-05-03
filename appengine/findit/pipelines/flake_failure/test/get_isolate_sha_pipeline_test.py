# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock

from dto.flake_try_job_report import FlakeTryJobReport
from dto.flake_try_job_result import FlakeTryJobResult
from dto.isolated_tests import IsolatedTests
from gae_libs import pipelines
from gae_libs.pipeline_wrapper import pipeline_handlers
from libs.list_of_basestring import ListOfBasestring
from model.flake.flake_try_job import FlakeTryJob
from model.flake.master_flake_analysis import MasterFlakeAnalysis
from pipelines.flake_failure.get_isolate_sha_pipeline import (
    GetIsolateShaForBuildParameters)
from pipelines.flake_failure.get_isolate_sha_pipeline import (
    GetIsolateShaForBuildPipeline)
from pipelines.flake_failure.get_isolate_sha_pipeline import (
    GetIsolateShaForCommitPositionParameters)
from pipelines.flake_failure.get_isolate_sha_pipeline import (
    GetIsolateShaForCommitPositionPipeline)
from pipelines.flake_failure.get_isolate_sha_pipeline import (
    GetIsolateShaForTryJobParameters)
from pipelines.flake_failure.get_isolate_sha_pipeline import (
    GetIsolateShaForTryJobPipeline)
from pipelines.flake_failure.get_isolate_sha_pipeline import (
    GetIsolateShaOutput)
from pipelines.flake_failure.run_flake_try_job_pipeline import (
    RunFlakeTryJobParameters)
from pipelines.flake_failure.run_flake_try_job_pipeline import (
    RunFlakeTryJobPipeline)
from services import step_util
from services import swarming
from services import swarmbot_util
from waterfall import build_util
from waterfall import buildbot
from waterfall import waterfall_config
from waterfall.build_info import BuildInfo
from waterfall.test.wf_testcase import WaterfallTestCase


class GetIsolateShaPipelineTest(WaterfallTestCase):
  app_module = pipeline_handlers._APP

  @mock.patch.object(swarming, 'GetIsolatedShaForStep')
  def testGetIsolateShaForBuildPipeline(self, mocked_get_isolate_sha):
    master_name = 'm'
    builder_name = 'b'
    build_number = 100
    step_name = 's'
    url = 'url'
    sha = 'sha1'
    mocked_get_isolate_sha.return_value = sha

    get_build_sha_parameters = GetIsolateShaForBuildParameters(
        master_name=master_name,
        builder_name=builder_name,
        build_number=build_number,
        url=url,
        step_name=step_name)

    pipeline_job = GetIsolateShaForBuildPipeline(get_build_sha_parameters)
    pipeline_job.start()
    self.execute_queued_tasks()

    self.assertTrue(mocked_get_isolate_sha.called)

  def testGetIsolateShaForTryJobPipeline(self):
    step_name = 'interactive_ui_tests'
    expected_sha = 'ddd3e494ed97366f99453e5a8321b77449f26a58'
    url = ('https://ci.chromium.org/p/chromium/builders/luci.chromium.findit/'
           'findit_variable/1391')
    expected_output = GetIsolateShaOutput(
        isolate_sha=expected_sha, build_url=None,
        try_job_url=url).ToSerializable()

    try_job_result = FlakeTryJobResult.FromSerializable({
        'report': {
            'previously_checked_out_revision':
                'f5dc74a384d48f1a0929dc056cadae2a0019f8b5',
            'previously_cached_revision':
                '17b7ff2ff8e107b0e9cebcd9d6894072acc98639',
            'result': {
                'f5dc74a384d48f1a0929dc056cadae2a0019f8b5': {
                    'interactive_ui_tests': {
                        'status': 'skipped',
                        'valid': True
                    }
                }
            },
            'isolated_tests': {
                'interactive_ui_tests':
                    'ddd3e494ed97366f99453e5a8321b77449f26a58'
            },
            'last_checked_out_revision':
                None,
            'metadata': {}
        },
        'url': url,
        'try_job_id': '8951342990533358272'
    })

    get_try_job_sha_parameters = GetIsolateShaForTryJobParameters(
        try_job_result=try_job_result, step_name=step_name)

    pipeline_job = GetIsolateShaForTryJobPipeline(get_try_job_sha_parameters)
    pipeline_job.start()
    self.execute_queued_tasks()

    pipeline_job = pipelines.pipeline.Pipeline.from_id(pipeline_job.pipeline_id)
    pipeline_output = pipeline_job.outputs.default.value

    self.assertEqual(expected_output, pipeline_output)

  @mock.patch.object(step_util, 'GetValidBoundingBuildsForStep')
  @mock.patch.object(buildbot, 'CreateBuildUrl')
  def testGetIsolateShaForCommitPositionPipelineBuildLevel(
      self, mocked_url, mocked_build_info):
    master_name = 'm'
    builder_name = 'b'
    build_number = 100
    step_name = 's'
    test_name = 't'
    requested_commit_position = 1000
    requested_revision = 'r1000'
    expected_sha = 'sha1'
    build_url = 'url'
    mocked_url.return_value = build_url

    expected_output = GetIsolateShaOutput(
        isolate_sha=expected_sha, build_url=build_url, try_job_url=None)

    build = BuildInfo(master_name, builder_name, build_number)
    build.commit_position = requested_commit_position
    mocked_build_info.return_value = (None, build)

    analysis = MasterFlakeAnalysis.Create(master_name, builder_name,
                                          build_number, step_name, test_name)
    analysis.Save()

    get_build_sha_parameters = GetIsolateShaForBuildParameters(
        master_name=master_name,
        builder_name=builder_name,
        build_number=build_number,
        url=build_url,
        step_name=step_name)

    get_sha_input = GetIsolateShaForCommitPositionParameters(
        analysis_urlsafe_key=unicode(analysis.key.urlsafe()),
        commit_position=requested_commit_position,
        revision=requested_revision)

    self.MockSynchronousPipeline(GetIsolateShaForBuildPipeline,
                                 get_build_sha_parameters, expected_output)

    pipeline_job = GetIsolateShaForCommitPositionPipeline(get_sha_input)
    pipeline_job.start()
    self.execute_queued_tasks()

    pipeline_job = pipelines.pipeline.Pipeline.from_id(pipeline_job.pipeline_id)
    pipeline_output = pipeline_job.outputs.default.value

    self.assertEqual(expected_output.ToSerializable(), pipeline_output)

  @mock.patch.object(waterfall_config, 'GetTrybotDimensions')
  @mock.patch.object(swarmbot_util, 'GetCacheName')
  @mock.patch.object(step_util, 'GetValidBoundingBuildsForStep')
  @mock.patch.object(build_util, 'GetBuildInfo')
  def testGetIsolateShaForCommitPositionPipelineCommitLevel(
      self, mocked_reference_build, mocked_bounding_builds, mocked_cache,
      mocked_dimensions):
    master_name = 'm'
    builder_name = 'b'
    build_number = 100
    step_name = 's'
    test_name = 't'
    dimensions = ['dimensions']
    requested_commit_position = 1000
    containing_build_commit_position = 1001
    requested_revision = 'r1000'
    expected_sha = 'sha1'
    cache_name = 'cache'
    try_job_id = 'try_job_id'
    url = 'url'
    mocked_cache.return_value = cache_name
    mocked_dimensions.return_value = dimensions
    expected_isolated_tests = IsolatedTests()
    expected_isolated_tests[test_name] = expected_sha

    build = BuildInfo(master_name, builder_name, build_number)
    build.commit_position = containing_build_commit_position
    mocked_reference_build.return_value = (None, build)
    mocked_bounding_builds.return_value = (None, build)

    analysis = MasterFlakeAnalysis.Create(master_name, builder_name,
                                          build_number, step_name, test_name)
    analysis.Save()

    try_job = FlakeTryJob.Create(master_name, builder_name, step_name,
                                 test_name, requested_revision)
    try_job.put()

    run_flake_try_job_parameters = RunFlakeTryJobParameters(
        analysis_urlsafe_key=analysis.key.urlsafe(),
        revision=requested_revision,
        flake_cache_name=cache_name,
        dimensions=ListOfBasestring.FromSerializable(dimensions),
        urlsafe_try_job_key=try_job.key.urlsafe())

    get_sha_input = GetIsolateShaForCommitPositionParameters(
        analysis_urlsafe_key=unicode(analysis.key.urlsafe()),
        commit_position=requested_commit_position,
        revision=requested_revision)

    expected_try_job_report = FlakeTryJobReport(
        result={},
        isolated_tests=expected_isolated_tests,
        last_checked_out_revision=None,
        previously_cached_revision=None,
        previously_checked_out_revision=None,
        metadata=None)

    expected_try_job_result = FlakeTryJobResult(
        report=expected_try_job_report, url=url, try_job_id=try_job_id)

    get_isolate_sha_for_try_job_pipeline = GetIsolateShaForTryJobParameters(
        try_job_result=expected_try_job_result, step_name=step_name)

    self.MockAsynchronousPipeline(RunFlakeTryJobPipeline,
                                  run_flake_try_job_parameters,
                                  expected_try_job_report)

    self.MockSynchronousPipeline(GetIsolateShaForTryJobPipeline,
                                 get_isolate_sha_for_try_job_pipeline,
                                 expected_sha)

    pipeline_job = GetIsolateShaForCommitPositionPipeline(get_sha_input)
    pipeline_job.start()
    self.execute_queued_tasks()
