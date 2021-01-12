# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Functions for operating on tests run in swarming."""

import json
import logging
import time

from google.appengine.ext import ndb

from common.findit_http_client import FinditHttpClient
from dto import swarming_task_error
from dto.swarming_task_error import SwarmingTaskError
from dto.test_location import TestLocation
from infra_api_clients.swarming import swarming_util
from libs.test_results import test_results_util
from libs.test_results.resultdb_test_results import ResultDBTestResults
from services import isolate
from services import constants
from services import resultdb
from services import swarming
from waterfall import waterfall_config

_FINDIT_HTTP_CLIENT = FinditHttpClient()


def GetOutputJsonByOutputsRef(outputs_ref, http_client):
  """Downloads failure log from isolated server."""
  isolated_data = swarming_util.GenerateIsolatedData(outputs_ref)
  file_content, error = isolate.DownloadFileFromIsolatedServer(
      isolated_data, http_client, 'output.json')
  return json.loads(file_content) if file_content else None, error


def GetSwarmingTaskDataAndResult(task_id,
                                 http_client=_FINDIT_HTTP_CLIENT,
                                 use_resultdb=False):
  """Gets information about a swarming task.

  Returns: A tuple of 3 elements
    Data of swarming task: dict
    Test results for swarming task: subclass of BaseTestResults
    Error for swarming task: dict
  """
  data, error = swarming_util.GetSwarmingTaskResultById(swarming.SwarmingHost(),
                                                        task_id, http_client)

  error = SwarmingTaskError.FromSerializable(error)
  if not data:
    return None, None, error

  task_state = data['state']
  test_results = None
  output_json = None
  if task_state not in constants.STATE_NOT_STOP:
    if task_state == constants.STATE_COMPLETED:
      if use_resultdb:
        inv_name = swarming_util.GetInvocationNameForSwarmingTask(
            swarming.SwarmingHost(), task_id)
        res = resultdb.query_resultdb(
            inv_name, only_variants_with_unexpected_results=False)
        if res:
          test_results = ResultDBTestResults(res)
      else:
        outputs_ref = data.get('outputs_ref')

        # If swarming task aborted because of errors in request arguments,
        # it's possible that there is no outputs_ref.
        if not outputs_ref:
          error = error or SwarmingTaskError.GenerateError(
              swarming_task_error.NO_TASK_OUTPUTS)
        else:
          output_json, error = GetOutputJsonByOutputsRef(
              outputs_ref, http_client)
          if not output_json:
            error = error or SwarmingTaskError.GenerateError(
                swarming_task_error.NO_OUTPUT_JSON)
          test_results = test_results_util.GetTestResultObject(output_json)
    else:
      # The swarming task did not complete successfully.
      logging.error('Swarming task %s stopped with status: %s', task_id,
                    task_state)
      error = SwarmingTaskError.GenerateError(
          swarming_task_error.STATES_NOT_RUNNING_TO_ERROR_CODES[task_state])
  return data, test_results, error


def GetTestResultForSwarmingTask(task_id, http_client=_FINDIT_HTTP_CLIENT):
  """Get test results object for a swarming task based on it's id."""
  _data, test_results, _error = GetSwarmingTaskDataAndResult(
      task_id, http_client)
  return test_results


def GetTestLocation(task_id, test_name):
  """Gets the filepath and line number of a test from swarming.

  Args:
    task_id (str): The swarming task id to query.
    test_name (str): The name of the test whose location to return.

  Returns:
    (TestLocation): The file path and line number of the test, or None
        if the test location was not be retrieved.

  """
  test_results = GetTestResultForSwarmingTask(task_id, _FINDIT_HTTP_CLIENT)
  test_location, error = test_results.GetTestLocation(
      test_name) if test_results else (None, constants.WRONG_FORMAT_LOG)
  if error:
    return None
  return TestLocation.FromSerializable(test_location or {})


def RetrieveShardedTestResultsFromIsolatedServer(list_isolated_data,
                                                 http_client):
  """Gets test results from isolated server and merge the results."""
  shard_results = []
  for isolated_data in list_isolated_data:
    test_result_log, _ = isolate.DownloadFileFromIsolatedServer(
        isolated_data, http_client, 'output.json')
    if not test_result_log:
      return None
    shard_results.append(json.loads(test_result_log))

  if not shard_results:
    return []

  test_results = test_results_util.GetTestResultObject(shard_results[0])
  return test_results.GetMergedTestResults(
      shard_results) if test_results else None


def GetTaskIdFromSwarmingTaskEntity(urlsafe_task_key):
  """Gets swarming task id from SwarmingTask. Waits and polls if needed."""
  swarming_settings = waterfall_config.GetSwarmingSettings()
  wait_seconds = swarming_settings.get('get_swarming_task_id_wait_seconds')
  timeout_seconds = swarming_settings.get(
      'get_swarming_task_id_timeout_seconds')
  deadline = time.time() + timeout_seconds

  while time.time() < deadline:
    swarming_task = ndb.Key(urlsafe=urlsafe_task_key).get()

    if not swarming_task:
      raise Exception('Swarming task was deleted unexpectedly!')

    if swarming_task.task_id:
      return swarming_task.task_id
    # Wait for the existing pipeline to start the Swarming task.
    time.sleep(wait_seconds)

  raise Exception('Timed out waiting for task_id.')
