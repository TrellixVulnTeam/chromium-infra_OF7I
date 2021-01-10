# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import base64
import mock

from go.chromium.org.luci.resultdb.proto.v1 import common_pb2, test_result_pb2
from libs.test_results.resultdb_test_results import (ResultDBTestResults,
                                                     ResultDBTestType)
from waterfall.test import wf_testcase

_SAMPLE_TEST_RESULTS = [
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic"),
            common_pb2.StringPair(key="gtest_status", value="CRASH"),
        ],
        status=test_result_pb2.TestStatus.CRASH,
        summary_html="summary",
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic"),
            common_pb2.StringPair(key="gtest_status", value="PASS"),
        ],
        status="PASS",
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic1",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic1"),
            common_pb2.StringPair(key="gtest_status", value="FAIL"),
        ],
        status="FAIL",
        summary_html="summary1",
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic1",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic1"),
            common_pb2.StringPair(key="gtest_status", value="FAIL"),
        ],
        status="FAIL",
        summary_html="summary2",
    ),
]


class ResultDBTestResultsTest(wf_testcase.WaterfallTestCase):

  def testTestTypeForSingleResult(self):
    test_result = test_result_pb2.TestResult(
        test_id="ninja://:blink_web_tests/animations/keyframes-integer.html")
    self.assertEqual(
        ResultDBTestResults.test_type_for_test_result(test_result),
        ResultDBTestType.BLINK)
    test_result = test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic",
        tags=[common_pb2.StringPair(key="gtest_status", value="CRASH")])
    self.assertEqual(
        ResultDBTestResults.test_type_for_test_result(test_result),
        ResultDBTestType.GTEST)
    test_result = test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic")
    self.assertEqual(
        ResultDBTestResults.test_type_for_test_result(test_result),
        ResultDBTestType.OTHER)

  def testTestType(self):
    rdb_test_results = ResultDBTestResults([])
    self.assertEqual(rdb_test_results.test_type(), ResultDBTestType.OTHER)
    rdb_test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    self.assertEqual(rdb_test_results.test_type(), ResultDBTestType.GTEST)

  def testTestName(self):
    test_result = test_result_pb2.TestResult(
        test_id="ninja://:blink_web_tests/animations/keyframes-integer.html")
    self.assertIsNone(
        ResultDBTestResults.test_name_for_test_result(test_result))
    test_result = test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic",
        tags=[common_pb2.StringPair(key="gtest_status", value="CRASH")])
    self.assertIsNone(
        ResultDBTestResults.test_name_for_test_result(test_result))
    test_result = test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic")
        ])
    self.assertEqual(
        ResultDBTestResults.test_name_for_test_result(test_result),
        "SharedImageTest.Basic")

  def testGroupTestResultsByTestId(self):
    rdb_test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    self.assertEqual(
        rdb_test_results.test_results, {
            u"ninja://gpu:gl_tests/SharedImageTest.Basic": {
                "test_name": u"SharedImageTest.Basic",
                "reliable_failure": False,
                "failure_logs": [u"summary"],
                "test_type": ResultDBTestType.GTEST,
            },
            u"ninja://gpu:gl_tests/SharedImageTest.Basic1": {
                "test_name": u"SharedImageTest.Basic1",
                "reliable_failure": True,
                "failure_logs": [u"summary1", u"summary2"],
                "test_type": ResultDBTestType.GTEST,
            },
        })

  def testGetFailedTestsInformation(self):
    rdb_test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    self.assertTrue(rdb_test_results.contains_all_tests)
    failed_test_log, reliable_failed_tests = rdb_test_results.GetFailedTestsInformation()  # pylint: disable=line-too-long
    self.assertEqual(failed_test_log, {
        "SharedImageTest.Basic1": base64.b64encode("summary1\nsummary2"),
    })
    self.assertEqual(reliable_failed_tests, {
        "SharedImageTest.Basic1": "SharedImageTest.Basic1",
    })
