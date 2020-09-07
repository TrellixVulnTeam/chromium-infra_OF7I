# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest

from deepdiff import DeepDiff

from chromeperf.services import swarming


@pytest.fixture(autouse=True)
def setup_default_response(request_json):
  request_json.return_value = {'content': {}}

def _assert_correct_response(content):
  assert content == {'content': {}}


def _assert_request_made_once(request_json, path, *args, **kwargs):
  request_json.assert_called_once_with(
      'https://server/_ah/api/swarming/v1/' + path, *args, **kwargs)


def test_Swarming_Bot_Get(request_json):
  response = swarming.Swarming('https://server').Bot('bot_id').Get()
  _assert_correct_response(response)
  _assert_request_made_once(request_json, 'bot/bot_id/get')


def test_Swarming_Bot_Tasks(request_json):
  response = swarming.Swarming('https://server').Bot('bot_id').Tasks()
  _assert_correct_response(response)
  _assert_request_made_once(request_json, 'bot/bot_id/tasks')


def test_Swarming_Bots_List(request_json):
  response = swarming.Swarming('https://server').Bots().List(
      'CkMSPWoQ', {
          'pool': 'Chrome-perf',
          'a': 'b'
      }, False, 1, True)
  _assert_correct_response(response)

  path = ('bots/list')
  _assert_request_made_once(
      request_json,
      path,
      cursor='CkMSPWoQ',
      dimensions=('a:b', 'pool:Chrome-perf'),
      is_dead=False,
      limit=1,
      quarantined=True)


def test_Swarming_Task_Cancel(request_json):
  response = swarming.Swarming('https://server').Task('task_id').Cancel()
  _assert_correct_response(response)
  _assert_request_made_once(request_json, 'task/task_id/cancel', method='POST')


def test_Swarming_Task_Request(request_json):
  response = swarming.Swarming('https://server').Task('task_id').Request()
  _assert_correct_response(response)
  _assert_request_made_once(request_json, 'task/task_id/request')


def test_Swarming_Task_Result(request_json):
  response = swarming.Swarming('https://server').Task('task_id').Result()
  _assert_correct_response(response)
  _assert_request_made_once(request_json, 'task/task_id/result')


def test_Swarming_Task_ResultWithPerformanceStats(request_json):
  response = swarming.Swarming('https://server').Task('task_id').Result(True)
  _assert_correct_response(response)
  _assert_request_made_once(
      request_json, 'task/task_id/result', include_performance_stats=True)


def test_Swarming_Task_Stdout(request_json):
  response = swarming.Swarming('https://server').Task('task_id').Stdout()
  _assert_correct_response(response)
  _assert_request_made_once(request_json, 'task/task_id/stdout')


def test_Swarming_Tasks_New(request_json):
  body = {
      'name': 'name',
      'user': 'user',
      'priority': '100',
      'expiration_secs': '600',
      'properties': {
          'inputs_ref': {
              'isolated': 'isolated_hash',
          },
          'extra_args': ['--output-format=histograms'],
          'dimensions': [
              {
                  'key': 'id',
                  'value': 'bot_id'
              },
              {
                  'key': 'pool',
                  'value': 'Chrome-perf'
              },
          ],
          'execution_timeout_secs': '3600',
          'io_timeout_secs': '3600',
      },
      'tags': [
          'id:bot_id',
          'pool:Chrome-perf',
      ],
  }

  response = swarming.Swarming('https://server').Tasks().New(body)
  _assert_correct_response(response)
  _assert_request_made_once(request_json, 'tasks/new', method='POST', body=body)
