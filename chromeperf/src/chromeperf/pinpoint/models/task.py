# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import dataclasses
import datetime
import functools
import itertools
import logging
import typing

from google.cloud import datastore
from google.protobuf import any_pb2

from chromeperf.engine import evaluator as evaluator_module

__all__ = (
    'populate_task_graph',
    'append_task_log',
)


# These are internal-only models, used as an implementation detail of the
# execution engine.  These correspond to the entity kinds in Datastore.
@dataclasses.dataclass
class Task:
    """A Task associated with a Pinpoint Job.

    Task instances are always associated with a Job. Tasks represent units of
    work that are in well-defined states. Updates to Task instances are
    transactional and need to be.
    """
    key: datastore.Key
    task_type: str
    status: str
    payload: any_pb2.Any
    dependencies: list = dataclasses.field(default_factory=list)
    created: datetime.datetime = dataclasses.field(
        default_factory=datetime.datetime.now)

    # updated: datetime.datetime = datetime.datetime.now()

    def ToEntity(self) -> datastore.Entity:
        d = dataclasses.asdict(self)
        del d['key']
        d['payload'] = d['payload'].SerializeToString()
        entity = datastore.Entity(self.key)
        entity.update(d)
        return entity

    @staticmethod
    def FromEntity(entity):
        d = dict(entity)
        d['payload'] = any_pb2.Any.FromString(d.get('payload', ''))
        d['dependencies'] = list(d.get('dependencies', []))
        d['key'] = entity.key
        return Task(**d)


def populate_task_graph(client, job, graph):
    """Populate the Datastore with Task instances associated with a Job.

    The `graph` argument must have two properties: a collection of
    `TaskVertex` instances named `vertices` and a collection of `Dependency`
    instances named `dependencies`.
    """
    if job is None:
        raise ValueError('job must not be None.')

    job_key = job.key
    tasks = {
        v.id: Task(key=client.key('Task', v.id, parent=job_key),
                   task_type=v.vertex_type,
                   payload=v.payload,
                   status='pending')
        for v in graph.vertices
    }
    dependencies = set()
    for dependency in graph.edges:
        dependency_key = client.key('Task', dependency.to, parent=job_key)
        if dependency not in dependencies:
            tasks[dependency.from_].dependencies.append(dependency_key)
            dependencies.add(dependency)

    with client.transaction():
        client.put_multi(t.ToEntity() for t in tasks.values())


def task_graph_loader(
    client: datastore.Client,
    job: typing.Any,
) -> typing.Callable[[], evaluator_module.TaskGraph]:
    def load_task_graph() -> evaluator_module.TaskGraph:
        with client.transaction():
            task_entities = list(
                client.query(kind='Task', ancestor=job.key).fetch())
        tasks = [Task.FromEntity(te) for te in task_entities]

        vertices = [
            evaluator_module.TaskVertex(
                id=t.key.name,
                vertex_type=t.task_type,
                payload=t.payload,
                state=t.status,
            ) for t in tasks
        ]
        return evaluator_module.TaskGraph(
            vertices=vertices,
            edges=[
                evaluator_module.Dependency(from_.key.name, to.name)
                for from_ in tasks for to in from_.dependencies
            ],
        )

    return load_task_graph
