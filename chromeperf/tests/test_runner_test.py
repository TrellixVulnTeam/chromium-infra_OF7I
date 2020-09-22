# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import functools
import uuid

import pytest

from google.protobuf import empty_pb2
from google.protobuf import json_format

from chromeperf.engine import combinators
from chromeperf.engine import evaluator
from chromeperf.engine import event as event_module
from chromeperf.engine import predicates
from chromeperf.pinpoint import test_runner_payload_pb2
from chromeperf.pinpoint import change_pb2
from chromeperf.pinpoint.evaluators import isolate_finder
from chromeperf.pinpoint.evaluators import test_runner
from chromeperf.pinpoint.models import change as change_module
from chromeperf.pinpoint.models import task as task_module

from . import test_utils
from . import bisection_test_util

DIMENSIONS = [
    test_runner_payload_pb2.TestRunnerPayload.Dimension(
        key='pool', value='Chrome-perf-pinpoint'),
    test_runner_payload_pb2.TestRunnerPayload.Dimension(key='key',
                                                        value='value'),
]


@pytest.fixture
def swarming_tasks_new(mocker):
    return mocker.patch('chromeperf.services.swarming.Tasks.New')


@pytest.fixture
def swarming_task_result(mocker):
    return mocker.patch('chromeperf.services.swarming.Task.Result')


@pytest.fixture
def swarming_task_stdout(mocker):
    return mocker.patch('chromeperf.services.swarming.Task.Stdout')


@pytest.fixture
def populated_job(datastore_client, request):
    job = test_utils.MockJob(datastore_client.key('Job', str(uuid.uuid4())))
    git_hash = request.node.get_closest_marker('git_hash')
    if not git_hash:
        git_hash = '7c7e90be'
    else:
        git_hash = git_hash.args[0]

    task_module.populate_task_graph(
        datastore_client, job,
        test_runner.create_graph(
            test_runner.TaskOptions(
                build_options=isolate_finder.TaskOptions(
                    builder='Some Builder',
                    target='telemetry_perf_tests',
                    bucket='luci.bucket',
                    change=change_module.reconstitute_change(
                        datastore_client,
                        {
                            'commits': [{
                                'repository': 'chromium',
                                'git_hash': 'aaaaaaa',
                            }]
                        },
                    )),
                swarming_server='some_server',
                dimensions=DIMENSIONS,
                extra_args=[],
                attempts=10,
            )))
    return job


def test_TestRunner_EvaluateToCompletion(
    datastore_client,
    swarming_task_result,
    swarming_tasks_new,
    populated_job,
):
    swarming_tasks_new.return_value = {'task_id': 'task id'}
    test_evaluator = combinators.SequenceEvaluator(evaluators=(
        combinators.FilteringEvaluator(
            predicate=predicates.TaskTypeEq('find_isolate'),
            delegate=combinators.SequenceEvaluator(evaluators=(
                bisection_test_util.FakeFoundIsolate(
                    datastore_client,
                    populated_job,
                ),
                combinators.TaskPayloadLiftingEvaluator(),
            ))),
        test_runner.Evaluator(populated_job, datastore_client),
    ))
    test_event = event_module.build_event(type='initiate',
                                          target_task=None,
                                          payload=empty_pb2.Empty())
    context = evaluator.evaluate_graph(
        test_event,
        test_evaluator,
        task_module.task_graph_loader(datastore_client, populated_job),
    )
    assert set(task.state
               for task in context.values()) == set(['ongoing', 'completed'])

    # Ensure that we've found all the 'test_runner' tasks.
    select_context = evaluator.evaluate_graph(
        event_module.select_event(),
        combinators.Selector(task_type='run_test'),
        task_module.task_graph_loader(datastore_client, populated_job))

    # Check that the output in the payload is populated as we expect.
    for task_id, task_context in select_context.items():
        task_payload = test_runner_payload_pb2.TestRunnerPayload()
        assert task_context.payload.Unpack(task_payload)
        assert task_id == f'run_test_chromium@aaaaaaa_{task_payload.index}'
        assert task_context.state == 'ongoing'
        # FIXME: Check more pertinent information here.
        assert json_format.MessageToDict(task_payload.output) != {}

    # Ensure that we've actually made the calls to the Swarming service.
    swarming_tasks_new.assert_called()
    assert swarming_tasks_new.call_count == 10

    # Then we propagate an event for each of the test_runner tasks in the graph.
    swarming_task_result.return_value = {
        'bot_id': 'bot id',
        'exit_code': 0,
        'failure': False,
        'outputs_ref': {
            'isolatedserver': 'output isolate server',
            'isolated': 'output isolate hash',
        },
        'state': 'COMPLETED',
    }
    for attempt in range(10):
        task_id = f'run_test_chromium@aaaaaaa_{attempt}'
        context = evaluator.evaluate_graph(
            event_module.build_event(payload=empty_pb2.Empty(),
                                     type='update',
                                     target_task=task_id), test_evaluator,
            task_module.task_graph_loader(datastore_client, populated_job))
        assert task_id in context
        assert context[task_id].state == 'completed'

    # Ensure that we've polled the status of each of the tasks, and that we've
    # marked the tasks completed.
    select_context = evaluator.evaluate_graph(
        event_module.select_event(),
        combinators.Selector(task_type='run_test'),
        task_module.task_graph_loader(datastore_client, populated_job))
    for attempt in range(10):
        task_id = f'run_test_chromium@aaaaaaa_{attempt}'
        task_context = select_context[task_id]
        task_payload = test_runner_payload_pb2.TestRunnerPayload()
        assert task_context.payload.Unpack(task_payload)
        assert task_id == f'run_test_chromium@aaaaaaa_{task_payload.index}'
        assert task_context.state == 'completed'
        # FIXME: Check more pertinent information here.
        assert json_format.MessageToDict(task_payload.output) != {}
        assert (task_payload.output.task_output.isolate_server ==
                'output isolate server')
        assert (task_payload.output.task_output.isolate_hash ==
                'output isolate hash')

    # Ensure that we've actually made the calls to the Swarming service.
    swarming_task_result.assert_called()
    assert swarming_task_result.call_count == 10


def test_TestRunner_EvaluateFailedDependency(
    datastore_client,
    swarming_task_result,
    swarming_tasks_new,
    populated_job,
):
    test_evaluator = combinators.SequenceEvaluator(evaluators=(
        combinators.FilteringEvaluator(
            predicate=predicates.TaskTypeEq('find_isolate'),
            delegate=combinators.SequenceEvaluator(evaluators=(
                bisection_test_util.FakeFindIsolateFailed(
                    datastore_client=datastore_client,
                    job=populated_job,
                ),
                combinators.TaskPayloadLiftingEvaluator(),
            ))),
        test_runner.Evaluator(populated_job, datastore_client),
    ))

    # When we initiate the test_runner tasks, we should immediately see the
    # tasks failing because the dependency has a hard failure status.
    context = evaluator.evaluate_graph(
        event_module.build_event(payload=empty_pb2.Empty(),
                                 type='initiate',
                                 target_task=None), test_evaluator,
        task_module.task_graph_loader(datastore_client, populated_job))
    for attempt in range(10):
        task_id = f'run_test_chromium@aaaaaaa_{attempt}'
        assert task_id in context
        task_context = context[task_id]
        assert task_context.state == 'failed'
        task_payload = test_runner_payload_pb2.TestRunnerPayload()
        assert task_context.payload.Unpack(task_payload)
        assert len(task_payload.errors) == 1


def test_TestRunner_EvaluatePendingDependency(datastore_client, populated_job):
    # Ensure that tasks stay pending in the event of an update.
    context = evaluator.evaluate_graph(
        event_module.build_event(
            payload=empty_pb2.Empty(),
            type='update',
            target_task=None,
        ),
        test_runner.Evaluator(
            datastore_client=datastore_client,
            job=populated_job,
        ),
        task_module.task_graph_loader(
            datastore_client=datastore_client,
            job=populated_job,
        ),
    )
    assert 'find_isolate_chromium@aaaaaaa' in context
    assert context['find_isolate_chromium@aaaaaaa'].state == 'pending'
    for attempt in range(10):
        task_id = f'run_test_chromium@aaaaaaa_{attempt}'
        assert task_id in context
        assert context[task_id].state == 'pending'


def test_TestRunner_EvaluateHandleFailures_Hard(
    datastore_client,
    swarming_task_stdout,
    swarming_task_result,
    swarming_tasks_new,
    populated_job,
):
    swarming_tasks_new.return_value = {'task_id': 'task id'}
    test_evaluator = combinators.SequenceEvaluator(evaluators=(
        combinators.FilteringEvaluator(
            predicate=predicates.TaskTypeEq('find_isolate'),
            delegate=combinators.SequenceEvaluator(evaluators=(
                bisection_test_util.FakeFoundIsolate(
                    datastore_client=datastore_client,
                    job=populated_job,
                ),
                combinators.TaskPayloadLiftingEvaluator(),
            ))),
        test_runner.Evaluator(
            datastore_client=datastore_client,
            job=populated_job,
        ),
    ))
    context = evaluator.evaluate_graph(
        event_module.build_event(payload=empty_pb2.Empty(),
                                 type='initiate',
                                 target_task=None), test_evaluator,
        task_module.task_graph_loader(datastore_client=datastore_client,
                                      job=populated_job))
    assert context != {}

    # We set it up so that when we poll the swarming task, that we're going to
    # get an error status. We're expecting that hard failures are detected.
    swarming_task_stdout.return_value = {
        'output':
        """Traceback (most recent call last):
    File "../../testing/scripts/run_performance_tests.py", line 282, in <module>
    sys.exit(main())
    File "../../testing/scripts/run_performance_tests.py", line 226, in main
    benchmarks = args.benchmark_names.split(',')
    AttributeError: 'Namespace' object has no attribute 'benchmark_names'"""
    }
    swarming_task_result.return_value = {
        'bot_id': 'bot id',
        'exit_code': 1,
        'failure': True,
        'outputs_ref': {
            'isolatedserver': 'output isolate server',
            'isolated': 'output isolate hash',
        },
        'state': 'COMPLETED',
    }
    for attempt in range(10):
        task_id = f'run_test_chromium@aaaaaaa_{attempt}'
        context = evaluator.evaluate_graph(
            event_module.build_event(payload=empty_pb2.Empty(),
                                     type='update',
                                     target_task=task_id),
            test_evaluator,
            task_module.task_graph_loader(
                datastore_client=datastore_client,
                job=populated_job,
            ),
        )
        assert context != {}
        assert task_id in context
        select_context = evaluator.evaluate_graph(
            event_module.build_event(payload=empty_pb2.Empty(),
                                     type='select',
                                     target_task=task_id),
            combinators.Selector(task_type='run_test'),
            task_module.task_graph_loader(
                datastore_client=datastore_client,
                job=populated_job,
            ))
        assert select_context != {}
        task_context = select_context[task_id]
        task_payload = test_runner_payload_pb2.TestRunnerPayload()
        task_context.payload.Unpack(task_payload)
        assert task_context.state == 'failed'
        assert len(task_payload.errors) == 1


def test_TestRunner_EvaluateHandleFailures_Expired(
    datastore_client,
    swarming_task_stdout,
    swarming_task_result,
    swarming_tasks_new,
    populated_job,
):
    swarming_tasks_new.return_value = {'task_id': 'task id'}
    test_evaluator = combinators.SequenceEvaluator(evaluators=(
        combinators.FilteringEvaluator(
            predicate=predicates.TaskTypeEq('find_isolate'),
            delegate=combinators.SequenceEvaluator(evaluators=(
                bisection_test_util.FakeFoundIsolate(
                    datastore_client=datastore_client, job=populated_job),
                combinators.TaskPayloadLiftingEvaluator(),
            ))),
        test_runner.Evaluator(datastore_client=datastore_client,
                              job=populated_job),
    ))
    context = evaluator.evaluate_graph(
        event_module.build_event(payload=empty_pb2.Empty(),
                                 type='initiate',
                                 target_task=None), test_evaluator,
        task_module.task_graph_loader(datastore_client=datastore_client,
                                      job=populated_job))
    assert context != {}
    swarming_task_result.return_value = {
        'state': 'EXPIRED',
    }
    for attempt in range(10):
        task_id = f'run_test_chromium@aaaaaaa_{attempt}'
        context = evaluator.evaluate_graph(
            event_module.build_event(
                payload=empty_pb2.Empty(),
                type='update',
                target_task=task_id,
            ),
            test_evaluator,
            task_module.task_graph_loader(datastore_client=datastore_client,
                                          job=populated_job),
        )
        assert context != {}
        select_context = evaluator.evaluate_graph(
            event_module.select_event(target_task=task_id),
            combinators.Selector(task_type='run_test'),
            task_module.task_graph_loader(datastore_client=datastore_client,
                                          job=populated_job),
        )
        assert select_context != {}
        assert task_id in select_context
        task_context = context[task_id]
        task_payload = test_runner_payload_pb2.TestRunnerPayload()
        assert task_context.payload.Unpack(task_payload)
        assert task_context.state == 'failed'
        assert len(task_payload.errors) == 1


@pytest.mark.skip('Deferring implementation pending design.')
def test_TestRunner_EvaluateHandleFailures_Retry():
    pass
