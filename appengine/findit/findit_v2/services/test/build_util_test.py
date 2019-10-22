# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock

from buildbucket_proto import common_pb2
from buildbucket_proto.build_pb2 import Build
from buildbucket_proto.build_pb2 import BuilderID
from buildbucket_proto.rpc_pb2 import SearchBuildsResponse
from buildbucket_proto.step_pb2 import Step

from findit_v2.model import luci_build
from findit_v2.model.compile_failure import CompileFailure
from findit_v2.model.gitiles_commit import Culprit
from findit_v2.services import build_util
from findit_v2.services.context import Context
from findit_v2.services.failure_type import StepTypeEnum
from services import git
from waterfall.test.wf_testcase import WaterfallTestCase


class BuildUtilTest(WaterfallTestCase):

  def testGetFailedStepsInBuild(self):
    build_id = 8000000000123
    build_number = 123
    builder = BuilderID(project='chromium', bucket='try', builder='linux-rel')
    build = Build(
        id=build_id,
        builder=builder,
        number=build_number,
        status=common_pb2.FAILURE)
    step1 = Step(name='s1', status=common_pb2.SUCCESS)
    step2 = Step(name='compile', status=common_pb2.FAILURE)
    build.steps.extend([step1, step2])

    context = Context(
        luci_project_name='chromium',
        gitiles_host='gitiles.host.com',
        gitiles_project='project/name',
        gitiles_ref='ref/heads/master',
        gitiles_id='git_sha')

    failed_steps = build_util.GetFailedStepsInBuild(context, build)
    self.assertEqual(1, len(failed_steps))
    self.assertEqual('compile', failed_steps[0][0].name)
    self.assertEqual(StepTypeEnum.COMPILE, failed_steps[0][1])

  def testGetAnalyzedBuildIdFromRerunBuild(self):
    analyzed_build_id = 8000000000123
    build = Build(tags=[{
        'key': 'analyzed_build_id',
        'value': str(analyzed_build_id)
    }])
    self.assertEqual(analyzed_build_id,
                     build_util.GetAnalyzedBuildIdFromRerunBuild(build))

  def testGetAnalyzedBuildIdFromRerunBuildNoAnalyzedBuildId(self):
    self.assertIsNone(build_util.GetAnalyzedBuildIdFromRerunBuild(Build()))

  @mock.patch(
      'common.waterfall.buildbucket_client.GetV2Build', return_value=None)
  def testGetBuildAndContextForAnalysisNoBuild(self, _):
    self.assertEqual((None, None),
                     build_util.GetBuildAndContextForAnalysis('chromium', 123))

  @mock.patch.object(git, 'GetCommitPositionFromRevision', return_value=12345)
  @mock.patch('common.waterfall.buildbucket_client.GetV2Build')
  @mock.patch.object(build_util, 'GetRecentCompletedBuilds')
  def testAllLaterBuildsHaveOverlappingFailure(self, mock_builds, mock_get, _):
    context = Context(
        luci_project_name='chromium',
        gitiles_host='gitiles.host.com',
        gitiles_project='project/name',
        gitiles_ref='ref/heads/master',
        gitiles_id='git_sha')

    build_id = 8000000000123
    build_number = 123
    builder = BuilderID(project='chromium', bucket='try', builder='linux-rel')
    original_build = Build(
        id=build_id,
        builder=builder,
        number=build_number,
        status=common_pb2.FAILURE)
    step1 = Step(name='s1', status=common_pb2.SUCCESS)
    step2 = Step(name='compile', status=common_pb2.FAILURE)
    original_build.steps.extend([step1, step2])
    mock_get.return_value = original_build

    build_entity = luci_build.SaveFailedBuild(context, original_build,
                                              'COMPILE')
    failure = CompileFailure.Create(build_entity.key, 'compile', ['target1'])
    failure.put()
    culprit = Culprit.GetOrCreate(context.gitiles_host, context.gitiles_host,
                                  context.gitiles_host, 'badc0de', 12345,
                                  [failure.key.urlsafe()])

    # Continued failure.
    later_build = Build(
        id=build_id - 1,
        builder=builder,
        number=build_number + 1,
        status=common_pb2.FAILURE)
    step1 = Step(name='s1', status=common_pb2.SUCCESS)
    step2 = Step(name='compile', status=common_pb2.FAILURE)
    later_build.steps.extend([step1, step2])

    mock_builds.return_value = [later_build, original_build]
    self.assertTrue(
        build_util.AllLaterBuildsHaveOverlappingFailure(context, build_id,
                                                        culprit))

    # Build succeeds later.
    later_build = Build(
        id=build_id - 1,
        builder=builder,
        number=build_number + 1,
        status=common_pb2.SUCCESS)
    step1 = Step(name='s1', status=common_pb2.SUCCESS)
    step2 = Step(name='compile', status=common_pb2.SUCCESS)
    later_build.steps.extend([step1, step2])

    mock_builds.return_value = [later_build, original_build]
    self.assertFalse(
        build_util.AllLaterBuildsHaveOverlappingFailure(context, original_build,
                                                        culprit))

    # Later build fails differently.
    later_build = Build(
        id=build_id - 1,
        builder=builder,
        number=build_number + 1,
        status=common_pb2.FAILURE)
    step1 = Step(name='s1', status=common_pb2.FAILURE)
    step2 = Step(name='compile', status=common_pb2.SUCCESS)
    later_build.steps.extend([step1, step2])

    mock_builds.return_value = [later_build, original_build]
    self.assertFalse(
        build_util.AllLaterBuildsHaveOverlappingFailure(context, original_build,
                                                        culprit))

  @mock.patch('common.waterfall.buildbucket_client.SearchV2BuildsOnBuilder')
  def testGetRecentCompletedBuilds(self, mock_builds):
    builder = BuilderID(project='chromium', bucket='try', builder='linux-rel')
    mock_builds.return_value = SearchBuildsResponse(builds=[
        Build(id=3, builder=builder, number=1, status=common_pb2.FAILURE),
        Build(id=2, builder=builder, number=2, status=common_pb2.FAILURE),
        Build(id=1, builder=builder, number=3, status=common_pb2.FAILURE)
    ])

    recents = build_util.GetRecentCompletedBuilds(builder)
    self.assertEqual([3, 2, 1], [x.number for x in recents])

    recents = build_util.GetRecentCompletedBuilds(builder, at_or_after_build=2)
    self.assertEqual([3, 2], [x.number for x in recents])
