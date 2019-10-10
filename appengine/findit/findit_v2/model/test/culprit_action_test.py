# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import hashlib
from datetime import datetime

from findit_v2.model.gitiles_commit import Culprit
from findit_v2.model.culprit_action import CulpritAction
from waterfall.test import wf_testcase


class CulpritActionTest(wf_testcase.WaterfallTestCase):

  def testGetRecentActions(self):
    culprits = []
    for i in range(6):
      c = Culprit.Create('x.googlesource.com', 'project', 'refs/heads/master',
                         hashlib.sha1(str(i)).hexdigest(), i)
      c.put()
      culprits.append(c)

    # Create different numbers of each action type
    CulpritAction.Create(
        culprits[0],
        CulpritAction.REVERT,
        revert_committed=True,
        revert_change=10).put()
    CulpritAction.Create(
        culprits[1],
        CulpritAction.REVERT,
        revert_committed=False,
        revert_change=1).put()
    CulpritAction.Create(
        culprits[2],
        CulpritAction.REVERT,
        revert_committed=False,
        revert_change=2).put()
    CulpritAction.Create(culprits[3], CulpritAction.CULPRIT_NOTIFIED).put()
    CulpritAction.Create(culprits[4], CulpritAction.CULPRIT_NOTIFIED).put()
    CulpritAction.Create(culprits[5], CulpritAction.CULPRIT_NOTIFIED).put()

    self.assertEquals(
        3,
        len(
            CulpritAction.GetRecentActionsByType(
                CulpritAction.CULPRIT_NOTIFIED)))
    self.assertEquals(
        2,
        len(
            CulpritAction.GetRecentActionsByType(
                CulpritAction.REVERT, revert_committed=False)))
    self.assertEquals(
        1,
        len(
            CulpritAction.GetRecentActionsByType(
                CulpritAction.REVERT, revert_committed=True)))

  def testCreateAction(self):
    c = Culprit.Create('x.googlesource.com', 'project', 'refs/heads/master',
                       hashlib.sha1('42').hexdigest(), 42)
    c.put()
    params = [
        [None, CulpritAction.REVERT],
        [c, 42],
    ]
    for p in params:
      with self.assertRaises(Exception):  # BadValueError, AttributeError.
        CulpritAction.Create(*p).put()

  def testGetAction(self):
    c = Culprit.Create('x.googlesource.com', 'project', 'refs/heads/master',
                       hashlib.sha1('666').hexdigest(), 666)
    c.put()
    self.assertIsNone(CulpritAction.CreateKey(c).get())
    action = CulpritAction.Create(c, CulpritAction.REVERT, revert_change=666)
    # To avoid conflict with testGetRecentActions.
    action.create_timestamp = datetime(2010, 1, 1, 0, 0, 0)
    action.put()
    retrieved = CulpritAction.CreateKey(c).get()
    self.assertEqual(retrieved, action)
    self.assertEqual('https://x-review.googlesource.com/c/project/+/666',
                     retrieved.GetRevertURL())
