# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Collection of evaluator predicates.

This module contains common filters/predicates that are useful in composing
combinators.FilteringEvaluator instances that deal with tasks and events.
Filters are callables that return a boolean while Evaluators are callables
that return an iterable of actions (or None).
"""


class Not(object):

  def __init__(self, filter_):
    self._filter = filter_

  def __call__(self, *args):
    return not self._filter(*args)


class All(object):

  def __init__(self, *filters):
    self._filters = filters

  def __call__(self, *args):
    return all(f(*args) for f in self._filters)


class Any(object):

  def __init__(self, *filters):
    self._filters = filters

  def __call__(self, *args):
    return any(f(*args) for f in self._filters)


class TaskTypeEq(object):

  def __init__(self, task_type_filter):
    self._task_type_filter = task_type_filter

  def __call__(self, task, *_):
    return task.task_type == self._task_type_filter


class TaskStateIn(object):

  def __init__(self, include_types):
    self._include_types = include_types

  def __call__(self, task, *_):
    return task.state in self._include_types


class TaskIsEventTarget(object):

  def __call__(self, task, event, _):
    return event.target_task is None or event.target_task == task.id
