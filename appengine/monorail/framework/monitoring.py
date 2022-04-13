# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

"""Monitoring ts_mon custom to monorail."""

import os
import sys
lib_path = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'lib')
sys.path.insert(0, lib_path)

import google.auth
import google.auth.transport.requests

from google.cloud import logging
from infra_libs import ts_mon
from framework import framework_helpers
import settings


def GetCommonFields(status, name, is_robot=False):
  # type: (int, str, bool=False) -> Dict[str, Union[int, str, bool]]
  return {
      'status': status,
      'name': name,
      'is_robot': is_robot,
  }


API_REQUESTS_COUNT = ts_mon.CounterMetric(
    'monorail/api_requests',
    'Number of requests to Monorail APIs',
    [ts_mon.StringField('client_id'),
     ts_mon.StringField('client_email'),
     ts_mon.StringField('version')])


def IncrementAPIRequestsCount(
    version, client_id, client_email=None, handler='none'):
  # type: (str, str, Optional[str], Optional[str]) -> None
  """Increment the request count in ts_mon."""
  if not client_email:
    client_email = 'anonymous'
  elif not framework_helpers.IsServiceAccount(client_email):
    # Avoid value explosion and protect PII info
    client_email = 'user@email.com'

  fields = {
      'client_id': client_id,
      'client_email': client_email,
      'version': version
  }
  API_REQUESTS_COUNT.increment_by(1, fields)

  if not settings.unit_test_mode:
    logging_client = logging.Client()
    logger = logging_client.logger("request_log")
    logger.log_struct(
        {
            'log_type': "IncrementAPIRequestsCount",
            'client_id': client_id,
            'client_email': client_email,
            'requests_count': str(API_REQUESTS_COUNT.get(fields)),
            'endpoint': handler
        })


# 90% of durations are in the range 11-1873ms.  Growth factor 10^0.06 puts that
# range into 37 buckets.  Max finite bucket value is 12 minutes.
DURATION_BUCKETER = ts_mon.GeometricBucketer(10**0.06)

# 90% of sizes are in the range 0.17-217014 bytes.  Growth factor 10^0.1 puts
# that range into 54 buckets.  Max finite bucket value is 6.3GB.
SIZE_BUCKETER = ts_mon.GeometricBucketer(10**0.1)

# TODO(https://crbug.com/monorail/9281): Differentiate internal/external calls.
SERVER_DURATIONS = ts_mon.CumulativeDistributionMetric(
    'monorail/server_durations',
    'Time elapsed between receiving a request and sending a'
    ' response (including parsing) in milliseconds.', [
        ts_mon.IntegerField('status'),
        ts_mon.StringField('name'),
        ts_mon.BooleanField('is_robot'),
    ],
    bucketer=DURATION_BUCKETER)


def AddServerDurations(elapsed_ms, fields):
  # type: (int, Dict[str, Union[int, bool]]) -> None
  SERVER_DURATIONS.add(elapsed_ms, fields=fields)


SERVER_RESPONSE_STATUS = ts_mon.CounterMetric(
    'monorail/server_response_status',
    'Number of responses sent by HTTP status code.', [
        ts_mon.IntegerField('status'),
        ts_mon.StringField('name'),
        ts_mon.BooleanField('is_robot'),
    ])


def IncrementServerResponseStatusCount(fields):
  # type: (Dict[str, Union[int, bool]]) -> None
  SERVER_RESPONSE_STATUS.increment(fields=fields)


SERVER_REQUEST_BYTES = ts_mon.CumulativeDistributionMetric(
    'monorail/server_request_bytes',
    'Bytes received per http request (body only).', [
        ts_mon.IntegerField('status'),
        ts_mon.StringField('name'),
        ts_mon.BooleanField('is_robot'),
    ],
    bucketer=SIZE_BUCKETER)


def AddServerRequesteBytes(request_length, fields):
  # type: (int, Dict[str, Union[int, bool]]) -> None
  SERVER_REQUEST_BYTES.add(request_length, fields=fields)


SERVER_RESPONSE_BYTES = ts_mon.CumulativeDistributionMetric(
    'monorail/server_response_bytes',
    'Bytes sent per http request (content only).', [
        ts_mon.IntegerField('status'),
        ts_mon.StringField('name'),
        ts_mon.BooleanField('is_robot'),
    ],
    bucketer=SIZE_BUCKETER)


def AddServerResponseBytes(response_length, fields):
  SERVER_RESPONSE_BYTES.add(response_length, fields=fields)
