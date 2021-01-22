# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import base64
import mock

from go.chromium.org.luci.resultdb.proto.v1 import (artifact_pb2, common_pb2,
                                                    test_result_pb2,
                                                    test_metadata_pb2)
from infra_api_clients import http_client_util
from libs.test_results.resultdb_test_results import (ResultDBTestResults,
                                                     ResultDBTestType)
from services import resultdb
from waterfall.test import wf_testcase

_SAMPLE_TEST_RESULTS = [
    test_result_pb2.TestResult(
        test_id="ninja://gpu:gl_tests/SharedImageTest.Basic",
        name="name1",
        tags=[
            common_pb2.StringPair(
                key="test_name", value="SharedImageTest.Basic"),
            common_pb2.StringPair(key="gtest_status", value="CRASH"),
        ],
        status=test_result_pb2.TestStatus.CRASH,
        summary_html="summary",
        test_metadata=test_metadata_pb2.TestMetadata(
            name="SharedImageTest.Basic",
            location=test_metadata_pb2.TestLocation(
                repo="https://chromium.googlesource.com/chromium/src",
                file_name="/path/to/test",
                line=123))),
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
        name="name2",
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
        name="name3",
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
        name="name4",
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

  def testGroupTestResultsByTestName(self):
    rdb_test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    self.assertEqual(
        rdb_test_results.test_results, {
            u"SharedImageTest.Basic": {
                "reliable_failure": False,
                "failure_logs": [{
                    "name": "name1",
                    "summary_html": "summary"
                }],
                "test_type": ResultDBTestType.GTEST,
                "test_location": {
                    "file": "/path/to/test",
                    "line": 123
                },
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
            u"SharedImageTest.Basic1": {
                "reliable_failure": True,
                "failure_logs": [{
                    "name": "name2",
                    "summary_html": "summary1"
                }, {
                    "name": "name3",
                    "summary_html": "summary2"
                }],
                "test_type": ResultDBTestType.GTEST,
                "test_location": None,
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
            u"SharedImageTest.Basic2": {
                "reliable_failure": True,
                "failure_logs": [{
                    "name": "name4",
                    "summary_html": "summary3"
                }],
                "test_type": ResultDBTestType.GTEST,
                "test_location": None,
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
            u"SharedImageTest.Basic3": {
                "reliable_failure": False,
                "failure_logs": [],
                "test_type": ResultDBTestType.GTEST,
                "test_location": None,
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
            u"SharedImageTest.Basic4": {
                "reliable_failure": False,
                "failure_logs": [],
                "test_type": ResultDBTestType.GTEST,
                "test_location": None,
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

  @mock.patch.object(ResultDBTestResults, 'get_detailed_failure_log')
  def testGetFailedTestsInformation(self, mock_get_log):
    mock_get_log.side_effect = ["content1", "content2", "content3"]
    rdb_test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    self.assertTrue(rdb_test_results.contains_all_tests)
    failed_test_log, reliable_failed_tests = rdb_test_results.GetFailedTestsInformation()  # pylint: disable=line-too-long
    self.assertEqual(
        failed_test_log, {
            "SharedImageTest.Basic1": base64.b64encode("content1\ncontent2"),
            "SharedImageTest.Basic2": base64.b64encode("content3"),
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

  def testGetTestLocation(self):
    rdb_test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    location, err = rdb_test_results.GetTestLocation("SharedImageTest.Basic")
    self.assertEqual(location, {"file": "/path/to/test", "line": 123})
    self.assertIsNone(err)
    location, err = rdb_test_results.GetTestLocation("InvalidTest")
    self.assertIsNone(location)
    self.assertIsNotNone(err)

  def testGetDetailedFailureLogNonGtestTestType(self):
    test_type = ResultDBTestType.BLINK
    log = {"name": "test_result_name", "summary_html": "summary_html"}
    self.assertEqual(
        ResultDBTestResults.get_detailed_failure_log(test_type, log),
        "summary_html")

  def testGetDetailedFailureLogStackTraceNotInSummaryHTML(self):
    test_type = ResultDBTestType.GTEST
    log = {"name": "test_result_name", "summary_html": "summary_html"}
    self.assertEqual(
        ResultDBTestResults.get_detailed_failure_log(test_type, log),
        "summary_html")

  @mock.patch.object(resultdb, 'list_artifacts')
  def testGetDetailedFailureLogNoArtifact(self, mock_list_artifacts):
    test_type = ResultDBTestType.GTEST
    log = {
        "name": "test_result_name",
        "summary_html": "Please see stack_trace artifact"
    }
    mock_list_artifacts.return_value = []
    self.assertEqual(
        ResultDBTestResults.get_detailed_failure_log(test_type, log),
        "Please see stack_trace artifact")
    mock_list_artifacts.assert_called_once_with("test_result_name")

  @mock.patch.object(resultdb, 'list_artifacts')
  def testGetDetailedFailureLogNoStackTraceArtifact(self, mock_list_artifacts):
    test_type = ResultDBTestType.GTEST
    log = {
        "name": "test_result_name",
        "summary_html": "Please see stack_trace artifact"
    }
    mock_list_artifacts.return_value = [
        artifact_pb2.Artifact(
            artifact_id="some_artifact", fetch_url="https://abc.xyz/some_url")
    ]
    self.assertEqual(
        ResultDBTestResults.get_detailed_failure_log(test_type, log),
        "Please see stack_trace artifact")
    mock_list_artifacts.assert_called_once_with("test_result_name")

  @mock.patch.object(resultdb, 'list_artifacts')
  @mock.patch.object(http_client_util, 'SendRequestToServer')
  def testGetDetailedFailureLogFetchError(self, mock_http_client,
                                          mock_list_artifacts):
    test_type = ResultDBTestType.GTEST
    log = {
        "name": "test_result_name",
        "summary_html": "Please see stack_trace artifact"
    }
    mock_list_artifacts.return_value = [
        artifact_pb2.Artifact(
            artifact_id="some_artifact", fetch_url="https://abc.xyz/some_url"),
        artifact_pb2.Artifact(
            artifact_id="stack_trace",
            fetch_url="https://abc.xyz/some_other_url"),
    ]
    mock_http_client.return_value = (None, "some error")
    self.assertEqual(
        ResultDBTestResults.get_detailed_failure_log(test_type, log),
        "Please see stack_trace artifact")
    mock_list_artifacts.assert_called_once_with("test_result_name")

  @mock.patch.object(resultdb, 'list_artifacts')
  @mock.patch.object(http_client_util, 'SendRequestToServer')
  def testGetDetailedFailureLog(self, mock_http_client, mock_list_artifacts):
    test_type = ResultDBTestType.GTEST
    log = {
        "name": "test_result_name",
        "summary_html": "Please see stack_trace artifact"
    }
    mock_list_artifacts.return_value = [
        artifact_pb2.Artifact(
            artifact_id="some_artifact", fetch_url="https://abc.xyz/some_url"),
        artifact_pb2.Artifact(
            artifact_id="stack_trace",
            fetch_url="https://abc.xyz/some_other_url"),
    ]
    mock_http_client.return_value = ("content", None)
    self.assertEqual(
        ResultDBTestResults.get_detailed_failure_log(test_type, log), "content")

  def TestDoesTestExist(self):
    test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    self.assertFalse(test_results.DoesTestExist('some_name'))
    self.assertTrue(test_results.DoesTestExist('SharedImageTest.Basic'))

  def TestIsTestResultUseful(self):
    test_results = ResultDBTestResults(_SAMPLE_TEST_RESULTS)
    self.assertTrue(test_results.IsTestResultUseful())
    test_results = ResultDBTestResults([])
    self.assertFalse(test_results.IsTestResultUseful())
