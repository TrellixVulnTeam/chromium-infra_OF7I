# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Defines the APIs that each supported project must implement."""

import json
import textwrap

from google.appengine.api import taskqueue
from google.appengine.ext import ndb

from findit_v2.services.gerrit_actions import GerritActions
from infra_api_clients.codereview import gerrit
from services import git


class ProjectAPI(object):  # pragma: no cover.

  # Default auto action messaging templates, override these constants for
  # custom/project-specific messages.
  REQUEST_REVIEW = textwrap.dedent("""\
      Sheriffs, CL owner or CL reviewers:
      Please submit this revert if it is correct.

      If it is a false positive, please abandon and report it
      at {bug_link}

      For more information about Findit auto-revert: https://goo.gl/adB34D.""")

  REQUEST_CONFIRMATION = textwrap.dedent("""\
      Sheriffs, CL owner or CL reviewers:
      Please confirm if this revert is correct.

      If it is a false positive, please reland the original CL and report this
      at {bug_link}

      For more information about Findit auto-revert: https://goo.gl/adB34D.""")

  ACTION_REASON = textwrap.dedent("""\
      Findit (https://goo.gl/kROfz5) {verb} this CL at revision {revision} as
      the culprit for failures in the continuous build including:

      Sample Failed Build: {build}
      Sample Failed Step: {step}

      If it is a false positive, please report it at {bug_link}""")


  def __init__(self, project_name=None):
    self.project_name = project_name
    self.gerrit_actions = GerritActions(self)

  def ClassifyStepType(self, build, step):
    """ Returns the failure type of the given build step.

    Args:
      build (buildbucket build.proto): ALL info about the build.
      step (buildbucket step.proto): ALL info about the build step.

    Returns:
      findit_v2.services.failure_type.StepTypeEnum
    """
    # pylint: disable=unused-argument
    raise NotImplementedError

  def GetCompileFailures(self, build, compile_steps):
    """Returns the detailed compile failures from a failed build.

    Args:
      build (buildbucket build.proto): ALL info about the build.
      compile_steps (buildbucket step.proto): The failed compile steps.

    Returns:
      (dict): Information about detailed compile_failures.
      {
        'step_name': {
          'failures': {
            frozenset(['target1', 'target2']): {
              'first_failed_build': {
                'id': 8765432109,
                'number': 123,
                'commit_id': 654321
              },
              'last_passed_build': None,
              'properties': {
                # Arbitrary information about the failure if exists.
              }
            },
          'first_failed_build': {
            'id': 8765432109,
            'number': 123,
            'commit_id': 654321
          },
          'last_passed_build': None,
          'properties': {
            # Arbitrary information about the failure if exists.
          }
        },
      }
    """
    # pylint: disable=unused-argument
    raise NotImplementedError

  def GetTestFailures(self, build, test_steps):
    """Returns the detailed test failures from a failed build.

    Args:
      build (buildbucket build.proto): ALL info about the build.
      test_steps (list of buildbucket step.proto): The failed test steps.

    Returns:
      (dict): Information about detailed test failures.
      {
        'step_name1': {
          'failures': {
            frozenset(['test_name']): {
              'first_failed_build': {
                'id': 8765432109,
                'number': 123,
                'commit_id': 654321
              },
              'last_passed_build': None,
              'properties': {
                # Arbitrary information about the failure if exists.
              }
            },
            ...
          },
          'first_failed_build': {
            'id': 8765432109,
            'number': 123,
            'commit_id': 654321
          },
          'last_passed_build': None
          'properties': None,
        },
        'step_name2': {
          # No test level information.
          'failures': {},
          'first_failed_build': {
            'id': 8765432109,
            'number': 123,
            'commit_id': 654321
          },
          'last_passed_build': None
          'properties': {
            # Arbitrary information about the failure if exists.
          },
        },
      }
    """
    # pylint: disable=unused-argument
    raise NotImplementedError

  def GetRerunDimensions(self, analyzed_build_id):
    """Dimensions for rerun builds.

    Some projects may use a single re-run builder supporting multiple
    configurations and dimensions, they may override this function to provide
    custom dimensions to the rerun job request.

    The expected return value is a list of dictionaries where each dictionary
    has two key/value pairs describing one dimension. E.g.
    [{
        'key': 'os',
        'value': 'Mac'
    }, {
        'key': ...,
        'value': ...
    }, ...]
    """
    # pylint: disable=unused-argument
    return None

  def GetRerunBuilderId(self, build):
    """Gets builder id to run the rerun builds.

    Args:
      build (buildbucket build.proto): ALL info about the build.

    Returns:
      (str): Builder id in the format luci_project/luci_bucket/luci_builder
    """
    # pylint: disable=unused-argument
    raise NotImplementedError

  def GetCompileRerunBuildInputProperties(self, failed_targets,
                                          analyzed_build_id):
    """Gets input properties of a rerun build for compile failures.

    Args:
      failed_targets (list of str): Targets Findit wants to rerun in the build.
      analyzed_build_id (int): Buildbucket Id of the analyzed build, may be used
          to derive properties for reruns.

    Returns:
      (dict): input properties of the rerun build."""
    # pylint: disable=unused-argument
    return NotImplementedError

  def GetFailuresWithMatchingCompileFailureGroups(
      self, context, build, first_failures_in_current_build):
    """Gets reusable failure groups for given compile failure(s).

    Args:
      context (findit_v2.services.context.Context): Scope of the analysis.
      build (buildbucket build.proto): ALL info about the build.
      first_failures_in_current_build (dict): A dict for failures that happened
      the first time in current build.
      {
        'failures': {
          'step': {
            'atomic_failures': [
              frozenset(['target4']),
              frozenset(['target1', 'target2'])],
            'last_passed_build': {
              'id': 8765432109,
              'number': 122,
              'commit_id': 'git_sha1'
            },
          },
        },
        'last_passed_build': {
          # In this build all the failures that happened in the build being
          # analyzed passed.
          'id': 8765432108,
          'number': 121,
          'commit_id': 'git_sha0'
        }
      }
    """
    # For projects that don't need to group failures (e.g. chromium), this is
    # a no-op.
    # pylint: disable=unused-argument
    return {}

  def GetFailuresWithMatchingTestFailureGroups(self, context, build,
                                               first_failures_in_current_build):
    """Gets reusable failure groups for given test failure(s).

    This method is a placeholder for projects that might need this feature,
    though it's actually a no-op for currently supported projects:
    + For chromium, failure grouping is not required at the moment;
    + For chromeos, All tests are run on the same builder, this grouping is not
      needed at all.

    Args:
      context (findit_v2.services.context.Context): Scope of the analysis.
      build (buildbucket build.proto): ALL info about the build.
      first_failures_in_current_build (dict): A dict for failures that happened
        the first time in current build.
        {
          'failures': {
            'step': {
              'atomic_failures': ['test1', 'test2', ...],
              'last_passed_build': {
                'id': 8765432109,
                'number': 122,
                'commit_id': 'git_sha1'
              },
            },
          },
          'last_passed_build': {
            # In this build all the failures that happened in the build being
            # analyzed passed.
            'id': 8765432108,
            'number': 121,
            'commit_id': 'git_sha0'
          }
        }
      }
    """
    # pylint: disable=unused-argument
    return {}

  def GetTestRerunBuildInputProperties(self, tests, analyzed_build_id):
    """Gets input properties of a rerun build for test failures.

    Args:
      tests (dict): Tests Findit wants to rerun in the build.
      {
        'step': {
          'tests': [
            {
              'name': 'test',
              'properties': {
                # Properties for this test.
              },
            },
          ],
          'properties': {
            # Properties for this step.
          },
        },
      }
      analyzed_build_id (int): Buildbucket Id of the analyzed build, may be used
          to derive properties for reruns.

    Returns:
      (dict): input properties of the rerun build."""
    # pylint: disable=unused-argument
    raise NotImplementedError

  def GetFailureKeysToAnalyzeTestFailures(self, failure_entities):
    """Gets failures that'll actually be analyzed in the analysis."""
    return [f.key for f in failure_entities]

  def GetCompileFailureInfo(self, context, build,
                            first_failures_in_current_build):
    """Creates input object required by heuristic analysis for compile."""
    # pylint: disable=unused-argument
    return {}

  def FailureShouldBeAnalyzed(self, failure_entity):
    """Checks if the failure is supposed to be analyzed."""
    # pylint: disable=unused-argument
    return True

  def ClearSkipFlag(self, failure_entities):
    """For failures that were skipped on purpose then require to be analyzed,
      updates them to be picked up by an analysis.

    So far this is a special case for CrOS: CrOS can tell Findit to skip
    analyzing a failed build if there are too many failures. Those failures
    will have a flag in properties indicates that they don't need analysis.
    But a following build with failures that need analysis might be merged into
    some of the skipped failures, if so those particular failures need to update
    to be analyzed.
    """
    # pylint: disable=unused-argument
    return

  def GetTestFailureInfo(self, context, build, first_failures_in_current_build):
    """Creates input object required by heuristic analysis for test."""
    # pylint: disable=unused-argument
    return {}


  def AsyncCommitRevert(self, culprit, revert_description,
                        request_confirmation_message):
    """Enqueues a task to revert a culprit and land it.

    This is needed so that the wrapping transaction can complete without waiting
    for an external RPC.
    """
    self._ScheduleTask(
        'CommitRevert', culprit, {
            'revert_description': revert_description,
            'request_confirmation_message': request_confirmation_message,
        })

  def AsyncRequestReview(self, culprit, revert_description,
                         request_review_message):
    """Enqueues a task to revert a culprit and request it to be reviewed.

    This is needed so that the wrapping transaction can complete without waiting
    for an external RPC.
    """
    self._ScheduleTask(
        'RequestReview', culprit, {
            'revert_description': revert_description,
            'request_review_message': request_review_message,
        })

  def AsyncNotifyCulprit(self, culprit, message, silent_notification=True):
    """Enqueues a task to notify a culprit.

    This is needed so that the wrapping transaction can complete without waiting
    for an external RPC.
    """
    self._ScheduleTask('NotifyCulprit', culprit, {
        'message': message,
        'silent_notification': silent_notification,
    })


  def GetAutoRevertReviewers(self):
    """Returns a list of reviewers to be notified of automated reverts."""
    # pylint: disable=unused-argument
    return []

  def _ScheduleTask(self, action, culprit, action_params):
    """Transactionally schedules a task to take an action later.

    Args:
      action (str): An action understood by findit_v2/handlers/async_action.py
      culprit: Datastore entity for the culprit to take action on.
      action_params: Parameters specific to the type of action as a
          json-serializable dict.
    """
    assert ndb.in_transaction()
    payload = {
        'project': self.project_name,
        'action': action,
        'culprit_key': culprit.key.urlsafe(),
    }
    payload.update(action_params)

    taskqueue.add(
        queue_name='findit-v2-actions',
        url='/findit/internal/v2/task/async-action',
        method='POST',
        target='findit-backend',
        payload=json.dumps(payload),
        transactional=True,
    )
