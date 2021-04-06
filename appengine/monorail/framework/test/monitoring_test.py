# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

"""Unit tests for the monitoring module."""

import unittest
from framework import monitoring

COMMON_TEST_FIELDS = monitoring.GetCommonFields(200, 'monorail.v3.MethodName')


class MonitoringTest(unittest.TestCase):

  def testIncrementAPIRequestsCount(self):
    # Non-service account email gets hidden.
    monitoring.IncrementAPIRequestsCount(
        'v3', 'monorail-prod', client_email='client-email@chicken.com')
    self.assertEqual(
        1,
        monitoring.API_REQUESTS_COUNT.get(
            fields={
                'client_id': 'monorail-prod',
                'client_email': 'user@email.com',
                'version': 'v3'
            }))

    # None email address gets replaced by 'anonymous'.
    monitoring.IncrementAPIRequestsCount('v3', 'monorail-prod')
    self.assertEqual(
        1,
        monitoring.API_REQUESTS_COUNT.get(
            fields={
                'client_id': 'monorail-prod',
                'client_email': 'anonymous',
                'version': 'v3'
            }))

    # Service account email is not hidden
    monitoring.IncrementAPIRequestsCount(
        'endpoints',
        'monorail-prod',
        client_email='123456789@developer.gserviceaccount.com')
    self.assertEqual(
        1,
        monitoring.API_REQUESTS_COUNT.get(
            fields={
                'client_id': 'monorail-prod',
                'client_email': '123456789@developer.gserviceaccount.com',
                'version': 'endpoints'
            }))

  def testGetCommonFields(self):
    fields = monitoring.GetCommonFields(200, 'monorail.v3.TestName')
    self.assertEqual(
        {
            'status': 200,
            'name': 'monorail.v3.TestName',
            'is_robot': False
        }, fields)

  def testAddServerDurations(self):
    self.assertIsNone(
        monitoring.SERVER_DURATIONS.get(fields=COMMON_TEST_FIELDS))
    monitoring.AddServerDurations(500, COMMON_TEST_FIELDS)
    self.assertIsNotNone(
        monitoring.SERVER_DURATIONS.get(fields=COMMON_TEST_FIELDS))

  def testIncrementServerResponseStatusCount(self):
    monitoring.IncrementServerResponseStatusCount(COMMON_TEST_FIELDS)
    self.assertEqual(
        1, monitoring.SERVER_RESPONSE_STATUS.get(fields=COMMON_TEST_FIELDS))

  def testAddServerRequesteBytes(self):
    self.assertIsNone(
        monitoring.SERVER_REQUEST_BYTES.get(fields=COMMON_TEST_FIELDS))
    monitoring.AddServerRequesteBytes(1234, COMMON_TEST_FIELDS)
    self.assertIsNotNone(
        monitoring.SERVER_REQUEST_BYTES.get(fields=COMMON_TEST_FIELDS))

  def testAddServerResponseBytes(self):
    self.assertIsNone(
        monitoring.SERVER_RESPONSE_BYTES.get(fields=COMMON_TEST_FIELDS))
    monitoring.AddServerResponseBytes(9876, COMMON_TEST_FIELDS)
    self.assertIsNotNone(
        monitoring.SERVER_RESPONSE_BYTES.get(fields=COMMON_TEST_FIELDS))
