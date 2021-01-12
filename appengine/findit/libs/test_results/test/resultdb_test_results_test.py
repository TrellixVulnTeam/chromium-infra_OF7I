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
        status=test_result_pb2.TestStatus.PASS,
        expected=True,
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic1",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic1"),
            common_pb2.StringPair(key="gtest_status", value="FAIL"),
        ],
        status=test_result_pb2.TestStatus.FAIL,
        summary_html="summary1",
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic1",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic1"),
            common_pb2.StringPair(key="gtest_status", value="FAIL"),
        ],
        status=test_result_pb2.TestStatus.FAIL,
        summary_html="summary2",
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic2",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic2"),
            common_pb2.StringPair(key="gtest_status", value="ABORT"),
        ],
        status=test_result_pb2.TestStatus.ABORT,
        summary_html="summary3",
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic3",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic3"),
            common_pb2.StringPair(key="gtest_status", value="SKIP"),
        ],
        status=test_result_pb2.TestStatus.SKIP,
        expected=True,
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic4",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic4"),
            common_pb2.StringPair(key="gtest_status", value="UNSPECIFIED"),
        ],
        status=test_result_pb2.TestStatus.STATUS_UNSPECIFIED,
    ),
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic4",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic4"),
            common_pb2.StringPair(key="gtest_status", value="SKIP"),
        ],
        status=test_result_pb2.TestStatus.SKIP,
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
    test_result = test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic",
        tags=[common_pb2.StringPair(key="k", value="v")])
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
                "total_run": 2,
                "num_expected_results": 1,
                "num_unexpected_results": 1,
                "num_passed": 1,
                "num_failed": 0,
                "num_crashed": 1,
                "num_aborted": 0,
                "num_skipped": 0,
                "num_notrun": 0,
                "num_unspecified": 0
            },
            u"ninja://gpu:gl_tests/SharedImageTest.Basic1": {
                "test_name": u"SharedImageTest.Basic1",
                "reliable_failure": True,
                "failure_logs": [u"summary1", u"summary2"],
                "test_type": ResultDBTestType.GTEST,
                "total_run": 2,
                "num_expected_results": 0,
                "num_unexpected_results": 2,
                "num_passed": 0,
                "num_failed": 2,
                "num_crashed": 0,
                "num_aborted": 0,
                "num_skipped": 0,
                "num_notrun": 0,
                "num_unspecified": 0
            },
            u"ninja://gpu:gl_tests/SharedImageTest.Basic2": {
                "test_name": u"SharedImageTest.Basic2",
                "reliable_failure": True,
                "failure_logs": ['summary3'],
                "test_type": ResultDBTestType.GTEST,
                "total_run": 1,
                "num_expected_results": 0,
                "num_unexpected_results": 1,
                "num_passed": 0,
                "num_failed": 0,
                "num_crashed": 0,
                "num_aborted": 1,
                "num_skipped": 0,
                "num_notrun": 0,
                "num_unspecified": 0
            },
            u"ninja://gpu:gl_tests/SharedImageTest.Basic3": {
                "test_name": u"SharedImageTest.Basic3",
                "reliable_failure": False,
                "failure_logs": [],
                "test_type": ResultDBTestType.GTEST,
                "total_run": 1,
                "num_expected_results": 1,
                "num_unexpected_results": 0,
                "num_passed": 0,
                "num_failed": 0,
                "num_crashed": 0,
                "num_aborted": 0,
                "num_skipped": 1,
                "num_notrun": 0,
                "num_unspecified": 0
            },
            u"ninja://gpu:gl_tests/SharedImageTest.Basic4": {
                "test_name": u"SharedImageTest.Basic4",
                "reliable_failure": False,
                "failure_logs": [],
                "test_type": ResultDBTestType.GTEST,
                "total_run": 2,
                "num_expected_results": 0,
                "num_unexpected_results": 2,
                "num_passed": 0,
                "num_failed": 0,
                "num_crashed": 0,
                "num_aborted": 0,
                "num_skipped": 0,
                "num_notrun": 1,
                "num_unspecified": 1
            },
        })

  def testGetFailedTestsInformation(self):
    rdb_test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    self.assertTrue(rdb_test_results.contains_all_tests)
    failed_test_log, reliable_failed_tests = rdb_test_results.GetFailedTestsInformation()  # pylint: disable=line-too-long
    self.assertEqual(
        failed_test_log, {
            "SharedImageTest.Basic1": base64.b64encode("summary1\nsummary2"),
            "SharedImageTest.Basic2": base64.b64encode("summary3"),
        })
    self.assertEqual(
        reliable_failed_tests, {
            "SharedImageTest.Basic1": "SharedImageTest.Basic1",
            "SharedImageTest.Basic2": "SharedImageTest.Basic2",
        })

  def testGetClassifiedTestResults(self):
    expected_statuses = {
        'SharedImageTest.Basic': {
            'total_run': 2,
            'num_expected_results': 1,
            'num_unexpected_results': 1,
            'results': {
                'passes': {
                    'PASS': 1
                },
                'failures': {
                    'CRASH': 1
                },
                'skips': {},
                'unknowns': {},
                'notruns': {},
            }
        },
        'SharedImageTest.Basic1': {
            'total_run': 2,
            'num_expected_results': 0,
            'num_unexpected_results': 2,
            'results': {
                'passes': {},
                'failures': {
                    'FAIL': 2
                },
                'skips': {},
                'unknowns': {},
                'notruns': {},
            }
        },
        'SharedImageTest.Basic2': {
            'total_run': 1,
            'num_expected_results': 0,
            'num_unexpected_results': 1,
            'results': {
                'passes': {},
                'failures': {
                    'ABORT': 1
                },
                'skips': {},
                'unknowns': {},
                'notruns': {},
            }
        },
        'SharedImageTest.Basic3': {
            'total_run': 1,
            'num_expected_results': 1,
            'num_unexpected_results': 0,
            'results': {
                'passes': {},
                'failures': {},
                'skips': {
                    'SKIP': 1
                },
                'unknowns': {},
                'notruns': {},
            }
        },
        'SharedImageTest.Basic4': {
            'total_run': 2,
            'num_expected_results': 0,
            'num_unexpected_results': 2,
            'results': {
                'passes': {},
                'failures': {},
                'skips': {},
                'unknowns': {
                    'UNSPECIFIED': 1
                },
                'notruns': {
                    'SKIP': 1
                },
            }
        },
    }
    test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    classified_results = test_results.GetClassifiedTestResults()
    for test_name, expected_test_result in expected_statuses.iteritems():
      self.assertEqual(expected_test_result,
                       classified_results[test_name].ToDict())
