# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This module is for processing test results from resultdb"""

from libs.test_results.base_test_results import BaseTestResults


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
    self.test_results = test_results
