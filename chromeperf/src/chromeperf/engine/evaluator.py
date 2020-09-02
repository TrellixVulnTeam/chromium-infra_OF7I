# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import copy
import dataclasses
import logging

from chromeperf.engine import actions

# Traversal states used in the graph traversal. We use these as marks for when
# vertices are traversed, as how we would implement graph colouring in a graph
# traversal (like Depth First Search).
_NOT_EVALUATED, _CHILDREN_PENDING, _EVALUATION_DONE = (0, 1, 2)


@dataclasses.dataclass
class TaskVertex:
  id: str
  vertex_type: str
  payload: str
  state: str = 'unknown'


# TODO(fancl): Migrate all these namedtuple to dataclass
class Dependency(collections.namedtuple('Dependency', ('from_', 'to'))):
  __slots__ = ()


class TaskGraph(collections.namedtuple('TaskGraph', ('vertices', 'edges'))):
  __slots__ = ()


class _AdjacencyPair(
    collections.namedtuple('_AdjacencyPair', ('task', 'dependencies'))):
  __slots__ = ()


class NormalizedTask(
    collections.namedtuple(
        'NormalizedTask',
        ('id', 'task_type', 'payload', 'state', 'dependencies'))):
  __slots__ = ()


class _ReconstitutedTaskGraph(
    collections.namedtuple('_ReconstitutedTaskGraph',
                           ('terminal_tasks', 'tasks'))):
  __slots__ = ()


class Error(Exception):
  pass


class InvalidDependencyError(Error):
  pass


class MalformedGraphError(Error):
  pass


def _preprocess_graph(graph):
  if not graph.vertices:
    raise MalformedGraphError('No vertices in graph.')
  vertices = {
      vertex.id: _AdjacencyPair(task=vertex, dependencies=[])
      for vertex in graph.vertices
  }
  for from_, to in graph.edges:
    if from_ not in vertices:
      raise InvalidDependencyError(
          'Invalid dependency: {} not a known vertex'.format(from_))
    if to not in vertices:
      raise InvalidDependencyError(
          'Invalid dependency: {} not a known vertex'.format(to))
    vertices[from_].dependencies.append(to)
  has_dependents = set()
  for _, deps in vertices.values():
    has_dependents |= set(deps)
  terminal_tasks = [
      v.id for v, _ in vertices.values() if v.id not in has_dependents
  ]
  if not terminal_tasks:
    raise MalformedGraphError('Provided graph has no terminal vertices.')
  return _ReconstitutedTaskGraph(terminal_tasks=terminal_tasks, tasks=vertices)


def evaluate_graph(event, evaluator, load_graph):
  """Applies an evaluator given a task in the task graph and an event as input.

  This function implements a depth-first search traversal of the task graph and
  applies the `evaluator` given a task and the event input in post-order
  traversal. We start the DFS from the terminal tasks (those that don't have
  dependencies) and call the `evaluator` function with a representation of the
  task in the graph, an `event` as input, and an accumulator argument.

  `load_graph` is a callable which turns a `TaskGraph` instance, or one that
  has similar properties.

  The `evaluator` must be a callable which accepts three arguments:

    - task: a NormalizedTask instance, representing a task in the graph.
    - event: an object whose shape/type is defined by the caller of the
      `Evaluate` function and that the evaluator can handle.
    - accumulator: a dictionary which is mutable which is valid in the scope of
      a traversal of the graph.

  The `evaluator` must return either None or an iterable of callables which take
  a single argument, which is the accumulator at the end of a traversal.

  Events are free-form but usually are dictionaries which constitute inputs that
  are external to the task graph evaluation. This could model events in an
  event-driven evaluation of tasks, or synthetic inputs to the system. It is
  more important that the `event` information is known to the evaluator
  implementation, and is provided as-is to the evaluator in this function.

  The Evaluate function will keep iterating while there are actions still being
  produced by the evaluator. When there are no more actions to run, the Evaluate
  function will return the most recent traversal's accumulator.
  """

  context = {}
  collected_actions = [actions.Noop()]
  while collected_actions:
    for action in collected_actions:
      logging.debug('Running action: %s', action)

      # Each action should be a callable which takes the context as an input.
      # We want to run each action in their own transaction as well. This must
      # not be called in a transaction.
      action(context)

    # Clear the actions and accumulator for this traversal.
    del collected_actions[:]
    context.clear()

    # Load the graph, and preprocess it.
    graph = _preprocess_graph(load_graph())

    if not graph.tasks:
      logging.debug('Task graph was empty.')
      return

    # First get all the "terminal" tasks, and traverse the dependencies in a
    # depth-first-search.
    task_stack = [graph.tasks[task_id] for task_id in graph.terminal_tasks]

    vertex_states = {}
    while task_stack:
      task, deps = task_stack[-1]
      state = vertex_states.get(task.id, _NOT_EVALUATED)
      if state == _CHILDREN_PENDING:
        # We provide a deep copy of the task, to ensure that the evaluator
        # cannot modify the graph we're working with directly.
        result_actions = evaluator(
            NormalizedTask(
                copy.copy(task.id), copy.copy(task.vertex_type),
                copy.deepcopy(task.payload), copy.copy(task.state),
                copy.deepcopy(deps)), event, context)
        if result_actions:
          collected_actions.extend(result_actions)
        vertex_states[task.id] = _EVALUATION_DONE
      elif state == _NOT_EVALUATED:
        # This vertex has not been evaluated yet, we should traverse the
        # dependencies.
        vertex_states[task.id] = _CHILDREN_PENDING
        for dependency in deps:
          if vertex_states.get(dependency, _NOT_EVALUATED) == _NOT_EVALUATED:
            task_stack.append(graph.tasks[dependency])
      else:
        assert state == _EVALUATION_DONE
        task_stack.pop()

  return context
