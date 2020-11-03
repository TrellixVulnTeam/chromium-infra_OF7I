# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest

from chromeperf.pinpoint import service


@pytest.fixture
def test_client(datastore_client):
    app = service.create_app(datastore_client)
    with app.test_client() as c:
        yield c


def test_Post_Test_Runner_Job(test_client, mocker):
    ret = test_client.post(
        '/debug/jobs/test_job_123',
        json={
            'type': 'run_test',
            'options': {
                'build_options': {
                    'builder': 'Some Builder',
                    'target': 'telemetry_perf_tests',
                    'bucket': 'luci.bucket',
                    'change': {
                        'commits': [{
                            'repository': {
                                'name': 'chromium',
                            },
                            'git_hash': 'aaaaaaa',
                        }],
                    },
                },
                'swarming_server': 'some_server',
                'dimensions': [{
                    'key': 'pool',
                    'value': 'Chrome-perf-pinpoint'
                }],
                'extra_args': [],
                'attempts': 10,
            },
        },
    )
    assert ret.status_code == 200

    ret = test_client.get('/debug/jobs/test_job_123')
    assert ret.status_code == 200
    assert ret.get_json() == [{
        'created': mocker.ANY,
        'dependencies': [],
        'key': 'find_isolate_chromium@aaaaaaa',
        'payload': {
            'bucket': 'luci.bucket',
            'builder': 'Some Builder',
            'change': {
                'commits': [{
                    'gitHash': 'aaaaaaa',
                    'repository': 'chromium'
                }]
            },
            'errors': [],
            'isolateHash': '',
            'isolateServer': '',
            'target': 'telemetry_perf_tests',
            'tries': 0
        },
        'status': 'pending',
        'task_type': 'find_isolate'
    }] + [{
        'created': mocker.ANY,
        'dependencies': ['find_isolate_chromium@aaaaaaa'],
        'key': f'run_test_chromium@aaaaaaa_{i}',
        'payload': {
            'errors': [],
            'index': i,
            'input': {
                'change': {
                    'commits': [{
                        'gitHash': 'aaaaaaa',
                        'repository': 'chromium'
                    }]
                },
                'dimensions': [{
                    'key': 'pool',
                    'value': 'Chrome-perf-pinpoint'
                }],
                'executionTimeoutSecs': '',
                'expirationSecs': '',
                'extraArgs': [],
                'ioTimeoutSecs': '',
                'swarmingServer': 'some_server'
            }
        },
        'status': 'pending',
        'task_type': 'run_test'
    } for i in range(10)]
