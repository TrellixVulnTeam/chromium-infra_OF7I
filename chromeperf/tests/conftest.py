# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import urllib
import pytest

from google.cloud import datastore


# @pytest.fixture(autouse=True, scope='session')
# def datastoreEmulatorReset():
#   yield
#   url = f'http://{os.getenv("DATASTORE_EMULATOR_HOST")}/reset'
#   urllib.request.urlopen(urllib.request.Request(url, method="POST"))

@pytest.fixture
def datastore_client(request):
  """A datastore.Client with a psuedorandom suffix appended to its project.

  The suffix is derived from the test name.  Use this to get some isolation
  between tests using datastore, even when running tests in parallel.
  """
  # Project IDs are pretty constrained (6-30 chars, only lowercase, digits and
  # hyphen), so append a hex string of the hash of the test name to get a
  # sufficiently unique name that is still valid.
  return datastore.Client(project='chromeperf-' + hex(hash(request.node.name)))

@pytest.fixture
def request_json(mocker):
  return mocker.patch('chromeperf.services.request.request_json')


@pytest.fixture
def service_request(mocker):
  return mocker.patch('chromeperf.services.request.request')
