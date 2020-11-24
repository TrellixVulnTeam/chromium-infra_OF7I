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


def test_Update_Test_Runner_Job(test_client, mocker):
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

    ret = test_client.patch(
        '/debug/jobs/test_job_123',
        json={
            'evaluator': 'run_test',
            'event': {
                'type': 'initiate',
                'payload_type': 'none',
                'payload': {},
            }
        },
    )
    assert ret.status_code == 200
    res = {
        'find_isolate_chromium@aaaaaaa': {
            'payload': {
                '@type':
                'type.googleapis.com/chromeperf.pinpoint.FindIsolateTaskPayload',
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
            'state': 'pending'
        }
    }
    res.update({
        f'run_test_chromium@aaaaaaa_{i}': {
            'payload': {
                '@type':
                'type.googleapis.com/chromeperf.pinpoint.TestRunnerPayload',
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
                    'executionTimeoutSecs':
                    '',
                    'expirationSecs':
                    '',
                    'extraArgs': [],
                    'ioTimeoutSecs':
                    '',
                    'swarmingServer':
                    'some_server'
                }
            },
            'state': 'pending'
        }
        for i in range(10)
    })
    assert ret.get_json() == res


def test_Post_Culprit_Finder_Job(test_client, mocker):
    ret = test_client.post(
        '/debug/jobs/test_job_123',
        json={
            'type': 'culprit_finder',
            'options': {
                'build_options': {
                    'builder': 'Some Builder',
                    'target': 'telemetry_perf_tests',
                    'bucket': 'luci.bucket',
                    'change': None,
                },
                'test_options': {
                    'swarming_server': 'some_server',
                    'dimensions': [{
                        'key': 'pool',
                        'value': 'Chrome-perf-pinpoint'
                    }],
                    'extra_args': [],
                    'build_options': None,
                    'attempts': None,
                },
                'read_options': {
                    'benchmark': 'some.benchmark',
                    'histogram_options': {
                        'grouping_label': 'some_group',
                        'story': 'some story',
                        'statistic': 'some stat',
                        'histogram_name': 'some chart',
                    },
                    'mode': 'histogram_sets',
                    'results_filename': 'perf_results.json',
                    'test_options': None,
                    'graph_json_options': None,
                },
                'analysis_options': {
                    'comparison_magnitude': 1.0,
                    'min_attempts': 10,
                    'max_attempts': 60,
                },
                'start_change': {
                    'commits': [{
                        'git_hash': 'aaaaaaa',
                        'repository': {
                            'name': 'chromium',
                        },
                    }]
                },
                'end_change': {
                    'commits': [{
                        'git_hash': 'fffffff',
                        'repository': {
                            'name': 'chromium',
                        },
                    }]
                },
            },
        },
    )
    assert ret.status_code == 200

    ret = test_client.get('/debug/jobs/test_job_123')
    assert ret.status_code == 200
    key_fn = lambda d: d.get('key')
    assert sorted(ret.get_json(), key=key_fn) == sorted([{
        'created': mocker.ANY,
        'dependencies': [f'read_value_chromium@{git_hash}_{i}'
                         for git_hash in ('aaaaaaa', 'fffffff')
                         for i in range(10)],
        'key': 'performance_bisection',
        'payload': {
            'errors': [],
            'input': {
                'analysisOptions': {
                    'comparisonMagnitude': 1.0,
                    'maxAttempts': 60,
                    'minAttempts': 10,
                },
                'buildOptionTemplate': mocker.ANY,
                'endChange': mocker.ANY,
                'readOptionTemplate': mocker.ANY,
                'startChange': mocker.ANY,
                'testOptionTemplate': mocker.ANY,
            },
        },
        'status': 'pending',
        'task_type': 'find_culprit',
    }] + [{
        'created': mocker.ANY,
        'dependencies': [],
        'key': f'find_isolate_chromium@{git_hash}',
        'payload': {
            'bucket': 'luci.bucket',
            'builder': 'Some Builder',
            'change': {
                'commits': [{
                    'gitHash': git_hash,
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
    } for git_hash in ('aaaaaaa', 'fffffff')] + [{
        'created': mocker.ANY,
        'dependencies': [f'find_isolate_chromium@{git_hash}'],
        'key': f'run_test_chromium@{git_hash}_{i}',
        'payload': {
            'errors': [],
            'index': i,
            'input': {
                'change': {
                    'commits': [{
                        'gitHash': git_hash,
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
    } for i in range(10) for git_hash in ('aaaaaaa', 'fffffff')] + [{
        'created': mocker.ANY,
        'dependencies': [f'run_test_chromium@{git_hash}_{i}'],
        'key': f'read_value_chromium@{git_hash}_{i}',
        'payload': {
            'errors': [],
            'index': i,
            'input': {
                'benchmark': 'some.benchmark',
                'change': {
                    'commits': [{
                        'gitHash': git_hash,
                        'repository': 'chromium'
                    }]
                },
                'histogramOptions': {
                    'groupingLabel': 'some_group',
                    'histogramName': 'some chart',
                    'statistic': 'some stat',
                    'story': 'some story',
                },
                'mode': 'histogram_sets',
                'resultsFilename': 'some.benchmark/perf_results.json',
            },
            'tries': 0,
        },
        'status': 'pending',
        'task_type': 'read_value',
    } for i in range(10) for git_hash in ('aaaaaaa', 'fffffff')],
    key=key_fn)
