# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Special logic of pre compile analysis.

Build with compile failures will be pre-processed to determine if a new compile
analysis is needed or not.
"""
import logging

from google.appengine.ext import ndb
from google.protobuf.field_mask_pb2 import FieldMask

from services import gerrit
from services import git
from services import deps
from waterfall import waterfall_config

from common.waterfall import buildbucket_client
from findit_v2.model import luci_build
from findit_v2.model import compile_failure
from findit_v2.model.compile_failure import CompileFailure
from findit_v2.model.compile_failure import CompileFailureAnalysis
from findit_v2.model.compile_failure import CompileFailureGroup
from findit_v2.model.compile_failure import CompileRerunBuild
from findit_v2.model.culprit_action import CulpritAction
from findit_v2.services import build_util
from findit_v2.services import constants
from findit_v2.services import projects
from findit_v2.services.analysis.analysis_api import AnalysisAPI
from findit_v2.services.failure_type import StepTypeEnum


class CompileAnalysisAPI(AnalysisAPI):

  @property
  def step_type(self):
    return StepTypeEnum.COMPILE

  def _GetMergedFailureKey(self, failure_entities, referred_build_id,
                           step_ui_name, atomic_failure):
    return CompileFailure.GetMergedFailureKey(
        failure_entities, referred_build_id, step_ui_name, atomic_failure)

  def _GetFailuresInBuild(self, project_api, build, failed_steps):
    return project_api.GetCompileFailures(build, failed_steps)

  def _GetFailuresWithMatchingFailureGroups(self, project_api, context, build,
                                            first_failures_in_current_build):
    return project_api.GetFailuresWithMatchingCompileFailureGroups(
        context, build, first_failures_in_current_build)

  def _CreateFailure(self, failed_build_key, step_ui_name,
                     first_failed_build_id, last_passed_build_id,
                     merged_failure_key, atomic_failure, properties):
    """Creates a CompileFailure entity."""
    return CompileFailure.Create(
        failed_build_key=failed_build_key,
        step_ui_name=step_ui_name,
        output_targets=list(atomic_failure or []),
        rule=(properties or {}).get('rule'),
        first_failed_build_id=first_failed_build_id,
        last_passed_build_id=last_passed_build_id,
        # Default to first_failed_build_id, will be updated later if matching
        # group exists.
        failure_group_build_id=first_failed_build_id,
        merged_failure_key=merged_failure_key)

  def GetFailureEntitiesForABuild(self, build):
    compile_failure_entities = CompileFailure.query(
        ancestor=ndb.Key(luci_build.LuciFailedBuild, build.id)).fetch()
    assert compile_failure_entities, (
        'No compile failure saved in datastore for build {}'.format(build.id))
    return compile_failure_entities

  def _CreateFailureGroup(self, context, build, compile_failure_keys,
                          last_passed_gitiles_id, last_passed_commit_position,
                          first_failed_commit_position):
    group_entity = CompileFailureGroup.Create(
        luci_project=context.luci_project_name,
        luci_bucket=build.builder.bucket,
        build_id=build.id,
        gitiles_host=context.gitiles_host,
        gitiles_project=context.gitiles_project,
        gitiles_ref=context.gitiles_ref,
        last_passed_gitiles_id=last_passed_gitiles_id,
        last_passed_commit_position=last_passed_commit_position,
        first_failed_gitiles_id=context.gitiles_id,
        first_failed_commit_position=first_failed_commit_position,
        compile_failure_keys=compile_failure_keys)
    return group_entity

  def _CreateFailureAnalysis(
      self, luci_project, context, build, last_passed_gitiles_id,
      last_passed_commit_position, first_failed_commit_position,
      rerun_builder_id, compile_failure_keys):
    analysis = CompileFailureAnalysis.Create(
        luci_project=luci_project,
        luci_bucket=build.builder.bucket,
        luci_builder=build.builder.builder,
        build_id=build.id,
        gitiles_host=context.gitiles_host,
        gitiles_project=context.gitiles_project,
        gitiles_ref=context.gitiles_ref,
        last_passed_gitiles_id=last_passed_gitiles_id,
        last_passed_commit_position=last_passed_commit_position,
        first_failed_gitiles_id=context.gitiles_id,
        first_failed_commit_position=first_failed_commit_position,
        rerun_builder_id=rerun_builder_id,
        compile_failure_keys=compile_failure_keys)
    return analysis

  def _GetFailuresInAnalysis(self, analysis):
    return ndb.get_multi(analysis.compile_failure_keys)

  def _FetchRerunBuildsOfAnalysis(self, analysis):
    return CompileRerunBuild.query(ancestor=analysis.key).order(
        CompileRerunBuild.gitiles_commit.commit_position).fetch()

  def _GetFailureAnalysis(self, analyzed_build_id):
    analysis = CompileFailureAnalysis.GetVersion(analyzed_build_id)
    assert analysis, 'Failed to get CompileFailureAnalysis for build {}'.format(
        analyzed_build_id)
    return analysis

  def _GetFailuresToRerun(self, failure_entities):
    return compile_failure.GetFailedTargets(failure_entities)

  def _GetExistingRerunBuild(self, analysis_key, rerun_commit):
    return CompileRerunBuild.SearchBuildOnCommit(analysis_key, rerun_commit)

  def _CreateRerunBuild(self, rerun_builder, new_build, rerun_commit,
                        analysis_key):
    return CompileRerunBuild.Create(
        luci_project=rerun_builder.project,
        luci_bucket=rerun_builder.bucket,
        luci_builder=rerun_builder.builder,
        build_id=new_build.id,
        legacy_build_number=new_build.number,
        gitiles_host=rerun_commit.gitiles_host,
        gitiles_project=rerun_commit.gitiles_project,
        gitiles_ref=rerun_commit.gitiles_ref,
        gitiles_id=rerun_commit.gitiles_id,
        commit_position=rerun_commit.commit_position,
        status=new_build.status,
        create_time=new_build.create_time.ToDatetime(),
        parent_key=analysis_key)

  def _GetRerunBuildTags(self, analyzed_build_id):
    return [
        {
            'key': constants.RERUN_BUILD_PURPOSE_TAG_KEY,
            'value': constants.COMPILE_RERUN_BUILD_PURPOSE,
        },
        {
            'key': constants.ANALYZED_BUILD_ID_TAG_KEY,
            'value': str(analyzed_build_id),
        },
    ]

  def _GetRerunBuildInputProperties(self, project_api, rerun_failures,
                                    analyzed_build_id):
    return project_api.GetCompileRerunBuildInputProperties(
        rerun_failures, analyzed_build_id)

  def GetSuspectedCulprits(self, project_api, context, build,
                           first_failures_in_current_build):

    failure_info = project_api.GetCompileFailureInfo(
        context, build, first_failures_in_current_build)
    # Projects that support heuristic analysis for compile must implement
    # GetCompileFailureInfo.
    if failure_info:
      signals = project_api.ExtractSignalsForCompileFailure(failure_info)

      change_logs = git.PullChangeLogs(
          first_failures_in_current_build['last_passed_build']['commit_id'],
          context.gitiles_id)

      deps_info = deps.ExtractDepsInfo(failure_info, change_logs)

      return project_api.HeuristicAnalysisForCompile(failure_info, change_logs,
                                                     deps_info, signals)
    return None

  def _GetFailureGroupByContext(self, context):
    groups = CompileFailureGroup.query(
        CompileFailureGroup.luci_project == context.luci_project_name).filter(
            CompileFailureGroup.first_failed_commit.gitiles_id == context
            .gitiles_id).fetch()
    return groups[0] if groups else None

  def OnCulpritFound(self, context, analyzed_build_id, culprit):
    """Decides and executes the action for the found culprit change.

    This possible actions include:

     - No action.
     - Notify the culprit CL.
     - Create revert and request that it's reviewed.
     - Create a revert and submit it.

    Selecting the appropriate action will be based on the project's configured
    options and daily limits as well as whether the action can be taken safely.

    Refer to the code below for details.

    Args:
      context (findit_v2.services.context.Context): Scope of the analysis.
      analyzed_build_id: Buildbucket id of the continuous build being analyzed.
      culprit: The Culprit entity for the change identified as causing the
          failures.

    Returns:
      The CulpritAction entity describing the action taken, None if no action
      was performed.
    """

    project_api = projects.GetProjectAPI(context.luci_project_name)
    project_config = projects.PROJECT_CFG.get(context.luci_project_name, {})
    action_settings = waterfall_config.GetActionSettings()

    if not project_config.get('auto_actions_enabled_for_project', False):
      return self._NoAction(culprit, 'Auto-actions disabled for project')

    if not build_util.AllLaterBuildsHaveOverlappingFailure(
        context, analyzed_build_id, culprit):
      return self._NoAction(culprit, 'Build has recovered')

    change_info, gerrit_client = project_api.ChangeInfoAndClientFromCommit(
        culprit)
    cl_details = gerrit_client.GetClDetails(change_info['review_change_id'])
    if bool(cl_details.revert_of):
      return self._Notify(project_api, culprit, 'The culprit is a revert')

    reverted, by_findit = self._CheckIfReverted(
        cl_details, culprit,
        project_config.get('auto_actions_service_account', ''))
    if reverted and by_findit:
      return self._NoAction(culprit,
                            'We already created a revert for this culprit')

    if reverted:
      return self._Notify(
          project_api,
          culprit,
          'A revert was manually created for this culprit',
          silent=True)

    if CulpritAction.GetRecentActionsByType(
        CulpritAction.REVERT, revert_committed=False) >= action_settings.get(
            'auto_create_revert_daily_threshold_compile', 10):
      return self._Notify(project_api, culprit, 'Reached revert creation quota')

    if not project_config.get('auto_revert_enabled_for_project', False):
      return self._Notify(project_api, culprit,
                          'Auto-revert disabled for this project')

    if cl_details.auto_revert_off:
      return self._Notify(project_api, culprit,
                          'The culprit has been tagged with NOAUTOREVERT=True')

    if gerrit.ExistCQedDependingChanges(change_info):
      return self._Notify(project_api, culprit,
                          'Changes already in the CQ depend on culprit')

    if not git.ChangeCommittedWithinTime(
        culprit.gitiles_id,
        repo_url=git.GetRepoUrlFromContext(context),
        hours=project_config.get('max_revertible_culprit_age_hours', 24)):
      return self._Notify(project_api, culprit,
                          'Culprit is too old to auto-revert')

    if cl_details.owner_email in project_config.get(
        'automated_account_whitelist', []):
      return self._Notify(project_api, culprit,
                          'Culprit was created by a whitelisted account')

    revert = project_api.CreateRevert(
        culprit, self._ComposeRevertDescription(project_api, culprit))
    if project_config.get('auto_commit_enabled_for_project', False):
      if CulpritAction.GetRecentActionsByType(
          CulpritAction.REVERT, revert_committed=True) < action_settings.get(
              'auto_commit_revert_daily_threshold_compile', 4):
        logging.info('Submitting revert %s for %s', revert['id'],
                     culprit.key.id())
        action = self._CommitRevert(project_api, revert, culprit)
        if action:
          return action
        logging.info(
            'Could not land revert automatically, requesting manual review')
      else:
        logging.info('Reached auto-commit quota, requesting manual review')
    else:
      logging.info('Auto-committing disabled, requesting manual review')

    return self._RequestReview(project_api, revert, culprit)
