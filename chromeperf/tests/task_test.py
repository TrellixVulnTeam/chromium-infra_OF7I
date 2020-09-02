# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import functools
import logging
import pytest

from google.cloud import datastore

from chromeperf.engine import evaluator as evaluator_module
from chromeperf.pinpoint.models import task as task_module

FakeEvent = collections.namedtuple('Event', ('type', 'status', 'payload'))

MockJob = collections.namedtuple('MockJob', ('key'))


def testPopulateAndEvaluateAdderGraph():
  client = datastore.Client()
  job = MockJob(client.key('Job', 'key'))
  task_graph = evaluator_module.TaskGraph(
      vertices=[
          evaluator_module.TaskVertex(
              id='input0', vertex_type='constant', payload={'value': 0}),
          evaluator_module.TaskVertex(
              id='input1', vertex_type='constant', payload={'value': 1}),
          evaluator_module.TaskVertex(
              id='plus', vertex_type='operator+', payload={}),
      ],
      edges=[
          evaluator_module.Dependency(from_='plus', to='input0'),
          evaluator_module.Dependency(from_='plus', to='input1'),
      ],
  )
  task_module.populate_task_graph(client, job, task_graph)

  def AdderEvaluator(task, _, accumulator):
    if task.task_type == 'constant':
      accumulator[task.id] = task.payload.get('value', 0)
    elif task.task_type == 'operator+':
      inputs = [accumulator.get(dep) for dep in task.dependencies]
      accumulator[task.id] = functools.reduce(lambda a, v: a + v, inputs)

  accumulator = evaluator_module.evaluate_graph(
      {},
      AdderEvaluator,
      task_module.TaskGraphLoader(client, job),
  )
  assert accumulator.get('plus') == 1


def testPouplateAndEvaluateGrowingGraph():
  client = datastore.Client()
  job = MockJob(client.key('Job', 'key'))
  task_module.populate_task_graph(
      client,
      job,
      evaluator_module.TaskGraph(
          vertices=[
              evaluator_module.TaskVertex(
                  id='rev_0',
                  vertex_type='revision',
                  payload={
                      'revision': '0',
                      'position': 0
                  }),
              evaluator_module.TaskVertex(
                  id='rev_100',
                  vertex_type='revision',
                  payload={
                      'revision': '100',
                      'position': 100
                  }),
              evaluator_module.TaskVertex(
                  id='bisection', vertex_type='bisection', payload={}),
          ],
          edges=[
              evaluator_module.Dependency(from_='bisection', to='rev_0'),
              evaluator_module.Dependency(from_='bisection', to='rev_100'),
          ],
      ),
  )


def testPopulateEvaluateCallCounts():
  client = datastore.Client()
  job = MockJob(client.key('Job', 'key'))
  task_module.populate_task_graph(
      client,
      job,
      evaluator_module.TaskGraph(
          vertices=[
              evaluator_module.TaskVertex(
                  id='leaf_0', vertex_type='node', payload={}),
              evaluator_module.TaskVertex(
                  id='leaf_1', vertex_type='node', payload={}),
              evaluator_module.TaskVertex(
                  id='parent', vertex_type='node', payload={}),
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
      task_module.TaskGraphLoader(client, job),
  )
  assert calls == {
      'leaf_0': 1,
      'leaf_1': 1,
      'parent': 1,
  }


def testPopulateEmptyGraph(mocker):
  client = datastore.Client()
  job = MockJob(client.key('Job', 'key'))
  task_graph = evaluator_module.TaskGraph(vertices=[], edges=[])
  task_module.populate_task_graph(client, job, task_graph)
  evaluator = mocker.MagicMock()
  evaluator.assert_not_called()
  with pytest.raises(evaluator_module.MalformedGraphError):
    evaluator_module.evaluate_graph(
        'test',
        evaluator,
        task_module.TaskGraphLoader(client, job),
    )


def testPopulateCycles(mocker):
  client = datastore.Client()
  job = MockJob(client.key('Job', 'key'))
  task_graph = evaluator_module.TaskGraph(
      vertices=[
          evaluator_module.TaskVertex(
              id='node_0', vertex_type='process', payload={}),
          evaluator_module.TaskVertex(
              id='node_1', vertex_type='process', payload={})
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
        task_module.TaskGraphLoader(client, job),
    )


@pytest.mark.skip(reason='Not implemented yet')
def testPopulateIslands():
  pass


def update_task(job, task_id, new_state, _):
  logging.debug('Updating task "%s" to "%s"', task_id, new_state)
  task_module.update_task(datastore.Client(), job, task_id, new_state=new_state)


def TransitionEvaluator(job, task, event, accumulator):
  accumulator[task.id] = task.state
  if task.id != event.get('target'):
    if task.dependencies and any(
        accumulator.get(dep) == 'ongoing'
        for dep in task.dependencies) and task.state != 'ongoing':
      return [functools.partial(update_task, job, task.id, 'ongoing')]
    if len(task.dependencies) and all(
        accumulator.get(dep) == 'completed'
        for dep in task.dependencies) and task.state != 'completed':
      return [functools.partial(update_task, job, task.id, 'completed')]
    return None

  if task.state == event.get('current_state'):
    return [
        functools.partial(update_task, job, task.id, event.get('new_state'))
    ]


class SetupGraph():

  def __init__(self):
    client = datastore.Client()
    self.job = MockJob(client.key('Job', 'key'))
    task_module.populate_task_graph(
        client,
        self.job,
        evaluator_module.TaskGraph(
            vertices=[
                evaluator_module.TaskVertex(
                    id='task_0', vertex_type='task', payload={}),
                evaluator_module.TaskVertex(
                    id='task_1', vertex_type='task', payload={}),
                evaluator_module.TaskVertex(
                    id='task_2', vertex_type='task', payload={}),
            ],
            edges=[
                evaluator_module.Dependency(from_='task_2', to='task_0'),
                evaluator_module.Dependency(from_='task_2', to='task_1'),
            ],
        ),
    )
    self.graph = task_module.TaskGraphLoader(client, self.job)


@pytest.fixture
def setupGraph():
  return SetupGraph()


def testEvaluateStateTransitionProgressions(setupGraph):
  assert evaluator_module.evaluate_graph(
      {
          'target': 'task_0',
          'current_state': 'pending',
          'new_state': 'ongoing'
      },
      functools.partial(TransitionEvaluator, setupGraph.job),
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
      functools.partial(TransitionEvaluator, setupGraph.job),
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
      functools.partial(TransitionEvaluator, setupGraph.job),
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
      functools.partial(TransitionEvaluator, setupGraph.job),
      setupGraph.graph,
  ) == {
      'task_0': 'completed',
      'task_1': 'completed',
      'task_2': 'completed'
  }


def testEvaluateInvalidTransition(setupGraph):
  with pytest.raises(task_module.InvalidTransition):
    assert evaluator_module.evaluate_graph(
        {
            'target': 'task_0',
            'current_state': 'pending',
            'new_state': 'failed'
        },
        functools.partial(TransitionEvaluator, setupGraph.job),
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
        functools.partial(TransitionEvaluator, setupGraph.job),
        setupGraph.graph,
    )


def testEvaluateInvalidAmendment_ExistingTask(setupGraph):
  with pytest.raises(task_module.InvalidAmendment):

    def AddExistingTaskEvaluator(task, event, _):
      if event.get('target') == task.id:

        def GraphExtender(_):
          task_module.extend_task_graph(
              datastore.Client(),
              setupGraph.job,
              vertices=[
                  evaluator_module.TaskVertex(
                      id=task.id, vertex_type='duplicate', payload={})
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


def testEvaluateInvalidAmendment_BrokenDependency(setupGraph):
  with pytest.raises(ValueError):

    def AddExistingTaskEvaluator(task, event, _):
      if event.get('target') == task.id:

        def GraphExtender(_):
          task_module.extend_task_graph(
              datastore.Client(),
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
