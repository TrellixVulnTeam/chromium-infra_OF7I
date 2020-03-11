# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the hotlists servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from api import resource_name_converters as rnc
from api.v1 import hotlists_servicer
from api.v1 import converters
from api.v1.api_proto import hotlists_pb2
from framework import exceptions
from framework import monorailcontext
from framework import permissions
from testing import fake
from services import service_manager


class HotlistsServicerTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        features=fake.FeaturesService(),
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        config=fake.ConfigService(),
        user=fake.UserService(),
        usergroup=fake.UserGroupService())
    self.hotlists_svcr = hotlists_servicer.HotlistsServicer(
        self.services, make_rate_limiter=False)
    self.PAST_TIME = 12345
    self.user_1 = self.services.user.TestAddUser('user_111@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('user_222@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('user_333@example.com', 333)

    self.project_1 = self.services.project.TestAddProject(
        'proj', project_id=789)

    self.issue_1 = fake.MakeTestIssue(
        self.project_1.project_id, 1, 'sum', 'New', 111,
        project_name=self.project_1.project_name)
    self.issue_2 = fake.MakeTestIssue(
        self.project_1.project_id, 2, 'sum', 'New', 111,
        project_name=self.project_1.project_name)
    self.issue_3 = fake.MakeTestIssue(
        self.project_1.project_id, 3, 'sum', 'New', 111,
        project_name=self.project_1.project_name)
    self.issue_4 = fake.MakeTestIssue(
        self.project_1.project_id, 4, 'sum', 'New', 111,
        project_name=self.project_1.project_name)
    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)
    self.services.issue.TestAddIssue(self.issue_3)
    self.services.issue.TestAddIssue(self.issue_4)
    issue_ids = [self.issue_1.issue_id, self.issue_2.issue_id,
                 self.issue_3.issue_id, self.issue_4.issue_id]
    self.issue_ids_to_name = rnc.ConvertIssueNames(
        self.cnxn, issue_ids, self.services)

    hotlist_items = [
        (
            self.issue_4.issue_id, 31, self.user_3.user_id, self.PAST_TIME,
            'note5'),
        (
            self.issue_3.issue_id, 21, self.user_1.user_id, self.PAST_TIME,
            'note1'),
        (
            self.issue_2.issue_id, 11, self.user_2.user_id, self.PAST_TIME,
            'note2'),
        (
            self.issue_1.issue_id, 1, self.user_1.user_id, self.PAST_TIME,
            'note4')
    ]
    self.hotlist_1 = self.services.features.TestAddHotlist(
        'HotlistName', owner_ids=[self.user_1.user_id],
        editor_ids=[self.user_2.user_id], hotlist_item_fields=hotlist_items)
    self.hotlist_resource_name = rnc.ConvertHotlistName(
        self.hotlist_1.hotlist_id)

  def CallWrapped(self, wrapped_handler, *args, **kwargs):
    return wrapped_handler.wrapped(self.hotlists_svcr, *args, **kwargs)

  # TODO(crbug/monorail/7104): Add page_token tests when implemented.
  def testListHotlistItems(self):
    """We can list a Hotlist's HotlistItems."""
    request = hotlists_pb2.ListHotlistItemsRequest(
        parent=self.hotlist_resource_name, page_size=2, order_by='note,stars')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(
        self.hotlists_svcr.ListHotlistItems, mc, request)
    expected_items = converters.ConvertHotlistItems(
        mc.cnxn, mc.auth, self.hotlist_1.hotlist_id,
        [self.hotlist_1.items[1], self.hotlist_1.items[2]],
        self.services)
    self.assertEqual(
        response, hotlists_pb2.ListHotlistItemsResponse(items=expected_items))

  def testListHotlistItems_Empty(self):
    """We can return a response if the Hotlist has no items"""
    empty_hotlist = self.services.features.TestAddHotlist(
        'Empty',
        owner_ids=[self.user_1.user_id],
        editor_ids=[self.user_2.user_id],
        hotlist_item_fields=[])
    hotlist_resource_name = rnc.ConvertHotlistName(empty_hotlist.hotlist_id)
    request = hotlists_pb2.ListHotlistItemsRequest(parent=hotlist_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(
        self.hotlists_svcr.ListHotlistItems, mc, request)
    self.assertEqual(response, hotlists_pb2.ListHotlistItemsResponse(items=[]))

  def testRerankHotlistItems(self):
    """We can rerank a Hotlist."""
    item_names_dict = rnc.ConvertHotlistItemNames(
        self.cnxn, self.hotlist_1.hotlist_id,
        [item.issue_id for item in self.hotlist_1.items], self.services)
    request = hotlists_pb2.RerankHotlistItemsRequest(
        name=self.hotlist_resource_name,
        hotlist_items=[
            item_names_dict[self.issue_4.issue_id],
            item_names_dict[self.issue_3.issue_id]
        ],
        target_position=0)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.hotlists_svcr.RerankHotlistItems, mc, request)
    updated_hotlist = self.services.features.GetHotlist(
        self.cnxn, self.hotlist_1.hotlist_id)
    self.assertEqual(
        [item.issue_id for item in updated_hotlist.items],
        [self.issue_4.issue_id, self.issue_3.issue_id,
         self.issue_1.issue_id, self.issue_2.issue_id])

  def testRemoveHotlistItems(self):
    """We can remove items from a Hotlist."""
    issue_1_name = self.issue_ids_to_name[self.issue_1.issue_id]
    issue_2_name = self.issue_ids_to_name[self.issue_2.issue_id]
    request = hotlists_pb2.RemoveHotlistItemsRequest(
        parent=self.hotlist_resource_name, issues=[issue_1_name, issue_2_name])

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.hotlists_svcr.RemoveHotlistItems, mc, request)
    updated_hotlist = self.services.features.GetHotlist(
        self.cnxn, self.hotlist_1.hotlist_id)
    # The hotlist used to have 4 items and we've removed two.
    self.assertEqual(len(updated_hotlist.items), 2)

  def testGetHotlist(self):
    """We can get a Hotlist."""
    request = hotlists_pb2.GetHotlistRequest(name=self.hotlist_resource_name)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    api_hotlist = self.CallWrapped(self.hotlists_svcr.GetHotlist, mc, request)
    self.assertEqual(api_hotlist,
                     converters.ConvertHotlist(
                         mc.cnxn, mc.auth, self.hotlist_1, self.services))
