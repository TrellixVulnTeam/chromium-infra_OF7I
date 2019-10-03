# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import unittest
from collections import namedtuple

import infra.services.bugdroid.log_parser as log_parser

LogEntry = namedtuple('LogEntry', 'msg')

class BugLineParserTest(unittest.TestCase):
  def test_matching_bug(self):
    for bug, bug_line in [
        # Keep distinct bug numbers for easy search in case of test failures.
        (123, 'BUG=123'),
        (124, 'Bug: 124'),
        ('chromium:125', 'Bugs: chromium:125'),
        (126, 'Fixed: 126'),
        ('chromium:127', 'Fixed : chromium:127'),
    ]:
      m = log_parser.BUG_LINE_REGEX.match(bug_line)
      self.assertIsNotNone(m, '"%s" line must be matched' % bug_line)
      self.assertEqual(m.groups()[-1], str(bug),
                       '"%s" line matched to %s but %s expected.' % (
                       bug_line, m.groups()[-1], str(bug)))

  def test_not_matching_bug(self):
    for bug_line in [
        # Keep distinct bug numbers for easy search in case of test failures.
        'BUGr=123',
        'BUGS/124',
        'someBugs:',
        'FIXED=126',  # FIXED= is not supported; only Fixed: is.
        'someFixed:',
    ]:
      m = log_parser.BUG_LINE_REGEX.match(bug_line)
      self.assertIsNone(m, '"%s" line must not be matched (got %s)' %
                           (bug_line, m.groups()) if m else None)

  def test_get_issues(self):
    test_cases = [
        ({'bugs': {'default': [123]}, 'fixed': {}}, 'Bug: 123'),
        ({'bugs': {'default': [123]}, 'fixed': {}}, 'Bug: #123'),
        ({'bugs': {'default': [123]}, 'fixed': {}}, 'Bug: crbug.com/123'),
        ({'bugs': {'proj': [123]}, 'fixed': {}}, 'Bug: proj:123'),
        ({'bugs': {'proj': [123]}, 'fixed': {}}, 'Bug: proj:#123'),
        ({'bugs': {'proj': [123]}, 'fixed': {}}, 'Bug: crbug.com/proj/123'),
        ({'bugs': {'default': [11]}, 'fixed': {'default': [11]}}, 'Fixed: 11'),
        ({'bugs': {'default': [12]}, 'fixed': {'default': [12]}}, 'Fixed: #12'),
        ({'bugs': {'default': [126]}, 'fixed': {'default': [126]}},
         'Fixed: crbug.com/126'),
        ({'bugs': {'proj': [127]}, 'fixed': {'proj': [127]}},
         'Fixed: proj:127'),
        ({'bugs': {'proj': [128]}, 'fixed': {'proj': [128]}},
         'Fixed: proj:#128'),
        ({'bugs': {'proj': [129]}, 'fixed': {'proj': [129]}},
         'Fixed: crbug.com/proj/129'),
    ]
    for expected, bug_line in test_cases:
      log_entry = LogEntry(msg=bug_line)
      self.assertEqual(expected, log_parser.get_issues(log_entry, 'default'))

  def test_not_get_issues(self):
    test_cases = [
        'Bug: foo123',
        'Bug: 123.5',
        'Bug: foocrbug.com/123',
        'Bug: invalid_name:123',
        'Bug: proj:#123.5',
        'Bug: foocrbug.com/proj/123',
        'Fixed: foo123',
        'Fixed: 123.5',
        'Fixed: foocrbug.com/123',
        'Fixed: invalid_name:123',
        'Fixed: proj:#123.5',
        'Fixed: foocrbug.com/proj/123',
    ]
    for bug_line in test_cases:
      log_entry = LogEntry(msg=bug_line)
      self.assertEqual({'bugs': {}, 'fixed': {}},
                       log_parser.get_issues(log_entry, 'default'))

  def test_should_send_email(self):
    for test_case, result in [
      (None, True),
      ("Random stuff\nhereman\nBug: 12", True),
      ("Bugdroid-Send-Email: yaaaman", True),
      ("Bugdroid-Send-Email: no", False),
      ("Bugdroid-Send-Email: false", False),
      ("""
Whitespace CL to test bugdroid

BUG=637024

Change-Id: Ib273794c41ea206f11c33fceac2182a0b8e637ee
Bugdroid-Send-Email: False
Reviewed-on: https://chromium-review.googlesource.com/367879
Reviewed-by: Daniel Jacques <dnj@chromium.org>
Commit-Queue: Stephen Martinis <martiniss@chromium.org>
       """, False),
      ("""
Whitespace CL to test bugdroid

BUG=637024
I love that Bugdroid-Send-Email: False doesn't work if it's not
in proper git footer style!
Bugdroid-Send-Email: False

Change-Id: Ib273794c41ea206f11c33fceac2182a0b8e637ee
Reviewed-on: https://chromium-review.googlesource.com/367879
Reviewed-by: Daniel Jacques <dnj@chromium.org>
Commit-Queue: Stephen Martinis <martiniss@chromium.org>
       """, True),
    ]:
      self.assertEqual(
          result, log_parser.should_send_email(test_case), test_case)
