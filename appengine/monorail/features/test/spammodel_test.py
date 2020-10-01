# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Tests for the spammodel module."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import mock
import unittest
import webapp2

from features import spammodel
from framework import urls


class TrainingDataExportTest(unittest.TestCase):

  def test_handler_definition(self):
    instance = spammodel.TrainingDataExport()
    self.assertIsInstance(instance, webapp2.RequestHandler)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def test_enqueues_task(self, get_client_mock):
    spammodel.TrainingDataExport().get()
    task = {
        'app_engine_http_request':
            {
                'relative_uri': urls.SPAM_DATA_EXPORT_TASK + '.do',
                'body': '',
                'headers': {
                    'Content-type': 'application/x-www-form-urlencoded'
                }
            }
    }
    get_client_mock().create_task.assert_called_once()
    ((_parent, called_task), _kwargs) = get_client_mock().create_task.call_args
    self.assertEqual(called_task, task)
