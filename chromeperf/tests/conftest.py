# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import urllib

import pytest


@pytest.fixture(autouse=True)
def datastoreEmulatorReset():
  url = f'http://{os.getenv("DATASTORE_EMULATOR_HOST")}/reset'
  urllib.request.urlopen(urllib.request.Request(url, method="POST"))


@pytest.fixture
def request_json(mocker):
  return mocker.patch('chromeperf.services.request.request_json')


@pytest.fixture
def service_request(mocker):
  return mocker.patch('chromeperf.services.request.request')
