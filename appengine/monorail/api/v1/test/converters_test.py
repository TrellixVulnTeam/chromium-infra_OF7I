# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

"""Tests for converting internal protorpc to external protoc."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from google.protobuf import timestamp_pb2

from api.v1 import converters
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import user_objects_pb2
from testing import fake
from testing import testing_helpers
from services import service_manager

class ConverterFunctionsTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        user=fake.UserService())
    self.cnxn = fake.MonorailConnection()
    self.PAST_TIME = 12345
    self.project_1 = self.services.project.TestAddProject(
        'proj', project_id=789)
    self.project_2 = self.services.project.TestAddProject(
        'goose', project_id=788)
    self.issue_1 = fake.MakeTestIssue(
        self.project_1.project_id,
        1,
        'sum',
        'New',
        111,
        project_name=self.project_1.project_name)
    self.issue_2 = fake.MakeTestIssue(
        self.project_2.project_id,
        2,
        'sum',
        'New',
        111,
        project_name=self.project_2.project_name)
    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)
    self.user_1 = self.services.user.TestAddUser('one@example.com', 111)

  def testConvertHotlist(self):
    """We can convert a Hotlist."""
    hotlist = fake.Hotlist(
        'Hotlist-Name', 240, default_col_spec='chicken goose',
        is_private=False, owner_ids=[111], editor_ids=[222, 333],
        summary='Hotlist summary', description='Hotlist Description')
    expected_api_hotlist = feature_objects_pb2.Hotlist(
        name='hotlists/240',
        display_name=hotlist.name,
        owner='users/111',
        summary=hotlist.summary,
        description=hotlist.description,
        editors=['users/222', 'users/333'],
        hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
            'PUBLIC'),
        default_columns=[
            issue_objects_pb2.IssuesListColumn(column='chicken'),
            issue_objects_pb2.IssuesListColumn(column='goose')])
    self.assertEqual(converters.ConvertHotlist(hotlist), expected_api_hotlist)

  def testConvertHotlist_DefaultValues(self):
    """We can convert a Hotlist with some empty or default values."""
    hotlist = fake.Hotlist(
        'Hotlist-Name', 241, is_private=True, owner_ids=[111],
        summary='Hotlist summary', description='Hotlist Description',
        default_col_spec='')
    expected_api_hotlist = feature_objects_pb2.Hotlist(
        name='hotlists/241',
        display_name=hotlist.name,
        owner='users/111',
        summary=hotlist.summary,
        description=hotlist.description)
    self.assertEqual(converters.ConvertHotlist(hotlist), expected_api_hotlist)

  def testConvertHotlistItems(self):
    """We can convert HotlistItems."""
    hotlist_item_fields = [
        (self.issue_1.issue_id, 21, 111, self.PAST_TIME, 'note2'),
        (78900, 11, 222, self.PAST_TIME, 'note3'),  # Does not exist.
        (self.issue_2.issue_id, 1, 222, None, 'note1'),
    ]
    hotlist = fake.Hotlist(
        'Hotlist-Name', 241, hotlist_item_fields=hotlist_item_fields)
    api_items = converters.ConvertHotlistItems(
        self.cnxn, hotlist.hotlist_id, hotlist.items, self.services)
    expected_create_time = timestamp_pb2.Timestamp()
    expected_create_time.FromSeconds(self.PAST_TIME)
    expected_items = [
        feature_objects_pb2.HotlistItem(
            name='hotlists/241/items/proj.1',
            issue='projects/proj/issues/1',
            rank=1,
            adder='users/111',
            create_time=expected_create_time,
            note='note2'),
        feature_objects_pb2.HotlistItem(
            name='hotlists/241/items/goose.2',
            issue='projects/goose/issues/2',
            rank=0,
            adder='users/222',
            note='note1')
    ]
    self.assertEqual(api_items, expected_items)

  def testConvertHotlistItems_Empty(self):
    hotlist = fake.Hotlist('Hotlist-Name', 241)
    api_items = converters.ConvertHotlistItems(
        self.cnxn, hotlist.hotlist_id, hotlist.items, self.services)
    self.assertEqual(api_items, [])

  def testConvertUsers(self):
    expected_api_user = [
        user_objects_pb2.User(
            name='users/111',
            display_name='one@example.com',
            availability_message='non-empty-string')
    ]

    user_pb = testing_helpers.Blank(
        user_id=111,
        display_name='one@example.com',
        email='one@example.com',
        banned=False,
        is_site_admin=True,
        linked_parent_id=None,
        vacation_message='non-empty-string')

    users = [user_pb]

    user_auth = testing_helpers.Blank(user_pb=user_pb)
    project = None

    self.assertEqual(
        converters.ConvertUsers(users, user_auth, project), expected_api_user)
