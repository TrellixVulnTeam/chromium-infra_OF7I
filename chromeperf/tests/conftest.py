# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import urllib
import logging

import pytest

from google.api_core import client_options
from google.auth import credentials
from google.cloud import datastore

from . import test_utils  # pylint: disable=relative-beyond-top-level
from chromeperf.pinpoint.models import repository
from chromeperf.pinpoint.models import change


@pytest.fixture
def datastore_client(request, datastore_emulator):
    """A datastore.Client with a psuedorandom suffix appended to its project.

    The suffix is derived from the test name.  Use this to get some isolation
    between tests using datastore, even when running tests in parallel.
    """
    emulator_host = datastore_emulator.get('DATASTORE_EMULATOR_HOST')
    # Project IDs are pretty constrained (6-30 chars, only lowercase, digits and
    # hyphen), so append a hex string of the hash of the test name to get a
    # sufficiently unique name that is still valid.
    return datastore.Client(
        project='chromeperf-' + hex(hash(request.node.name)),
        credentials=credentials.AnonymousCredentials(),
        client_options=client_options.ClientOptions(
            api_endpoint=f'http://{emulator_host}'),
    )


@pytest.fixture(autouse=True)
def pinpoint_seeded_data(datastore_client):
    # Add some test repositories.
    repository.add_repository(
        datastore_client,
        'catapult',
        'https://chromium.googlesource.com/catapult',
    )
    repository.add_repository(
        datastore_client,
        'chromium',
        test_utils.CHROMIUM_URL,
    )


@pytest.fixture
def request_json(mocker):
    return mocker.patch('chromeperf.services.request.request_json')


@pytest.fixture
def service_request(mocker):
    return mocker.patch('chromeperf.services.request.request')


@pytest.fixture(scope='session')
def datastore_emulator(worker_id):
    # TODO(fancl): Repick the port if it's occupied
    port = 8081 + hash(worker_id) % 6000
    with test_utils.with_emulator('datastore', port) as envs:
        yield envs


# What follows here are bisection-related fixtures, also useful for testing
# individual evaluators.
@pytest.fixture
def start_change():
    return change.reconstitute_change(
        {'commits': [{
            'repository': 'chromium',
            'git_hash': 'commit_0'
        }]})


@pytest.fixture
def end_change():
    return change.reconstitute_change(
        {'commits': [{
            'repository': 'chromium',
            'git_hash': 'commit_5'
        }]})
