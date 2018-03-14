# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import mock

from dto.test_location import TestLocation
from infra_api_clients.swarming import swarming_util
from services import gtest
from services import isolate
from services import swarmed_test_util
from services import test_results
from waterfall.test import wf_testcase


class SwarmedTestUtilTest(wf_testcase.WaterfallTestCase):

  @mock.patch.object(
      swarmed_test_util, 'GetTestResultForSwarmingTask', return_value={})
  def testGetTestLocationNoTestLocations(self, _):
    self.assertIsNone(swarmed_test_util.GetTestLocation('task', 'test'))

  @mock.patch.object(gtest, 'IsTestResultsInExpectedFormat', return_value=True)
  @mock.patch.object(swarmed_test_util, 'GetTestResultForSwarmingTask')
  def testGetTestLocation(self, mock_get_isolated_output, _):
    test_name = 'test'
    expected_test_location = {
        'line': 123,
        'file': '/path/to/test_file.cc',
    }
    mock_get_isolated_output.return_value = {
        'test_locations': {
            test_name: expected_test_location,
        }
    }

    self.assertEqual(
        TestLocation.FromSerializable(expected_test_location),
        swarmed_test_util.GetTestLocation('task', test_name))

  @mock.patch.object(
      isolate,
      'DownloadFileFromIsolatedServer',
      return_value=(json.dumps({
          'all_tests': ['test1']
      }), None))
  def testGetOutputJsonByOutputsRef(self, _):
    outputs_ref = {
        'isolatedserver': 'isolated_server',
        'namespace': 'default-gzip',
        'isolated': 'shard1_isolated'
    }

    result, error = swarmed_test_util.GetOutputJsonByOutputsRef(
        outputs_ref, None)

    self.assertEqual({'all_tests': ['test1']}, result)
    self.assertIsNone(error)

  @mock.patch.object(
      swarming_util,
      'GetSwarmingTaskResultById',
      return_value=({
          'outputs_ref': 'ref'
      }, None))
  @mock.patch.object(
      swarmed_test_util,
      'GetOutputJsonByOutputsRef',
      return_value=(None, 'error'))
  def testGetTestResultForSwarmingTaskIsolatedError(self, *_):
    self.assertIsNone(
        swarmed_test_util.GetTestResultForSwarmingTask(None, None))

  @mock.patch.object(
      swarming_util,
      'GetSwarmingTaskResultById',
      return_value=({
          'a': []
      }, None))
  def testGetTestResultForSwarmingTaskNoOutputRef(self, _):
    self.assertIsNone(
        swarmed_test_util.GetTestResultForSwarmingTask(None, None))

  @mock.patch.object(
      swarming_util, 'GetSwarmingTaskResultById', return_value=(None, 'error'))
  def testGetTestResultForSwarmingTaskDataError(self, _):
    self.assertIsNone(
        swarmed_test_util.GetTestResultForSwarmingTask(None, None))

  @mock.patch.object(
      swarmed_test_util,
      'GetOutputJsonByOutputsRef',
      return_value=('content', None))
  @mock.patch.object(
      swarming_util,
      'GetSwarmingTaskResultById',
      return_value=({
          'outputs_ref': 'ref'
      }, None))
  def testGetTestResultForSwarmingTask(self, mock_fn, _):
    task_id = '2944afa502297110'
    result = swarmed_test_util.GetTestResultForSwarmingTask(task_id, None)

    self.assertEqual('content', result)
    mock_fn.assert_called_once_with('chromium-swarm.appspot.com', task_id, None)

  @mock.patch.object(
      swarmed_test_util,
      'GetTestResultForSwarmingTask',
      return_value='test_result_log')
  @mock.patch.object(test_results, 'IsTestEnabled', return_value=True)
  def testIsTestEnabled(self, *_):
    self.assertTrue(swarmed_test_util.IsTestEnabled('test', '123'))

  def testRetrieveShardedTestResultsFromIsolatedServerNoLog(self):
    self.assertEqual(
        [],
        swarmed_test_util.RetrieveShardedTestResultsFromIsolatedServer([],
                                                                       None))

  @mock.patch.object(gtest, 'IsTestResultsInExpectedFormat', return_value=True)
  @mock.patch.object(isolate, 'DownloadFileFromIsolatedServer')
  def testRetrieveShardedTestResultsFromIsolatedServer(self, mock_data, _):
    isolated_data = [{
        'digest': 'shard1_isolated',
        'namespace': 'default-gzip',
        'isolatedserver': 'isolated_server'
    }, {
        'digest': 'shard2_isolated',
        'namespace': 'default-gzip',
        'isolatedserver': 'isolated_server'
    }]

    mock_data.side_effect = [(json.dumps({
        'all_tests': ['test1', 'test2'],
        'per_iteration_data': [{
            'test1': [{
                'output_snippet': '[ RUN ] test1.\\r\\n',
                'output_snippet_base64': 'WyBSVU4gICAgICBdIEFjY291bnRUcm',
                'status': 'SUCCESS'
            }]
        }]
    }), 200), (json.dumps({
        'all_tests': ['test1', 'test2'],
        'per_iteration_data': [{
            'test2': [{
                'output_snippet': '[ RUN ] test2.\\r\\n',
                'output_snippet_base64': 'WyBSVU4gICAgICBdIEFjY291bnRUcm',
                'status': 'SUCCESS'
            }]
        }]
    }), 200)]
    result = swarmed_test_util.RetrieveShardedTestResultsFromIsolatedServer(
        isolated_data, None)
    expected_result = {
        'all_tests': ['test1', 'test2'],
        'per_iteration_data': [{
            'test1': [{
                'output_snippet': '[ RUN ] test1.\\r\\n',
                'output_snippet_base64': 'WyBSVU4gICAgICBdIEFjY291bnRUcm',
                'status': 'SUCCESS'
            }],
            'test2': [{
                'output_snippet': '[ RUN ] test2.\\r\\n',
                'output_snippet_base64': 'WyBSVU4gICAgICBdIEFjY291bnRUcm',
                'status': 'SUCCESS'
            }]
        }]
    }

    self.assertEqual(expected_result, result)

  @mock.patch.object(gtest, 'IsTestResultsInExpectedFormat', return_value=True)
  @mock.patch.object(isolate, 'DownloadFileFromIsolatedServer')
  def testRetrieveShardedTestResultsFromIsolatedServerOneShard(
      self, mock_data, _):
    isolated_data = [{
        'digest': 'shard1_isolated',
        'namespace': 'default-gzip',
        'isolatedserver': 'isolated_server'
    }]
    data_json = {'all_tests': ['test'], 'per_iteration_data': []}
    data_str = json.dumps(data_json)
    mock_data.return_value = (data_str, 200)

    result = swarmed_test_util.RetrieveShardedTestResultsFromIsolatedServer(
        isolated_data, None)

    self.assertEqual(data_json, result)

  @mock.patch.object(gtest, 'IsTestResultsInExpectedFormat', return_value=True)
  @mock.patch.object(isolate, 'DownloadFileFromIsolatedServer')
  def testRetrieveShardedTestResultsFromIsolatedServerFailed(
      self, mock_data, _):
    isolated_data = [{
        'digest': 'shard1_isolated',
        'namespace': 'default-gzip',
        'isolatedserver': 'isolated_server'
    }]
    mock_data.return_value = (None, 404)

    result = swarmed_test_util.RetrieveShardedTestResultsFromIsolatedServer(
        isolated_data, None)

    self.assertIsNone(result)
