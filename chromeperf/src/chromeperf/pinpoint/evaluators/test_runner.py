# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses
import itertools
import json
import logging
import typing

from google.cloud import datastore
from google.protobuf import any_pb2

from chromeperf.engine import combinators
from chromeperf.engine import evaluator
from chromeperf.engine import predicates
from chromeperf.pinpoint import change_pb2
from chromeperf.pinpoint import find_isolate_task_payload_pb2
from chromeperf.pinpoint import test_runner_payload_pb2
from chromeperf.pinpoint.actions import tests
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.evaluators import isolate_finder
from chromeperf.pinpoint.models import task as task_module

# FIXME: Make this an environment variable, or an input.
_TESTER_SERVICE_ACCOUNT = (
    'chrome-tester@chops-service-accounts.iam.gserviceaccount.com')


def _swarming_tags_from_job(job) -> dict:
    return {
        'pinpoint_job_id': job.job_id,
        'url': job.url,
        'comparison_mode': job.comparison_mode,
        'pinpoint_task_kind': 'test',
        'pinpoint_user': job.user,
    }


# Everything after this point aims to define an evaluator for the 'run_test'
# tasks.
@dataclasses.dataclass
class InitiateEvaluator(task_module.PayloadUnpackingMixin):
    job: typing.Any
    datastore_client: datastore.Client

    def __call__(self, task, event, context):
        # Outline:
        #   - Check dependencies to see if they're 'completed', looking for:
        #     - Isolate server
        #     - Isolate hash
        dep_map = {}
        for dep in task.dependencies:
            dep_task_context = context.get(dep)
            if not dep_task_context:
                raise task_module.FatalError(
                    f'Missing dependency "{dep}" for task "{task.id}"')
            dep_map[dep] = dep_task_context

        if not dep_map:
            logging.error(
                'No dependencies for "run_test" task, unlikely to proceed; task = %s',
                task)
            raise task_module.FatalError(
                f'No dependencies for task {task.id}, this task '
                f'is unlikely to complete.')

        dep_context = None
        if len(dep_map) > 1:
            # TODO(dberris): Figure out whether it's a valid use-case to have
            # multiple isolate inputs to Swarming.
            logging.error(('Found multiple dependencies for run_test; '
                           'picking a random input; task = %s'), task)
        _, dep_context = dep_map.popitem()

        if dep_context.state == 'failed':
            return [
                tests.MarkTestFailedAction(
                    datastore_client=self.datastore_client,
                    job=self.job,
                    task=task,
                    reason='BuildIsolateNotFound',
                    message=('The build task this depends on failed, '
                             'so we cannot proceed to running the tests.'))
            ]
        elif dep_context.state == 'completed':
            task_payload = self.unpack(
                test_runner_payload_pb2.TestRunnerPayload, task.payload)
            dep_payload = self.unpack(
                find_isolate_task_payload_pb2.FindIsolateTaskPayload,
                dep_context.payload,
            )
            for tag in ('%s:%s' % (k, v)
                        for k, v in _swarming_tags_from_job(self.job).items()):
                task_payload.output.swarming_request.tags.append(tag)
            task_payload.output.swarming_request.pubsub_auth_token = 'UNUSED'
            task_payload.output.swarming_request.pubsub_topic = (
                'projects/chromeperf/topics/pinpoint-swarming-updates')
            task_payload.output.swarming_request.pubsub_userdata = json.dumps({
                'job_id':
                self.job.job_id,
                'task': {
                    'type': 'run_test',
                    'id': task.id,
                },
            })
            task_payload.output.swarming_request.name = 'Pinpoint job'
            task_payload.output.swarming_request.user = 'Pinpoint'
            task_payload.output.swarming_request.priority = '100'
            task_payload.output.swarming_request.task_slices.append(
                test_runner_payload_pb2.TestRunnerPayload.Output.
                SwarmingRequest.TaskSlice(
                    properties=test_runner_payload_pb2.TestRunnerPayload.
                    Output.SwarmingRequest.TaskSlice.Properties(
                        inputs_ref=test_runner_payload_pb2.TestRunnerPayload.
                        Output.SwarmingRequest.TaskSlice.Properties.InputsRef(
                            isolate_server=dep_payload.isolate_server,
                            isolate_hash=dep_payload.isolate_hash,
                        ),
                        extra_args=task_payload.input.extra_args,
                        dimensions=task_payload.input.dimensions,
                        execution_timeout_secs=(
                            task_payload.input.execution_timeout_secs
                            or '2700'),
                        io_timeout_secs=(task_payload.input.io_timeout_secs
                                         or '2700'),
                    ),
                    expiration_secs=(task_payload.input.expiration_secs
                                     or '86400'),
                ))
            task.payload.Pack(task_payload)
            return [
                tests.ScheduleTestAction(
                    job=self.job,
                    task=task,
                    datastore_client=self.datastore_client,
                )
            ]
        elif dep_context.state in {'pending', 'ongoing'}:
            # Skip pending/ongoing dependencies.
            return

        raise task_module.FatalError(
            f'Unhandled event ({event}) in {type(self).__name__}; '
            f'unlikely to finish!')


@dataclasses.dataclass
class UpdateEvaluator(task_module.PayloadUnpackingMixin):
    job: typing.Any
    datastore_client: datastore.Client

    def __call__(self, task, event, context):
        if not event.target_task or event.target_task != task.id:
            return None

        task_payload = self.unpack(
            test_runner_payload_pb2.TestRunnerPayload,
            task.payload,
        )

        # Check that the task has the required information to poll Swarming. In
        # this handler we're going to look for the 'swarming_task_id' key in
        # the payload.
        # TODO(dberris): Move this out, when we incorporate validation
        # properly.
        if not task_payload.output.swarming_response.task_id:
            return [
                tests.MarkTestFailedAction(
                    job=self.job,
                    task=task,
                    datastore_client=self.datastore_client,
                )
            ]

        return [
            tests.PollSwarmingTaskAction(
                job=self.job,
                task=task,
                datastore_client=self.datastore_client)
        ]


class Evaluator(combinators.SequenceEvaluator):
    def __init__(self, job, datastore_client: datastore.Client):
        super(Evaluator, self).__init__(evaluators=(
            combinators.FilteringEvaluator(
                predicate=predicates.All(predicates.TaskTypeEq('run_test'), ),
                delegate=combinators.DispatchByEventTypeEvaluator({
                    'initiate':
                    combinators.FilteringEvaluator(predicate=predicates.Not(
                        predicates.TaskStateIn(
                            {'ongoing', 'failed', 'completed'})),
                                                   delegate=InitiateEvaluator(
                                                       job, datastore_client)),
                    # For updates, we want to ensure that the initiate evaluator
                    # has a chance to run on 'pending' tasks.
                    'update':
                    combinators.SequenceEvaluator([
                        combinators.FilteringEvaluator(
                            predicate=predicates.Not(
                                predicates.TaskStateIn(
                                    {'ongoing', 'failed', 'completed'})),
                            delegate=InitiateEvaluator(job, datastore_client)),
                        combinators.FilteringEvaluator(
                            predicate=predicates.TaskStateIn({'ongoing'}),
                            delegate=UpdateEvaluator(job, datastore_client)),
                    ])
                })),
            combinators.TaskPayloadLiftingEvaluator(),
        ))


def TestSerializer(task, _, accumulator):
    results = accumulator.setdefault(task.id, {})
    results.update({
        'completed': (task.status in {
            'completed',
            'failed',
            'cancelled',
        }),
        'exception':
        (','.join(e.get('reason') for e in task.payload.get(
            'errors',
            [],
        )) or None),
        'details': [],
    })

    swarming_task_result = task.payload.get('swarming_task_result', {})
    swarming_server = task.payload.get('swarming_server')
    bot_id = swarming_task_result.get('bot_id')
    if bot_id:
        results['details'].append({
            'key': 'bot',
            'value': bot_id,
            'url': swarming_server + '/bot?id=' + bot_id
        })
    task_id = task.payload.get('swarming_task_id')
    if task_id:
        results['details'].append({
            'key': ('task'),
            'value': (task_id),
            'url': (swarming_server + '/task?id=' + task_id),
        })
    isolate_hash = task.payload.get('isolate_hash')
    isolate_server = task.payload.get('isolate_server')
    if isolate_hash and isolate_server:
        results['details'].append({
            'key': ('isolate'),
            'value': (isolate_hash),
            'url': ('%s/browse?digest=%s' % (
                isolate_server,
                isolate_hash,
            ))
        })


class Serializer(combinators.FilteringEvaluator):
    def __init__(self):
        super(Serializer,
              self).__init__(predicate=predicates.TaskTypeEq('run_test'),
                             delegate=TestSerializer)


def task_id(change, attempt):
    return 'run_test_%s_%s' % (change, attempt)


@dataclasses.dataclass
class TaskOptions:
    build_options: isolate_finder.TaskOptions
    swarming_server: str
    dimensions: list
    extra_args: list
    attempts: int


def create_graph(options: TaskOptions) -> evaluator.TaskGraph:
    if not isinstance(options, TaskOptions):
        raise ValueError('options is not an instance of run_test.TaskOptions')
    subgraph = isolate_finder.create_graph(options.build_options)
    find_isolate_tasks = [
        task for task in subgraph.vertices
        if task.vertex_type == 'find_isolate'
    ]
    assert len(find_isolate_tasks) == 1
    find_isolate_task = find_isolate_tasks[0]

    def _create_payload(index):
        test_runner_payload = test_runner_payload_pb2.TestRunnerPayload(
            input=test_runner_payload_pb2.TestRunnerPayload.Input(
                swarming_server=options.swarming_server,
                extra_args=options.extra_args,
                change=change_pb2.Change(
                    commits=[
                        change_pb2.Commit(
                            repository=c.repository.name,
                            git_hash=c.git_hash,
                        ) for c in options.build_options.change.commits
                    ],
                    patch=(change_pb2.GerritPatch(
                        server=options.build_options.change.patch.server,
                        change=options.build_options.change.patch.change,
                        revision=options.build_options.change.patch.revision)
                           if options.build_options.change.patch else None),
                ),
                dimensions=options.dimensions,
            ),
            index=index)
        encoded_payload = any_pb2.Any()
        encoded_payload.Pack(test_runner_payload)
        return encoded_payload

    subgraph.vertices.extend([
        evaluator.TaskVertex(
            id=task_id(
                isolate_finder.change_id(options.build_options.change),
                attempt,
            ),
            vertex_type='run_test',
            payload=_create_payload(attempt),
        ) for attempt in range(options.attempts)
    ])
    subgraph.edges.extend([
        evaluator.Dependency(from_=task.id, to=find_isolate_task.id)
        for task in subgraph.vertices if task.vertex_type == 'run_test'
    ])
    return subgraph
