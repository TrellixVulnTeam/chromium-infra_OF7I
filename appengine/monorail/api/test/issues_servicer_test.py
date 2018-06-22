# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is govered by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the issues servicer."""

import logging
import unittest
from mock import Mock, patch

from google.protobuf import empty_pb2

from components.prpc import codes
from components.prpc import context
from components.prpc import server

from api import issues_servicer
from api.api_proto import common_pb2
from api.api_proto import issues_pb2
from api.api_proto import issue_objects_pb2
from api.api_proto import common_pb2
from businesslogic import work_env
from features import send_notifications
from framework import authdata
from framework import monorailcontext
from framework import permissions
from proto import tracker_pb2
from testing import fake
from services import service_manager
from proto import tracker_pb2


class IssuesServicerTest(unittest.TestCase):

  NOW = 1234567890

  def setUp(self):
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        config=fake.ConfigService(),
        issue=fake.IssueService(),
        user=fake.UserService(),
        usergroup=fake.UserGroupService(),
        project=fake.ProjectService(),
        features=fake.FeaturesService())
    self.project = self.services.project.TestAddProject(
        'proj', project_id=789, owner_ids=[111L], contrib_ids=[222L, 333L])
    self.user_1 = self.services.user.TestAddUser('owner@example.com', 111L)
    self.user_2 = self.services.user.TestAddUser('approver2@example.com', 222L)
    self.user_3 = self.services.user.TestAddUser('approver3@example.com', 333L)
    self.issue_1 = fake.MakeTestIssue(
        789, 1, 'sum', 'New', 111L, project_name='proj',
        opened_timestamp=self.NOW)
    self.issue_2 = fake.MakeTestIssue(
        789, 2, 'sum', 'New', 111L, project_name='proj')
    self.issue_1.blocked_on_iids.append(self.issue_2.issue_id)
    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)
    self.issues_svcr = issues_servicer.IssuesServicer(
        self.services, make_rate_limiter=False)
    self.prpc_context = context.ServicerContext()
    self.prpc_context.set_code(server.StatusCode.OK)
    self.auth = authdata.AuthData(user_id=333L, email='approver3@example.com')

    self.fd_1 = tracker_pb2.FieldDef(
        field_name='FirstField', field_id=1,
        field_type=tracker_pb2.FieldTypes.STR_TYPE,
        applicable_type='')
    self.fd_2 = tracker_pb2.FieldDef(
        field_name='SecField', field_id=2,
        field_type=tracker_pb2.FieldTypes.INT_TYPE,
        applicable_type='')
    self.fd_3 = tracker_pb2.FieldDef(
        field_name='LegalApproval', field_id=3,
        field_type=tracker_pb2.FieldTypes.APPROVAL_TYPE,
        applicable_type='')
    self.fd_4 = tracker_pb2.FieldDef(
        field_name='UserField', field_id=4,
        field_type=tracker_pb2.FieldTypes.USER_TYPE,
        applicable_type='')

  def CallWrapped(self, wrapped_handler, *args, **kwargs):
    return wrapped_handler.wrapped(self.issues_svcr, *args, **kwargs)

  def testCreateIssue_Normal(self):
    """We can create an issue."""
    request = issues_pb2.CreateIssueRequest(
        project_name='proj',
        issue=issue_objects_pb2.Issue(
            project_name='proj', local_id=1, summary='sum'))
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')

    response = self.CallWrapped(self.issues_svcr.CreateIssue, mc, request)

    self.assertEqual('proj', response.project_name)

  def testGetIssue_Normal(self):
    """We can get an issue."""
    request = issues_pb2.GetIssueRequest()
    request.issue_ref.project_name = 'proj'
    request.issue_ref.local_id = 1
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    mc.LookupLoggedInUserPerms(self.project)

    response = self.CallWrapped(self.issues_svcr.GetIssue, mc, request)

    actual = response.issue
    self.assertEqual('proj', actual.project_name)
    self.assertEqual(1, actual.local_id)
    self.assertEqual(1, len(actual.blocked_on_issue_refs))
    self.assertEqual('proj', actual.blocked_on_issue_refs[0].project_name)
    self.assertEqual(2, actual.blocked_on_issue_refs[0].local_id)

  def testUpdateIssue_Denied(self):
    """We reject requests to update an issue when the user lacks perms."""
    request = issues_pb2.UpdateIssueRequest()
    request.issue_ref.project_name = 'proj'
    request.issue_ref.local_id = 1

    # Anon user can never update.
    mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)
    mc.LookupLoggedInUserPerms(self.project)
    with self.assertRaises(permissions.PermissionException):
      self.CallWrapped(self.issues_svcr.UpdateIssue, mc, request)

    # Signed in user cannot view this issue.
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='approver3@example.com')
    mc.LookupLoggedInUserPerms(self.project)
    self.issue_1.labels = ['Restrict-View-CoreTeam']
    with self.assertRaises(permissions.PermissionException):
      self.CallWrapped(self.issues_svcr.UpdateIssue, mc, request)

    # Signed in user cannot edit this issue.
    self.issue_1.labels = ['Restrict-EditIssue-CoreTeam']
    with self.assertRaises(permissions.PermissionException):
      self.CallWrapped(self.issues_svcr.UpdateIssue, mc, request)

  @patch('features.send_notifications.PrepareAndSendIssueChangeNotification')
  def testUpdateIssue_Normal(self, fake_pasicn):
    """We can update an issue."""
    request = issues_pb2.UpdateIssueRequest()
    request.issue_ref.project_name = 'proj'
    request.issue_ref.local_id = 1
    request.delta.summary.value = 'New summary'
    request.delta.label_refs_add.extend([
        common_pb2.LabelRef(label='Hot')])
    request.comment_content = 'test comment'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    mc.LookupLoggedInUserPerms(self.project)

    response = self.CallWrapped(self.issues_svcr.UpdateIssue, mc, request)

    actual = response.issue
    # Intended stuff was changed.
    self.assertEqual(1, len(actual.label_refs))
    self.assertEqual('Hot', actual.label_refs[0].label)
    self.assertEqual('New summary', actual.summary)

    # Other stuff didn't change.
    self.assertEqual('proj', actual.project_name)
    self.assertEqual(1, actual.local_id)
    self.assertEqual(1, len(actual.blocked_on_issue_refs))
    self.assertEqual('proj', actual.blocked_on_issue_refs[0].project_name)
    self.assertEqual(2, actual.blocked_on_issue_refs[0].local_id)

    # A comment was added.
    fake_pasicn.assert_called_once()
    comments = self.services.issue.GetCommentsForIssue(
        self.cnxn, self.issue_1.issue_id)
    self.assertEqual(2, len(comments))
    self.assertEqual('test comment', comments[1].content)

  @patch('features.send_notifications.PrepareAndSendIssueChangeNotification')
  def testUpdateIssue_CommentOnly(self, fake_pasicn):
    """We can update an issue with a comment w/o making any other changes."""
    request = issues_pb2.UpdateIssueRequest()
    request.issue_ref.project_name = 'proj'
    request.issue_ref.local_id = 1
    request.comment_content = 'test comment'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    mc.LookupLoggedInUserPerms(self.project)

    self.CallWrapped(self.issues_svcr.UpdateIssue, mc, request)

    # A comment was added.
    fake_pasicn.assert_called_once()
    comments = self.services.issue.GetCommentsForIssue(
        self.cnxn, self.issue_1.issue_id)
    self.assertEqual(2, len(comments))
    self.assertEqual('test comment', comments[1].content)

  @patch('features.send_notifications.PrepareAndSendIssueChangeNotification')
  def testUpdateIssue_NoOp(self, fake_pasicn):
    """We gracefully ignore requests that have no delta or comment."""
    request = issues_pb2.UpdateIssueRequest()
    request.issue_ref.project_name = 'proj'
    request.issue_ref.local_id = 1
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    mc.LookupLoggedInUserPerms(self.project)

    response = self.CallWrapped(self.issues_svcr.UpdateIssue, mc, request)

    actual = response.issue
    # Other stuff didn't change.
    self.assertEqual('proj', actual.project_name)
    self.assertEqual(1, actual.local_id)
    self.assertEqual('sum', actual.summary)
    self.assertEqual('New', actual.status_ref.status)

    # No comment was added.
    fake_pasicn.assert_not_called()
    comments = self.services.issue.GetCommentsForIssue(
        self.cnxn, self.issue_1.issue_id)
    self.assertEqual(1, len(comments))

  def testListComments_Normal(self):
    """We can get comments on an issue."""
    comment = tracker_pb2.IssueComment(
        user_id=111L, timestamp=self.NOW, content='second',
        project_id=789, issue_id=self.issue_1.issue_id, sequence=1)
    self.services.issue.TestAddComment(comment, self.issue_1.local_id)
    request = issues_pb2.ListCommentsRequest()
    request.issue_ref.project_name = 'proj'
    request.issue_ref.local_id = 1
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    mc.LookupLoggedInUserPerms(self.project)

    response = self.CallWrapped(self.issues_svcr.ListComments, mc, request)

    actual_0 = response.comments[0]
    actual_1 = response.comments[1]
    expected_0 = issue_objects_pb2.Comment(
        project_name='proj', local_id=1, sequence_num=0, is_deleted=False,
        commenter=common_pb2.UserRef(
            user_id=111L, display_name='owner@example.com'),
        timestamp=self.NOW, content='sum', is_spam=False,
        description_num=1)
    expected_1 = issue_objects_pb2.Comment(
        project_name='proj', local_id=1, sequence_num=1, is_deleted=False,
        commenter=common_pb2.UserRef(
            user_id=111L, display_name='owner@example.com'),
        timestamp=self.NOW, content='second')
    self.assertEqual(expected_0, actual_0)
    self.assertEqual(expected_1, actual_1)

  def testDeleteIssueComment_Normal(self):
    """We can delete a comment."""
    request = issues_pb2.DeleteIssueCommentRequest(
        project_name='proj', local_id=1, comment_id=11, delete=True)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')

    response = self.CallWrapped(
        self.issues_svcr.DeleteIssueComment, mc, request)

    self.assertTrue(isinstance(response, empty_pb2.Empty))

  @patch('businesslogic.work_env.WorkEnv.UpdateIssueApproval')
  @patch('features.send_notifications.PrepareAndSendApprovalChangeNotification')
  def testUpdateApproval(self, _mockPrepareAndSend, mockUpdateIssueApproval):
    """We can update an approval."""

    av_3 = tracker_pb2.ApprovalValue(
            approval_id=3,
            status=tracker_pb2.ApprovalStatus.NEEDS_REVIEW,
            approver_ids=[333L]
    )
    self.issue_1.approval_values = [av_3]

    config = self.services.config.GetProjectConfig(
        self.cnxn, 789)
    config.field_defs = [self.fd_1, self.fd_3]

    self.services.config.StoreConfig(self.cnxn, config)

    issue_ref = common_pb2.IssueRef(project_name='proj', local_id=1)
    field_ref = common_pb2.FieldRef(field_name='LegalApproval')
    approval_delta = issues_pb2.ApprovalDelta(
        status=issue_objects_pb2.REVIEW_REQUESTED,
        approver_refs_add=[
          common_pb2.UserRef(user_id=222L, display_name='approver2@example.com')
          ],
        field_vals_add=[
          issue_objects_pb2.FieldValue(
              field_ref=common_pb2.FieldRef(field_name='FirstField'),
              value='string')
          ]
    )

    request = issues_pb2.UpdateApprovalRequest(
        issue_ref=issue_ref, field_ref=field_ref, approval_delta=approval_delta,
        comment_content='Well, actually'
    )
    request.issue_ref.project_name = 'proj'
    request.issue_ref.local_id = 1
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='approver3@example.com',
        auth=self.auth)

    mockUpdateIssueApproval.return_value = [
        tracker_pb2.ApprovalValue(
            approval_id=3,
            status=tracker_pb2.ApprovalStatus.REVIEW_REQUESTED,
            setter_id=333L,
            approver_ids=[333L, 222L]),
        'comment_pb']

    actual = self.CallWrapped(self.issues_svcr.UpdateApproval, mc, request)

    expected = issues_pb2.UpdateApprovalResponse()
    expected.approval.CopyFrom(
      issue_objects_pb2.Approval(
          field_ref=common_pb2.FieldRef(
              field_name='LegalApproval', type=common_pb2.APPROVAL_TYPE),
          approver_refs=[
              common_pb2.UserRef(
                  user_id=333, display_name='approver3@example.com'),
              common_pb2.UserRef(
                  user_id=222, display_name='approver2@example.com')
              ],
          status=issue_objects_pb2.REVIEW_REQUESTED,
          setter_ref=common_pb2.UserRef(
                  user_id=333, display_name='approver3@example.com'),
          phase_ref=issue_objects_pb2.PhaseRef()
      )
      )

    work_env.WorkEnv(mc, self.services).UpdateIssueApproval.assert_called_once()
    self.assertEqual(actual, expected)
