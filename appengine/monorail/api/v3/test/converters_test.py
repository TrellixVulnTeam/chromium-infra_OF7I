# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
"""Tests for converting internal protorpc to external protoc."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import difflib
import logging
import unittest

from mock import patch
from google.protobuf import timestamp_pb2

from api import resource_name_converters as rnc
from api.v3 import converters
from api.v3.api_proto import feature_objects_pb2
from api.v3.api_proto import issue_objects_pb2
from api.v3.api_proto import user_objects_pb2
from api.v3.api_proto import project_objects_pb2
from framework import authdata
from framework import exceptions
from framework import monorailcontext
from testing import fake
from testing import testing_helpers
from services import service_manager
from proto import tracker_pb2
from tracker import tracker_bizobj as tbo

EXPLICIT_DERIVATION = issue_objects_pb2.Derivation.Value('EXPLICIT')
RULE_DERIVATION = issue_objects_pb2.Derivation.Value('RULE')
Choice = project_objects_pb2.FieldDef.EnumTypeSettings.Choice


class ConverterFunctionsTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        usergroup=fake.UserGroupService(),
        user=fake.UserService(),
        config=fake.ConfigService(),
        template=fake.TemplateService(),
        features=fake.FeaturesService())
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
        self.project_1.project_id,
        self.field_def_1_name,
        'STR_TYPE',
        admin_ids=[self.user_1.user_id],
        is_required=True,
        is_multivalued=True,
        is_phase_field=True,
        regex='abc')
    self.field_def_2_name = 'test_field_2'
    self.field_def_2 = self._CreateFieldDef(
        self.project_1.project_id,
        self.field_def_2_name,
        'INT_TYPE',
        max_value=37,
        is_niche=True)
    self.field_def_3_name = 'days'
    self.field_def_3 = self._CreateFieldDef(
        self.project_1.project_id, self.field_def_3_name, 'ENUM_TYPE')
    self.field_def_4_name = 'OS'
    self.field_def_4 = self._CreateFieldDef(
        self.project_1.project_id, self.field_def_4_name, 'ENUM_TYPE')
    self.field_def_5_name = 'yellow'
    self.field_def_5 = self._CreateFieldDef(
        self.project_1.project_id, self.field_def_5_name, 'ENUM_TYPE')
    self.field_def_7_name = 'redredred'
    self.field_def_7 = self._CreateFieldDef(
        self.project_1.project_id,
        self.field_def_7_name,
        'ENUM_TYPE',
        is_restricted_field=True,
        editor_ids=[self.user_1.user_id])
    self.field_def_8_name = 'dogandcat'
    self.field_def_8 = self._CreateFieldDef(
        self.project_1.project_id,
        self.field_def_8_name,
        'USER_TYPE',
        needs_member=True,
        needs_perm='EDIT_PROJECT',
        notify_on=tracker_pb2.NotifyTriggers.ANY_COMMENT)
    self.field_def_9_name = 'catanddog'
    self.field_def_9 = self._CreateFieldDef(
        self.project_1.project_id,
        self.field_def_9_name,
        'DATE_TYPE',
        date_action_str='ping_owner_only')
    self.field_def_10_name = 'url'
    self.field_def_10 = self._CreateFieldDef(
        self.project_1.project_id, self.field_def_10_name, 'URL_TYPE')
    self.field_def_project2_name = 'lorem'
    self.field_def_project2 = self._CreateFieldDef(
        self.project_2.project_id, self.field_def_project2_name, 'ENUM_TYPE')
    self.approval_def_1_name = 'approval_field_1'
    self.approval_def_1_id = self._CreateFieldDef(
        self.project_1.project_id,
        self.approval_def_1_name,
        'APPROVAL_TYPE',
        docstring='ad_1_docstring',
        admin_ids=[self.user_1.user_id])
    self.approval_def_1 = tracker_pb2.ApprovalDef(
        approval_id=self.approval_def_1_id,
        approver_ids=[self.user_2.user_id],
        survey='approval_def_1 survey')
    approval_defs = [self.approval_def_1]
    self.field_def_6_name = 'simonsays'
    self.field_def_6 = self._CreateFieldDef(
        self.project_1.project_id,
        self.field_def_6_name,
        'STR_TYPE',
        approval_id=self.approval_def_1_id)
    self.dne_field_def_id = 999999
    self.fv_1_value = 'some_string_field_value'
    self.fv_1 = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value=self.fv_1_value, derived=False)
    self.fv_1_derived = fake.MakeFieldValue(
        field_id=self.field_def_1, str_value=self.fv_1_value, derived=True)
    self.fv_6 = fake.MakeFieldValue(
        field_id=self.field_def_6, str_value='touch-nose')
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
    self.labeldef_1 = tracker_pb2.LabelDef(
        label='white-mountain',
        label_docstring='test label doc string for white-mountain')
    self.labeldef_2 = tracker_pb2.LabelDef(
        label='yellow-submarine',
        label_docstring='Submarine choice for yellow enum field')
    self.labeldef_3 = tracker_pb2.LabelDef(
        label='yellow-basket',
        label_docstring='Basket choice for yellow enum field')
    self.labeldef_4 = tracker_pb2.LabelDef(
        label='yellow-tasket',
        label_docstring='Deprecated tasket choice for yellow enum field',
        deprecated=True)
    self.labeldef_5 = tracker_pb2.LabelDef(
        label='mont-blanc',
        label_docstring='test label doc string for mont-blanc',
        deprecated=True)
    self.predefined_labels = [
        self.labeldef_1, self.labeldef_2, self.labeldef_3, self.labeldef_4,
        self.labeldef_5
    ]
    test_label_ids = {}
    for index, ld in enumerate(self.predefined_labels):
      test_label_ids[ld.label] = index
    self.services.config.TestAddLabelsDict(test_label_ids)
    self.status_1 = tracker_pb2.StatusDef(
        status='New', means_open=True, status_docstring='status_1 docstring')
    self.status_2 = tracker_pb2.StatusDef(
        status='Duplicate',
        means_open=False,
        status_docstring='status_2 docstring')
    self.status_3 = tracker_pb2.StatusDef(
        status='Accepted',
        means_open=True,
        status_docstring='status_3_docstring')
    self.status_4 = tracker_pb2.StatusDef(
        status='Gibberish',
        means_open=True,
        status_docstring='status_4_docstring',
        deprecated=True)
    self.predefined_statuses = [
        self.status_1, self.status_2, self.status_3, self.status_4
    ]
    self.component_def_1_path = 'foo'
    self.component_def_1_id = self.services.config.CreateComponentDef(
        self.cnxn, self.project_1.project_id, self.component_def_1_path,
        'cd1_docstring', False, [self.user_1.user_id], [self.user_2.user_id],
        self.PAST_TIME, self.user_1.user_id, [0, 1, 2, 3, 4])
    self.component_def_2_path = 'foo>bar'
    self.component_def_2_id = self.services.config.CreateComponentDef(
        self.cnxn, self.project_1.project_id, self.component_def_2_path,
        'cd2_docstring', True, [self.user_1.user_id], [self.user_2.user_id],
        self.PAST_TIME, self.user_1.user_id, [])
    self.services.config.UpdateConfig(
        self.cnxn,
        self.project_1,
        statuses_offer_merge=[self.status_2.status],
        excl_label_prefixes=['type', 'priority'],
        default_template_for_developers=self.template_2.template_id,
        default_template_for_users=self.template_1.template_id,
        list_prefs=('ID Summary', 'ID', 'status', 'owner', 'owner:me'),
        # UpdateConfig accepts tuples rather than protorpc *Defs
        well_known_labels=[
            (ld.label, ld.label_docstring, ld.deprecated)
            for ld in self.predefined_labels
        ],
        approval_defs=[
            (ad.approval_id, ad.approver_ids, ad.survey) for ad in approval_defs
        ],
        well_known_statuses=[
            (sd.status, sd.status_docstring, sd.means_open, sd.deprecated)
            for sd in self.predefined_statuses
        ])
    # base_query_id 2 equates to "is:open", defined in tracker_constants.
    self.psq_1 = tracker_pb2.SavedQuery(
        query_id=2, name='psq1 name', base_query_id=2, query='foo=bar')
    self.psq_2 = tracker_pb2.SavedQuery(
        query_id=3, name='psq2 name', query='fizz=buzz')
    self.services.features.UpdateCannedQueries(
        self.cnxn, self.project_1.project_id, [self.psq_1, self.psq_2])

  def _CreateFieldDef(
      self,
      project_id,
      field_name,
      field_type_str,
      docstring=None,
      min_value=None,
      max_value=None,
      regex=None,
      needs_member=None,
      needs_perm=None,
      grants_perm=None,
      notify_on=None,
      date_action_str=None,
      admin_ids=None,
      editor_ids=None,
      is_required=False,
      is_niche=False,
      is_multivalued=False,
      is_phase_field=False,
      approval_id=None,
      is_restricted_field=False):
    """Calls CreateFieldDef with reasonable defaults, returns the ID."""
    if admin_ids is None:
      admin_ids = []
    if editor_ids is None:
      editor_ids = []
    return self.services.config.CreateFieldDef(
        self.cnxn,
        project_id,
        field_name,
        field_type_str,
        None,
        None,
        is_required,
        is_niche,
        is_multivalued,
        min_value,
        max_value,
        regex,
        needs_member,
        needs_perm,
        grants_perm,
        notify_on,
        date_action_str,
        docstring,
        admin_ids,
        editor_ids,
        is_phase_field=is_phase_field,
        approval_id=approval_id,
        is_restricted_field=is_restricted_field)

  def _GetFieldDefById(self, project_id, fd_id):
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    return [fd for fd in config.field_defs if fd.field_id == fd_id][0]

  def _GetApprovalDefById(self, project_id, ad_id):
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    return [ad for ad in config.approval_defs if ad.approval_id == ad_id][0]

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
        owner= 'users/111',
        summary=hotlist.summary,
        description=hotlist.description,
        editors=['users/222', 'users/333'],
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
        owner='users/111',
        summary=hotlist.summary,
        description=hotlist.description,
        hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
            'PRIVATE'))
    self.converter.user_auth = authdata.AuthData.FromUser(
        self.cnxn, self.user_1, self.services)
    self.assertEqual(
        expected_api_hotlist, self.converter.ConvertHotlist(hotlist))

  def testConvertHotlists(self):
    """We can convert several Hotlists."""
    hotlists = [
        fake.Hotlist(
            'Hotlist-Name',
            241,
            owner_ids=[111],
            summary='Hotlist summary',
            description='Hotlist Description'),
        fake.Hotlist(
            'Hotlist-Name',
            241,
            owner_ids=[111],
            summary='Hotlist summary',
            description='Hotlist Description')
    ]
    self.assertEqual(2, len(self.converter.ConvertHotlists(hotlists)))

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
            adder= 'users/111',
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
            issue_objects_pb2.FieldValue(
                derivation=EXPLICIT_DERIVATION,
                field='projects/proj/fieldDefs/test_field_1',
                value=self.fv_1_value,
            ),
            issue_objects_pb2.FieldValue(
                derivation=RULE_DERIVATION,
                field='projects/proj/fieldDefs/test_field_1',
                value=self.fv_1_value,
            ),
            issue_objects_pb2.FieldValue(
                derivation=EXPLICIT_DERIVATION,
                field='projects/proj/fieldDefs/days',
                value='1',
            ),
            issue_objects_pb2.FieldValue(
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

  def testIngestIssue(self):
    ingest = issue_objects_pb2.Issue(
        summary='sum',
        status=issue_objects_pb2.Issue.StatusValue(
            status='new', derivation=RULE_DERIVATION),
        owner=issue_objects_pb2.Issue.UserValue(
            derivation=EXPLICIT_DERIVATION, user='users/111'),
        cc_users=[
            issue_objects_pb2.Issue.UserValue(
                derivation=EXPLICIT_DERIVATION, user='users/new@user.com'),
            issue_objects_pb2.Issue.UserValue(
                derivation=RULE_DERIVATION, user='users/333')
        ],
        components=[
            issue_objects_pb2.Issue.ComponentValue(
                component='projects/proj/componentDefs/%d' %
                self.component_def_1_id),
            issue_objects_pb2.Issue.ComponentValue(
                component='projects/proj/componentDefs/%d' %
                self.component_def_2_id),
        ],
        labels=[
            issue_objects_pb2.Issue.LabelValue(
                derivation=EXPLICIT_DERIVATION, label='a'),
            issue_objects_pb2.Issue.LabelValue(
                derivation=EXPLICIT_DERIVATION, label='key-explicit'),
            issue_objects_pb2.Issue.LabelValue(
                derivation=RULE_DERIVATION, label='derived1'),
            issue_objects_pb2.Issue.LabelValue(
                derivation=RULE_DERIVATION, label='key-derived')
        ],
        field_values=[
            issue_objects_pb2.FieldValue(
                derivation=EXPLICIT_DERIVATION,
                field='projects/proj/fieldDefs/test_field_1',
                value='multivalue1',
            ),
            issue_objects_pb2.FieldValue(
                derivation=RULE_DERIVATION,
                field='projects/proj/fieldDefs/test_field_1',
                value='multivalue2',
            ),
            issue_objects_pb2.FieldValue(
                derivation=EXPLICIT_DERIVATION,
                field='projects/proj/fieldDefs/days',
                value='1',
            ),
            issue_objects_pb2.FieldValue(
                derivation=RULE_DERIVATION,
                field='projects/proj/fieldDefs/OS',
                value='mac',
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/test_field_2',
                value='38',  # Max value not checked.
            ),
            issue_objects_pb2.FieldValue(  # Multivalue not checked.
                field='projects/proj/fieldDefs/test_field_2',
                value='0'  # Confirm we ingest 0 rather than None.
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/dogandcat',
                value='users/111',
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/dogandcat',
                value='users/404',  # User lookup not attempted.
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/dogandcat',
                value='users/nobody@no.com',  # Parsed to '-1' for INVALID_USER.
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/catanddog',
                value='2020-01-01',
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/catanddog',
                value='2100-01-01',
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/catanddog',
                value='1000-01-01',
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/url',
                value='garbage',
            ),
            # TODO(crbug/monorail/8050): FieldValue mistakes are ignored.
            issue_objects_pb2.FieldValue(),
            issue_objects_pb2.FieldValue(field='garbage'),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/test_field_2',
                value='garbage',
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/dogandcat',
                value='garbage',
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/dogandcat',
            ),
            issue_objects_pb2.FieldValue(
                field='projects/proj/fieldDefs/catanddog',
                value='garbagedate',
            ),
            issue_objects_pb2.FieldValue(  # Project does not exist.
                field='projects/noproject/fieldDefs/something',
                value='something'
            ),
            issue_objects_pb2.FieldValue(  # Different project.
                field='projects/goose/fieldDefs/lorem',
                value='something'
            ),
        ],
        merged_into_issue_ref=issue_objects_pb2.IssueRef(ext_identifier='b/1'),
        blocked_on_issue_refs=[
            # Reversing natural ordering to ensure order is respected.
            issue_objects_pb2.IssueRef(issue='projects/goose/issues/4'),
            issue_objects_pb2.IssueRef(issue='projects/proj/issues/3'),
            issue_objects_pb2.IssueRef(ext_identifier='b/555'),
            issue_objects_pb2.IssueRef(ext_identifier='b/2')
        ],
        blocking_issue_refs=[
            issue_objects_pb2.IssueRef(issue='projects/goose/issues/5'),
            issue_objects_pb2.IssueRef(ext_identifier='b/3')
        ],
        # All the following fields should be ignored.
        name='projects/proj/issues/1',
        state=issue_objects_pb2.IssueContentState.Value('SPAM'),
        reporter='users/111',
        create_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        component_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        status_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        owner_modify_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        star_count=1,
        attachment_count=5,
        phases=[self.phase_1.name])

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

    actual = self.converter.IngestIssue(ingest, self.project_1.project_id)

    expected_cc1_id = self.services.user.LookupUserID(
        self.cnxn, 'new@user.com', autocreate=False)
    expected_field_values = [
        tracker_pb2.FieldValue(
            field_id=self.field_def_1,
            str_value=u'multivalue1',
            derived=False,
        ),
        tracker_pb2.FieldValue(
            field_id=self.field_def_1,
            str_value=u'multivalue2',
            derived=False,
        ),
        tracker_pb2.FieldValue(
            field_id=self.field_def_2, int_value=38, derived=False),
        tracker_pb2.FieldValue(
            field_id=self.field_def_2, int_value=0, derived=False),
        tracker_pb2.FieldValue(
            field_id=self.field_def_8, user_id=111, derived=False),
        tracker_pb2.FieldValue(
            field_id=self.field_def_8, user_id=404, derived=False),
        tracker_pb2.FieldValue(
            field_id=self.field_def_8, user_id=-1, derived=False),
        tracker_pb2.FieldValue(
            field_id=self.field_def_9, date_value=1577836800, derived=False),
        tracker_pb2.FieldValue(
            field_id=self.field_def_9, date_value=4102444800, derived=False),
        tracker_pb2.FieldValue(
            field_id=self.field_def_9, date_value=-30610224000, derived=False),
        tracker_pb2.FieldValue(
            field_id=self.field_def_10,
            url_value=u'http://garbage',
            derived=False),
    ]
    expected = tracker_pb2.Issue(
        summary=u'sum',
        status=u'new',
        owner_id=111,
        cc_ids=[expected_cc1_id, 333],
        component_ids=[self.component_def_1_id, self.component_def_2_id],
        merged_into_external=u'b/1',
        labels=[
            u'a', u'key-explicit', u'derived1', u'key-derived', u'days-1',
            u'OS-mac'
        ],
        field_values=expected_field_values,
        blocked_on_iids=[blocked_on_2.issue_id, blocked_on_1.issue_id],
        blocking_iids=[blocking.issue_id],
        dangling_blocked_on_refs=[
            tracker_pb2.DanglingIssueRef(ext_issue_identifier=u'b/555'),
            tracker_pb2.DanglingIssueRef(ext_issue_identifier=u'b/2')
        ],
        dangling_blocking_refs=[
            tracker_pb2.DanglingIssueRef(ext_issue_identifier=u'b/3')
        ],
    )
    self.AssertProtosEqual(actual, expected)

  def AssertProtosEqual(self, actual, expected):
    """Asserts equal, printing a diff if not."""
    # TODO(jessan): If others find this useful, move to a shared testing lib.
    try:
      self.assertEqual(actual, expected)
    except AssertionError as e:
      # Append a diff to the normal error message.
      expected_str = str(expected).splitlines(1)
      actual_str = str(actual).splitlines(1)
      diff = difflib.unified_diff(actual_str, expected_str)
      err_msg = '%s\nProto actual vs expected diff:\n %s' % (e, ''.join(diff))
      raise AssertionError(err_msg)

  def testIngestIssue_Minimal(self):
    """Test IngestIssue with as few fields set as possible."""
    minimal = issue_objects_pb2.Issue(
        status=issue_objects_pb2.Issue.StatusValue(status='new')
    )
    expected = tracker_pb2.Issue(
        summary='', # Summary gets set to empty str on conversion.
        status='new'
    )
    actual = self.converter.IngestIssue(minimal, self.project_1.project_id)
    self.assertEqual(actual, expected)

  def testIngestIssue_NoSuchProject(self):
    self.services.config.strict = True
    ingest = issue_objects_pb2.Issue(
        status=issue_objects_pb2.Issue.StatusValue(status='new'))
    with self.assertRaises(exceptions.NoSuchProjectException):
      self.converter.IngestIssue(ingest, -1)

  def testIngestIssue_Errors(self):
    invalid_issue_ref = issue_objects_pb2.IssueRef(
        ext_identifier='b/1',
        issue='projects/proj/issues/1')
    ingest = issue_objects_pb2.Issue(
        summary='sum',
        owner=issue_objects_pb2.Issue.UserValue(
            derivation=EXPLICIT_DERIVATION, user='users/nonexisting@user.com'),
        cc_users=[
            issue_objects_pb2.Issue.UserValue(
                derivation=EXPLICIT_DERIVATION, user='invalidFormat1'),
            issue_objects_pb2.Issue.UserValue(
                derivation=RULE_DERIVATION, user='invalidFormat2')
        ],
        components=[
            issue_objects_pb2.Issue.ComponentValue(
                component='projects/proj/componentDefs/404')
        ],
        merged_into_issue_ref=invalid_issue_ref,
        blocked_on_issue_refs=[
            issue_objects_pb2.IssueRef(),
            issue_objects_pb2.IssueRef(issue='projects/404/issues/1')
        ],
        blocking_issue_refs=[
            issue_objects_pb2.IssueRef(issue='projects/proj/issues/404')
        ],
    )
    error_messages = [
        r'.+not found when ingesting owner',
        r'.+cc_users: Invalid resource name: invalidFormat1.',
        r'Status is required when creating an issue',
        r'.+components: Component not found: 404.',
        r'.+merged_into_ref: IssueRefs MUST NOT have both.+',
        r'.+blocked_on_issue_refs: IssueRefs MUST have.+',
        r'.+blocked_on_issue_refs: Project 404 not found.',
        r'.+blocking_issue_refs: Issue.+404.+not found'
    ]
    error_messages_re = '\n'.join(error_messages)
    with self.assertRaisesRegexp(exceptions.InputException, error_messages_re):
      self.converter.IngestIssue(ingest, self.project_1.project_id)

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
    expected_value = issue_objects_pb2.FieldValue(
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
    expected_1 = issue_objects_pb2.FieldValue(
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
    expected_2 = issue_objects_pb2.FieldValue(
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
    expected_1 = issue_objects_pb2.FieldValue(
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
    expected = rnc.ConvertUserName(user_id)
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

  def testConvertApprovalValues_Issue(self):
    """We can convert issue approval_values."""
    name = rnc.ConvertApprovalValueNames(
        self.cnxn, self.issue_1.issue_id, self.services)[self.av_1.approval_id]
    approval_def_name = rnc.ConvertApprovalDefNames(
        self.cnxn, [self.approval_def_1_id], self.project_1.project_id,
        self.services)[self.approval_def_1_id]
    approvers = [rnc.ConvertUserName(self.user_2.user_id)]
    status = issue_objects_pb2.ApprovalValue.ApprovalStatus.Value(
        'APPROVAL_STATUS_UNSPECIFIED')
    set_time = timestamp_pb2.Timestamp()
    set_time.FromSeconds(self.PAST_TIME)
    setter = rnc.ConvertUserName(self.user_1.user_id)
    api_fvs = self.converter.ConvertFieldValues(
        [self.fv_6], self.project_1.project_id, [self.phase_1])

    output = self.converter.ConvertApprovalValues(
        [self.av_1], [self.fv_1, self.fv_6], [self.phase_1],
        issue_id=self.issue_1.issue_id)
    expected = issue_objects_pb2.ApprovalValue(
        name=name,
        approval_def=approval_def_name,
        approvers=approvers,
        status=status,
        set_time=set_time,
        setter=setter,
        phase=self.phase_1.name,
        field_values=api_fvs)
    self.assertEqual([expected], output)

  def testConvertApprovalValues_Templates(self):
    """We can convert template approval_values."""
    approval_def_name = rnc.ConvertApprovalDefNames(
        self.cnxn, [self.approval_def_1_id], self.project_1.project_id,
        self.services)[self.approval_def_1_id]
    approvers = [rnc.ConvertUserName(self.user_2.user_id)]
    status = issue_objects_pb2.ApprovalValue.ApprovalStatus.Value(
        'APPROVAL_STATUS_UNSPECIFIED')
    set_time = timestamp_pb2.Timestamp()
    set_time.FromSeconds(self.PAST_TIME)
    setter = rnc.ConvertUserName(self.user_1.user_id)
    api_fvs = self.converter.ConvertFieldValues(
        [self.fv_6], self.project_1.project_id, [self.phase_1])

    output = self.converter.ConvertApprovalValues(
        [self.av_1], [self.fv_1, self.fv_6], [self.phase_1],
        project_id=self.project_1.project_id)
    expected = issue_objects_pb2.ApprovalValue(
        approval_def=approval_def_name,
        approvers=approvers,
        status=status,
        set_time=set_time,
        setter=setter,
        phase=self.phase_1.name,
        field_values=api_fvs)
    self.assertEqual([expected], output)

  def testConvertApprovalValues_NoPhase(self):
    approval_def_name = rnc.ConvertApprovalDefNames(
        self.cnxn, [self.approval_def_1_id], self.project_1.project_id,
        self.services)[self.approval_def_1_id]
    approvers = [rnc.ConvertUserName(self.user_2.user_id)]
    status = issue_objects_pb2.ApprovalValue.ApprovalStatus.Value(
        'APPROVAL_STATUS_UNSPECIFIED')
    set_time = timestamp_pb2.Timestamp()
    set_time.FromSeconds(self.PAST_TIME)
    setter = rnc.ConvertUserName(self.user_1.user_id)
    expected = issue_objects_pb2.ApprovalValue(
        approval_def=approval_def_name,
        approvers=approvers,
        status=status,
        set_time=set_time,
        setter=setter)

    output = self.converter.ConvertApprovalValues(
        [self.av_1], [], [], project_id=self.project_1.project_id)
    self.assertEqual([expected], output)

  def testConvertApprovalValues_Empty(self):
    output = self.converter.ConvertApprovalValues(
        [], [], [], project_id=self.project_1.project_id)
    self.assertEqual([], output)

  def testConvertApprovalValues_IgnoresNullFieldDefs(self):
    """It ignores approval values referencing a non-existent field"""
    av = fake.MakeApprovalValue(self.dne_field_def_id)

    output = self.converter.ConvertApprovalValues(
        [av], [], [], issue_id=self.issue_1.issue_id)
    self.assertEqual([], output)

  def test_ComputeApprovalValueStatus_NOT_SET(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NOT_SET),
        issue_objects_pb2.ApprovalValue.ApprovalStatus.Value(
            'APPROVAL_STATUS_UNSPECIFIED'))

  def test_ComputeApprovalValueStatus_NEEDS_REVIEW(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NEEDS_REVIEW),
        issue_objects_pb2.ApprovalValue.ApprovalStatus.Value('NEEDS_REVIEW'))

  def test_ComputeApprovalValueStatus_NA(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NA),
        issue_objects_pb2.ApprovalValue.ApprovalStatus.Value('NA'))

  def test_ComputeApprovalValueStatus_REVIEW_REQUESTED(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.REVIEW_REQUESTED),
        issue_objects_pb2.ApprovalValue.ApprovalStatus.Value(
            'REVIEW_REQUESTED'))

  def test_ComputeApprovalValueStatus_REVIEW_STARTED(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.REVIEW_STARTED),
        issue_objects_pb2.ApprovalValue.ApprovalStatus.Value('REVIEW_STARTED'))

  def test_ComputeApprovalValueStatus_NEED_INFO(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NEED_INFO),
        issue_objects_pb2.ApprovalValue.ApprovalStatus.Value('NEED_INFO'))

  def test_ComputeApprovalValueStatus_APPROVED(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.APPROVED),
        issue_objects_pb2.ApprovalValue.ApprovalStatus.Value('APPROVED'))

  def test_ComputeApprovalValueStatus_NOT_APPROVED(self):
    self.assertEqual(
        self.converter._ComputeApprovalValueStatus(
            tracker_pb2.ApprovalStatus.NOT_APPROVED),
        issue_objects_pb2.ApprovalValue.ApprovalStatus.Value('NOT_APPROVED'))

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
        issue_objects_pb2.FieldValue(
            field=expected_name,
            value=self.template_1_label1_value,
            derivation=EXPLICIT_DERIVATION))
    self.assertFalse(result.blocked_on_issue_refs)
    self.assertFalse(result.blocking_issue_refs)
    self.assertFalse(result.attachment_count)
    self.assertFalse(result.star_count)
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
    self.assertEqual(len(result.phases), 0)

  def testConvertIssueTemplates(self):
    result = self.converter.ConvertIssueTemplates(
        self.project_1.project_id, [self.template_1])
    self.assertEqual(len(result), 1)
    actual = result[0]
    self.assertEqual(
        actual.name, 'projects/{}/templates/{}'.format(
            self.project_1.project_name, self.template_1.name))
    self.assertEqual(actual.display_name, self.template_1.name)
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
    self.assertListEqual(
        [av for av in actual.approval_values],
        self.converter.ConvertApprovalValues(
            self.template_1.approval_values, self.template_1.field_values,
            self.template_1.phases, project_id=self.project_1.project_id))

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
    expected = issue_objects_pb2.FieldValue(
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
    expected = issue_objects_pb2.FieldValue(
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
    expected = issue_objects_pb2.FieldValue(
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
    expected_0 = issue_objects_pb2.FieldValue(
        field=expected_0_name,
        value=expected_value_0,
        derivation=EXPLICIT_DERIVATION)
    self.assertEqual(result[0], expected_0)
    expected_1_name = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_4], self.project_1.project_id,
        self.services).get(self.field_def_4)
    expected_1 = issue_objects_pb2.FieldValue(
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

  def test_ConvertDateAction(self):
    """We can convert from protorpc to protoc FieldDef.DateAction"""
    date_type_settings = project_objects_pb2.FieldDef.DateTypeSettings

    input_type = tracker_pb2.DateAction.NO_ACTION
    actual = self.converter._ConvertDateAction(input_type)
    expected = date_type_settings.DateAction.Value('NO_ACTION')
    self.assertEqual(expected, actual)

    input_type = tracker_pb2.DateAction.PING_OWNER_ONLY
    actual = self.converter._ConvertDateAction(input_type)
    expected = date_type_settings.DateAction.Value('NOTIFY_OWNER')
    self.assertEqual(expected, actual)

    input_type = tracker_pb2.DateAction.PING_PARTICIPANTS
    actual = self.converter._ConvertDateAction(input_type)
    expected = date_type_settings.DateAction.Value('NOTIFY_PARTICIPANTS')
    self.assertEqual(expected, actual)

  def test_ConvertRoleRequirements(self):
    """We can convert from protorpc to protoc FieldDef.RoleRequirements"""
    user_type_settings = project_objects_pb2.FieldDef.UserTypeSettings

    actual = self.converter._ConvertRoleRequirements(False)
    expected = user_type_settings.RoleRequirements.Value('NO_ROLE_REQUIREMENT')
    self.assertEqual(expected, actual)

    actual = self.converter._ConvertRoleRequirements(True)
    expected = user_type_settings.RoleRequirements.Value('PROJECT_MEMBER')
    self.assertEqual(expected, actual)

  def test_ConvertNotifyTriggers(self):
    """We can convert from protorpc to protoc FieldDef.NotifyTriggers"""
    user_type_settings = project_objects_pb2.FieldDef.UserTypeSettings

    input_type = tracker_pb2.NotifyTriggers.NEVER
    actual = self.converter._ConvertNotifyTriggers(input_type)
    expected = user_type_settings.NotifyTriggers.Value('NEVER')
    self.assertEqual(expected, actual)

    input_type = tracker_pb2.NotifyTriggers.ANY_COMMENT
    actual = self.converter._ConvertNotifyTriggers(input_type)
    expected = user_type_settings.NotifyTriggers.Value('ANY_COMMENT')
    self.assertEqual(expected, actual)

  def test_ConvertFieldDefType(self):
    """We can convert from protorpc FieldType to protoc FieldDef.Type"""
    input_type = tracker_pb2.FieldTypes.ENUM_TYPE
    actual = self.converter._ConvertFieldDefType(input_type)
    expected = project_objects_pb2.FieldDef.Type.Value('ENUM')
    self.assertEqual(expected, actual)

    input_type = tracker_pb2.FieldTypes.INT_TYPE
    actual = self.converter._ConvertFieldDefType(input_type)
    expected = project_objects_pb2.FieldDef.Type.Value('INT')
    self.assertEqual(expected, actual)

    input_type = tracker_pb2.FieldTypes.STR_TYPE
    actual = self.converter._ConvertFieldDefType(input_type)
    expected = project_objects_pb2.FieldDef.Type.Value('STR')
    self.assertEqual(expected, actual)

    input_type = tracker_pb2.FieldTypes.USER_TYPE
    actual = self.converter._ConvertFieldDefType(input_type)
    expected = project_objects_pb2.FieldDef.Type.Value('USER')
    self.assertEqual(expected, actual)

    input_type = tracker_pb2.FieldTypes.DATE_TYPE
    actual = self.converter._ConvertFieldDefType(input_type)
    expected = project_objects_pb2.FieldDef.Type.Value('DATE')
    self.assertEqual(expected, actual)

    input_type = tracker_pb2.FieldTypes.URL_TYPE
    actual = self.converter._ConvertFieldDefType(input_type)
    expected = project_objects_pb2.FieldDef.Type.Value('URL')
    self.assertEqual(expected, actual)

  def test_ConvertFieldDefType_BOOL(self):
    """We raise exception for unsupported input type BOOL"""
    input_type = tracker_pb2.FieldTypes.BOOL_TYPE
    with self.assertRaises(ValueError) as cm:
      self.converter._ConvertFieldDefType(input_type)
    self.assertEqual(
        'Unsupported tracker_pb2.FieldType enum. Boolean types '
        'are unsupported and approval types are found in ApprovalDefs',
        str(cm.exception))

  def test_ConvertFieldDefType_APPROVAL(self):
    """We raise exception for input type APPROVAL"""
    input_type = tracker_pb2.FieldTypes.APPROVAL_TYPE
    with self.assertRaises(ValueError) as cm:
      self.converter._ConvertFieldDefType(input_type)
    self.assertEqual(
        'Unsupported tracker_pb2.FieldType enum. Boolean types '
        'are unsupported and approval types are found in ApprovalDefs',
        str(cm.exception))

  def testConvertFieldDefs(self):
    """We can convert field defs"""
    project_config = self.services.config.GetProjectConfig(
        self.cnxn, self.project_1.project_id)
    input_fds = project_config.field_defs
    output = self.converter.ConvertFieldDefs(
        input_fds, self.project_1.project_id)
    fd1_rn = rnc.ConvertFieldDefNames(
        self.cnxn, [self.field_def_1], self.project_1.project_id,
        self.services).get(self.field_def_1)
    self.assertEqual(fd1_rn, output[0].name)
    self.assertEqual(self.field_def_1_name, output[0].display_name)
    self.assertEqual('', output[0].docstring)
    self.assertEqual(
        project_objects_pb2.FieldDef.Type.Value('STR'), output[0].type)
    self.assertEqual(
        project_objects_pb2.FieldDef.Type.Value('INT'), output[1].type)
    self.assertEqual('', output[1].applicable_issue_type)
    fd1_admin_editor = [rnc.ConvertUserName(self.user_1.user_id)]
    self.assertEqual(fd1_admin_editor, output[0].admins)
    self.assertEqual(fd1_admin_editor, output[5].editors)

  def testConvertFieldDefs_Traits(self):
    """We can convert FieldDefs with traits"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_1)
    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(1, len(output))
    expected_traits = [
        project_objects_pb2.FieldDef.Traits.Value('REQUIRED'),
        project_objects_pb2.FieldDef.Traits.Value('MULTIVALUED'),
        project_objects_pb2.FieldDef.Traits.Value('PHASE')
    ]
    self.assertEqual(expected_traits, output[0].traits)

    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_2)
    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(1, len(output))
    expected_traits = [
        project_objects_pb2.FieldDef.Traits.Value('DEFAULT_HIDDEN')
    ]
    self.assertEqual(expected_traits, output[0].traits)

  def testConvertFieldDefs_ApprovalParent(self):
    """We can convert FieldDef with approval parents"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_6)
    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(1, len(output))

    approval_names_dict = rnc.ConvertApprovalDefNames(
        self.cnxn, [self.approval_def_1_id], self.project_1.project_id,
        self.services)
    expected_approval_parent = approval_names_dict.get(input_fd.approval_id)
    self.assertEqual(expected_approval_parent, output[0].approval_parent)

  def testConvertFieldDefs_EnumTypeSettings(self):
    """We can convert enum FieldDef and its settings"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_5)
    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(1, len(output))

    expected_settings = project_objects_pb2.FieldDef.EnumTypeSettings(
        choices=[
            Choice(
                value='submarine', docstring=self.labeldef_2.label_docstring),
            Choice(value='basket', docstring=self.labeldef_3.label_docstring)
        ])
    self.assertEqual(expected_settings, output[0].enum_settings)

  def testConvertFieldDefs_IntTypeSettings(self):
    """We can convert int FieldDef and its settings"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_2)
    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(1, len(output))

    expected_settings = project_objects_pb2.FieldDef.IntTypeSettings(
        max_value=37)
    self.assertEqual(expected_settings, output[0].int_settings)

  def testConvertFieldDefs_StrTypeSettings(self):
    """We can convert str FieldDef and its settings"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_1)
    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(1, len(output))

    expected_settings = project_objects_pb2.FieldDef.StrTypeSettings(
        regex='abc')
    self.assertEqual(expected_settings, output[0].str_settings)

  def testConvertFieldDefs_UserTypeSettings(self):
    """We can convert user FieldDef and its settings"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_8)
    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(1, len(output))

    user_settings = project_objects_pb2.FieldDef.UserTypeSettings
    expected_settings = project_objects_pb2.FieldDef.UserTypeSettings(
        role_requirements=user_settings.RoleRequirements.Value(
            'PROJECT_MEMBER'),
        needs_perm='EDIT_PROJECT',
        notify_triggers=user_settings.NotifyTriggers.Value('ANY_COMMENT'))
    self.assertEqual(expected_settings, output[0].user_settings)

  def testConvertFieldDefs_DateTypeSettings(self):
    """We can convert user FieldDef and its settings"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_9)
    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(1, len(output))

    date_settings = project_objects_pb2.FieldDef.DateTypeSettings
    expected_settings = project_objects_pb2.FieldDef.DateTypeSettings(
        date_action=date_settings.DateAction.Value('NOTIFY_OWNER'))
    self.assertEqual(expected_settings, output[0].date_settings)

  def testConvertFieldDefs_SkipsApprovals(self):
    """We skip over approval defs"""
    project_config = self.services.config.GetProjectConfig(
        self.cnxn, self.project_1.project_id)
    input_fds = project_config.field_defs
    # project_1 is set up to have 11 non-approval fields and 1 approval field
    self.assertEqual(11, len(input_fds))
    output = self.converter.ConvertFieldDefs(
        input_fds, self.project_1.project_id)
    # assert we skip approval fields
    self.assertEqual(10, len(output))

  def testConvertFieldDefs_NonexistentID(self):
    """We skip over any field defs whose ID does not exist."""
    input_fd = tracker_pb2.FieldDef(
        field_id=self.dne_field_def_id,
        project_id=self.project_1.project_id,
        field_name='foobar',
        field_type=tracker_pb2.FieldTypes('STR_TYPE'))

    output = self.converter.ConvertFieldDefs(
        [input_fd], self.project_1.project_id)
    self.assertEqual(0, len(output))

  def testConvertFieldDefs_Empty(self):
    """We can handle empty list input"""
    self.assertEqual(
        [], self.converter.ConvertFieldDefs([], self.project_1.project_id))

  def test_ComputeFieldDefTraits(self):
    """We can get Sequence of Traits for a FieldDef"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_1)
    actual = self.converter._ComputeFieldDefTraits(input_fd)
    expected = [
        project_objects_pb2.FieldDef.Traits.Value('REQUIRED'),
        project_objects_pb2.FieldDef.Traits.Value('MULTIVALUED'),
        project_objects_pb2.FieldDef.Traits.Value('PHASE')
    ]
    self.assertEqual(expected, actual)

    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_2)
    actual = self.converter._ComputeFieldDefTraits(input_fd)
    expected = [project_objects_pb2.FieldDef.Traits.Value('DEFAULT_HIDDEN')]
    self.assertEqual(expected, actual)

    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_7)
    actual = self.converter._ComputeFieldDefTraits(input_fd)
    expected = [project_objects_pb2.FieldDef.Traits.Value('RESTRICTED')]
    self.assertEqual(expected, actual)

  def test_ComputeFieldDefTraits_Empty(self):
    """We return an empty Sequence of Traits for plain FieldDef"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_3)
    actual = self.converter._ComputeFieldDefTraits(input_fd)
    self.assertEqual([], actual)

  def test_GetEnumFieldChoices(self):
    """We can get all choices for an enum field"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_5)
    actual = self.converter._GetEnumFieldChoices(input_fd)
    expected = [
        Choice(
            value=self.labeldef_2.label.split('-')[1],
            docstring=self.labeldef_2.label_docstring),
        Choice(
            value=self.labeldef_3.label.split('-')[1],
            docstring=self.labeldef_3.label_docstring),
    ]
    self.assertEqual(expected, actual)

  def test_GetEnumFieldChoices_NotEnumField(self):
    """We raise exception for non-enum-field"""
    input_fd = self._GetFieldDefById(
        self.project_1.project_id, self.field_def_1)
    with self.assertRaises(ValueError) as cm:
      self.converter._GetEnumFieldChoices(input_fd)
    self.assertEqual(
        'Cannot get value from label for non-enum-type field', str(
            cm.exception))

  def testConvertApprovalDefs(self):
    """We can convert ApprovalDefs"""
    input_ad = self._GetApprovalDefById(
        self.project_1.project_id, self.approval_def_1_id)
    actual = self.converter.ConvertApprovalDefs(
        [input_ad], self.project_1.project_id)

    resource_names_dict = rnc.ConvertApprovalDefNames(
        self.cnxn, [self.approval_def_1_id], self.project_1.project_id,
        self.services)
    expected_name = resource_names_dict.get(self.approval_def_1_id)
    self.assertEqual(actual[0].name, expected_name)
    self.assertEqual(actual[0].display_name, self.approval_def_1_name)
    matching_fd = self._GetFieldDefById(
        self.project_1.project_id, self.approval_def_1_id)
    expected_docstring = matching_fd.docstring
    self.assertEqual(actual[0].docstring, expected_docstring)
    self.assertEqual(actual[0].survey, self.approval_def_1.survey)
    expected_approvers = [rnc.ConvertUserName(self.user_2.user_id)]
    self.assertEqual(actual[0].approvers, expected_approvers)
    expected_admins = [rnc.ConvertUserName(self.user_1.user_id)]
    self.assertEqual(actual[0].admins, expected_admins)

  def testConvertApprovalDefs_Empty(self):
    """We can handle empty case"""
    actual = self.converter.ConvertApprovalDefs([], self.project_1.project_id)
    self.assertEqual(actual, [])

  def testConvertApprovalDefs_SkipsNonApprovalDefs(self):
    """We skip if no matching field def exists"""
    input_ad = tracker_pb2.ApprovalDef(
        approval_id=self.dne_field_def_id,
        approver_ids=[self.user_2.user_id],
        survey='anything goes')
    actual = self.converter.ConvertApprovalDefs(
        [input_ad], self.project_1.project_id)
    self.assertEqual(actual, [])

  def testConvertLabelDefs(self):
    """We can convert LabelDefs"""
    actual = self.converter.ConvertLabelDefs(
        [self.labeldef_1, self.labeldef_5], self.project_1.project_id)
    resource_names_dict = rnc.ConvertLabelDefNames(
        self.cnxn, [self.labeldef_1.label, self.labeldef_5.label],
        self.project_1.project_id, self.services)
    expected_0_name = resource_names_dict.get(self.labeldef_1.label)
    expected_0 = project_objects_pb2.LabelDef(
        name=expected_0_name,
        value=self.labeldef_1.label,
        docstring=self.labeldef_1.label_docstring,
        state=project_objects_pb2.LabelDef.LabelDefState.Value('ACTIVE'))
    self.assertEqual(expected_0, actual[0])
    expected_1_name = resource_names_dict.get(self.labeldef_5.label)
    expected_1 = project_objects_pb2.LabelDef(
        name=expected_1_name,
        value=self.labeldef_5.label,
        docstring=self.labeldef_5.label_docstring,
        state=project_objects_pb2.LabelDef.LabelDefState.Value('DEPRECATED'))
    self.assertEqual(expected_1, actual[1])

  def testConvertLabelDefs_Empty(self):
    """We can handle empty input case"""
    actual = self.converter.ConvertLabelDefs([], self.project_1.project_id)
    self.assertEqual([], actual)

  def testConvertStatusDefs(self):
    """We can convert StatusDefs"""
    actual = self.converter.ConvertStatusDefs(
        self.predefined_statuses, self.project_1.project_id)
    self.assertEqual(len(actual), 4)

    input_names = [sd.status for sd in self.predefined_statuses]
    names = rnc.ConvertStatusDefNames(
        self.cnxn, input_names, self.project_1.project_id, self.services)
    self.assertEqual(names[self.status_1.status], actual[0].name)
    self.assertEqual(names[self.status_2.status], actual[1].name)
    self.assertEqual(names[self.status_3.status], actual[2].name)
    self.assertEqual(names[self.status_4.status], actual[3].name)

    self.assertEqual(self.status_1.status, actual[0].value)
    self.assertEqual(
        project_objects_pb2.StatusDef.StatusDefType.Value('OPEN'),
        actual[0].type)
    self.assertEqual(0, actual[0].rank)
    self.assertEqual(self.status_1.status_docstring, actual[0].docstring)
    self.assertEqual(
        project_objects_pb2.StatusDef.StatusDefState.Value('ACTIVE'),
        actual[0].state)

  def testConvertStatusDefs_Empty(self):
    """Can handle empty input case"""
    actual = self.converter.ConvertStatusDefs([], self.project_1.project_id)
    self.assertEqual([], actual)

  def testConvertStatusDefs_Rank(self):
    """Rank is indepdendent of input order"""
    input_sds = [self.status_2, self.status_4, self.status_3, self.status_1]
    actual = self.converter.ConvertStatusDefs(
        input_sds, self.project_1.project_id)
    self.assertEqual(1, actual[0].rank)
    self.assertEqual(3, actual[1].rank)

  def testConvertStatusDefs_type_MERGED(self):
    """Includes mergeable status when parsed from project config"""
    actual = self.converter.ConvertStatusDefs(
        [self.status_2], self.project_1.project_id)
    self.assertEqual(
        project_objects_pb2.StatusDef.StatusDefType.Value('MERGED'),
        actual[0].type)

  def testConvertStatusDefs_state_DEPRECATED(self):
    """Includes deprecated status"""
    actual = self.converter.ConvertStatusDefs(
        [self.status_4], self.project_1.project_id)
    self.assertEqual(
        project_objects_pb2.StatusDef.StatusDefState.Value('DEPRECATED'),
        actual[0].state)

  def testConvertComponentDefs(self):
    """We can convert ComponentDefs"""
    project_config = self.services.config.GetProjectConfig(
        self.cnxn, self.project_1.project_id)
    self.assertEqual(len(project_config.component_defs), 2)

    actual = self.converter.ConvertComponentDefs(
        project_config.component_defs, self.project_1.project_id)
    self.assertEqual(2, len(actual))

    resource_names_dict = rnc.ConvertComponentDefNames(
        self.cnxn, [self.component_def_1_id, self.component_def_2_id],
        self.project_1.project_id, self.services)
    self.assertEqual(
        resource_names_dict.get(self.component_def_1_id), actual[0].name)
    self.assertEqual(
        resource_names_dict.get(self.component_def_2_id), actual[1].name)
    self.assertEqual(self.component_def_1_path, actual[0].value)
    self.assertEqual(self.component_def_2_path, actual[1].value)
    self.assertEqual('cd1_docstring', actual[0].docstring)
    self.assertEqual(
        project_objects_pb2.ComponentDef.ComponentDefState.Value('ACTIVE'),
        actual[0].state)
    self.assertEqual(
        project_objects_pb2.ComponentDef.ComponentDefState.Value('DEPRECATED'),
        actual[1].state)
    # component_def 1 and 2 have the same admins, ccs, creator, and crate_time
    expected_admins = [rnc.ConvertUserName(self.user_1.user_id)]
    self.assertEqual(expected_admins, actual[0].admins)
    expected_ccs = [rnc.ConvertUserName(self.user_2.user_id)]
    self.assertEqual(expected_ccs, actual[0].ccs)
    expected_creator = rnc.ConvertUserName(self.user_1.user_id)
    self.assertEqual(expected_creator, actual[0].creator)
    expected_create_time = timestamp_pb2.Timestamp(seconds=self.PAST_TIME)
    self.assertEqual(expected_create_time, actual[0].create_time)

    expected_labels = [ld.label for ld in self.predefined_labels]
    self.assertEqual(expected_labels, actual[0].labels)
    self.assertEqual([], actual[1].labels)

  def testConvertComponentDefs_Empty(self):
    """Can handle empty input case"""
    actual = self.converter.ConvertComponentDefs([], self.project_1.project_id)
    self.assertEqual([], actual)

  def testConvertProjectSavedQueries(self):
    """We can convert ProjectSavedQueries"""
    input_psqs = [self.psq_2]
    actual = self.converter.ConvertProjectSavedQueries(
        input_psqs, self.project_1.project_id)
    self.assertEqual(1, len(actual))

    resource_names_dict = rnc.ConvertProjectSavedQueryNames(
        self.cnxn, [self.psq_2.query_id], self.project_1.project_id,
        self.services)
    self.assertEqual(
        resource_names_dict.get(self.psq_2.query_id), actual[0].name)
    self.assertEqual(self.psq_2.name, actual[0].display_name)
    self.assertEqual(self.psq_2.query, actual[0].query)

  def testConvertProjectSavedQueries_ExpandsBasedOn(self):
    """We expand query to include base_query_id"""
    actual = self.converter.ConvertProjectSavedQueries(
        [self.psq_1], self.project_1.project_id)
    expected_query = '{} {}'.format(
        tbo.GetBuiltInQuery(self.psq_1.base_query_id), self.psq_1.query)
    self.assertEqual(expected_query, actual[0].query)

  def testConvertProjectSavedQueries_NotInProject(self):
    """We skip over saved queries that don't belong to this project"""
    psq_not_registered = tracker_pb2.SavedQuery(
        query_id=4, name='psq no registered name', query='no registered')
    actual = self.converter.ConvertProjectSavedQueries(
        [psq_not_registered], self.project_1.project_id)
    self.assertEqual([], actual)

  def testConvertProjectSavedQueries_Empty(self):
    """We can handle empty inputs"""
    actual = self.converter.ConvertProjectSavedQueries(
        [], self.project_1.project_id)
    self.assertEqual([], actual)
