# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This module is for processing test results from resultdb"""

import base64
import logging

from collections import defaultdict
from go.chromium.org.luci.resultdb.proto.v1 import test_result_pb2
from libs.test_results.base_test_results import BaseTestResults
from libs.test_results.classified_test_results import ClassifiedTestResults

_FAILURE_STATUSES = [
    test_result_pb2.TestStatus.FAIL, test_result_pb2.TestStatus.CRASH,
    test_result_pb2.TestStatus.ABORT
]


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
    self.test_results = ResultDBTestResults.group_test_results_by_test_id(
        test_results)

  def GetFailedTestsInformation(self):
    failed_test_log = {}
    reliable_failed_tests = {}
    for _, result in self.test_results.items():
      test_name = result["test_name"]
      if result["reliable_failure"] and test_name:
        merged_test_log = '\n'.join(result["failure_logs"])
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
    for _, test_info in self.test_results.items():
      test_name = test_info["test_name"]
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

  @staticmethod
  def group_test_results_by_test_id(test_results):
    """Returns a dictionary of
    {
      <test_id>:{
        "test_name": <test name>
        "reliable_failure": whether the test fail consistently
        "failure_logs": array of failure logs
        "test_type": type of test
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
      test_id = test_result.test_id
      is_failure = ResultDBTestResults.is_failure(test_result)
      log = ResultDBTestResults.summary_html_for_test_result(test_result)
      if not results.get(test_id):
        results[test_id] = {
            "test_name":
                ResultDBTestResults.test_name_for_test_result(test_result),
            "reliable_failure":
                is_failure,
            "failure_logs": [log] if is_failure else [],
            "test_type":
                ResultDBTestResults.test_type_for_test_result(test_result),
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
        results[test_id]["reliable_failure"] = results[test_id][
            "reliable_failure"] and is_failure
        if is_failure:
          results[test_id]["failure_logs"].append(log)
      ResultDBTestResults._update_classified_test_results(
          results[test_id], test_result)
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
    elif test_result.status == test_result_pb2.TestStatus.STATUS_UNSPECIFIED:
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
