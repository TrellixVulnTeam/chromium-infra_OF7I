# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses
import functools
import itertools
import logging
import typing

from google.cloud import datastore
from google.protobuf import any_pb2
from google.protobuf import message

from chromeperf.engine import task_pb2
from chromeperf.pinpoint.models import task as task_module

VALID_TRANSITIONS = {
    'pending': {'ongoing', 'completed', 'failed', 'cancelled'},
    'ongoing': {'completed', 'failed', 'cancelled'},
    'cancelled': {'pending'},
    'completed': {'pending'},
    'failed': {'pending'},
}


class Error(Exception):
    pass


class InvalidAmendment(Error):
    pass


class TaskNotFound(Error):
    pass


class InvalidTransition(Error):
    pass


def update_task(datastore_client,
                job,
                task_id,
                new_state=None,
                payload: typing.Optional[any_pb2.Any] = None):
    """Update a task.

    This enforces that the status transitions are semantically correct, where
    only the transitions defined in the VALID_TRANSITIONS map are allowed.

    When either new_state or payload are not None, this function performs the
    update transactionally. At least one of `new_state` or `payload` must be
    provided in calls to this function.
    """
    if new_state is None and payload is None:
        raise ValueError('Set one of `new_state` or `payload`.')

    if new_state and new_state not in VALID_TRANSITIONS:
        raise InvalidTransition('Unknown state: %s' % (new_state, ))

    with datastore_client.transaction():
        task = datastore_client.get(
            datastore_client.key(
                'Task',
                task_id,
                parent=job.key,
            ), )
        if not task:
            raise TaskNotFound('Task with id "%s" not found for job "%s".' %
                               (task_id, job.job_id))

        if new_state:
            valid_transitions = VALID_TRANSITIONS.get(task['status'])
            task_status = task['status']
            if new_state not in valid_transitions:
                raise InvalidTransition(
                    f'Attempting transition from "{task_status}"'
                    f'to "{new_state}" not in {valid_transitions}'
                    f'; task = {task}')
            task['status'] = new_state

        if payload:
            task['payload'] = payload.SerializeToString()

        datastore_client.put(task)


@dataclasses.dataclass(frozen=True)
class UpdateTaskAction:
    datastore_client: datastore.Client
    job: typing.Any
    task_id: str
    new_state: typing.Optional[str] = None
    payload: typing.Optional[message.Message] = None

    def __str__(self):
        return (
            f'UpdateTaskAction(task_id={self.task_id}, '
            f'new_state={self.new_state}, '
            f'payload={None if self.payload is None else "..."})')

    def __call__(self, context):
        del context
        update_task(self.datastore_client, self.job, self.task_id,
                    new_state=self.new_state, payload=self.payload)


def extend_task_graph(client, job, vertices, dependencies):
    """Add new vertices and dependency links to the graph.

    Args:
      job: a dashboard.pinpoint.model.job.Job instance.
      vertices: an iterable of TaskVertex instances.
      dependencies: an iterable of Dependency instances.
    """
    if job is None:
        raise ValueError('job must not be None.')
    if not vertices and not dependencies:
        return

    job_key = job.key
    amendment_task_graph = {
        v.id: task_module.Task(key=client.key('Task', v.id, parent=job_key),
                               task_type=v.vertex_type,
                               status='pending',
                               payload=v.payload)
        for v in vertices
    }

    with client.transaction():
        # Ensure that the keys we're adding are not in the graph yet.
        current_tasks = list(
            client.query(kind='Task', ancestor=job_key).fetch())
        current_task_keys = set(t.key for t in current_tasks)
        new_task_keys = set(t.key for t in amendment_task_graph.values())
        overlap = new_task_keys & current_task_keys
        if overlap:
            raise InvalidAmendment('vertices (%r) already in task graph.' %
                                   (overlap, ))

        # Then we add the dependencies.
        current_task_graph = {t.key.id_or_name: t for t in current_tasks}
        handled_dependencies = set()
        update_filter = set(amendment_task_graph)
        for dependency in dependencies:
            dependency_key = client.key('Task', dependency.to, parent=job_key)
            if dependency not in handled_dependencies:
                current_task = current_task_graph.get(dependency.from_)
                amendment_task = amendment_task_graph.get(dependency.from_)
                if current_task is None and amendment_task is None:
                    raise InvalidAmendment(
                        'dependency `from` (%s) not in amended graph.' %
                        (dependency.from_, ))
                if current_task:
                    current_task_graph[dependency.from_]['dependencies'].append(
                        dependency_key)
                if amendment_task:
                    amendment_task_graph[dependency.from_].dependencies.append(
                        dependency_key)
                handled_dependencies.add(dependency)
                update_filter.add(dependency.from_)

        client.put_multi(itertools.chain(
            (t.ToEntity() for t in amendment_task_graph.values()),
            (t for id_, t in current_task_graph.items()
             if id_ in update_filter)))


def log_transition_failures(wrapped_action):
    """Decorator to log state transition failures.

  This is a convenience decorator to handle state transition failures, and
  suppress further exception propagation of the transition failure.
  """
    @functools.wraps(wrapped_action)
    def ActionWrapper(*args, **kwargs):
        try:
            return wrapped_action(*args, **kwargs)
        except InvalidTransition as e:
            logging.error('State transition failed: %s', e)
            return None

    return ActionWrapper


class ErrorAppendingMixin:
    def update_task_with_error(
        self,
        datastore_client: datastore.Client,
        job: typing.Any,
        task: task_module.Task,
        payload: message.Message,
        reason: str,
        message: str,
    ):
        # Here we're assuming that the payload message has an `errors` field.
        payload.errors.append(
            task_pb2.ErrorMessage(reason=reason, message=message))
        task.payload.Pack(payload)
        return update_task(
            datastore_client,
            job,
            task.id,
            new_state='failed',
            payload=task.payload,
        )
