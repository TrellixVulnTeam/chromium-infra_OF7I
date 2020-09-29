# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Tests for the cloud tasks helper module."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from google.api_core import exceptions

import mock
import unittest

from framework import cloud_tasks_helpers
import settings


class CloudTasksHelpersTest(unittest.TestCase):

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def test_create_task(self, get_client_mock):

    queue = 'somequeue'
    task = {
        'app_engine_http_request':
            {
                'http_method': 'GET',
                'relative_uri': '/some_url'
            }
    }
    cloud_tasks_helpers.create_task(task, queue=queue)

    get_client_mock().queue_path.assert_called_with(
        settings.app_id, settings.CLOUD_TASKS_REGION, queue)
    get_client_mock().create_task.assert_called_once()
    ((_parent, called_task), _kwargs) = get_client_mock().create_task.call_args
    self.assertEqual(called_task, task)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def test_create_task_raises(self, get_client_mock):
    task = {'app_engine_http_request': {}}

    get_client_mock().create_task.side_effect = exceptions.GoogleAPICallError(
        'oh no!')

    with self.assertRaises(exceptions.GoogleAPICallError):
      cloud_tasks_helpers.create_task(task)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def test_create_task_retries(self, get_client_mock):
    task = {'app_engine_http_request': {}}

    cloud_tasks_helpers.create_task(task)

    (_args, kwargs) = get_client_mock().create_task.call_args
    self.assertEqual(kwargs.get('retry'), cloud_tasks_helpers._DEFAULT_RETRY)

  def test_generate_simple_task(self):
    actual = cloud_tasks_helpers.generate_simple_task(
        '/alphabet/letters', {
            'a': 'a',
            'b': 'b'
        })
    expected = {
        'app_engine_http_request': {
            'relative_uri': '/alphabet/letters?a=a&b=b'
        }
    }
    self.assertEqual(actual, expected)

    actual = cloud_tasks_helpers.generate_simple_task('/alphabet/letters', {})
    expected = {
        'app_engine_http_request': {
            'relative_uri': '/alphabet/letters'
        }
    }
    self.assertEqual(actual, expected)
