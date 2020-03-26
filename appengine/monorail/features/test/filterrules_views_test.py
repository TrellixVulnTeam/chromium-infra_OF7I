# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Unittest for issue tracker views."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from features import filterrules_views
from proto import tracker_pb2
from testing import testing_helpers


class RuleViewTest(unittest.TestCase):

  def setUp(self):
    self.rule = tracker_pb2.FilterRule()
    self.rule.predicate = 'label:a label:b'

  def testNone(self):
    view = filterrules_views.RuleView(None, {})
    self.assertEqual('', view.action_type)
    self.assertEqual('', view.action_value)

  def testEmpty(self):
    view = filterrules_views.RuleView(self.rule, {})
    self.rule.predicate = ''
    self.assertEqual('', view.predicate)
    self.assertEqual('', view.action_type)
    self.assertEqual('', view.action_value)

  def testDefaultStatus(self):
    self.rule.default_status = 'Unknown'
    view = filterrules_views.RuleView(self.rule, {})
    self.assertEqual('label:a label:b', view.predicate)
    self.assertEqual('default_status', view.action_type)
    self.assertEqual('Unknown', view.action_value)

  def testDefaultOwner(self):
    self.rule.default_owner_id = 111
    view = filterrules_views.RuleView(
        self.rule, {
            111: testing_helpers.Blank(email='jrobbins@chromium.org')})
    self.assertEqual('label:a label:b', view.predicate)
    self.assertEqual('default_owner', view.action_type)
    self.assertEqual('jrobbins@chromium.org', view.action_value)

  def testAddCCs(self):
    self.rule.add_cc_ids.extend([111, 222])
    view = filterrules_views.RuleView(
        self.rule, {
            111: testing_helpers.Blank(email='jrobbins@chromium.org'),
            222: testing_helpers.Blank(email='jrobbins@gmail.com')})
    self.assertEqual('label:a label:b', view.predicate)
    self.assertEqual('add_ccs', view.action_type)
    self.assertEqual(
        'jrobbins@chromium.org, jrobbins@gmail.com', view.action_value)

  def testAddLabels(self):
    self.rule.add_labels.extend(['Hot', 'Cool'])
    view = filterrules_views.RuleView(self.rule, {})
    self.assertEqual('label:a label:b', view.predicate)
    self.assertEqual('add_labels', view.action_type)
    self.assertEqual('Hot, Cool', view.action_value)

  def testAlsoNotify(self):
    self.rule.add_notify_addrs.extend(['a@dom.com', 'b@dom.com'])
    view = filterrules_views.RuleView(self.rule, {})
    self.assertEqual('label:a label:b', view.predicate)
    self.assertEqual('also_notify', view.action_type)
    self.assertEqual('a@dom.com, b@dom.com', view.action_value)
