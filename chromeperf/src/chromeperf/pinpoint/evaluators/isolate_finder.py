# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Evaluators for finding isolates.

This module includes evaluators that follow the contract defined by
chromeperf.engine.evaluator.evaluate_graph.
"""

import logging
import dataclasses

from google.protobuf import any_pb2

from chromeperf.engine import combinators
from chromeperf.engine import evaluator
from chromeperf.engine import predicates
from chromeperf.engine import task_pb2
from chromeperf.pinpoint import change_pb2
from chromeperf.pinpoint import find_isolate_task_payload_pb2
from chromeperf.pinpoint.actions import builds
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.models import change as change_module
from chromeperf.pinpoint.models import commit as commit_module
from chromeperf.pinpoint.models import isolate as isolate_module
from chromeperf.pinpoint.models import task as task_module


class InitiateEvaluator(task_module.PayloadUnpackingMixin):
    def __init__(self, job, datastore_client):
        """Creates an InitiateEvaluator.

        Args:
        - job: a Pinpoint Job.
        """
        self._job = job
        self._datastore_client = datastore_client

    @updates.log_transition_failures
    def __call__(self, task, _, change):
        if task.state == 'ongoing':
            logging.warning(
                'Ignoring an initiate event on an ongoing task; task = %s',
                task.id)
            return None

        # Outline:
        #   - Check if we can find the isolate for this revision.
        #     - If found, update the payload of the task and update accumulator
        #       with result for this task.
        #     - If not found, schedule a build for this revision, update the
        #       task payload with the build details, wait for updates.
        payload = self.unpack(
            find_isolate_task_payload_pb2.FindIsolateTaskPayload, task.payload)
        try:
            change = change_module.Change.FromProto(self._datastore_client,
                                                    payload.change)
            logging.debug('Looking up isolate for change = %s', change)
            payload.isolate_server, payload.isolate_hash = isolate_module.get(
                payload.builder, change, payload.target,
                self._datastore_client)
            task.payload.Pack(payload)

            # At this point we've found an isolate from a previous build, so we mark
            # the task 'completed' and allow tasks that depend on the isolate to see
            # this information.
            @updates.log_transition_failures
            def _complete_with_cached_isolate(_):
                updates.update_task(self._datastore_client,
                                    self._job,
                                    task.id,
                                    new_state='completed',
                                    payload=task.payload)

            return [_complete_with_cached_isolate]
        except KeyError as e:
            logging.error('Failed to find isolate for task = %s;\nError: %s',
                          task.id, e)
            return [
                builds.ScheduleBuildBucketBuild(
                    self._job,
                    task,
                    change,
                    self._datastore_client,
                )
            ]
        return None


class UpdateEvaluator(task_module.PayloadUnpackingMixin):
    def __init__(self, job, datastore_client):
        self._job = job
        self._datastore_client = datastore_client

    def __call__(self, task, event, _):
        # Outline:
        #   - Check build state payload.
        #     - If successful, update the task payload with state and relevant
        #       information, propagate information into the accumulator.
        #     - If unsuccessful:
        #       - Retry if the failure is a retryable error (update payload with
        #         retry information)
        #       - Fail if failure is non-retryable or we've exceeded retries.
        if event.type == 'update':
            payload = self.unpack(
                find_isolate_task_payload_pb2.FindIsolateTaskPayload,
                task.payload)
            change = change_module.Change.FromProto(self._datastore_client,
                                                    payload.change)
            return [
                builds.UpdateBuildStatus(
                    self._datastore_client,
                    self._job,
                    task,
                    change,
                    event,
                )
            ]
        return None


class Evaluator(combinators.SequenceEvaluator):
    def __init__(self, job, datastore_client):
        super(Evaluator, self).__init__(evaluators=(
            combinators.TaskPayloadLiftingEvaluator(),
            combinators.FilteringEvaluator(
                predicate=predicates.All(
                    predicates.TaskTypeEq('find_isolate'),
                    predicates.TaskIsEventTarget(),
                    predicates.Not(
                        predicates.TaskStateIn(
                            {'completed', 'failed', 'cancelled'})),
                ),
                delegate=combinators.DispatchByEventTypeEvaluator(
                    {
                        'initiate': InitiateEvaluator(job, datastore_client),
                        'update': UpdateEvaluator(job, datastore_client),
                    }, ),
            ),
        ))


class BuildSerializer(task_module.PayloadUnpackingMixin):
    def __call__(self, task, _, context):
        try:
            task_payload = self.unpack(
                find_isolate_task_payload_pb2.FindIsolateTaskPayload,
                task.payload,
            )
        except:
            return None

        results = context.get(task.id, {})
        results.update({
            'completed':
            task.state in {'completed', 'failed', 'cancelled'},
            'exception':
            ','.join(e.reason for e in task_payload.errors) or None,
            'details': [{
                'key': 'builder',
                'value': task_payload.builder,
                'url': task_payload.buildbucket_build.url or None,
            }]
        })

        if task_payload.buildbucket_build.id:
            results['details'].append({
                'key':
                'build',
                'value':
                task_payload.buildbucket_build.id,
                'url':
                task_payload.buildbucket_build.url or None
            })

        if task_payload.isolate_server and task_payload.isolate_hash:
            results['details'].append({
                'key':
                'isolate',
                'value':
                task_payload.isolate_hash,
                'url':
                f'{task_payload.isolate_server}/browse?digest={task_payload.isolate_hash}'
            })

        context.update({task.id: results})


class Serializer(combinators.FilteringEvaluator):
    def __init__(self):
        super(Serializer,
              self).__init__(predicate=predicates.TaskTypeEq('find_isolate'),
                             delegate=BuildSerializer())


@dataclasses.dataclass
class TaskOptions:
    builder: str
    target: str
    bucket: str
    change: change_module.Change


def change_id(change: change_module.Change) -> str:
    return change.id_string.replace(' ', '_')


def task_id(options: TaskOptions) -> str:
    return f'find_isolate_{change_id(options.change)}'


def create_graph(options: TaskOptions) -> evaluator.TaskGraph:
    # We create the proto payload from the options.
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload(
        builder=options.builder,
        target=options.target,
        bucket=options.bucket,
        change=change_pb2.Change(
            commits=[
                change_pb2.Commit(repository=c.repository.name,
                                  git_hash=c.git_hash)
                for c in options.change.commits
            ],
            patch=(change_pb2.GerritPatch(
                server=options.change.patch.server,
                change=options.change.patch.change,
                revision=options.change.patch.revision)
                   if options.change.patch else None),
        ),
    )
    encoded_payload = any_pb2.Any()
    encoded_payload.Pack(task_payload)
    return evaluator.TaskGraph(
        vertices=[
            evaluator.TaskVertex(
                id=task_id(options),
                vertex_type='find_isolate',
                payload=encoded_payload,
                state='pending',
            )
        ],
        edges=[],
    )
