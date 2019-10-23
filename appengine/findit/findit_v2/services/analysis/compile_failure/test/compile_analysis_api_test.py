# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from datetime import datetime
import mock
import textwrap

from buildbucket_proto import common_pb2
from buildbucket_proto.build_pb2 import Build
from buildbucket_proto.build_pb2 import BuilderID

from findit_v2.model.compile_failure import CompileFailureGroup
from findit_v2.model.culprit_action import CulpritAction
from findit_v2.model.gitiles_commit import Culprit
from findit_v2.model.gitiles_commit import GitilesCommit
from findit_v2.model.luci_build import LuciFailedBuild
from findit_v2.services import build_util
from findit_v2.services import projects
from findit_v2.services.analysis.compile_failure.compile_analysis_api import (
    CompileAnalysisAPI)
from findit_v2.services.chromium_api import ChromiumProjectAPI
from findit_v2.services.context import Context
from findit_v2.services.failure_type import StepTypeEnum
from services import gerrit
from services import git
from waterfall.test import wf_testcase


class CompileAnalysisAPITest(wf_testcase.WaterfallTestCase):

  def _MockBuild(self,
                 build_id,
                 build_number,
                 gitiles_commit_id,
                 builder_name='Linux Tests',
                 build_status=common_pb2.FAILURE):
    builder = BuilderID(project='chromium', bucket='ci', builder=builder_name)
    build = Build(
        id=build_id, builder=builder, number=build_number, status=build_status)
    build.input.gitiles_commit.host = 'gitiles.host.com'
    build.input.gitiles_commit.project = 'project/name'
    build.input.gitiles_commit.ref = 'ref/heads/master'
    build.input.gitiles_commit.id = gitiles_commit_id
    build.create_time.FromDatetime(datetime(2019, 4, 9))
    build.start_time.FromDatetime(datetime(2019, 4, 9, 0, 1))
    build.end_time.FromDatetime(datetime(2019, 4, 9, 1))
    return build

  def setUp(self):
    super(CompileAnalysisAPITest, self).setUp()
    self.UpdateUnitTestConfigSettings(
        config_property='action_settings', override_data={'v2_actions': True})
    self.build_id = 8000000000123
    self.build_number = 123
    self.builder = BuilderID(
        project='chromium', bucket='ci', builder='Linux Tests')
    self.build = self._MockBuild(self.build_id, self.build_number,
                                 'git_sha_123')
    self.rerun_builder = BuilderID(
        project='chromium', bucket='try', builder='findit_variables')

    self.context = Context(
        luci_project_name='chromium',
        gitiles_host='gitiles.host.com',
        gitiles_project='project/name',
        gitiles_ref='ref/heads/master',
        gitiles_id='git_sha_123')

    self.build_entity = LuciFailedBuild.Create(
        luci_project=self.build.builder.project,
        luci_bucket=self.build.builder.bucket,
        luci_builder=self.build.builder.builder,
        build_id=self.build.id,
        legacy_build_number=self.build.number,
        gitiles_host=self.context.gitiles_host,
        gitiles_project=self.context.gitiles_project,
        gitiles_ref=self.context.gitiles_ref,
        gitiles_id=self.context.gitiles_id,
        commit_position=123,
        status=self.build.status,
        create_time=self.build.create_time.ToDatetime(),
        start_time=self.build.start_time.ToDatetime(),
        end_time=self.build.end_time.ToDatetime(),
        build_failure_type=StepTypeEnum.COMPILE)
    self.build_entity.put()

    self.analysis_api = CompileAnalysisAPI()

    self.compile_failure = self.analysis_api._CreateFailure(
        self.build_entity.key, 'compile', self.build_id, 8000000000122, None,
        frozenset(['a.o']), None)
    self.compile_failure.put()

  @mock.patch.object(git, 'GetCommitPositionFromRevision', return_value=67890)
  def testEntitiesCreation(self, _):
    group = self.analysis_api._CreateFailureGroup(
        self.context, self.build, [self.compile_failure.key], '122', 122, 123)
    group.put()
    groups = CompileFailureGroup.query().fetch()
    self.assertEqual(1, len(groups))
    self.assertEqual(self.build_id, groups[0].key.id())

    analysis = self.analysis_api._CreateFailureAnalysis(
        'chromium', self.context, self.build, 'git_sha_122', 122, 123,
        'preject/bucket/builder', [self.compile_failure.key])
    analysis.Save()
    analysis = self.analysis_api._GetFailureAnalysis(self.build_id)
    self.assertIsNotNone(analysis)
    self.assertEqual(self.build_id, analysis.build_id)
    self.assertEqual([self.compile_failure],
                     self.analysis_api._GetFailuresInAnalysis(analysis))

    rerun_commit = GitilesCommit(
        gitiles_host=self.context.gitiles_host,
        gitiles_project=self.context.gitiles_project,
        gitiles_ref=self.context.gitiles_ref,
        gitiles_id=self.context.gitiles_id,
        commit_position=123)
    rerun_build_id = 8000000000050
    self.analysis_api._CreateRerunBuild(self.rerun_builder,
                                        Build(id=rerun_build_id), rerun_commit,
                                        analysis.key).put()
    all_rerun_builds = self.analysis_api._FetchRerunBuildsOfAnalysis(analysis)
    self.assertEqual(1, len(all_rerun_builds))
    self.assertEqual(rerun_build_id, all_rerun_builds[0].build_id)

    existing_rerun_builds = self.analysis_api._GetExistingRerunBuild(
        analysis.key, rerun_commit)
    self.assertEqual(1, len(existing_rerun_builds))
    self.assertEqual(rerun_build_id, existing_rerun_builds[0].build_id)

  def testAPIStepType(self):
    self.assertEqual(StepTypeEnum.COMPILE, self.analysis_api.step_type)

  def testGetFailureEntitiesForABuild(self):
    failure_entities = self.analysis_api.GetFailureEntitiesForABuild(self.build)
    self.assertEqual(1, len(failure_entities))
    self.assertEqual(self.compile_failure, failure_entities[0])

  @mock.patch.object(git, 'GetCommitPositionFromRevision', return_value=67890)
  def testGetMergedFailureKey(self, _):
    with self.assertRaises(AssertionError):
      self.analysis_api._GetMergedFailureKey([self.compile_failure], None,
                                             'compile', None)

  @mock.patch.object(ChromiumProjectAPI, 'GetCompileFailures')
  def test_GetFailuresInBuild(self, mock_compile_failure):
    self.analysis_api._GetFailuresInBuild(ChromiumProjectAPI(), self.build,
                                          ['compile'])
    self.assertTrue(mock_compile_failure.called)

  @mock.patch.object(ChromiumProjectAPI,
                     'GetFailuresWithMatchingCompileFailureGroups')
  def test_GetFailuresWithMatchingFailureGroups(self, mock_failures_in_group):
    self.analysis_api._GetFailuresWithMatchingFailureGroups(
        ChromiumProjectAPI(), self.context, self.build, {})
    self.assertTrue(mock_failures_in_group.called)

  def testGetAtomicFailures(self):
    self.assertEqual({
        'compile': ['a.o']
    }, self.analysis_api._GetFailuresToRerun([self.compile_failure]))

  def testGetRerunBuildTags(self):
    expected_tags = [{
        'key': 'purpose',
        'value': 'compile-failure-culprit-finding'
    }, {
        'key': 'analyzed_build_id',
        'value': str(self.build_id)
    }]
    self.assertEqual(expected_tags,
                     self.analysis_api._GetRerunBuildTags(self.build_id))

  @mock.patch.object(ChromiumProjectAPI, 'GetCompileRerunBuildInputProperties')
  def testGetRerunBuildInputProperties(self, mock_input_properties):
    self.analysis_api._GetRerunBuildInputProperties(
        ChromiumProjectAPI(), {'compile': ['a.o']}, 8000000000122)
    self.assertTrue(mock_input_properties.called)

  def testGetFailureGroupOfBuild(self):
    group = self.analysis_api._CreateFailureGroup(
        self.context, self.build, [self.compile_failure.key], '122', 122, 123)
    group.put()
    self.assertEqual(
        group,
        CompileAnalysisAPI()._GetFailureGroupByContext(self.context))

  @mock.patch.object(build_util, 'AllLaterBuildsHaveOverlappingFailure')
  @mock.patch.object(CulpritAction, 'GetRecentActionsByType')
  @mock.patch.object(gerrit, 'ExistCQedDependingChanges')
  @mock.patch.object(git, 'ChangeCommittedWithinTime')
  @mock.patch.object(ChromiumProjectAPI, 'ChangeInfoAndClientFromCommit')
  @mock.patch.object(ChromiumProjectAPI, 'CreateRevert')
  @mock.patch.object(CompileAnalysisAPI, '_CheckIfReverted')
  @mock.patch.object(CompileAnalysisAPI, '_NoAction')
  @mock.patch.object(CompileAnalysisAPI, '_RequestReview')
  @mock.patch.object(CompileAnalysisAPI, '_CommitRevert')
  @mock.patch.object(CompileAnalysisAPI, '_Notify')
  def testOnCulpritFound(self, notify, commit, review, no_action, check_revert,
                         create_revert, change_info_and_client, changed_in_time,
                         cqed_changes, actions_by_type, ongoing_failure):

    class MockGerritClient(object):
      revert_of = False
      auto_revert_off = False

      def GetClDetails(self, *_args, **_kwargs):

        class MockClDetails(object):
          revert_of = MockGerritClient.revert_of
          auto_revert_off = MockGerritClient.auto_revert_off
          owner_email = 'dummy@account.org'

        return MockClDetails()

    change_info_and_client.return_value = ({
        'review_change_id': 1
    }, MockGerritClient())

    # In the following list of pairs, the first element is the list of values
    # for the mocks to return, and the second element is a dict indicating which
    # actions are expected to be taken by OnCulpritFound.
    scenarios = [
        # Create a revert and submit, with the default values.
        ([], {
            'create_revert': True,
            'commit': True
        }),
        # Auto-action disabled.
        ([False], {
            'no_action': True
        }),
        # Build recovered.
        ([True, False], {
            'no_action': True
        }),
        # The culprit is a revert.
        ([True, True, True], {
            'notify': True
        }),
        # The culprit is already reverted by findit.
        ([True, True, False, True, True], {
            'no_action': True
        }),
        # The culprit is already reverted by sheriff.
        ([True, True, False, True, False], {
            'notify': True
        }),
        # Reached the revert quota.
        ([True, True, False, False, False, 100], {
            'notify': True
        }),
        # Auto-revert disabled.
        ([True, True, False, False, False, 0, False], {
            'notify': True
        }),
        # Culprit tagged with NOAUTOREVERT=True
        ([True, True, False, False, False, 0, True, True], {
            'notify': True
        }),
        # CQed changes depend on the culprit.
        ([True, True, False, False, False, 0, True, False, True], {
            'notify': True
        }),
        # Culprit landed over 24 hours ago.
        ([True, True, False, False, False, 0, True, False, False, False], {
            'notify': True
        }),
        # Culprit author whitelisted.
        ([
            True, True, False, False, False, 0, True, False, False, True,
            ['dummy@account.org']
        ], {
            'notify': True
        }),
        # Auto-commit disabled.
        ([
            True, True, False, False, False, 0, True, False, False, True, [],
            False
        ], {
            'create_revert': True,
            'review': True
        }),
        # Auto-commit quota reached.
        ([
            True, True, False, False, False, 0, True, False, False, True, [],
            True, 100
        ], {
            'create_revert': True,
            'review': True
        }),
    ]
    for scenario_list, result in scenarios:
      # Make the list of flags into a dict to allow getting default values.
      scenario = dict(enumerate(scenario_list))

      # Reset action mocks to correctly check for calls later.
      for m in notify, create_revert, commit, review, no_action:
        m.reset_mock()

      # Set values of decision points according to list of flags.
      projects.PROJECT_CFG['chromium'][
          'auto_actions_enabled_for_project'] = scenario.get(0, True)
      ongoing_failure.return_value = scenario.get(1, True)
      MockGerritClient.revert_of = scenario.get(2, False)
      check_revert.return_value = scenario.get(3, False), scenario.get(4, False)
      actions_by_type.side_effect = [scenario.get(5, 0), scenario.get(12, 0)]
      projects.PROJECT_CFG['chromium'][
          'auto_revert_enabled_for_project'] = scenario.get(6, True)
      MockGerritClient.auto_revert_off = scenario.get(7, False)
      cqed_changes.return_value = scenario.get(8, False)
      changed_in_time.return_value = scenario.get(9, True)
      projects.PROJECT_CFG['chromium'][
          'automated_account_whitelist'] = scenario.get(10, [])
      projects.PROJECT_CFG['chromium'][
          'auto_commit_enabled_for_project'] = scenario.get(11, True)

      # Code under test.
      self.analysis_api.OnCulpritFound(
          self.context, self.build_id,
          Culprit.GetOrCreate(self.context.gitiles_host,
                              self.context.gitiles_project,
                              self.context.gitiles_ref, 'badc0de', 666,
                              [self.compile_failure.key.urlsafe()]))

      # Verify expected actions were called.
      self.assertEqual(result.get('no_action', False), bool(no_action.called))
      self.assertEqual(result.get('notify', False), bool(notify.called))
      self.assertEqual(
          result.get('create_revert', False), bool(create_revert.called))
      self.assertEqual(result.get('review', False), bool(review.called))
      self.assertEqual(result.get('commit', False), bool(commit.called))

  @mock.patch.object(ChromiumProjectAPI, 'RequestReview')
  @mock.patch.object(ChromiumProjectAPI, 'CommitRevert')
  @mock.patch.object(ChromiumProjectAPI, 'NotifyCulprit')
  def testActions(self, *_):
    # pylint:disable=line-too-long
    culprit = Culprit.GetOrCreate(self.context.gitiles_host,
                                  self.context.gitiles_project,
                                  self.context.gitiles_ref, 'badc0de', 666,
                                  [self.compile_failure.key.urlsafe()])
    self.assertIsNone(self.analysis_api._NoAction(culprit, 'no_action message'))
    self.assertEquals(
        textwrap.dedent("""\
      Findit (https://goo.gl/kROfz5) identified this CL at revision badc0de as
      the culprit for failures in the continuous build including:

      Sample Failed Build: https://ci.chromium.org/b/8000000000123
      Sample Failed Step: compile

      If it is a false positive, please report it at https://bugs.chromium.org/p/chromium/issues/entry?status=Available&comment=Detail+is+gitiles.host.com%2Fproject%2Fname%2Fref%2Fheads%2Fmaster%2Fbadc0de&labels=Test-Findit-Wrong&components=Tools%3ETest%3EFindIt&summary=Wrongly+blame+badc0de"""
                       ),
        self.analysis_api._ComposeRevertDescription(ChromiumProjectAPI(),
                                                    culprit))
    action = self.analysis_api._Notify(ChromiumProjectAPI(), culprit,
                                       'notify message')
    self.assertEqual(action.action_type, CulpritAction.CULPRIT_NOTIFIED)

    action = self.analysis_api._CommitRevert(ChromiumProjectAPI(),
                                             {'_number': 1}, culprit)
    self.assertEqual(action.action_type, CulpritAction.REVERT)
    self.assertEqual(action.revert_committed, True)

    action = self.analysis_api._RequestReview(ChromiumProjectAPI(),
                                              {'_number': 1}, culprit)
    self.assertEqual(action.action_type, CulpritAction.REVERT)
    self.assertEqual(action.revert_committed, False)
