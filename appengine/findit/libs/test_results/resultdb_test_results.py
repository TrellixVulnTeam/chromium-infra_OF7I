# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This module is for processing test results from resultdb"""

import base64
import logging

from collections import defaultdict
from common.findit_http_client import FinditHttpClient
from go.chromium.org.luci.resultdb.proto.v1 import test_result_pb2
from infra_api_clients import http_client_util
from libs.test_results.base_test_results import BaseTestResults
from libs.test_results.classified_test_results import ClassifiedTestResults
from services import resultdb

_FAILURE_STATUSES = [
    test_result_pb2.TestStatus.FAIL, test_result_pb2.TestStatus.CRASH,
    test_result_pb2.TestStatus.ABORT
]
_FINDIT_HTTP_CLIENT = FinditHttpClient()


class ResultDBTestType(object):
  OTHER = 'OTHER'
  GTEST = 'GTEST'
  BLINK = 'BLINK'


# TODO (crbug/981066): Implement this
# pylint: disable=abstract-method
class ResultDBTestResults(BaseTestResults):

  def __init__(self, test_results, partial_result=False):
    """Creates a ResultDBTestResults object from resultdb test results
    Arguments:
      test_results: Array of luci.resultdb.v1.TestResult object
      partial_result: False if the results are from a single shard, True if
      the results are from all shards
    """
    self.partial_result = partial_result
    self.test_results = ResultDBTestResults.group_test_results_by_test_name(
        test_results)

  def GetFailedTestsInformation(self):
    failed_test_log = {}
    reliable_failed_tests = {}
    for test_name, result in self.test_results.items():
      if result["reliable_failure"]:
        test_type = result["test_type"]
        # TODO(crbug.com/981066): Consider running this in parallel
        real_logs = map(
            lambda l: ResultDBTestResults.get_detailed_failure_log(
                test_type, l), result["failure_logs"])
        merged_test_log = '\n'.join(real_logs)
        failed_test_log[test_name] = base64.b64encode(merged_test_log)
        reliable_failed_tests[test_name] = test_name
    return failed_test_log, reliable_failed_tests

  @property
  def contains_all_tests(self):
    """
    True if the test result is merged results for all shards; False if it's a
    partial result.
    """
    return not self.partial_result

  def test_type(self):
    for _, result in self.test_results.items():
      return result["test_type"]
    return ResultDBTestType.OTHER

  def GetClassifiedTestResults(self):
    """Parses ResultDB results, counts and classifies test results.
    Also counts number of expected and unexpected results for each test.

    Returns:
      (ClassifiedTestResults) An object with information for each test:
      * total_run: total number of runs,
      * num_expected_results: total number of runs with expected results,
      * num_unexpected_results: total number of runs with unexpected results,
      * results: classified test results in 5 groups: passes, failures, skips,
        unknowns, notruns.
    """
    classified_results = ClassifiedTestResults()
    for test_name, test_info in self.test_results.items():
      classified_results[test_name].total_run = test_info["total_run"]
      classified_results[test_name].num_expected_results = test_info[
          "num_expected_results"]
      classified_results[test_name].num_unexpected_results = test_info[
          "num_unexpected_results"]
      if test_info["num_passed"]:
        classified_results[test_name].results.passes['PASS'] = test_info[
            "num_passed"]
      if test_info["num_failed"]:
        classified_results[test_name].results.failures['FAIL'] = test_info[
            "num_failed"]
      if test_info["num_crashed"]:
        classified_results[test_name].results.failures['CRASH'] = test_info[
            "num_crashed"]
      if test_info["num_aborted"]:
        classified_results[test_name].results.failures['ABORT'] = test_info[
            "num_aborted"]
      if test_info["num_skipped"]:
        classified_results[test_name].results.skips['SKIP'] = test_info[
            "num_skipped"]
      if test_info["num_notrun"]:
        classified_results[test_name].results.notruns['SKIP'] = test_info[
            "num_notrun"]
      if test_info["num_unspecified"]:
        classified_results[test_name].results.unknowns[
            'UNSPECIFIED'] = test_info["num_unspecified"]
    return classified_results

  def GetTestLocation(self, test_name):
    """Gets test location for a specific test.
    Returns: A tuple containing
      * A dictionary of {
          "line": line number of the test
          "file": file path to the test
        }
      * A possible error string
    """
    location = self.test_results.get(test_name, {}).get('test_location')
    if not location:
      return None, 'test location not found'
    return location, None

  def DoesTestExist(self, test_name):
    return test_name in self.test_results

  def IsTestResultUseful(self):
    return len(self.test_results) > 0

  @staticmethod
  def group_test_results_by_test_name(test_results):
    # pylint: disable=line-too-long
    """Returns a dictionary of
    {
      <test_name>:{
        "reliable_failure": whether the test fail consistently
        "failure_logs": array of dictionary {
          "name": test result name (e.g. invocations/task-chromium-swarm.appspot.com-508dcba4306cae11/tests/ninja:%2F%2Fgpu:gl_tests%2FSharedImageGLBackingProduceDawnTest.Basic/results/c649f775-00777)
          "summary_html": summary_html of a run
        }
        "test_type": type of test
        "test_location": location of the test
        "total_run": number of runs for the test
        "num_expected_results": number of expected runs
        "num_unexpected_results": number of unexpected runs
        "num_passed": number of passed results
        "num_failed": number of failed results
        "num_crashed": number of crashed results
        "num_aborted": number of aborted results
        "num_skipped": number of skipped results
        "num_notrun": number of not run results
        "num_unspecified": number of unspecified results
      }
    }
    Arguments:
      test_results: Array of ResultDB TestResult object
    """
    results = defaultdict(dict)
    for test_result in test_results:
      test_name = ResultDBTestResults.test_name_for_test_result(test_result)
      if not test_name:
        continue
      is_failure = ResultDBTestResults.is_failure(test_result)
      log = {
          "name":
              test_result.name,
          "summary_html":
              ResultDBTestResults.summary_html_for_test_result(test_result)
      }
      if not results.get(test_name):
        results[test_name] = {
            "reliable_failure":
                is_failure,
            "failure_logs": [log] if is_failure else [],
            "test_type":
                ResultDBTestResults.test_type_for_test_result(test_result),
            "test_location":
                ResultDBTestResults.test_location_for_test_result(test_result),
            "total_run":
                0,
            "num_expected_results":
                0,
            "num_unexpected_results":
                0,
            "num_passed":
                0,
            "num_failed":
                0,
            "num_crashed":
                0,
            "num_aborted":
                0,
            "num_skipped":
                0,
            "num_notrun":
                0,
            "num_unspecified":
                0,
        }
      else:
        results[test_name]["reliable_failure"] = results[test_name][
            "reliable_failure"] and is_failure
        if is_failure:
          results[test_name]["failure_logs"].append(log)
      ResultDBTestResults._update_classified_test_results(
          results[test_name], test_result)
    return results

  @staticmethod
  def _update_classified_test_results(classified_results, test_result):
    """Update classified_results with a test result object
    Arguments:
      classified_results: A dictionary containing results for a test ID
      test_result: A luci.resultdb.v1.TestResult object
    """
    classified_results["total_run"] += 1
    if test_result.expected:
      classified_results["num_expected_results"] += 1
    else:
      classified_results["num_unexpected_results"] += 1
    if test_result.status == test_result_pb2.TestStatus.PASS:
      classified_results["num_passed"] += 1
    elif test_result.status == test_result_pb2.TestStatus.FAIL:
      classified_results["num_failed"] += 1
    elif test_result.status == test_result_pb2.TestStatus.CRASH:
      classified_results["num_crashed"] += 1
    elif test_result.status == test_result_pb2.TestStatus.ABORT:
      classified_results["num_aborted"] += 1
    elif test_result.status == test_result_pb2.TestStatus.SKIP:
      if test_result.expected:
        classified_results["num_skipped"] += 1
      else:
        classified_results["num_notrun"] += 1
    else:
      classified_results["num_unspecified"] += 1

  @staticmethod
  def is_failure(test_result):
    return test_result.status in _FAILURE_STATUSES and not test_result.expected

  @staticmethod
  def test_name_for_test_result(test_result):
    """Returns the test name for luci.resultdb.v1.TestResult object
    Arguments:
      test_result: A luci.resultdb.v1.TestResult object
    """
    for tag in test_result.tags or []:
      if tag.key == "test_name":
        return tag.value
    logging.warning("There is no test name for test_id: %s",
                    test_result.test_id)
    return None

  @staticmethod
  def summary_html_for_test_result(test_result):
    return test_result.summary_html or ""

  @staticmethod
  def test_type_for_test_result(test_result):
    """Return a ResultDBTestType for test_result"""
    if "blink_web_tests" in test_result.test_id:
      return ResultDBTestType.BLINK
    if test_result.tags:
      for tag in test_result.tags:
        if "gtest" in tag.key:
          return ResultDBTestType.GTEST
    return ResultDBTestType.OTHER

  @staticmethod
  def test_location_for_test_result(test_result):
    """Return test location for test_result"""
    if (not test_result.test_metadata or
        not test_result.test_metadata.location or
        not test_result.test_metadata.location.file_name):
      return None
    return {
        "line": test_result.test_metadata.location.line,
        "file": test_result.test_metadata.location.file_name
    }

  @staticmethod
  def get_detailed_failure_log(test_type, failure_log):
    """Gets the detailed failure log from artifact if possible
      For gtest, if there is stack_trace artifact, download the content of the
      artifact. Otherwise, just return summaryHTML
    Argument:
      test_type: ResultDBTestType
      failure_log: Dictionary of {"name":..., "summary_html":...}
    Returns:
      A string for the detailed failure logs
    """
    summary_html = failure_log["summary_html"]
    if test_type != ResultDBTestType.GTEST:
      return summary_html
    # We only check for "stack_trace" artifact if "stack_trace" presents in
    # summary_html
    if "stack_trace" not in summary_html:
      return summary_html
    test_result_name = failure_log["name"]
    artifacts = resultdb.list_artifacts(test_result_name) or []
    stack_trace_artifact = next(
        (a for a in artifacts if a.artifact_id == "stack_trace"), None)
    if not stack_trace_artifact:
      return summary_html
    fetch_url = stack_trace_artifact.fetch_url
    content, error = http_client_util.SendRequestToServer(
        fetch_url, _FINDIT_HTTP_CLIENT)
    if not error:
      return content
    logging.warning("Unable to fetch content from %s: %s", fetch_url, error)
    return summary_html
