# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from shared.config import TRYJOBVERIFIER
from stats.analyzer import (
  AnalyzerGroup,
  CountAnalyzer,
  ListAnalyzer,
)

class PatchsetAnalyzer(AnalyzerGroup):
  def __init__(self):
    super(PatchsetAnalyzer, self).__init__(
      AttemptCount,
      AttemptDurations,
      BlockedOnClosedTreeDurations,
      BlockedOnThrottledTreeDurations,
      IssueCount,
      PatchsetCount,
      PatchsetCommitCount,
      PatchsetCommitDurations,
      PatchsetDurations,
      PatchsetFalseRejectCount,
      PatchsetRejectCount,
    )

class AttemptCount(CountAnalyzer):
  description = 'Number of CQ attempts made.'
  def new_patchset_attempts(self, issue, patchset, attempts):
    self.count += len(attempts)

class AttemptDurations(ListAnalyzer):
  description = 'Total time spent per CQ attempt.'
  unit = 'seconds'
  def new_patchset_attempts(self, issue, patchset, attempts):
    for attempt in attempts:
      delta = attempt[-1].timestamp - attempt[0].timestamp
      self.points.append((delta.total_seconds(), {
        'issue': issue,
        'patchset': patchset,
      }))

class BlockedOnClosedTreeDurations(ListAnalyzer):
  description = 'Time spent per committed patchset blocked on a closed tree.'
  unit = 'seconds'
  def new_patchset_attempts(self, issue, patchset, attempts):
    self.points.extend(duration_between_actions(issue, patchset, attempts,
      'patch_tree_closed', 'patch_ready_to_commit', False))

class BlockedOnThrottledTreeDurations(ListAnalyzer):
  description = 'Time spent per committed patchset blocked on a throttled tree.'
  unit = 'seconds'
  def new_patchset_attempts(self, issue, patchset, attempts):
    self.points.extend(duration_between_actions(issue, patchset, attempts,
      'patch_throttled', 'patch_ready_to_commit', False))

class IssueCount(CountAnalyzer):
  description = 'Number of issues processed by the CQ.'
  def __init__(self):
    super(IssueCount, self).__init__()
    self.issues = set()

  def new_patchset_attempts(self, issue, patchset, attempts):
    self.issues.add(issue)
    self.count = len(self.issues)

class PatchsetCount(CountAnalyzer):
  description = 'Number of patchsets processed by the CQ.'
  def __init__(self):
    super(PatchsetCount, self).__init__()
    self.patchsets = set()

  def new_patchset_attempts(self, issue, patchset, attempts):
    self.patchsets.add((issue, patchset))
    self.count = len(self.patchsets)

class PatchsetCommitCount(CountAnalyzer):
  description = 'Number of patchsets committed by the CQ.'
  def new_patchset_attempts(self, issue, patchset, attempts):
    if has_any_actions(attempts, ('patch_committed',)):
      self.count += 1

class PatchsetCommitDurations(ListAnalyzer):
  description = 'Time taken by the CQ to land a patch after passing all checks.'
  unit = 'seconds'
  def new_patchset_attempts(self, issue, patchset, attempts):
    self.points.extend(duration_between_actions(issue, patchset, attempts,
      'patch_committing', 'patch_committed', True))

class PatchsetDurations(ListAnalyzer):
  description = ('Total time spent in the CQ per patchset, '
                 'counts multiple CQ attempts as one.')
  unit = 'seconds'
  def new_patchset_attempts(self, issue, patchset, attempts):
    duration = 0
    for attempt in attempts:
      delta = attempt[-1].timestamp - attempt[0].timestamp
      duration += delta.total_seconds()
    self.points.append((duration, {
      'issue': issue,
      'patchset': patchset,
    }))

class PatchsetFalseRejectCount(CountAnalyzer):
  description = ('Number of patchsets rejected by the trybots '
                 'that eventually passed.')
  def new_patchset_attempts(self, issue, patchset, attempts):
    if (has_any_actions(attempts, ('verifier_retry', 'verifier_fail')) and
        has_any_actions(attempts, ('verifier_pass',))):
      self.count += 1

class PatchsetRejectCount(CountAnalyzer):
  description = 'Number of patchsets rejected by the trybots at least once.'
  def new_patchset_attempts(self, issue, patchset, attempts):
    if has_any_actions(attempts, ('verifier_retry', 'verifier_fail')):
      self.count += 1

def has_any_actions(attempts, actions):
  for attempt in attempts:
    for record in attempt:
      action = record.fields.get('action')
      if action in actions:
        if (action.startswith('verifier_') and
            record.fields.get('verifier') != TRYJOBVERIFIER):
          continue
        return True
  return False

def duration_between_actions(issue, patchset, attempts,
    action_start, action_end, requires_start):
  '''Counts the duration between start and end actions per patchset

  The end action must be present for the duration to be recorded.
  It is optional whether the start action needs to be present.
  An absent start action counts as a 0 duration.'''
  start_found = False
  duration_valid = False
  duration = 0
  for attempt in attempts:
    start_timestamp = None
    for record in attempt:
      if not start_timestamp and record.fields.get('action') == action_start:
        start_found = True
        start_timestamp = record.timestamp
      if ((start_found or not requires_start) and
          record.fields.get('action') == action_end):
        duration_valid = True
        if start_found:
          duration += (record.timestamp - start_timestamp).total_seconds()
          start_found = False
          start_timestamp = None
  if duration_valid:
    return ((duration, {
      'issue': issue,
      'patchset': patchset,
    }),)
  return ()
