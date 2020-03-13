# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

"""Tests for converting internal protorpc to external protoc."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from google.protobuf import timestamp_pb2

from api import resource_name_converters as rnc
from api.v1 import converters
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import user_objects_pb2
from framework import authdata
from framework import exceptions
from testing import fake
from testing import testing_helpers
from services import service_manager
from proto import tracker_pb2


class ConverterFunctionsTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        usergroup=fake.UserGroupService(),
        user=fake.UserService(),
        config=fake.ConfigService())
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
        project_name=self.project_1.project_name,
        star_count=1)
    self.issue_2 = fake.MakeTestIssue(
        self.project_2.project_id,
        2,
        'sum2',
        'New',
        111,
        project_name=self.project_2.project_name)
    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)
    self.user_1 = self.services.user.TestAddUser('one@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('two@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('three@example.com', 333)

    self.field_def_1_name = 'test_field_1'
    self.field_def_1 = self.services.config.CreateFieldDef(
        self.cnxn, self.project_1.project_id, self.field_def_1_name, 'STR_TYPE',
        None, None, None, None, None, None, None, None, None, None, None, None,
        None, None, [], [])
    self.field_def_2_name = 'test_field_2'
    self.field_def_2 = self.services.config.CreateFieldDef(
        self.cnxn, self.project_1.project_id, self.field_def_2_name, 'INT_TYPE',
        None, None, None, None, None, None, None, None, None, None, None, None,
        None, None, [], [])
    self.approval_def_1_name = 'approval_field_1'
    self.approval_def_1_id = self.services.config.CreateFieldDef(
        self.cnxn, self.project_1.project_id, self.approval_def_1_name,
        'APPROVAL_TYPE', None, None, None, None, None, None, None, None, None,
        None, None, None, None, None, [], [])
    self.dne_field_def_id = 999999

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

  def testConvertIssues(self):
    """We can convert Issues."""
    # TODO(jessan): Add self.issue_2 once method fully implemented.
    issues = [self.issue_1]
    expected_issues = [
        issue_objects_pb2.Issue(
            name='projects/proj/issues/1',
            summary='sum',
            state=issue_objects_pb2.IssueContentState.Value('ACTIVE'),
            star_count=1)
    ]
    self.assertEqual(
        converters.ConvertIssues(self.cnxn, issues, self.services),
        expected_issues)

  def testConvertIssues_Empty(self):
    """ConvertIssues works with no issues passed in."""
    self.assertEqual(converters.ConvertIssues(self.cnxn, [], self.services), [])

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

  def testIngestIssuesListColumns(self):
    columns = [
        issue_objects_pb2.IssuesListColumn(column='chicken'),
        issue_objects_pb2.IssuesListColumn(column='boiled-egg')
    ]
    self.assertEqual(
        converters.IngestIssuesListColumns(columns), 'chicken boiled-egg')

  def testIngestIssuesListColumns_Empty(self):
    self.assertEqual(converters.IngestIssuesListColumns([]), '')

  def testConvertFieldValues(self):
    """It ignores field values referencing a non-existent field"""
    expected_str = 'some_string_field_value'
    fv = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value=expected_str, derived=False)
    expected_name = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_1], self.project_1.project_id,
        self.services)[self.field_def_1]
    expected_value = issue_objects_pb2.Issue.FieldValue(
        field=expected_name,
        value=expected_str,
        derivation=issue_objects_pb2.Issue.Derivation.Value('EXPLICIT'),
        phase=None)
    output = converters.ConvertFieldValues(
        self.cnxn, [fv], self.project_1.project_id, [], self.services)
    self.assertEqual([expected_value], output)

  def testConvertFieldValues_Empty(self):
    output = converters.ConvertFieldValues(
        self.cnxn, [], self.project_1.project_id, [], self.services)
    self.assertEqual([], output)

  def testConvertFieldValues_PreservesOrder(self):
    """It ignores field values referencing a non-existent field"""
    expected_str = 'some_string_field_value'
    fv_1 = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value=expected_str, derived=False)
    name_1 = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_1], self.project_1.project_id,
        self.services)[self.field_def_1]
    expected_1 = issue_objects_pb2.Issue.FieldValue(
        field=name_1,
        value=expected_str,
        derivation=issue_objects_pb2.Issue.Derivation.Value('EXPLICIT'),
        phase=None)

    expected_int = 111111
    fv_2 = fake.MakeFieldValue(
        field_id=self.field_def_2, int_value=expected_int, derived=True)
    name_2 = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_2], self.project_1.project_id,
        self.services).get(self.field_def_2)
    expected_2 = issue_objects_pb2.Issue.FieldValue(
        field=name_2,
        value=str(expected_int),
        derivation=issue_objects_pb2.Issue.Derivation.Value('RULE'),
        phase=None)
    output = converters.ConvertFieldValues(
        self.cnxn, [fv_1, fv_2], self.project_1.project_id, [], self.services)
    self.assertEqual([expected_1, expected_2], output)

  def testConvertFieldValues_IgnoresNullFieldDefs(self):
    """It ignores field values referencing a non-existent field"""
    expected_str = 'some_string_field_value'
    fv_1 = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value=expected_str, derived=False)
    name_1 = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_1], self.project_1.project_id,
        self.services)[self.field_def_1]
    expected_1 = issue_objects_pb2.Issue.FieldValue(
        field=name_1,
        value=expected_str,
        derivation=issue_objects_pb2.Issue.Derivation.Value('EXPLICIT'),
        phase=None)

    fv_2 = fake.MakeFieldValue(
        field_id=self.dne_field_def_id, int_value=111111, derived=True)
    output = converters.ConvertFieldValues(
        self.cnxn, [fv_1, fv_2], self.project_1.project_id, [], self.services)
    self.assertEqual([expected_1], output)

  def test_ComputeFieldValueString_None(self):
    with self.assertRaises(exceptions.InputException):
      converters._ComputeFieldValueString(None)

  def test_ComputeFieldValueString_INT_TYPE(self):
    expected = 123158
    fv = fake.MakeFieldValue(field_id=self.field_def_2, int_value=expected)
    output = converters._ComputeFieldValueString(fv)
    self.assertEqual(str(expected), output)

  def test_ComputeFieldValueString_STR_TYPE(self):
    expected = 'some_string_field_value'
    fv = fake.MakeFieldValue(field_id=self.field_def_1, str_value=expected)
    output = converters._ComputeFieldValueString(fv)
    self.assertEqual(expected, output)

  def test_ComputeFieldValueString_USER_TYPE(self):
    user_id = self.user_1.user_id
    expected = rnc.ConvertUserNames([user_id]).get(user_id)
    fv = fake.MakeFieldValue(field_id=self.dne_field_def_id, user_id=user_id)
    output = converters._ComputeFieldValueString(fv)
    self.assertEqual(expected, output)

  def test_ComputeFieldValueString_DATE_TYPE(self):
    expected = 1234567890
    fv = fake.MakeFieldValue(
        field_id=self.dne_field_def_id, date_value=expected)
    output = converters._ComputeFieldValueString(fv)
    self.assertEqual(str(expected), output)

  def test_ComputeFieldValueString_URL_TYPE(self):
    expected = 'some URL'
    fv = fake.MakeFieldValue(field_id=self.dne_field_def_id, url_value=expected)
    output = converters._ComputeFieldValueString(fv)
    self.assertEqual(expected, output)

  def test_ComputeFieldValueDerivation_RULE(self):
    expected = issue_objects_pb2.Issue.Derivation.Value('RULE')
    fv = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value='something', derived=True)
    output = converters._ComputeFieldValueDerivation(fv)
    self.assertEqual(expected, output)

  def test_ComputeFieldValueDerivation_EXPLICIT(self):
    expected = issue_objects_pb2.Issue.Derivation.Value('EXPLICIT')
    fv = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value='something', derived=False)
    output = converters._ComputeFieldValueDerivation(fv)
    self.assertEqual(expected, output)

  def testConvertApprovalValues(self):
    name = rnc.ConvertApprovalDefNames(
        self.cnxn, [self.approval_def_1_id], self.project_1.project_id,
        self.services).get(self.approval_def_1_id)
    approvers = [
        rnc.ConvertUserNames([self.user_2.user_id]).get(self.user_2.user_id)
    ]
    status = issue_objects_pb2.Issue.ApprovalStatus.Value(
        'APPROVAL_STATUS_UNSPECIFIED')
    set_time = timestamp_pb2.Timestamp()
    set_time.FromSeconds(self.PAST_TIME)
    setter = rnc.ConvertUserNames([self.user_1.user_id]).get(
        self.user_1.user_id)
    phase_name = 'some phase name'
    phase_id = 123123
    phase = fake.MakePhase(phase_id, name=phase_name)
    expected = issue_objects_pb2.Issue.ApprovalValue(
        name=name,
        approvers=approvers,
        status=status,
        set_time=set_time,
        setter=setter,
        phase=phase_name)

    approval_value = fake.MakeApprovalValue(
        self.approval_def_1_id,
        setter_id=self.user_1.user_id,
        set_on=self.PAST_TIME,
        approver_ids=[self.user_2.user_id],
        phase_id=phase_id)

    output = converters.ConvertApprovalValues(
        self.cnxn, [approval_value], self.project_1.project_id, [phase],
        self.services)
    self.assertEqual([expected], output)

  def testConvertApprovalValues_Empty(self):
    output = converters.ConvertApprovalValues(
        self.cnxn, [], self.project_1.project_id, [], self.services)
    self.assertEqual([], output)

  def testConvertApprovalValues_IgnoresNullFieldDefs(self):
    """It ignores approval values referencing a non-existent field"""
    av = fake.MakeApprovalValue(self.dne_field_def_id)

    output = converters.ConvertApprovalValues(
        self.cnxn, [av], self.project_1.project_id, [], self.services)
    self.assertEqual([], output)

  def test_ComputeApprovalValueStatus_NOT_SET(self):
    self.assertEqual(
        converters._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NOT_SET),
        issue_objects_pb2.Issue.ApprovalStatus.Value(
            'APPROVAL_STATUS_UNSPECIFIED'))

  def test_ComputeApprovalValueStatus_NEEDS_REVIEW(self):
    self.assertEqual(
        converters._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NEEDS_REVIEW),
        issue_objects_pb2.Issue.ApprovalStatus.Value('NEEDS_REVIEW'))

  def test_ComputeApprovalValueStatus_NA(self):
    self.assertEqual(
        converters._ComputeApprovalValueStatus(tracker_pb2.ApprovalStatus.NA),
        issue_objects_pb2.Issue.ApprovalStatus.Value('NA'))

  def test_ComputeApprovalValueStatus_REVIEW_REQUESTED(self):
    self.assertEqual(
        converters._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.REVIEW_REQUESTED),
        issue_objects_pb2.Issue.ApprovalStatus.Value('REVIEW_REQUESTED'))

  def test_ComputeApprovalValueStatus_REVIEW_STARTED(self):
    self.assertEqual(
        converters._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.REVIEW_STARTED),
        issue_objects_pb2.Issue.ApprovalStatus.Value('REVIEW_STARTED'))

  def test_ComputeApprovalValueStatus_NEED_INFO(self):
    self.assertEqual(
        converters._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NEED_INFO),
        issue_objects_pb2.Issue.ApprovalStatus.Value('NEED_INFO'))

  def test_ComputeApprovalValueStatus_APPROVED(self):
    self.assertEqual(
        converters._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.APPROVED),
        issue_objects_pb2.Issue.ApprovalStatus.Value('APPROVED'))

  def test_ComputeApprovalValueStatus_NOT_APPROVED(self):
    self.assertEqual(
        converters._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NOT_APPROVED),
        issue_objects_pb2.Issue.ApprovalStatus.Value('NOT_APPROVED'))
