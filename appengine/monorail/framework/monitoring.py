# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

"""Monitoring metrics custom to monorail."""

from infra_libs import ts_mon
from framework import framework_helpers

API_REQUESTS_COUNT = ts_mon.CounterMetric(
    'monorail/api_requests',
    'Number of requests to Monorail APIs',
    [ts_mon.StringField('client_id'),
     ts_mon.StringField('client_email'),
     ts_mon.StringField('version')])

def IncrementAPIRequestsCount(version, client_id, client_email=None):
  # type: (str, str, Optional[str]) -> None
  """Increment the request count in ts_mon."""
  if not client_email:
    client_email = 'anonymous'
  elif not framework_helpers.IsServiceAccount(client_email):
    # Avoid value explosion and protect PII info
    client_email = 'user@email.com'

  fields = {'client_id': client_id,
            'client_email': client_email,
            'version': version}
  API_REQUESTS_COUNT.increment_by(1, fields)
