# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This module is for helper function for ResultDB."""

import logging

from infra_api_clients.swarming import swarming_util
from libs.test_results.resultdb_test_results import ResultDBTestResults
from services import resultdb
from services import swarming


def get_failed_tests_for_swarming_ids(swarming_ids):
  """Given a list of swarming_ids, queries ResultDB for failed test
  Arguments:
    swarming_ids: a list of swarming id
  Returns:
    A ResultDBTestResults instance containing the results, or None if no result
  """
  all_results = []
  # TODO (nqmtuan): Consider running this in parallel
  for swarming_id in swarming_ids:
    inv_name = swarming_util.GetInvocationNameForSwarmingTask(
        swarming.SwarmingHost(), swarming_id)
    if not inv_name:
      logging.error("Could not get invocation for swarming task %s",
                    swarming_id)
      continue
    res = resultdb.query_resultdb(inv_name)
    all_results.extend(res)
  if all_results:
    return ResultDBTestResults(all_results)
  return None


def get_failed_tests_in_step(failed_step):
  """Given a failed step, queries ResultDB for failed test
  Arguments:
    failed_step: a TestFailedStep instance
  Returns:
    A ResultDBTestResults instance containing the results, or None if no result
  """
  return get_failed_tests_for_swarming_ids(failed_step.swarming_ids)
