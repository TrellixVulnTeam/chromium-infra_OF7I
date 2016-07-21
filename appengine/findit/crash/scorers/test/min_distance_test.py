# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
from crash.callstack import StackFrame
from crash.results import Result, MatchResult
from crash.scorers.min_distance import MinDistance
from crash.scorers.test.scorer_test_suite import ScorerTestSuite


class MinDistanceTest(ScorerTestSuite):

  def testGetMetric(self):
    dummy_changelog = self._GetDummyChangeLog()
    match_result = MatchResult(dummy_changelog, 'src/', '')
    match_result.file_to_analysis_info = {
        'file': {'min_distance': 0, 'min_distance_frame': None}
    }

    self.assertEqual(MinDistance().GetMetric(match_result), 0)

    result = Result(dummy_changelog, 'src/', '')
    self.assertEqual(MinDistance().GetMetric(result), float('inf'))

  def testScore(self):
    self.assertEqual(MinDistance().Score(0), 1)
    self.assertEqual(MinDistance().Score(30), 0.8)
    self.assertEqual(MinDistance().Score(60), 0)

  def testReason(self):
    self.assertEqual(MinDistance().Reason(0, 1),
                     ('MinDistance', 1, 'Minimum distance is 0'))
    self.assertEqual(MinDistance().Reason(60, 0),
                     None)

  def testChangedFiles(self):
    dummy_changelog = self._GetDummyChangeLog()
    result = MatchResult(dummy_changelog, 'src/', '')
    frame = StackFrame(0, 'src/', 'func', 'f.cc', 'a/b/src/f.cc', [2],
                       repo_url='https://repo_url')
    result.file_to_stack_infos = {
        'src/f.cc': [(frame, 0)]
    }
    result.file_to_analysis_info = {
        'src/f.cc': {'min_distance': 0, 'min_distance_frame': frame}
    }

    self.assertEqual(MinDistance().ChangedFiles(result),
                     [{'file': 'f.cc',
                       'blame_url': ('https://repo_url/+blame/%s/f.cc#2' %
                                     dummy_changelog.revision),
                       'info': 'Minimum distance (LOC) 0, frame #0'}])

