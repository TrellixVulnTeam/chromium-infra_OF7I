# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import functools
import logging

from google.cloud import datastore
from google.protobuf import any_pb2
from google.protobuf import empty_pb2
from google.protobuf import struct_pb2
from google.protobuf import wrappers_pb2
import pytest

from chromeperf.engine import evaluator as evaluator_module
from chromeperf.pinpoint.models import task as task_module
from chromeperf.pinpoint.actions import updates

FakeEvent = collections.namedtuple('Event', ('type', 'status', 'payload'))

MockJob = collections.namedtuple('MockJob', ('key'))

def _Int32Payload(i):
  payload = any_pb2.Any()
  payload.Pack(wrappers_pb2.Int32Value(value=i))
  return payload

def _SimpleDictPayload(**kwargs):
  """Simple conversion of kwargs to Any-packed google.protobuf.Struct messages.

  Values need to be implicitly convertible to google.protobuf.Value messages
  (e.g. scalars like ints and strings are fine, classes and dicts probably not).
  """
  payload = any_pb2.Any()
  struct = struct_pb2.Struct()
  for key, value in kwargs.items():
    struct[key] = value
  payload.Pack(struct)
  return payload

def _EmptyPayload():
  payload = any_pb2.Any()
  payload.Pack(empty_pb2.Empty())
  return payload


@pytest.fixture
def client(request):
  """A datastore.Client with a psuedorandom suffix appended to its project.

  The suffix is derived from the test name.  Use this to get some isolation
  between tests using datastore, even when running tests in parallel.
  """
  # Project IDs are pretty constrained (6-30 chars, only lowercase, digits and
  # hyphen), so append a hex string of the hash of the test name to get a
  # sufficiently unique name that is still valid.
  return datastore.Client(project='chromeperf-' + hex(hash(request.node.name)))


def testPopulateAndEvaluateAdderGraph(client):
  job = MockJob(client.key('Job', 'Key'))
  task_graph = evaluator_module.TaskGraph(
      vertices=[
          evaluator_module.TaskVertex(
              id='input2', vertex_type='constant', payload=_Int32Payload(2)),
          evaluator_module.TaskVertex(
              id='input3', vertex_type='constant', payload=_Int32Payload(3)),
          evaluator_module.TaskVertex(
              id='plus', vertex_type='operator+', payload=_EmptyPayload()),
      ],
      edges=[
          evaluator_module.Dependency(from_='plus', to='input2'),
          evaluator_module.Dependency(from_='plus', to='input3'),
      ],
  )
  task_module.populate_task_graph(client, job, task_graph)

  def AdderEvaluator(task, _, accumulator):
    if task.task_type == 'constant':
      int_payload = wrappers_pb2.Int32Value()
      assert task.payload.Unpack(int_payload)
      accumulator[task.id] = int_payload.value
    elif task.task_type == 'operator+':
      inputs = [accumulator.get(dep) for dep in task.dependencies]
      accumulator[task.id] = functools.reduce(lambda a, v: a + v, inputs)

  accumulator = evaluator_module.evaluate_graph(
      {},
      AdderEvaluator,
      task_module.task_graph_loader(client, job),
  )
  assert accumulator.get('plus') == 5


def testPouplateAndEvaluateGrowingGraph(client):
  job = MockJob(client.key('Job', 'key'))
  task_module.populate_task_graph(
      client,
      job,
      evaluator_module.TaskGraph(
          vertices=[
              evaluator_module.TaskVertex(
                  id='rev_0',
                  vertex_type='revision',
                  payload=_SimpleDictPayload(revision='0', position=0),
                  ),
              evaluator_module.TaskVertex(
                  id='rev_100',
                  vertex_type='revision',
                  payload=_SimpleDictPayload(revision='100', position=100),
                  ),
              evaluator_module.TaskVertex(
                  id='bisection', vertex_type='bisection',
                  payload=_EmptyPayload()),
          ],
          edges=[
              evaluator_module.Dependency(from_='bisection', to='rev_0'),
              evaluator_module.Dependency(from_='bisection', to='rev_100'),
          ],
      ),
  )


def testPopulateEvaluateCallCounts(client):
  job = MockJob(client.key('Job', 'key'))
  task_module.populate_task_graph(
      client,
      job,
      evaluator_module.TaskGraph(
          vertices=[
              evaluator_module.TaskVertex(
                  id='leaf_0', vertex_type='node', payload=_EmptyPayload()),
              evaluator_module.TaskVertex(
                  id='leaf_1', vertex_type='node', payload=_EmptyPayload()),
              evaluator_module.TaskVertex(
                  id='parent', vertex_type='node', payload=_EmptyPayload()),
          ],
          edges=[
              evaluator_module.Dependency(from_='parent', to='leaf_0'),
              evaluator_module.Dependency(from_='parent', to='leaf_1'),
          ],
      ),
  )
  calls = {}

  def CallCountEvaluator(task, event, accumulator):
    logging.debug('Evaluate(%s, %s, %s) called.', task.id, event, accumulator)
    calls[task.id] = calls.get(task.id, 0) + 1
    return None

  evaluator_module.evaluate_graph(
      'test',
      CallCountEvaluator,
      task_module.task_graph_loader(client, job),
  )
  assert calls == {
      'leaf_0': 1,
      'leaf_1': 1,
      'parent': 1,
  }


def testPopulateEmptyGraph(mocker, client):
  job = MockJob(client.key('Job', 'key'))
  task_graph = evaluator_module.TaskGraph(vertices=[], edges=[])
  task_module.populate_task_graph(client, job, task_graph)
  evaluator = mocker.MagicMock()
  evaluator.assert_not_called()
  with pytest.raises(evaluator_module.MalformedGraphError):
    evaluator_module.evaluate_graph(
        'test',
        evaluator,
        task_module.task_graph_loader(client, job),
    )


def testPopulateCycles(mocker, client):
  job = MockJob(client.key('Job', 'key'))
  task_graph = evaluator_module.TaskGraph(
      vertices=[
          evaluator_module.TaskVertex(
              id='node_0', vertex_type='process', payload=_EmptyPayload()),
          evaluator_module.TaskVertex(
              id='node_1', vertex_type='process', payload=_EmptyPayload())
      ],
      edges=[
          evaluator_module.Dependency(from_='node_0', to='node_1'),
          evaluator_module.Dependency(from_='node_1', to='node_0')
      ])
  task_module.populate_task_graph(client, job, task_graph)

  evaluator = mocker.MagicMock()
  evaluator.assert_not_called()
  with pytest.raises(evaluator_module.MalformedGraphError):
    evaluator_module.evaluate_graph(
        'test',
        evaluator,
        task_module.task_graph_loader(client, job),
    )


@pytest.mark.skip(reason='Not implemented yet')
def testPopulateIslands():
  pass


def update_task(client, job, task_id, new_state, _):
  logging.debug('Updating task "%s" to "%s"', task_id, new_state)
  updates.update_task(client, job, task_id, new_state=new_state)


def TransitionEvaluator(client, job, task, event, accumulator):
  accumulator[task.id] = task.state
  if task.id != event.get('target'):
    if task.dependencies and any(
        accumulator.get(dep) == 'ongoing'
        for dep in task.dependencies) and task.state != 'ongoing':
      return [functools.partial(update_task, client, job, task.id, 'ongoing')]
    if len(task.dependencies) and all(
        accumulator.get(dep) == 'completed'
        for dep in task.dependencies) and task.state != 'completed':
      return [functools.partial(update_task, client, job, task.id, 'completed')]
    return None

  if task.state == event.get('current_state'):
    return [
        functools.partial(update_task, client, job, task.id,
                          event.get('new_state'))
    ]


class SetupGraph():

  def __init__(self, client):
    self.job = MockJob(client.key('Job', 'key'))
    task_module.populate_task_graph(
        client,
        self.job,
        evaluator_module.TaskGraph(
            vertices=[
                evaluator_module.TaskVertex(
                    id='task_0', vertex_type='task', payload=_EmptyPayload()),
                evaluator_module.TaskVertex(
                    id='task_1', vertex_type='task', payload=_EmptyPayload()),
                evaluator_module.TaskVertex(
                    id='task_2', vertex_type='task', payload=_EmptyPayload()),
            ],
            edges=[
                evaluator_module.Dependency(from_='task_2', to='task_0'),
                evaluator_module.Dependency(from_='task_2', to='task_1'),
            ],
        ),
    )
    self.graph = task_module.task_graph_loader(client, self.job)


@pytest.fixture
def setupGraph(client):
  return SetupGraph(client)


def testEvaluateStateTransitionProgressions(setupGraph, client):
  assert evaluator_module.evaluate_graph(
      {
          'target': 'task_0',
          'current_state': 'pending',
          'new_state': 'ongoing'
      },
      functools.partial(TransitionEvaluator, client, setupGraph.job),
      setupGraph.graph,
  ) == {
      'task_0': 'ongoing',
      'task_1': 'pending',
      'task_2': 'ongoing'
  }
  assert evaluator_module.evaluate_graph(
      {
          'target': 'task_1',
          'current_state': 'pending',
          'new_state': 'ongoing'
      },
      functools.partial(TransitionEvaluator, client, setupGraph.job),
      setupGraph.graph,
  ) == {
      'task_0': 'ongoing',
      'task_1': 'ongoing',
      'task_2': 'ongoing'
  }
  assert evaluator_module.evaluate_graph(
      {
          'target': 'task_0',
          'current_state': 'ongoing',
          'new_state': 'completed'
      },
      functools.partial(TransitionEvaluator, client, setupGraph.job),
      setupGraph.graph,
  ) == {
      'task_0': 'completed',
      'task_1': 'ongoing',
      'task_2': 'ongoing'
  }
  assert evaluator_module.evaluate_graph(
      {
          'target': 'task_1',
          'current_state': 'ongoing',
          'new_state': 'completed'
      },
      functools.partial(TransitionEvaluator, client, setupGraph.job),
      setupGraph.graph,
  ) == {
      'task_0': 'completed',
      'task_1': 'completed',
      'task_2': 'completed'
  }


def testEvaluateInvalidTransition(setupGraph, client):
  with pytest.raises(updates.InvalidTransition):
    assert evaluator_module.evaluate_graph(
        {
            'target': 'task_0',
            'current_state': 'pending',
            'new_state': 'failed'
        },
        functools.partial(TransitionEvaluator, client, setupGraph.job),
        setupGraph.graph,
    ) == {
        'task_0': 'failed',
        'task_1': 'pending',
        'task_2': 'pending',
    }

    evaluator_module.evaluate_graph(
        {
            'target': 'task_0',
            'current_state': 'failed',
            'new_state': 'ongoing'
        },
        functools.partial(TransitionEvaluator, client, setupGraph.job),
        setupGraph.graph,
    )


def testEvaluateInvalidAmendment_ExistingTask(setupGraph, client):
  with pytest.raises(updates.InvalidAmendment):

    def AddExistingTaskEvaluator(task, event, _):
      if event.get('target') == task.id:

        def GraphExtender(_):
          updates.extend_task_graph(
              client,
              setupGraph.job,
              vertices=[
                  evaluator_module.TaskVertex(
                      id=task.id,
                      vertex_type='duplicate',
                      payload=_EmptyPayload())
              ],
              dependencies=[
                  evaluator_module.Dependency(from_='task_2', to=task.id)
              ],
          )

        return [GraphExtender]

    evaluator_module.evaluate_graph(
        {'target': 'task_0'},
        AddExistingTaskEvaluator,
        setupGraph.graph,
    )


def testEvaluateInvalidAmendment_BrokenDependency(setupGraph, client):
  with pytest.raises(ValueError):

    def AddExistingTaskEvaluator(task, event, _):
      if event.get('target') == task.id:

        def GraphExtender(_):
          updates.extend_task_graph(
              client,
              setupGraph.job,
              vertices=[],
              dependencies=[
                  evaluator_module.Dependency(from_='unknown', to=task.id)
              ],
          )

        return [GraphExtender]

    evaluator_module.evaluate_graph(
        {'target': 'task_0'},
        AddExistingTaskEvaluator,
        setupGraph.graph,
    )


@pytest.mark.skip(reason='Not implemented yet')
def testEvaluateConverges(self):
  pass


def TaskStatusGetter(task_status, task, event, _):
  if event.type == 'test':
    task_status[task.id] = task.status
  return None
