# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import collections
import dataclasses
import datetime
import functools
import itertools
import logging

from google.cloud import datastore

from chromeperf.engine import evaluator as evaluator_module

__all__ = (
    'populate_task_graph',
    'extend_task_graph',
    'update_task',
    'append_task_log',
)

# These InMemoryTask instances are meant to isolate the Task model which is
# actually persisted in Datastore.
InMemoryTask = collections.namedtuple(
    'InMemoryTask', ('id', 'task_type', 'payload', 'status', 'dependencies'))

VALID_TRANSITIONS = {
    'pending': {'ongoing', 'completed', 'failed', 'cancelled'},
    'ongoing': {'completed', 'failed', 'cancelled'},
    'cancelled': {'pending'},
    'completed': {'pending'},
    'failed': {'pending'},
}

# Traversal states used in the graph traversal. We use these as marks for when
# vertices are traversed, as how we would implement graph colouring in a graph
# traversal (like Depth First Search).
NOT_EVALUATED, CHILDREN_PENDING, EVALUATION_DONE = (0, 1, 2)


class Error(Exception):
  pass


class InvalidAmendment(Error):
  pass


class TaskNotFound(Error):
  pass


class InvalidTransition(Error):
  pass


# These are internal-only models, used as an implementation detail of the
# execution engine.
@dataclasses.dataclass
class Task:
  """A Task associated with a Pinpoint Job.

  Task instances are always associated with a Job. Tasks represent units of work
  that are in well-defined states. Updates to Task instances are transactional
  and need to be.
  """
  key: datastore.Key
  task_type: str
  status: str
  payload: str
  dependencies: list = dataclasses.field(default_factory=list)
  created: datetime.datetime = datetime.datetime.now()
  updated: datetime.datetime = datetime.datetime.now()

  def ToInMemoryTask(self):
    # We isolate the ndb model `Task` from the evaluator, to avoid accidentially
    # modifying the state in datastore.
    return InMemoryTask(
        id=self.key.id(),
        task_type=self.task_type,
        payload=self.payload,
        status=self.status,
        dependencies=[dep.id() for dep in self.dependencies])


class TaskLog(
    collections.namedtuple('TaskLog', ('timestamp', 'message', 'payload'))):
  """Log entries associated with Task instances.

  TaskLog instances are always associated with a Task. These entries are
  immutable once created.
  """
  __slots__ = ()


def populate_task_graph(client, job, graph):
  """Populate the Datastore with Task instances associated with a Job.

  The `graph` argument must have two properties: a collection of `TaskVertex`
  instances named `vertices` and a collection of `Dependency` instances named
  `dependencies`.
  """
  if job is None:
    raise ValueError('job must not be None.')

  job_key = job.key
  tasks = {
      v.id: Task(
          key=client.key('Task', v.id, parent=job_key),
          task_type=v.vertex_type,
          payload=v.payload,
          status='pending') for v in graph.vertices
  }
  dependencies = set()
  for dependency in graph.edges:
    dependency_key = client.key('Task', dependency.to, parent=job_key)
    if dependency not in dependencies:
      tasks[dependency.from_].dependencies.append(dependency_key)
      dependencies.add(dependency)

  def EncodeTaskEntity(task):
    e = datastore.Entity(task.key)
    e.update(dataclasses.asdict(task))
    del e['key']
    return e

  with client.transaction():
    client.put_multi(EncodeTaskEntity(t) for t in tasks.values())


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
      v.id: Task(
          key=client.key('Task', v.id, parent=job_key),
          task_type=v.vertex_type,
          status='pending',
          payload=v.payload) for v in vertices
  }

  with client.transaction():
    # Ensure that the keys we're adding are not in the graph yet.
    current_tasks = client.query(kind='Task', ancestor=job_key).fetch()
    current_task_keys = set(t.key for t in current_tasks)
    new_task_keys = set(t.key for t in amendment_task_graph.values())
    overlap = new_task_keys & current_task_keys
    if overlap:
      raise InvalidAmendment('vertices (%r) already in task graph.' %
                             (overlap,))

    # Then we add the dependencies.
    current_task_graph = {t.key.id(): t for t in current_tasks}
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
              (dependency.from_,))
        if current_task:
          current_task_graph[dependency.from_].dependencies.append(
              dependency_key)
        if amendment_task:
          amendment_task_graph[dependency.from_].dependencies.append(
              dependency_key)
        handled_dependencies.add(dependency)
        update_filter.add(dependency.from_)

    client.put_multi(
        itertools.chain(amendment_task_graph.values(), [
            t for id_, t in current_task_graph.items() if id_ in update_filter
        ]),
        use_cache=True)


def update_task(client, job, task_id, new_state=None, payload=None):
  """Update a task.

  This enforces that the status transitions are semantically correct, where only
  the transitions defined in the VALID_TRANSITIONS map are allowed.

  When either new_state or payload are not None, this function performs the
  update transactionally. At least one of `new_state` or `payload` must be
  provided in calls to this function.
  """
  if new_state is None and payload is None:
    raise ValueError('Set one of `new_state` or `payload`.')

  if new_state and new_state not in VALID_TRANSITIONS:
    raise InvalidTransition('Unknown state: %s' % (new_state,))

  with client.transaction():
    task = client.get(client.key('Task', task_id, parent=job.key))
    if not task:
      raise TaskNotFound('Task with id "%s" not found for job "%s".' %
                         (task_id, job.job_id))

    if new_state:
      valid_transitions = VALID_TRANSITIONS.get(task['status'])
      if new_state not in valid_transitions:
        raise InvalidTransition(
            'Attempting transition from "%s" to "%s" not in %s; task = %s' %
            (task['status'], new_state, valid_transitions, task))
      task['status'] = new_state

    if payload:
      task['payload'] = payload

    client.put(task)


def append_task_log(client, job, task_id, message, payload):
  task_log = TaskLog(
      parent=client.key('Task', task_id, parent=job.key),
      message=message,
      payload=payload)
  task_log.put()


def TaskGraphLoader(client, job):

  def LoadTaskGraph():
    with client.transaction():
      tasks = list(client.query(kind='Task', ancestor=job.key).fetch())

    vertices = [
        evaluator_module.TaskVertex(
            id=t.key.name,
            vertex_type=t['task_type'],
            payload=t['payload'],
            state=t['status'],
        ) for t in tasks
    ]
    return evaluator_module.TaskGraph(
        vertices=vertices,
        edges=[(from_.key.name, to.name)
               for from_ in tasks
               for to in from_['dependencies']],
    )

  return LoadTaskGraph
