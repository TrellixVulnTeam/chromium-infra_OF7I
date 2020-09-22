# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Collection of evaluator combinators.

This module exports common evaluator types which are used to compose multiple
specific evaluators. Use these combinators to compose a single evaluator from
multiple specific evaluator implementations, to be used when calling
chromeperf.engine.evaluator.evaluate_graph(...).
"""
import dataclasses
import itertools

from google.protobuf import any_pb2


class NoopEvaluator(object):
    def __call__(self, *_):
        return None

    def __str__(self):
        return 'NoopEvaluator()'


@dataclasses.dataclass(frozen=True)
class TaskContext:
    state: str
    payload: any_pb2.Any


class TaskPayloadLiftingEvaluator(object):
    """An evaluator that copies task payload and state to the accumulator.

    A common pattern in evaluators in Pinpoint is lifting, or copying, the
    task's payload into the accumulator potentially after changes to the
    task's in-memory payload has been made. This evaluator can be sequenced
    before or after other evaluators to copy the payload of a task to the
    accumulator.

    Side-effects:

      - Copies the payload and state of a task into an entry in the
        accumulator keyed by the task's id.

      - The state of the task becomes an entry in the dict with the key
        'state' as if it was part of the task's payload.

    Returns None.
    """
    def __init__(self,
                 exclude_keys=None,
                 exclude_event_types=None,
                 include_keys=None,
                 include_event_types=None):
        self._exclude_keys = exclude_keys or {}
        self._exclude_event_types = exclude_event_types or {}
        self._include_keys = include_keys
        self._include_event_types = include_event_types

    def __call__(self, task, event, context):
        if (self._include_event_types is not None
                and event.type not in self._include_event_types
            ) or event.type in self._exclude_event_types:
            return None

        context.update({
            task.id:
            TaskContext(state=task.state, payload=task.payload),
        })
        return None


class SequenceEvaluator(object):
    def __init__(self, evaluators):
        if not evaluators:
            raise ValueError('Argument `evaluators` must not be empty.')

        self._evaluators = evaluators

    def __call__(self, task, event, accumulator):
        def Flatten(seqs):
            return list(itertools.chain(*seqs))

        return Flatten([
            evaluator(task, event, accumulator) or []
            for evaluator in self._evaluators
        ])


class FilteringEvaluator(object):
    def __init__(self, predicate, delegate, alternative=None):
        if not predicate:
            raise ValueError('Argument `predicate` must not be empty.')
        if not delegate:
            raise ValueError('Argument `delegate` must not be empty.')

        self._predicate = predicate
        self._delegate = delegate
        self._alternative = alternative or NoopEvaluator()

    def __call__(self, *args):
        if self._predicate(*args):
            return self._delegate(*args)
        return self._alternative(*args)


class DispatchEvaluatorBase(object):
    def __init__(self, evaluator_map, default_evaluator=None):
        if not evaluator_map and not default_evaluator:
            raise ValueError(
                'Either one of evaluator_map or default_evaluator '
                'must be provided.')

        self._evaluator_map = evaluator_map
        self._default_evaluator = default_evaluator or NoopEvaluator()

    def _Key(self, task, event):
        raise NotImplementedError('Override this in the subclass.')

    def __call__(self, task, event, accumulator):
        handler = self._evaluator_map.get(self._Key(task, event),
                                          self._default_evaluator)
        return handler(task, event, accumulator)


class DispatchByEventTypeEvaluator(DispatchEvaluatorBase):
    @staticmethod
    def _Key(_, event):
        return event.type


class DispatchByTaskState(DispatchEvaluatorBase):
    @staticmethod
    def _Key(task, _):
        return task.state


class DispatchByTaskType(DispatchEvaluatorBase):
    @staticmethod
    def _Key(task, _):
        return task.task_type


class Selector(FilteringEvaluator):
    def __init__(self,
                 task_type=None,
                 event_type=None,
                 predicate=None,
                 **kwargs):
        def Predicate(task, event, accumulator):
            matches = False
            if task_type is not None:
                matches |= task_type == task.task_type
            if event_type is not None:
                matches |= event_type == event.type
            if predicate is not None:
                matches |= predicate(task, event, accumulator)
            return matches

        super(Selector, self).__init__(
            predicate=Predicate,
            delegate=TaskPayloadLiftingEvaluator(**kwargs),
        )
