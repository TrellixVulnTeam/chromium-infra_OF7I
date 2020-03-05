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
from framework import authdata
from testing import fake
from testing import testing_helpers
from services import service_manager

class ConverterFunctionsTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        usergroup=fake.UserGroupService(),
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
    self.user_2 = self.services.user.TestAddUser('two@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('three@example.com', 333)

  def testConvertHotlist(self):
    """We can convert a Hotlist."""
    hotlist = fake.Hotlist(
        'Hotlist-Name', 240, default_col_spec='chicken goose',
        is_private=False, owner_ids=[111], editor_ids=[222, 333],
        summary='Hotlist summary', description='Hotlist Description')
    expected_api_hotlist = feature_objects_pb2.Hotlist(
        name='hotlists/240',
        display_name=hotlist.name,
        owner=user_objects_pb2.User(
            name='users/111',
            display_name=self.user_1.email,
            availability_message='User never visited'),
        summary=hotlist.summary,
        description=hotlist.description,
        editors=[
            user_objects_pb2.User(
                name='users/222',
                display_name=testing_helpers.ObscuredEmail(self.user_2.email),
                availability_message='User never visited'),
            user_objects_pb2.User(
                name='users/333',
                display_name=testing_helpers.ObscuredEmail(self.user_3.email),
                availability_message='User never visited')],
        hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
            'PUBLIC'),
        default_columns=[
            issue_objects_pb2.IssuesListColumn(column='chicken'),
            issue_objects_pb2.IssuesListColumn(column='goose')])
    user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    self.assertEqual(
        expected_api_hotlist,
        converters.ConvertHotlist(self.cnxn, user_auth, hotlist, self.services))

  def testConvertHotlist_DefaultValues(self):
    """We can convert a Hotlist with some empty or default values."""
    hotlist = fake.Hotlist(
        'Hotlist-Name', 241, is_private=True, owner_ids=[111],
        summary='Hotlist summary', description='Hotlist Description',
        default_col_spec='')
    expected_api_hotlist = feature_objects_pb2.Hotlist(
        name='hotlists/241',
        display_name=hotlist.name,
        owner=user_objects_pb2.User(
            name='users/111', display_name=self.user_1.email,
            availability_message='User never visited'),
        summary=hotlist.summary,
        description=hotlist.description)
    user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    self.assertEqual(
        expected_api_hotlist,
        converters.ConvertHotlist(self.cnxn, user_auth, hotlist, self.services))

  def testConvertHotlistItems(self):
    """We can convert HotlistItems."""
    hotlist_item_fields = [
        (self.issue_1.issue_id, 21, 111, self.PAST_TIME, 'note2'),
        (78900, 11, 222, self.PAST_TIME, 'note3'),  # Does not exist.
        (self.issue_2.issue_id, 1, 222, None, 'note1'),
    ]
    hotlist = fake.Hotlist(
        'Hotlist-Name', 241, hotlist_item_fields=hotlist_item_fields)
    user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    api_items = converters.ConvertHotlistItems(
        self.cnxn, user_auth, hotlist.hotlist_id, hotlist.items, self.services)
    expected_create_time = timestamp_pb2.Timestamp()
    expected_create_time.FromSeconds(self.PAST_TIME)
    expected_items = [
        feature_objects_pb2.HotlistItem(
            name='hotlists/241/items/proj.1',
            issue='projects/proj/issues/1',
            rank=1,
            adder=user_objects_pb2.User(
                name='users/111', display_name=self.user_1.email,
                availability_message='User never visited'),
            create_time=expected_create_time,
            note='note2'),
        feature_objects_pb2.HotlistItem(
            name='hotlists/241/items/goose.2',
            issue='projects/goose/issues/2',
            rank=0,
            adder=user_objects_pb2.User(
                name='users/222',
                display_name=testing_helpers.ObscuredEmail(self.user_2.email),
                availability_message='User never visited'),
            note='note1')
    ]
    self.assertEqual(api_items, expected_items)

  def testConvertHotlistItems_Empty(self):
    hotlist = fake.Hotlist('Hotlist-Name', 241)
    user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    api_items = converters.ConvertHotlistItems(
        self.cnxn, user_auth, hotlist.hotlist_id, hotlist.items, self.services)
    self.assertEqual(api_items, [])

  def testConvertUsers(self):
    self.user_1.vacation_message='non-empty-string'
    user_ids = [self.user_1.user_id]
    user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    project = None

    expected_user_dict = {
        self.user_1.user_id: user_objects_pb2.User(
            name='users/111',
            display_name='one@example.com',
            availability_message='non-empty-string')}
    self.assertEqual(
        converters.ConvertUsers(
            self.cnxn, user_ids, user_auth, project, self.services),
        expected_user_dict)
