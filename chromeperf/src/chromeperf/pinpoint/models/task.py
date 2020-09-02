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
    'append_task_log',
)


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
  created: datetime.datetime = dataclasses.field(
      default_factory=datetime.datetime.now)
  # updated: datetime.datetime = datetime.datetime.now()


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


def append_task_log(client, job, task_id, message, payload):
  task_log = TaskLog(
      parent=client.key('Task', task_id, parent=job.key),
      message=message,
      payload=payload)
  task_log.put()


def task_graph_loader(client, job):

  def load_task_graph():
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

  return load_task_graph
