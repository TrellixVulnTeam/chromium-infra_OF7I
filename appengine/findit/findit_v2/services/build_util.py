# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This module is for logic to process a buildbucket build."""

import logging

from buildbucket_proto import common_pb2

from google.appengine.ext import ndb
from google.protobuf.field_mask_pb2 import FieldMask

from common.waterfall import buildbucket_client

from findit_v2.services import projects
from findit_v2.services.constants import ANALYZED_BUILD_ID_TAG_KEY
from findit_v2.services.context import Context


def GetFailedStepsInBuild(context, build):
  """Gets failed steps and their types for a LUCI build.

  Args:
    context (findit_v2.services.context.Context): Scope of the analysis.
    build (buildbucket build.proto): ALL info about the build.

  Returns:
    A list of tuples, each tuple contains the information of a failed step and
    its type.
  """
  return _GetClassifiedStepsInBuildByStatus(context, build, common_pb2.FAILURE)


def GetPassingStepsInBuild(context, build):
  """Gets passing steps and their types for a LUCI build.

  Args:
    context (findit_v2.services.context.Context): Scope of the analysis.
    build (buildbucket build.proto): ALL info about the build.

  Returns:
    A list of tuples, each tuple contains the information of a passing step and
    its type.
  """
  return _GetClassifiedStepsInBuildByStatus(context, build, common_pb2.SUCCESS)


def _GetClassifiedStepsInBuildByStatus(context, build, wanted_status):
  """Gets steps in the given status and their types for a LUCI build.

  Args:
    context (findit_v2.services.context.Context): Scope of the analysis.
    build (buildbucket build.proto): ALL info about the build.

  Returns:
    A list of tuples, each tuple contains the information of a step and its
    type.
  """
  project_api = projects.GetProjectAPI(context.luci_project_name)

  filtered_steps = []
  for step in build.steps:
    if step.status != wanted_status:
      continue
    step_type = project_api.ClassifyStepType(build, step)
    filtered_steps.append((step, step_type))

  return filtered_steps


def GetAnalyzedBuildIdFromRerunBuild(build):
  """Gets analyzed build id from rerun_build's tag, otherwise None.

  Args:
    rerun_build (buildbucket build.proto): ALL info about the build.

  Returns:
    int: build_id of the analyzed build.
  """
  for tag in build.tags:
    if tag.key == ANALYZED_BUILD_ID_TAG_KEY:
      return int(tag.value)
  return None


def GetBuildAndContextForAnalysis(project, build_id):
  """Gets all information about a build and generates context from it.

  Args:
    project (str): Luci project of the build.
    build_id (int): Id of the build.

  Returns:
    (buildbucket build.proto): ALL info about the build.
    (Context): Context of an analysis.

  """
  build = buildbucket_client.GetV2Build(build_id, fields=FieldMask(paths=['*']))

  if not build:
    logging.error('Failed to get build info for build %d.', build_id)
    return None, None

  if (build.input.gitiles_commit.host !=
      projects.GERRIT_PROJECTS[project]['gitiles-host'] or
      build.input.gitiles_commit.project !=
      projects.GERRIT_PROJECTS[project]['name']):
    logging.warning('Unexpected gitiles project for build: %r', build_id)
    return None, None

  context = Context(
      luci_project_name=project,
      gitiles_host=build.input.gitiles_commit.host,
      gitiles_project=build.input.gitiles_commit.project,
      gitiles_ref=build.input.gitiles_commit.ref,
      gitiles_id=build.input.gitiles_commit.id)
  return build, context


def AllLaterBuildsHaveOverlappingFailure(context, build, culprit):
  """Gets later builds on the same builder with overlapping failed steps.

  Queries buildbucket for later builds on the same builder and checks if all of
  them fail with some overlap with the failures the culprit is responsible for,
  based on failed step names.

  Args:
    build (buildbucket build.proto): ALL info about the original build.
    culprit (findit_v2.model.gitiles_commit.Culprit): The culprit that
        introduces the failures we are checking.

  Returns:
    True if all completed builds on the builder after the original failure are
    also failed _and_ the failed steps of each overlap with the failed steps in
    the original failure. False if any of the builds completed successfully or
    with only warnings, or if any build fails, but succeeds at all the steps
    that the original failure failed at.
  """

  def _StepNamesOnly(step_type_tuples):
    return set(s.name for s, st in step_type_tuples)

  builder_id = build.builder
  latest_builds = GetRecentCompletedBuilds(
      builder_id, at_or_after_build=build.id)
  failures = [ndb.Key(urlsafe=k).get() for k in culprit.failure_urlsafe_keys]
  original_failed_steps = set(f.step_ui_name for f in failures)
  for newer_build in latest_builds:
    if newer_build.number <= build.number:
      break
    if newer_build.status == common_pb2.SUCCESS:
      logging.info('Found later build that succeeded BuildId:%d',
                   newer_build.id)
      return False
    new_failed_steps = _StepNamesOnly(
        GetFailedStepsInBuild(context, newer_build))
    new_passing_steps = _StepNamesOnly(
        GetPassingStepsInBuild(context, newer_build))
    if (not original_failed_steps & new_failed_steps and
        original_failed_steps.issubset(new_passing_steps)):
      logging.info(
          'All steps faild due to cuprit %s succeeded in later build %d',
          culprit.key.id(), newer_build.number)
      return False
  return True


def GetRecentCompletedBuilds(builder_id, page_size=20, at_or_after_build=0):
  """Gets the most recent <page_size> completed builds in the builder.

  If given, filter out builds with build id earlier than <at_or_after_build>.

  Args:
    builder_id (buildbucket_proto.build.BuilderID): project/bucket/builder.
    page_size (int): How many builds to retrieve (ending at the most recent).
    at_or_after_build (int): If greater than zero, exclude all builds with a
        build id earlier than this. N.B. Build id is monotonically decreasing.
  """
  search_builds_response = buildbucket_client.SearchV2BuildsOnBuilder(
      builder_id, status=common_pb2.ENDED_MASK, page_size=page_size)

  if search_builds_response:
    return sorted([
        build for build in search_builds_response.builds
        if (not at_or_after_build or build.id <= at_or_after_build)
    ],
                  key=lambda x: x.id)
  return None
