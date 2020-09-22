# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Tests for the componentexport module."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import mock
import unittest
import webapp2

import settings
from features import componentexport
from framework import urls


class ComponentTrainingDataExportTest(unittest.TestCase):

  def test_handler_definition(self):
    instance = componentexport.ComponentTrainingDataExport()
    self.assertIsInstance(instance, webapp2.RequestHandler)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def test_enqueues_task(self, get_client_mock):
    componentexport.ComponentTrainingDataExport().get()

    queue = 'componentexport'
    task = {
        'app_engine_http_request':
            {
                'http_method': 'GET',
                'relative_uri': urls.COMPONENT_DATA_EXPORT_TASK
            }
    }

    get_client_mock().queue_path.assert_called_with(
        settings.app_id, settings.CLOUD_TASKS_REGION, queue)
    get_client_mock().create_task.assert_called_once()
    ((_parent, called_task), _kwargs) = get_client_mock().create_task.call_args
    self.assertEqual(called_task, task)
