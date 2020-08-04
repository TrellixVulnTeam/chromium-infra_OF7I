# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import logging
import functools
import unittest
import sys

from chromeperf.engine import evaluator

FakeEvent = collections.namedtuple('Event', ('type', 'status', 'payload'))


class TestEvaluator:

  def test_simple_case(self):
    task_graph = evaluator.TaskGraph(
        vertices=[
            evaluator.TaskVertex(
                id='input0', vertex_type='constant', payload={'value': 0}),
            evaluator.TaskVertex(
                id='input1', vertex_type='constant', payload={'value': 1}),
            evaluator.TaskVertex(
                id='plus', vertex_type='operator+', payload={}),
        ],
        edges=[
            evaluator.Dependency(from_='plus', to='input0'),
            evaluator.Dependency(from_='plus', to='input1'),
        ],
    )

    def _adder_evaluator(task, _, context):
      if task.task_type == 'constant':
        context[task.id] = task.payload.get('value', 0)
      elif task.task_type == 'operator+':
        inputs = [context.get(dep) for dep in task.dependencies]
        context[task.id] = functools.reduce(lambda a, v: a + v, inputs)

    def _load_graph():
      return task_graph

    context = evaluator.evaluate_graph({}, _adder_evaluator, _load_graph)
    assert 1 == context.get('plus')

  def test_call_count_evaluator(self):

    def _load_graph():
      return evaluator.TaskGraph(
          vertices=[
              evaluator.TaskVertex(id='leaf_0', vertex_type='node', payload={}),
              evaluator.TaskVertex(id='leaf_1', vertex_type='node', payload={}),
              evaluator.TaskVertex(id='parent', vertex_type='node', payload={}),
          ],
          edges=[
              evaluator.Dependency(from_='parent', to='leaf_0'),
              evaluator.Dependency(from_='parent', to='leaf_1'),
          ])

    calls = {}

    def evaluate_call_counts(task, event, context):
      logging.info('Evaluate(%s, %s, %s) called.', task.id, event, context)
      calls[task.id] = calls.get(task.id, 0) + 1
      return None

    evaluator.evaluate_graph('test', evaluate_call_counts, _load_graph)
    assert {
        'leaf_0': 1,
        'leaf_1': 1,
        'parent': 1,
    } == calls
