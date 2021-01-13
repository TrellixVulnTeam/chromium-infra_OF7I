# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock
from services import parameters
from services import resultdb
from waterfall.test import wf_testcase

from go.chromium.org.luci.resultdb.proto.v1 import test_result_pb2
from infra_api_clients.swarming import swarming_util
from services import resultdb
from services import resultdb_util


class ResultDBTest(wf_testcase.WaterfallTestCase):

  @mock.patch.object(
      swarming_util,
      'GetInvocationNameForSwarmingTask',
      return_value="inv_name")
  @mock.patch.object(resultdb, 'query_resultdb')
  def testGetFailedTestInStep(self, mock_result_db, *_):
    failed_step = parameters.TestFailedStep()
    failed_step.swarming_ids = ["1", "2"]
    mock_result_db.side_effect = [
        [test_result_pb2.TestResult(test_id="test_id_1")],
        [test_result_pb2.TestResult(test_id="test_id_2")],
    ]
    test_results = resultdb_util.get_failed_tests_in_step(failed_step)
    self.assertEqual(len(test_results.test_results), 2)
    failed_step.swarming_ids = []
    test_results = resultdb_util.get_failed_tests_in_step(failed_step)
    self.assertIsNone(test_results)
