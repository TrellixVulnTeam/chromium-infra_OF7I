# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import copy
import dataclasses
import logging
import typing

from google.protobuf import any_pb2

from chromeperf.engine import actions
from chromeperf.engine import event as event_module

# Traversal states used in the graph traversal. We use these as marks for when
# vertices are traversed, as how we would implement graph colouring in a graph
# traversal (like Depth First Search).
_NOT_EVALUATED, _CHILDREN_PENDING, _EVALUATION_DONE = (0, 1, 2)


class TooManyIterationsError(Exception): pass


@dataclasses.dataclass
class TaskVertex:
    id: str
    vertex_type: str
    payload: any_pb2.Any
    state: str = 'unknown'


@dataclasses.dataclass(frozen=True)
class Dependency:
    from_: str
    to: str


@dataclasses.dataclass
class TaskGraph:
    vertices: typing.List[TaskVertex]
    edges: typing.List[Dependency]


# TODO(fancl): Migrate all these namedtuple to dataclass
class _AdjacencyPair(
        collections.namedtuple('_AdjacencyPair', ('task', 'dependencies'))):
    __slots__ = ()


@dataclasses.dataclass
class NormalizedTask:
    id: str
    task_type: str
    payload: typing.Any
    state: str
    dependencies: typing.List[str]


class _ReconstitutedTaskGraph(
        collections.namedtuple('_ReconstitutedTaskGraph',
                               ('terminal_tasks', 'tasks'))):
    # terminal_tasks: str
    # tasks: typing.Dict[str, _AdjacencyPair]
    __slots__ = ()


class Error(Exception):
    pass


class InvalidDependencyError(Error):
    pass


class MalformedGraphError(Error):
    pass


def _preprocess_graph(graph: TaskGraph) -> _ReconstitutedTaskGraph:
    if not graph.vertices:
        raise MalformedGraphError('No vertices in graph.')
    vertices = {
        vertex.id: _AdjacencyPair(task=vertex, dependencies=[])
        for vertex in graph.vertices
    }
    for dep in graph.edges:
        if dep.from_ not in vertices:
            raise InvalidDependencyError(
                'Invalid dependency: {} not a known vertex'.format(dep.from_))
        if dep.to not in vertices:
            raise InvalidDependencyError(
                'Invalid dependency: {} not a known vertex'.format(dep.to))
        vertices[dep.from_].dependencies.append(dep.to)
    has_dependents = set()
    for _, deps in vertices.values():
        has_dependents |= set(deps)
    terminal_tasks = [
        v.id for v, _ in vertices.values() if v.id not in has_dependents
    ]
    if not terminal_tasks:
        raise MalformedGraphError('Provided graph has no terminal vertices.')
    return _ReconstitutedTaskGraph(terminal_tasks=terminal_tasks,
                                   tasks=vertices)


Context = typing.Mapping[str, typing.Any]
Action = typing.Callable[[typing.Mapping[str, typing.Any]], None]
Evaluator = typing.Callable[[NormalizedTask, event_module.Event, Context],
                            typing.List[Action]]


def evaluate_graph(event: event_module.Event, evaluator: Evaluator,
                   load_graph: typing.Callable[[], TaskGraph],
                   max_iterations=10000):
    """Applies an evaluator given a task in the task graph and an event as input.

    This function implements a depth-first search traversal of the task graph
    and applies the `evaluator` given a task and the event input in
    post-order traversal. We start the DFS from the terminal tasks (those
    that don't have dependencies) and call the `evaluator` function with a
    representation of the task in the graph, an `event` as input, and an
    accumulator argument.

    `load_graph` is a callable which returns a `TaskGraph` instance, or one
    that has similar properties.

    The `evaluator` must be a callable which accepts three arguments:

      - task: a NormalizedTask instance, representing a task in the graph.
      - event: an object whose shape/type is defined by the caller of the
        `Evaluate` function and that the evaluator can handle.
      - accumulator: a dictionary which is mutable which is valid in the
        scope of a traversal of the graph.

    The `evaluator` must return either None or an iterable of callables which
    take a single argument, which is the accumulator at the end of a
    traversal.

    Events are free-form but usually are dictionaries which constitute inputs
    that are external to the task graph evaluation. This could model events
    in an event-driven evaluation of tasks, or synthetic inputs to the
    system. It is more important that the `event` information is known to the
    evaluator implementation, and is provided as-is to the evaluator in this
    function.

    The Evaluate function will keep iterating while there are actions still
    being produced by the evaluator. When there are no more actions to run,
    the Evaluate function will return the most recent traversal's
    accumulator.
    """

    context = {}
    collected_actions = [actions.Noop()]
    iter_count = 0
    while collected_actions:
        logging.debug(f'{len(collected_actions)} collected_actions')
        for action in collected_actions:
            iter_count += 1
            if iter_count > max_iterations: raise TooManyIterationsError
            #logging.debug('Running action: %s...', str(action)[:80])

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
            iter_count += 1
            if iter_count > max_iterations: raise TooManyIterationsError
            task, deps = task_stack[-1]
            state = vertex_states.get(task.id, _NOT_EVALUATED)
            if state == _CHILDREN_PENDING:
                # We provide a deep copy of the task, to ensure that the evaluator
                # cannot modify the graph we're working with directly.
                result_actions = evaluator(
                    NormalizedTask(task.id,
                                   task.vertex_type,
                                   copy.deepcopy(task.payload),
                                   task.state, copy.deepcopy(deps)),
                    event, context)
                if result_actions:
                    collected_actions.extend(result_actions)
                vertex_states[task.id] = _EVALUATION_DONE
            elif state == _NOT_EVALUATED:
                # This vertex has not been evaluated yet, we should traverse the
                # dependencies.
                vertex_states[task.id] = _CHILDREN_PENDING
                for dependency in deps:
                    if vertex_states.get(dependency,
                                         _NOT_EVALUATED) == _NOT_EVALUATED:
                        task_stack.append(graph.tasks[dependency])
            else:
                assert state == _EVALUATION_DONE
                task_stack.pop()

    return context
