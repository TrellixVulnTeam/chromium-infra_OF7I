# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
"""Tests for converting internal protorpc to external protoc."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from mock import patch
from google.protobuf import timestamp_pb2

from api import resource_name_converters as rnc
from api.v1 import converters
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import user_objects_pb2
from api.v1.api_proto import project_objects_pb2
from framework import authdata
from framework import exceptions
from framework import monorailcontext
from testing import fake
from testing import testing_helpers
from services import service_manager
from proto import tracker_pb2

EXPLICIT_DERIVATION = issue_objects_pb2.Issue.Derivation.Value('EXPLICIT')
RULE_DERIVATION = issue_objects_pb2.Issue.Derivation.Value('RULE')


class ConverterFunctionsTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        usergroup=fake.UserGroupService(),
        user=fake.UserService(),
        config=fake.ConfigService(),
        template=fake.TemplateService())
    self.cnxn = fake.MonorailConnection()
    self.mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)
    self.converter = converters.Converter(self.mc, self.services)
    self.PAST_TIME = 12345
    self.project_1 = self.services.project.TestAddProject(
        'proj', project_id=789)
    self.project_2 = self.services.project.TestAddProject(
        'goose', project_id=788)
    self.user_1 = self.services.user.TestAddUser('one@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('two@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('three@example.com', 333)
    self.services.project.TestAddProjectMembers(
        [self.user_1.user_id], self.project_1, 'CONTRIBUTOR_ROLE')

    self.field_def_1_name = 'test_field_1'
    self.field_def_1 = self._CreateFieldDef(
        self.project_1.project_id, self.field_def_1_name, 'STR_TYPE')
    self.field_def_2_name = 'test_field_2'
    self.field_def_2 = self._CreateFieldDef(
        self.project_1.project_id, self.field_def_2_name, 'INT_TYPE')
    self.field_def_3_name = 'days'
    self.field_def_3 = self._CreateFieldDef(
        self.project_1.project_id, self.field_def_3_name, 'ENUM_TYPE')
    self.field_def_4_name = 'OS'
    self.field_def_4 = self._CreateFieldDef(
        self.project_1.project_id, self.field_def_4_name, 'ENUM_TYPE')
    self.field_def_project2_name = 'lorem'
    self.field_def_project2 = self._CreateFieldDef(
        self.project_2.project_id, self.field_def_project2_name, 'ENUM_TYPE')
    self.approval_def_1_name = 'approval_field_1'
    self.approval_def_1_id = self._CreateFieldDef(
        self.project_1.project_id, self.approval_def_1_name, 'APPROVAL_TYPE')
    self.dne_field_def_id = 999999
    self.fv_1_value = 'some_string_field_value'
    self.fv_1 = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value=self.fv_1_value, derived=False)
    self.fv_1_derived = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value=self.fv_1_value, derived=True)
    self.phase_1_id = 123123
    self.phase_1 = fake.MakePhase(self.phase_1_id, name='some phase name')
    self.av_1 = fake.MakeApprovalValue(
        self.approval_def_1_id,
        setter_id=self.user_1.user_id,
        set_on=self.PAST_TIME,
        approver_ids=[self.user_2.user_id],
        phase_id=self.phase_1_id)
    self.av_2 = fake.MakeApprovalValue(
        self.approval_def_1_id,
        setter_id=self.user_1.user_id,
        set_on=self.PAST_TIME,
        approver_ids=[self.user_2.user_id])

    self.issue_1 = fake.MakeTestIssue(
        self.project_1.project_id,
        1,
        'sum',
        'New',
        self.user_1.user_id,
        cc_ids=[self.user_2.user_id],
        derived_cc_ids=[self.user_3.user_id],
        project_name=self.project_1.project_name,
        star_count=1,
        labels=['label-a', 'label-b', 'days-1'],
        derived_owner_id=self.user_2.user_id,
        derived_status='Fixed',
        derived_labels=['label-derived', 'OS-mac', 'label-derived-2'],
        component_ids=[1, 2],
        merged_into_external='b/1',
        derived_component_ids=[3, 4],
        attachment_count=5,
        field_values=[self.fv_1, self.fv_1_derived],
        opened_timestamp=self.PAST_TIME,
        modified_timestamp=self.PAST_TIME,
        approval_values=[self.av_1],
        phases=[self.phase_1])
    self.issue_2 = fake.MakeTestIssue(
        self.project_2.project_id,
        2,
        'sum2',
        None,
        None,
        reporter_id=self.user_1.user_id,
        project_name=self.project_2.project_name,
        merged_into=self.issue_1.issue_id,
        opened_timestamp=self.PAST_TIME,
        modified_timestamp=self.PAST_TIME,
        closed_timestamp=self.PAST_TIME,
        derived_status='Fixed',
        derived_owner_id=self.user_2.user_id,
        is_spam=True)
    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)

    self.template_0 = self.services.template.TestAddIssueTemplateDef(
        11110, self.project_1.project_id, 'template0')
    self.template_1_label1_value = '2'
    self.template_1_labels = [
        'pri-1', '{}-{}'.format(
            self.field_def_3_name, self.template_1_label1_value)
    ]
    self.template_1 = self.services.template.TestAddIssueTemplateDef(
        11111,
        self.project_1.project_id,
        'template1',
        content='foobar',
        summary='foo',
        admin_ids=[self.user_2.user_id],
        owner_id=self.user_1.user_id,
        labels=self.template_1_labels,
        component_ids=[654],
        field_values=[self.fv_1],
        approval_values=[self.av_1],
        phases=[self.phase_1])
    self.template_2 = self.services.template.TestAddIssueTemplateDef(
        11112,
        self.project_1.project_id,
        'template2',
        members_only=True,
        owner_defaults_to_member=True)
    self.template_3 = self.services.template.TestAddIssueTemplateDef(
        11113,
        self.project_1.project_id,
        'template3',
        field_values=[self.fv_1],
        approval_values=[self.av_2],
    )
    self.dne_template = tracker_pb2.TemplateDef(
        name='dne_template_name', template_id=11114)
    self.services.config.UpdateConfig(
        self.cnxn,
        self.project_1,
        statuses_offer_merge=['Duplicate'],
        excl_label_prefixes=['type', 'priority'],
        default_template_for_developers=self.template_2.template_id,
        default_template_for_users=self.template_1.template_id,
        list_prefs=('ID Summary', 'ID', 'status', 'owner', 'owner:me'))

  def _CreateFieldDef(self, project_id, field_name, field_type_str):
    """Calls CreateFieldDef with reasonable defaults, returns the ID."""
    return self.services.config.CreateFieldDef(
        self.cnxn, project_id, field_name, field_type_str, None, None, None,
        None, None, None, None, None, None, None, None, None, None, None, [],
        [])

  def testConvertHotlist(self):
    """We can convert a Hotlist."""
    hotlist = fake.Hotlist(
        'Hotlist-Name',
        240,
        default_col_spec='chicken goose',
        is_private=False,
        owner_ids=[111],
        editor_ids=[222, 333],
        summary='Hotlist summary',
        description='Hotlist Description')
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
                availability_message='User never visited')
        ],
        hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
            'PUBLIC'),
        default_columns=[
            issue_objects_pb2.IssuesListColumn(column='chicken'),
            issue_objects_pb2.IssuesListColumn(column='goose')
        ])
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    self.assertEqual(
        expected_api_hotlist, self.converter.ConvertHotlist(hotlist))

  def testConvertHotlist_DefaultValues(self):
    """We can convert a Hotlist with some empty or default values."""
    hotlist = fake.Hotlist(
        'Hotlist-Name',
        241,
        is_private=True,
        owner_ids=[111],
        summary='Hotlist summary',
        description='Hotlist Description',
        default_col_spec='')
    expected_api_hotlist = feature_objects_pb2.Hotlist(
        name='hotlists/241',
        display_name=hotlist.name,
        owner=user_objects_pb2.User(
            name='users/111',
            display_name=self.user_1.email,
            availability_message='User never visited'),
        summary=hotlist.summary,
        description=hotlist.description,
        hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
            'PRIVATE'))
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    self.assertEqual(
        expected_api_hotlist, self.converter.ConvertHotlist(hotlist))

  def testConvertHotlistItems(self):
    """We can convert HotlistItems."""
    hotlist_item_fields = [
        (self.issue_1.issue_id, 21, 111, self.PAST_TIME, 'note2'),
        (78900, 11, 222, self.PAST_TIME, 'note3'),  # Does not exist.
        (self.issue_2.issue_id, 1, 222, None, 'note1'),
    ]
    hotlist = fake.Hotlist(
        'Hotlist-Name', 241, hotlist_item_fields=hotlist_item_fields)
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    api_items = self.converter.ConvertHotlistItems(
        hotlist.hotlist_id, hotlist.items)
    expected_create_time = timestamp_pb2.Timestamp()
    expected_create_time.FromSeconds(self.PAST_TIME)
    expected_items = [
        feature_objects_pb2.HotlistItem(
            name='hotlists/241/items/proj.1',
            issue='projects/proj/issues/1',
            rank=1,
            adder=user_objects_pb2.User(
                name='users/111',
                display_name=self.user_1.email,
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
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    api_items = self.converter.ConvertHotlistItems(
        hotlist.hotlist_id, hotlist.items)
    self.assertEqual(api_items, [])

  @patch('tracker.attachment_helpers.SignAttachmentID')
  def testConvertComments(self, mock_SignAttachmentID):
    """We can convert comments."""
    mock_SignAttachmentID.return_value = 2
    attach = tracker_pb2.Attachment(
        attachment_id=1,
        mimetype='image/png',
        filename='example.png',
        filesize=12345)
    deleted_attach = tracker_pb2.Attachment(
        attachment_id=2,
        mimetype='image/png',
        filename='deleted_example.png',
        filesize=67890,
        deleted=True)
    initial_comment = tracker_pb2.IssueComment(
        project_id=self.issue_1.project_id,
        issue_id=self.issue_1.issue_id,
        user_id=self.issue_1.reporter_id,
        timestamp=self.PAST_TIME,
        content='initial description',
        sequence=0,
        is_description=True,
        description_num='1',
        attachments=[attach, deleted_attach])
    deleted_comment = tracker_pb2.IssueComment(
        project_id=self.issue_1.project_id,
        issue_id=self.issue_1.issue_id,
        timestamp=self.PAST_TIME,
        deleted_by=self.issue_1.reporter_id,
        sequence=1)
    amendments = [
        tracker_pb2.Amendment(
            field=tracker_pb2.FieldID.SUMMARY, newvalue='new', oldvalue='old'),
        tracker_pb2.Amendment(
            field=tracker_pb2.FieldID.OWNER, added_user_ids=[111]),
        tracker_pb2.Amendment(
            field=tracker_pb2.FieldID.CC,
            added_user_ids=[111],
            removed_user_ids=[222]),
        tracker_pb2.Amendment(
            field=tracker_pb2.FieldID.CUSTOM,
            custom_field_name='EstDays',
            newvalue='12')
    ]
    amendments_comment = tracker_pb2.IssueComment(
        project_id=self.issue_1.project_id,
        issue_id=self.issue_1.issue_id,
        user_id=self.issue_1.reporter_id,
        timestamp=self.PAST_TIME,
        content='some amendments',
        sequence=2,
        amendments=amendments,
        importer_id=1,  # Not used in conversion, so nothing to verify.
        approval_id=self.approval_def_1_id)
    inbound_spam_comment = tracker_pb2.IssueComment(
        project_id=self.issue_1.project_id,
        issue_id=self.issue_1.issue_id,
        user_id=self.issue_1.reporter_id,
        timestamp=self.PAST_TIME,
        content='content',
        sequence=3,
        inbound_message='inbound message',
        is_spam=True)
    expected_0 = issue_objects_pb2.Comment(
        name='projects/proj/issues/1/comments/0',
        state=issue_objects_pb2.IssueContentState.Value('ACTIVE'),
        content='initial description',
        commenter='users/111',
        create_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        attachments=[
            issue_objects_pb2.Comment.Attachment(
                filename='example.png',
                state=issue_objects_pb2.IssueContentState.Value('ACTIVE'),
                size=12345,
                media_type='image/png',
                thumbnail_uri='attachment?aid=1&signed_aid=2&inline=1&thumb=1',
                view_uri='attachment?aid=1&signed_aid=2&inline=1',
                download_uri='attachment?aid=1&signed_aid=2'),
            issue_objects_pb2.Comment.Attachment(
                filename='deleted_example.png',
                state=issue_objects_pb2.IssueContentState.Value('DELETED'),
                media_type='image/png')
        ])
    expected_1 = issue_objects_pb2.Comment(
        name='projects/proj/issues/1/comments/1',
        state=issue_objects_pb2.IssueContentState.Value('DELETED'),
        create_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME))
    expected_2 = issue_objects_pb2.Comment(
        name='projects/proj/issues/1/comments/2',
        state=issue_objects_pb2.IssueContentState.Value('ACTIVE'),
        content='some amendments',
        commenter='users/111',
        create_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        approval='projects/proj/approvalDefs/approval_field_1',
        amendments=[
            issue_objects_pb2.Comment.Amendment(
                field_name='Summary', new_or_delta_value='new',
                old_value='old'),
            issue_objects_pb2.Comment.Amendment(
                field_name='Owner', new_or_delta_value='o...@example.com'),
            issue_objects_pb2.Comment.Amendment(
                field_name='Cc',
                new_or_delta_value='-t...@example.com o...@example.com'),
            issue_objects_pb2.Comment.Amendment(
                field_name='EstDays', new_or_delta_value='12')
        ])
    expected_3 = issue_objects_pb2.Comment(
        name='projects/proj/issues/1/comments/3',
        state=issue_objects_pb2.IssueContentState.Value('SPAM'),
        content='content',
        commenter='users/111',
        create_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        inbound_message='inbound message')

    comments = [
        initial_comment, deleted_comment, amendments_comment,
        inbound_spam_comment
    ]
    actual = self.converter.ConvertComments(self.issue_1.issue_id, comments)
    self.assertEqual(actual, [expected_0, expected_1, expected_2, expected_3])

  def testConvertComments_Empty(self):
    """We can convert an empty list of comments."""
    self.assertEqual(
        self.converter.ConvertComments(self.issue_1.issue_id, []), [])

  def testConvertIssue(self):
    """We can convert a single issue."""
    self.assertEqual(self.converter.ConvertIssue(self.issue_1),
        self.converter.ConvertIssues([self.issue_1])[0])

  def testConvertIssues(self):
    """We can convert Issues."""
    blocked_on_1 = fake.MakeTestIssue(
        self.project_1.project_id,
        3,
        'sum3',
        'New',
        self.user_1.user_id,
        issue_id=301,
        project_name=self.project_1.project_name,
    )
    blocked_on_2 = fake.MakeTestIssue(
        self.project_2.project_id,
        4,
        'sum4',
        'New',
        self.user_1.user_id,
        issue_id=401,
        project_name=self.project_2.project_name,
    )
    blocking = fake.MakeTestIssue(
        self.project_2.project_id,
        5,
        'sum5',
        'New',
        self.user_1.user_id,
        issue_id=501,
        project_name=self.project_2.project_name,
    )
    self.services.issue.TestAddIssue(blocked_on_1)
    self.services.issue.TestAddIssue(blocked_on_2)
    self.services.issue.TestAddIssue(blocking)

    # Reversing natural ordering to ensure order is respected.
    self.issue_1.blocked_on_iids = [
        blocked_on_2.issue_id, blocked_on_1.issue_id
    ]
    self.issue_1.dangling_blocked_on_refs = [
        tracker_pb2.DanglingIssueRef(ext_issue_identifier='b/555'),
        tracker_pb2.DanglingIssueRef(ext_issue_identifier='b/2')
    ]
    self.issue_1.blocking_iids = [blocking.issue_id]
    self.issue_1.dangling_blocking_refs = [
        tracker_pb2.DanglingIssueRef(ext_issue_identifier='b/3')
    ]

    issues = [self.issue_1, self.issue_2]
    expected_1 = issue_objects_pb2.Issue(
        name='projects/proj/issues/1',
        summary='sum',
        state=issue_objects_pb2.IssueContentState.Value('ACTIVE'),
        status=issue_objects_pb2.Issue.StatusValue(
            derivation=EXPLICIT_DERIVATION, status='New'),
        reporter='users/111',
        owner=issue_objects_pb2.Issue.UserValue(
            derivation=EXPLICIT_DERIVATION, user='users/111'),
        cc_users=[
            issue_objects_pb2.Issue.UserValue(
                derivation=EXPLICIT_DERIVATION, user='users/222'),
            issue_objects_pb2.Issue.UserValue(
                derivation=RULE_DERIVATION, user='users/333')
        ],
        labels=[
            issue_objects_pb2.Issue.LabelValue(
                derivation=EXPLICIT_DERIVATION, label='label-a'),
            issue_objects_pb2.Issue.LabelValue(
                derivation=EXPLICIT_DERIVATION, label='label-b'),
            issue_objects_pb2.Issue.LabelValue(
                derivation=RULE_DERIVATION, label='label-derived'),
            issue_objects_pb2.Issue.LabelValue(
                derivation=RULE_DERIVATION, label='label-derived-2')
        ],
        components=[
            issue_objects_pb2.Issue.ComponentValue(
                derivation=EXPLICIT_DERIVATION,
                component='projects/proj/componentDefs/1'),
            issue_objects_pb2.Issue.ComponentValue(
                derivation=EXPLICIT_DERIVATION,
                component='projects/proj/componentDefs/2'),
            issue_objects_pb2.Issue.ComponentValue(
                derivation=RULE_DERIVATION,
                component='projects/proj/componentDefs/3'),
            issue_objects_pb2.Issue.ComponentValue(
                derivation=RULE_DERIVATION,
                component='projects/proj/componentDefs/4'),
        ],
        field_values=[
            issue_objects_pb2.Issue.FieldValue(
                derivation=EXPLICIT_DERIVATION,
                field='projects/proj/fieldDefs/test_field_1',
                value=self.fv_1_value,
            ),
            issue_objects_pb2.Issue.FieldValue(
                derivation=RULE_DERIVATION,
                field='projects/proj/fieldDefs/test_field_1',
                value=self.fv_1_value,
            ),
            issue_objects_pb2.Issue.FieldValue(
                derivation=EXPLICIT_DERIVATION,
                field='projects/proj/fieldDefs/days',
                value='1',
            ),
            issue_objects_pb2.Issue.FieldValue(
                derivation=RULE_DERIVATION,
                field='projects/proj/fieldDefs/OS',
                value='mac',
            )
        ],
        merged_into_issue_ref=issue_objects_pb2.IssueRef(ext_identifier='b/1'),
        blocked_on_issue_refs=[
            issue_objects_pb2.IssueRef(issue='projects/goose/issues/4'),
            issue_objects_pb2.IssueRef(issue='projects/proj/issues/3'),
            issue_objects_pb2.IssueRef(ext_identifier='b/555'),
            issue_objects_pb2.IssueRef(ext_identifier='b/2')
        ],
        blocking_issue_refs=[
            issue_objects_pb2.IssueRef(issue='projects/goose/issues/5'),
            issue_objects_pb2.IssueRef(ext_identifier='b/3')
        ],
        create_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        component_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        status_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        owner_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        star_count=1,
        attachment_count=5,
        approval_values=[
            issue_objects_pb2.Issue.ApprovalValue(
                approvers=['users/222'],
                name='projects/proj/approvalDefs/approval_field_1',
                phase=self.phase_1.name,
                set_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
                setter='users/111',
                status=issue_objects_pb2.Issue.ApprovalStatus.Value(
                    'APPROVAL_STATUS_UNSPECIFIED'))
        ],
        phases=[self.phase_1.name])
    expected_2 = issue_objects_pb2.Issue(
        name='projects/goose/issues/2',
        summary='sum2',
        state=issue_objects_pb2.IssueContentState.Value('SPAM'),
        status=issue_objects_pb2.Issue.StatusValue(
            derivation=RULE_DERIVATION, status='Fixed'),
        reporter='users/111',
        owner=issue_objects_pb2.Issue.UserValue(
            derivation=RULE_DERIVATION, user='users/222'),
        merged_into_issue_ref=issue_objects_pb2.IssueRef(
            issue='projects/proj/issues/1'),
        create_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        close_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        component_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        status_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        owner_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME))
    self.assertEqual(
        self.converter.ConvertIssues(issues), [expected_1, expected_2])

  def testConvertIssues_Empty(self):
    """ConvertIssues works with no issues passed in."""
    self.assertEqual(self.converter.ConvertIssues([]), [])

  def testConvertIssues_NegativeAttachmentCount(self):
    """Negative attachment counts are not set on issues."""
    issue = fake.MakeTestIssue(
        self.project_1.project_id,
        3,
        'sum',
        'New',
        owner_id=None,
        reporter_id=111,
        attachment_count=-10,
        project_name=self.project_1.project_name,
        opened_timestamp=self.PAST_TIME,
        modified_timestamp=self.PAST_TIME)
    self.services.issue.TestAddIssue(issue)
    expected_issue = issue_objects_pb2.Issue(
        name='projects/proj/issues/3',
        state=issue_objects_pb2.IssueContentState.Value('ACTIVE'),
        summary='sum',
        status=issue_objects_pb2.Issue.StatusValue(
            derivation=EXPLICIT_DERIVATION, status='New'),
        reporter='users/111',
        create_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        component_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        status_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        owner_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
    )
    self.assertEqual(self.converter.ConvertIssues([issue]), [expected_issue])

  def testConvertUser(self):
    """We can convert a single User."""
    self.user_1.vacation_message = 'non-empty-string'
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    project = None

    expected_user = user_objects_pb2.User(
        name='users/111',
        display_name='one@example.com',
        availability_message='non-empty-string')
    self.assertEqual(
        self.converter.ConvertUser(self.user_1, project), expected_user)


  def testConvertUsers(self):
    self.user_1.vacation_message = 'non-empty-string'
    user_ids = [self.user_1.user_id]
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    project = None

    expected_user_dict = {
        self.user_1.user_id:
            user_objects_pb2.User(
                name='users/111',
                display_name='one@example.com',
                availability_message='non-empty-string')
    }
    self.assertEqual(
        self.converter.ConvertUsers(user_ids, project), expected_user_dict)

  def testConvertProjectStars(self):
    expected_stars = [
        user_objects_pb2.ProjectStar(name='users/111/projectStars/proj'),
        user_objects_pb2.ProjectStar(name='users/111/projectStars/goose')
    ]
    self.assertEqual(
        self.converter.ConvertProjectStars(
            self.user_1.user_id, [self.project_1, self.project_2]),
        expected_stars)

  def testIngestIssuesListColumns(self):
    columns = [
        issue_objects_pb2.IssuesListColumn(column='chicken'),
        issue_objects_pb2.IssuesListColumn(column='boiled-egg')
    ]
    self.assertEqual(
        self.converter.IngestIssuesListColumns(columns), 'chicken boiled-egg')

  def testIngestIssuesListColumns_Empty(self):
    self.assertEqual(self.converter.IngestIssuesListColumns([]), '')

  def test_ComputeIssuesListColumns(self):
    """Can convert string to sequence of IssuesListColumns"""
    expected_columns = [
        issue_objects_pb2.IssuesListColumn(column='chicken'),
        issue_objects_pb2.IssuesListColumn(column='boiled-egg')
    ]
    self.assertEqual(
        expected_columns,
        self.converter._ComputeIssuesListColumns('chicken boiled-egg'))

  def test_ComputeIssuesListColumns_Empty(self):
    """Can handle empty strings"""
    self.assertEqual([], self.converter._ComputeIssuesListColumns(''))

  def test_Conversion_IssuesListColumns(self):
    """_Ingest and _Compute converts to and from each other"""
    expected_columns = 'foo bar fizz buzz'
    converted_columns = self.converter._ComputeIssuesListColumns(
        expected_columns)
    self.assertEqual(
        expected_columns,
        self.converter.IngestIssuesListColumns(converted_columns))

    expected_columns = [
        issue_objects_pb2.IssuesListColumn(column='foo'),
        issue_objects_pb2.IssuesListColumn(column='bar'),
        issue_objects_pb2.IssuesListColumn(column='fizz'),
        issue_objects_pb2.IssuesListColumn(column='buzz')
    ]
    converted_columns = self.converter.IngestIssuesListColumns(expected_columns)
    self.assertEqual(
        expected_columns,
        self.converter._ComputeIssuesListColumns(converted_columns))

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
        derivation=EXPLICIT_DERIVATION,
        phase=None)
    output = self.converter.ConvertFieldValues(
        [fv], self.project_1.project_id, [])
    self.assertEqual([expected_value], output)

  def testConvertFieldValues_Empty(self):
    output = self.converter.ConvertFieldValues(
        [], self.project_1.project_id, [])
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
        derivation=EXPLICIT_DERIVATION,
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
        derivation=RULE_DERIVATION,
        phase=None)
    output = self.converter.ConvertFieldValues(
        [fv_1, fv_2], self.project_1.project_id, [])
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
        derivation=EXPLICIT_DERIVATION,
        phase=None)

    fv_2 = fake.MakeFieldValue(
        field_id=self.dne_field_def_id, int_value=111111, derived=True)
    output = self.converter.ConvertFieldValues(
        [fv_1, fv_2], self.project_1.project_id, [])
    self.assertEqual([expected_1], output)

  def test_ComputeFieldValueString_None(self):
    with self.assertRaises(exceptions.InputException):
      self.converter._ComputeFieldValueString(None)

  def test_ComputeFieldValueString_INT_TYPE(self):
    expected = 123158
    fv = fake.MakeFieldValue(field_id=self.field_def_2, int_value=expected)
    output = self.converter._ComputeFieldValueString(fv)
    self.assertEqual(str(expected), output)

  def test_ComputeFieldValueString_STR_TYPE(self):
    expected = 'some_string_field_value'
    fv = fake.MakeFieldValue(field_id=self.field_def_1, str_value=expected)
    output = self.converter._ComputeFieldValueString(fv)
    self.assertEqual(expected, output)

  def test_ComputeFieldValueString_USER_TYPE(self):
    user_id = self.user_1.user_id
    expected = rnc.ConvertUserNames([user_id]).get(user_id)
    fv = fake.MakeFieldValue(field_id=self.dne_field_def_id, user_id=user_id)
    output = self.converter._ComputeFieldValueString(fv)
    self.assertEqual(expected, output)

  def test_ComputeFieldValueString_DATE_TYPE(self):
    expected = 1234567890
    fv = fake.MakeFieldValue(
        field_id=self.dne_field_def_id, date_value=expected)
    output = self.converter._ComputeFieldValueString(fv)
    self.assertEqual(str(expected), output)

  def test_ComputeFieldValueString_URL_TYPE(self):
    expected = 'some URL'
    fv = fake.MakeFieldValue(field_id=self.dne_field_def_id, url_value=expected)
    output = self.converter._ComputeFieldValueString(fv)
    self.assertEqual(expected, output)

  def test_ComputeFieldValueDerivation_RULE(self):
    expected = RULE_DERIVATION
    fv = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value='something', derived=True)
    output = self.converter._ComputeFieldValueDerivation(fv)
    self.assertEqual(expected, output)

  def test_ComputeFieldValueDerivation_EXPLICIT(self):
    expected = EXPLICIT_DERIVATION
    fv = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value='something', derived=False)
    output = self.converter._ComputeFieldValueDerivation(fv)
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
    phase = fake.MakePhase(self.phase_1_id, name=self.phase_1.name)
    expected = issue_objects_pb2.Issue.ApprovalValue(
        name=name,
        approvers=approvers,
        status=status,
        set_time=set_time,
        setter=setter,
        phase=self.phase_1.name)

    approval_value = fake.MakeApprovalValue(
        self.approval_def_1_id,
        setter_id=self.user_1.user_id,
        set_on=self.PAST_TIME,
        approver_ids=[self.user_2.user_id],
        phase_id=self.phase_1_id)

    output = self.converter.ConvertApprovalValues(
        [approval_value], self.project_1.project_id, [phase])
    self.assertEqual([expected], output)

  def testConvertApprovalValues_NoPhase(self):
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
    expected = issue_objects_pb2.Issue.ApprovalValue(
        name=name,
        approvers=approvers,
        status=status,
        set_time=set_time,
        setter=setter)

    approval_value = fake.MakeApprovalValue(
        self.approval_def_1_id,
        setter_id=self.user_1.user_id,
        set_on=self.PAST_TIME,
        approver_ids=[self.user_2.user_id],
        phase_id=self.phase_1_id)

    output = self.converter.ConvertApprovalValues(
        [approval_value], self.project_1.project_id, [])
    self.assertEqual([expected], output)

  def testConvertApprovalValues_Empty(self):
    output = self.converter.ConvertApprovalValues(
        [], self.project_1.project_id, [])
    self.assertEqual([], output)

  def testConvertApprovalValues_IgnoresNullFieldDefs(self):
    """It ignores approval values referencing a non-existent field"""
    av = fake.MakeApprovalValue(self.dne_field_def_id)

    output = self.converter.ConvertApprovalValues(
        [av], self.project_1.project_id, [])
    self.assertEqual([], output)

  def test_ComputeApprovalValueStatus_NOT_SET(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NOT_SET),
        issue_objects_pb2.Issue.ApprovalStatus.Value(
            'APPROVAL_STATUS_UNSPECIFIED'))

  def test_ComputeApprovalValueStatus_NEEDS_REVIEW(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NEEDS_REVIEW),
        issue_objects_pb2.Issue.ApprovalStatus.Value('NEEDS_REVIEW'))

  def test_ComputeApprovalValueStatus_NA(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NA),
        issue_objects_pb2.Issue.ApprovalStatus.Value('NA'))

  def test_ComputeApprovalValueStatus_REVIEW_REQUESTED(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.REVIEW_REQUESTED),
        issue_objects_pb2.Issue.ApprovalStatus.Value('REVIEW_REQUESTED'))

  def test_ComputeApprovalValueStatus_REVIEW_STARTED(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.REVIEW_STARTED),
        issue_objects_pb2.Issue.ApprovalStatus.Value('REVIEW_STARTED'))

  def test_ComputeApprovalValueStatus_NEED_INFO(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NEED_INFO),
        issue_objects_pb2.Issue.ApprovalStatus.Value('NEED_INFO'))

  def test_ComputeApprovalValueStatus_APPROVED(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.APPROVED),
        issue_objects_pb2.Issue.ApprovalStatus.Value('APPROVED'))

  def test_ComputeApprovalValueStatus_NOT_APPROVED(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NOT_APPROVED),
        issue_objects_pb2.Issue.ApprovalStatus.Value('NOT_APPROVED'))

  def test_ComputeTemplatePrivacy_PUBLIC(self):
    self.assertEqual(
        self.converter._ComputeTemplatePrivacy(self.template_1),
        project_objects_pb2.IssueTemplate.TemplatePrivacy.Value('PUBLIC'))

  def test_ComputeTemplatePrivacy_MEMBERS_ONLY(self):
    self.assertEqual(
        self.converter._ComputeTemplatePrivacy(self.template_2),
        project_objects_pb2.IssueTemplate.TemplatePrivacy.Value('MEMBERS_ONLY'))

  def test_ComputeTemplateDefaultOwner_UNSPECIFIED(self):
    self.assertEqual(
        self.converter._ComputeTemplateDefaultOwner(self.template_1),
        project_objects_pb2.IssueTemplate.DefaultOwner.Value(
            'DEFAULT_OWNER_UNSPECIFIED'))

  def test_ComputeTemplateDefaultOwner_REPORTER(self):
    self.assertEqual(
        self.converter._ComputeTemplateDefaultOwner(self.template_2),
        project_objects_pb2.IssueTemplate.DefaultOwner.Value(
            'PROJECT_MEMBER_REPORTER'))

  def test_ComputePhases(self):
    """It sorts by rank"""
    phase1 = fake.MakePhase(123111, name='phase1name', rank=3)
    phase2 = fake.MakePhase(123112, name='phase2name', rank=2)
    phase3 = fake.MakePhase(123113, name='phase3name', rank=1)
    expected = ['phase3name', 'phase2name', 'phase1name']
    self.assertEqual(
        self.converter._ComputePhases([phase1, phase2, phase3]), expected)

  def test_ComputePhases_EMPTY(self):
    self.assertEqual(self.converter._ComputePhases([]), [])

  def test_FillIssueFromTemplate(self):
    result = self.converter._FillIssueFromTemplate(
        self.template_1, self.project_1.project_id)
    self.assertFalse(result.name)
    self.assertEqual(result.summary, self.template_1.summary)
    self.assertEqual(
        result.state, issue_objects_pb2.IssueContentState.Value('ACTIVE'))
    self.assertEqual(result.status.status, 'New')
    self.assertFalse(result.reporter)
    self.assertEqual(result.owner.user, 'users/{}'.format(self.user_1.user_id))
    self.assertEqual(len(result.cc_users), 0)
    self.assertFalse(result.cc_users)
    self.assertEqual(len(result.labels), 1)
    self.assertEqual(result.labels[0].label, self.template_1.labels[0])
    self.assertEqual(result.labels[0].derivation, EXPLICIT_DERIVATION)
    self.assertEqual(len(result.components), 1)
    self.assertEqual(
        result.components[0].component, 'projects/{}/componentDefs/{}'.format(
            self.project_1.project_name, self.template_1.component_ids[0]))
    self.assertEqual(result.components[0].derivation, EXPLICIT_DERIVATION)
    self.assertEqual(len(result.field_values), 2)
    self.assertEqual(
        result.field_values[0].field, 'projects/{}/fieldDefs/{}'.format(
            self.project_1.project_name, self.field_def_1_name))
    self.assertEqual(result.field_values[0].value, self.fv_1_value)
    self.assertEqual(result.field_values[0].derivation, EXPLICIT_DERIVATION)
    expected_name = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_3], self.project_1.project_id,
        self.services).get(self.field_def_3)
    self.assertEqual(
        result.field_values[1],
        issue_objects_pb2.Issue.FieldValue(
            field=expected_name,
            value=self.template_1_label1_value,
            derivation=EXPLICIT_DERIVATION))
    self.assertFalse(result.blocked_on_issue_refs)
    self.assertFalse(result.blocking_issue_refs)
    self.assertFalse(result.attachment_count)
    self.assertFalse(result.star_count)
    self.assertEqual(len(result.approval_values), 1)
    self.assertEqual(len(result.approval_values[0].approvers), 1)
    self.assertEqual(
        result.approval_values[0].approvers[0], 'users/{}'.format(
            self.user_2.user_id))
    self.assertEqual(result.approval_values[0].phase, self.phase_1.name)
    self.assertEqual(len(result.phases), 1)
    self.assertEqual(result.phases[0], self.phase_1.name)

  def test_FillIssueFromTemplate_NoPhase(self):
    result = self.converter._FillIssueFromTemplate(
        self.template_3, self.project_1.project_id)
    self.assertEqual(len(result.field_values), 1)
    self.assertEqual(
        result.field_values[0].field, 'projects/{}/fieldDefs/{}'.format(
            self.project_1.project_name, self.field_def_1_name))
    self.assertEqual(result.field_values[0].value, self.fv_1_value)
    self.assertEqual(result.field_values[0].derivation, EXPLICIT_DERIVATION)
    self.assertEqual(len(result.approval_values), 1)
    self.assertEqual(len(result.approval_values[0].approvers), 1)
    self.assertEqual(
        result.approval_values[0].approvers[0], 'users/{}'.format(
            self.user_2.user_id))
    self.assertEqual(result.approval_values[0].phase, '')
    self.assertEqual(len(result.phases), 0)

  def testConvertIssueTemplates(self):
    result = self.converter.ConvertIssueTemplates(
        self.project_1.project_id, [self.template_1])
    self.assertEqual(len(result), 1)
    actual = result[0]
    self.assertEqual(
        actual.name, 'projects/{}/templates/{}'.format(
            self.project_1.project_name, self.template_1.name))
    self.assertEqual(actual.summary_must_be_edited, False)
    self.assertEqual(
        actual.template_privacy,
        project_objects_pb2.IssueTemplate.TemplatePrivacy.Value('PUBLIC'))
    self.assertEqual(
        actual.default_owner,
        project_objects_pb2.IssueTemplate.DefaultOwner.Value(
            'DEFAULT_OWNER_UNSPECIFIED'))
    self.assertEqual(actual.component_required, False)
    self.assertEqual(actual.admins, ['users/{}'.format(self.user_2.user_id)])
    self.assertEqual(
        actual.issue,
        self.converter._FillIssueFromTemplate(
            self.template_1, self.project_1.project_id))

  def testConvertIssueTemplates_IgnoresNonExistentTemplate(self):
    result = self.converter.ConvertIssueTemplates(
        self.project_1.project_id, [self.dne_template])
    self.assertEqual(len(result), 0)

  def testConvertLabels_OmitsFieldDefs(self):
    """It omits field def labels"""
    input_labels = ['pri-1', '{}-2'.format(self.field_def_3_name)]
    result = self.converter.ConvertLabels(
        input_labels, [], self.project_1.project_id)
    self.assertEqual(len(result), 1)
    expected = issue_objects_pb2.Issue.LabelValue(
        label=input_labels[0], derivation=EXPLICIT_DERIVATION)
    self.assertEqual(result[0], expected)

  def testConvertLabels_DerivedLabels(self):
    """It handles derived labels"""
    input_labels = ['pri-1']
    result = self.converter.ConvertLabels(
        [], input_labels, self.project_1.project_id)
    self.assertEqual(len(result), 1)
    expected = issue_objects_pb2.Issue.LabelValue(
        label=input_labels[0], derivation=RULE_DERIVATION)
    self.assertEqual(result[0], expected)

  def testConvertLabels(self):
    """It includes both non-derived and derived labels"""
    input_labels = ['pri-1', '{}-2'.format(self.field_def_3_name)]
    input_der_labels = ['{}-3'.format(self.field_def_3_name), 'job-secret']
    result = self.converter.ConvertLabels(
        input_labels, input_der_labels, self.project_1.project_id)
    self.assertEqual(len(result), 2)
    expected_0 = issue_objects_pb2.Issue.LabelValue(
        label=input_labels[0], derivation=EXPLICIT_DERIVATION)
    self.assertEqual(result[0], expected_0)
    expected_1 = issue_objects_pb2.Issue.LabelValue(
        label=input_der_labels[1], derivation=RULE_DERIVATION)
    self.assertEqual(result[1], expected_1)

  def testConvertLabels_Empty(self):
    result = self.converter.ConvertLabels([], [], self.project_1.project_id)
    self.assertEqual(result, [])

  def testConvertEnumFieldValues_OnlyFieldDefs(self):
    """It only returns enum field values"""
    expected_value = '2'
    input_labels = [
        'pri-1', '{}-{}'.format(self.field_def_3_name, expected_value)
    ]
    result = self.converter.ConvertEnumFieldValues(
        input_labels, [], self.project_1.project_id)
    self.assertEqual(len(result), 1)
    expected_name = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_3], self.project_1.project_id,
        self.services).get(self.field_def_3)
    expected = issue_objects_pb2.Issue.FieldValue(
        field=expected_name,
        value=expected_value,
        derivation=EXPLICIT_DERIVATION)
    self.assertEqual(result[0], expected)

  def testConvertEnumFieldValues_DerivedLabels(self):
    """It handles derived enum field values"""
    expected_value = '2'
    input_der_labels = [
        'pri-1', '{}-{}'.format(self.field_def_3_name, expected_value)
    ]
    result = self.converter.ConvertEnumFieldValues(
        [], input_der_labels, self.project_1.project_id)
    self.assertEqual(len(result), 1)
    expected_name = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_3], self.project_1.project_id,
        self.services).get(self.field_def_3)
    expected = issue_objects_pb2.Issue.FieldValue(
        field=expected_name, value=expected_value, derivation=RULE_DERIVATION)
    self.assertEqual(result[0], expected)

  def testConvertEnumFieldValues_Empty(self):
    result = self.converter.ConvertEnumFieldValues(
        [], [], self.project_1.project_id)
    self.assertEqual(result, [])

  def testConvertEnumFieldValues_ProjectSpecific(self):
    """It only considers field defs from specified project"""
    expected_value = '2'
    input_labels = [
        '{}-{}'.format(self.field_def_3_name, expected_value),
        '{}-ipsum'.format(self.field_def_project2_name)
    ]
    result = self.converter.ConvertEnumFieldValues(
        input_labels, [], self.project_1.project_id)
    self.assertEqual(len(result), 1)
    expected_name = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_3], self.project_1.project_id,
        self.services).get(self.field_def_3)
    expected = issue_objects_pb2.Issue.FieldValue(
        field=expected_name,
        value=expected_value,
        derivation=EXPLICIT_DERIVATION)
    self.assertEqual(result[0], expected)

  def testConvertEnumFieldValues(self):
    """It handles derived enum field values"""
    expected_value_0 = '2'
    expected_value_1 = 'macOS'
    input_labels = [
        'pri-1', '{}-{}'.format(self.field_def_3_name, expected_value_0),
        '{}-ipsum'.format(self.field_def_project2_name)
    ]
    input_der_labels = [
        '{}-{}'.format(self.field_def_4_name, expected_value_1), 'foo-bar'
    ]
    result = self.converter.ConvertEnumFieldValues(
        input_labels, input_der_labels, self.project_1.project_id)
    self.assertEqual(len(result), 2)
    expected_0_name = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_3], self.project_1.project_id,
        self.services).get(self.field_def_3)
    expected_0 = issue_objects_pb2.Issue.FieldValue(
        field=expected_0_name,
        value=expected_value_0,
        derivation=EXPLICIT_DERIVATION)
    self.assertEqual(result[0], expected_0)
    expected_1_name = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_4], self.project_1.project_id,
        self.services).get(self.field_def_4)
    expected_1 = issue_objects_pb2.Issue.FieldValue(
        field=expected_1_name,
        value=expected_value_1,
        derivation=RULE_DERIVATION)
    self.assertEqual(result[1], expected_1)

  @patch('project.project_helpers.GetThumbnailUrl')
  def testConvertProject(self, mock_GetThumbnailUrl):
    """We can convert a Project."""
    mock_GetThumbnailUrl.return_value = 'xyz'
    expected_api_project = project_objects_pb2.Project(
        name='projects/{}'.format(self.project_1.project_name),
        display_name=self.project_1.project_name,
        summary=self.project_1.summary,
        thumbnail_url='xyz')
    self.assertEqual(
        expected_api_project, self.converter.ConvertProject(self.project_1))

  @patch('project.project_helpers.GetThumbnailUrl')
  def testConvertProjects(self, mock_GetThumbnailUrl):
    """We can convert a Sequence of Projects."""
    mock_GetThumbnailUrl.return_value = 'xyz'
    expected_api_projects = [
        project_objects_pb2.Project(
            name='projects/{}'.format(self.project_1.project_name),
            display_name=self.project_1.project_name,
            summary=self.project_1.summary,
            thumbnail_url='xyz'),
        project_objects_pb2.Project(
            name='projects/{}'.format(self.project_2.project_name),
            display_name=self.project_2.project_name,
            summary=self.project_2.summary,
            thumbnail_url='xyz')
    ]
    self.assertEqual(
        expected_api_projects,
        self.converter.ConvertProjects([self.project_1, self.project_2]))

  def testConvertProjectConfig(self):
    """We can convert a project_config"""
    project_config = self.services.config.GetProjectConfig(
        self.cnxn, self.project_1.project_id)
    expected_grid_config = project_objects_pb2.ProjectConfig.GridViewConfig(
        default_x_attr=project_config.default_x_attr,
        default_y_attr=project_config.default_y_attr)
    template_names = rnc.ConvertTemplateNames(
        self.cnxn, project_config.project_id, [
            project_config.default_template_for_developers,
            project_config.default_template_for_users
        ], self.services)
    expected_api_config = project_objects_pb2.ProjectConfig(
        name=rnc.ConvertProjectConfigName(
            self.cnxn, self.project_1.project_id, self.services),
        exclusive_label_prefixes=project_config.exclusive_label_prefixes,
        member_default_query=project_config.member_default_query,
        default_sort=project_config.default_sort_spec,
        default_columns=[
            issue_objects_pb2.IssuesListColumn(column=col)
            for col in project_config.default_col_spec.split()
        ],
        project_grid_config=expected_grid_config,
        member_default_template=template_names.get(
            project_config.default_template_for_developers),
        non_members_default_template=template_names.get(
            project_config.default_template_for_users),
        revision_url_format=self.project_1.revision_url_format,
        custom_issue_entry_url=project_config.custom_issue_entry_url)
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    self.assertEqual(
        expected_api_config,
        self.converter.ConvertProjectConfig(project_config))

  def testConvertProjectConfig_NonMembers(self):
    """We can convert a project_config for non project members"""
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_2, self.services)
    project_config = self.services.config.GetProjectConfig(
        self.cnxn, self.project_1.project_id)
    api_config = self.converter.ConvertProjectConfig(project_config)

    expected_default_query = project_config.member_default_query
    self.assertEqual(expected_default_query, api_config.member_default_query)

    expected_member_default_template = rnc.ConvertTemplateNames(
        self.cnxn, project_config.project_id,
        [project_config.default_template_for_developers], self.services).get(
            project_config.default_template_for_developers)
    self.assertEqual(
        expected_member_default_template, api_config.member_default_template)

  def testCreateProjectMember(self):
    """We can create a ProjectMember."""
    expected_project_member = project_objects_pb2.ProjectMember(
        name='projects/proj/members/111',
        role=project_objects_pb2.ProjectMember.ProjectRole.Value('OWNER'))
    self.assertEqual(
        expected_project_member,
        self.converter.CreateProjectMember(self.cnxn, 789, 111, 'OWNER'))
