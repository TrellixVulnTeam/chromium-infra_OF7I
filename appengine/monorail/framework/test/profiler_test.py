# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Test for monorail.framework.profiler."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from framework import profiler


class MockPatchResponse(object):
  def execute(self):
    pass


class MockCloudTraceProjects(object):
  def __init__(self):
    self.patch_response = MockPatchResponse()
    self.project_id = None
    self.body = None

  def patchTraces(self, projectId, body):
    self.project_id = projectId
    self.body = body
    return self.patch_response


class MockCloudTraceApi(object):
  def __init__(self):
    self.mock_projects = MockCloudTraceProjects()

  def projects(self):
    return self.mock_projects


class ProfilerTest(unittest.TestCase):

  def testTopLevelPhase(self):
    prof = profiler.Profiler()
    self.assertEqual(prof.current_phase.name, 'overall profile')
    self.assertEqual(prof.current_phase.parent, None)
    self.assertEqual(prof.current_phase, prof.top_phase)
    self.assertEqual(prof.next_color, 0)

  def testSinglePhase(self):
    prof = profiler.Profiler()
    self.assertEqual(prof.current_phase.name, 'overall profile')
    with prof.Phase('test'):
      self.assertEqual(prof.current_phase.name, 'test')
      self.assertEqual(prof.current_phase.parent.name, 'overall profile')
    self.assertEqual(prof.current_phase.name, 'overall profile')
    self.assertEqual(prof.next_color, 1)

  def testSinglePhase_SuperLongName(self):
    prof = profiler.Profiler()
    self.assertEqual(prof.current_phase.name, 'overall profile')
    long_name = 'x' * 1000
    with prof.Phase(long_name):
      self.assertEqual(
          'x' * profiler.MAX_PHASE_NAME_LENGTH, prof.current_phase.name)

  def testSubphaseExecption(self):
    prof = profiler.Profiler()
    try:
      with prof.Phase('foo'):
        with prof.Phase('bar'):
          pass
        with prof.Phase('baz'):
          raise Exception('whoops')
    except Exception as e:
      self.assertEqual(e.message, 'whoops')
    finally:
      self.assertEqual(prof.current_phase.name, 'overall profile')
      self.assertEqual(prof.top_phase.subphases[0].subphases[1].name, 'baz')

  def testSpanJson(self):
    mock_trace_api = MockCloudTraceApi()
    mock_trace_context = '1234/5678;xxxxx'

    prof = profiler.Profiler(mock_trace_context, mock_trace_api)
    with prof.Phase('foo'):
      with prof.Phase('bar'):
        pass
      with prof.Phase('baz'):
        pass

    # Shouldn't this be automatic?
    prof.current_phase.End()

    self.assertEqual(prof.current_phase.name, 'overall profile')
    self.assertEqual(prof.top_phase.subphases[0].subphases[1].name, 'baz')
    span_json = prof.top_phase.SpanJson()
    self.assertEqual(len(span_json), 4)

    for span in span_json:
      self.assertTrue(span['endTime'] > span['startTime'])

    # pylint: disable=unbalanced-tuple-unpacking
    span1, span2, span3, span4 = span_json

    self.assertEqual(span1['name'], 'overall profile')
    self.assertEqual(span2['name'], 'foo')
    self.assertEqual(span3['name'], 'bar')
    self.assertEqual(span4['name'], 'baz')

    self.assertTrue(span1['startTime'] < span2['startTime'])
    self.assertTrue(span1['startTime'] < span3['startTime'])
    self.assertTrue(span1['startTime'] < span4['startTime'])

    self.assertTrue(span1['endTime'] > span2['endTime'])
    self.assertTrue(span1['endTime'] > span3['endTime'])
    self.assertTrue(span1['endTime'] > span4['endTime'])


  def testReportCloudTrace(self):
    mock_trace_api = MockCloudTraceApi()
    mock_trace_context = '1234/5678;xxxxx'

    prof = profiler.Profiler(mock_trace_context, mock_trace_api)
    with prof.Phase('foo'):
      with prof.Phase('bar'):
        pass
      with prof.Phase('baz'):
        pass

    # Shouldn't this be automatic?
    prof.current_phase.End()

    self.assertEqual(prof.current_phase.name, 'overall profile')
    self.assertEqual(prof.top_phase.subphases[0].subphases[1].name, 'baz')

    prof.ReportTrace()
    self.assertEqual(mock_trace_api.mock_projects.project_id, 'testing-app')
