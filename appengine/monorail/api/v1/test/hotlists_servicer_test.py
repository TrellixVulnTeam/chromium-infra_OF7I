# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the hotlists servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from google.protobuf import empty_pb2
from google.protobuf import field_mask_pb2

from api import resource_name_converters as rnc
from api.v1 import hotlists_servicer
from api.v1 import converters
from api.v1.api_proto import hotlists_pb2
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import user_objects_pb2
from framework import exceptions
from framework import monorailcontext
from framework import permissions
from features import features_constants
from testing import fake
from services import features_svc
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
    self.converter = None
    self.PAST_TIME = 12345
    self.user_1 = self.services.user.TestAddUser('user_111@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('user_222@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('user_333@example.com', 333)

    user_ids = [self.user_1.user_id, self.user_2.user_id, self.user_3.user_id]
    self.user_ids_to_name = rnc.ConvertUserNames(user_ids)

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
    self.issue_5 = fake.MakeTestIssue(
        self.project_1.project_id, 5, 'sum', 'New', 111,
        project_name=self.project_1.project_name)
    self.issue_6 = fake.MakeTestIssue(
        self.project_1.project_id, 6, 'sum', 'New', 111,
        project_name=self.project_1.project_name)
    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)
    self.services.issue.TestAddIssue(self.issue_3)
    self.services.issue.TestAddIssue(self.issue_4)
    self.services.issue.TestAddIssue(self.issue_5)
    self.services.issue.TestAddIssue(self.issue_6)
    issue_ids = [
        self.issue_1.issue_id, self.issue_2.issue_id, self.issue_3.issue_id,
        self.issue_4.issue_id, self.issue_5.issue_id, self.issue_6.issue_id
    ]
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
        'HotlistName',
        summary='summary',
        description='description',
        owner_ids=[self.user_1.user_id],
        editor_ids=[self.user_2.user_id],
        hotlist_item_fields=hotlist_items,
        default_col_spec='',
        is_private=True)
    self.hotlist_resource_name = rnc.ConvertHotlistName(
        self.hotlist_1.hotlist_id)

  def CallWrapped(self, wrapped_handler, mc, *args, **kwargs):
    self.converter = converters.Converter(mc, self.services)
    self.hotlists_svcr.converter = self.converter
    return wrapped_handler.wrapped(self.hotlists_svcr, mc, *args, **kwargs)

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
    expected_items = self.converter.ConvertHotlistItems(
        self.hotlist_1.hotlist_id,
        [self.hotlist_1.items[1], self.hotlist_1.items[2]])
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

  def testListHotlistItems_InvalidPageSize(self):
    """We raise an exception if `page_size` is negative."""
    request = hotlists_pb2.ListHotlistItemsRequest(
        parent=self.hotlist_resource_name, page_size=-1)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    with self.assertRaises(exceptions.InputException):
      self.CallWrapped(self.hotlists_svcr.ListHotlistItems, mc, request)

  def testListHotlistItems_DefaultPageSize(self):
    """We use our default page size when no `page_size` is given."""
    request = hotlists_pb2.ListHotlistItemsRequest(
        parent=self.hotlist_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(
        self.hotlists_svcr.ListHotlistItems, mc, request)
    self.assertEqual(
        len(response.items),
        min(
            features_constants.DEFAULT_RESULTS_PER_PAGE,
            len(self.hotlist_1.items)))

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

  def testAddHotlistItems(self):
    """We can add items to a Hotlist."""
    issue_5_name = self.issue_ids_to_name[self.issue_5.issue_id]
    issue_6_name = self.issue_ids_to_name[self.issue_6.issue_id]
    request = hotlists_pb2.AddHotlistItemsRequest(
        parent=self.hotlist_resource_name, issues=[issue_5_name, issue_6_name])

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.hotlists_svcr.AddHotlistItems, mc, request)
    updated_hotlist = self.services.features.GetHotlist(
        self.cnxn, self.hotlist_1.hotlist_id)
    # The hotlist used to have 4 items and we've added two.
    self.assertEqual(len(updated_hotlist.items), 6)

  def testRemoveHotlistEditors(self):
    """We can remove editors from a Hotlist."""
    user_2_name = self.user_ids_to_name[self.user_2.user_id]
    request = hotlists_pb2.RemoveHotlistEditorsRequest(
        name=self.hotlist_resource_name, editors=[user_2_name])

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.hotlists_svcr.RemoveHotlistEditors, mc, request)
    updated_hotlist = self.services.features.GetHotlist(
        self.cnxn, self.hotlist_1.hotlist_id)
    # User 2 was the only editor in the hotlist, and we removed them.
    self.assertEqual(len(updated_hotlist.editor_ids), 0)

  def testGetHotlist(self):
    """We can get a Hotlist."""
    request = hotlists_pb2.GetHotlistRequest(name=self.hotlist_resource_name)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    api_hotlist = self.CallWrapped(self.hotlists_svcr.GetHotlist, mc, request)
    self.assertEqual(api_hotlist, self.converter.ConvertHotlist(self.hotlist_1))

  def testGatherHotlistsForUser(self):
    """We can get all visible hotlists of a user."""
    request = hotlists_pb2.GatherHotlistsForUserRequest(
        user=self.user_ids_to_name[self.user_2.user_id])

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(
        self.hotlists_svcr.GatherHotlistsForUser, mc, request)

    user_names_by_id = rnc.ConvertUserNames(
        [self.user_2.user_id, self.user_1.user_id])
    expected_api_hotlists = [
        feature_objects_pb2.Hotlist(
            name=self.hotlist_resource_name,
            display_name='HotlistName',
            summary='summary',
            description='description',
            hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
                'PRIVATE'),
            owner=user_names_by_id[self.user_1.user_id],
            editors=[user_names_by_id[self.user_2.user_id]])
    ]
    self.assertEqual(
        response,
        hotlists_pb2.GatherHotlistsForUserResponse(
            hotlists=expected_api_hotlists))

  def testUpdateHotlist_AllFields(self):
    """We can update a Hotlist."""
    request = hotlists_pb2.UpdateHotlistRequest(
        update_mask=field_mask_pb2.FieldMask(
            paths=[
                'summary', 'description', 'default_columns', 'hotlist_privacy',
                'display_name'
            ]),
        hotlist=feature_objects_pb2.Hotlist(
            name=self.hotlist_resource_name,
            display_name='newName',
            summary='new summary',
            description='new description',
            default_columns=[
                issue_objects_pb2.IssuesListColumn(column='new-chicken-egg')
            ],
            hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
                'PUBLIC')))
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    api_hotlist = self.CallWrapped(
        self.hotlists_svcr.UpdateHotlist, mc, request)
    user_names_by_id = rnc.ConvertUserNames(
        [self.user_2.user_id, self.user_1.user_id])
    expected_hotlist = feature_objects_pb2.Hotlist(
        name=self.hotlist_resource_name,
        display_name='newName',
        summary='new summary',
        description='new description',
        default_columns=[
            issue_objects_pb2.IssuesListColumn(column='new-chicken-egg')
        ],
        hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
            'PUBLIC'),
        owner=user_names_by_id[self.user_1.user_id],
        editors=[user_names_by_id[self.user_2.user_id]])
    self.assertEqual(api_hotlist, expected_hotlist)

  def testUpdateHotlist_OneField(self):
    request = hotlists_pb2.UpdateHotlistRequest(
        update_mask=field_mask_pb2.FieldMask(paths=['summary']),
        hotlist=feature_objects_pb2.Hotlist(
            name=self.hotlist_resource_name,
            display_name='newName',
            summary='new summary',
            description='new description',
            default_columns=[
                issue_objects_pb2.IssuesListColumn(column='new-chicken-egg')
            ],
            hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
                'PUBLIC')))
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    api_hotlist = self.CallWrapped(
        self.hotlists_svcr.UpdateHotlist, mc, request)
    user_names_by_id = rnc.ConvertUserNames(
        [self.user_2.user_id, self.user_1.user_id])
    expected_hotlist = feature_objects_pb2.Hotlist(
        name=self.hotlist_resource_name,
        display_name='HotlistName',
        summary='new summary',
        description='description',
        default_columns=[],
        hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
            'PRIVATE'),
        owner=user_names_by_id[self.user_1.user_id],
        editors=[user_names_by_id[self.user_2.user_id]])
    self.assertEqual(api_hotlist, expected_hotlist)

  def testUpdateHotlist_EmptyFieldMask(self):
    request = hotlists_pb2.UpdateHotlistRequest(
        hotlist=feature_objects_pb2.Hotlist(summary='new'))
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    with self.assertRaises(exceptions.InputException):
      self.CallWrapped(self.hotlists_svcr.UpdateHotlist, mc, request)

  def testUpdateHotlist_EmptyHotlist(self):
    request = hotlists_pb2.UpdateHotlistRequest(
        update_mask=field_mask_pb2.FieldMask(paths=['summary']))
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    with self.assertRaises(exceptions.InputException):
      self.CallWrapped(self.hotlists_svcr.UpdateHotlist, mc, request)

  def testDeleteHotlist(self):
    """We can delete a Hotlist."""
    request = hotlists_pb2.GetHotlistRequest(name=self.hotlist_resource_name)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    api_response = self.CallWrapped(
        self.hotlists_svcr.DeleteHotlist, mc, request)
    self.assertEqual(api_response, empty_pb2.Empty())

    with self.assertRaises(features_svc.NoSuchHotlistException):
      self.services.features.GetHotlist(
          self.cnxn, self.hotlist_1.hotlist_id)
