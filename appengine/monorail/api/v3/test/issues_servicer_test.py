# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the issues servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest
import mock

from api.v3 import converters
from api.v3 import issues_servicer
from api.v3.api_proto import issues_pb2
from api.v3.api_proto import issue_objects_pb2
from framework import exceptions
from framework import monorailcontext
from proto import tracker_pb2
from testing import fake
from services import service_manager

from google.appengine.ext import testbed
from google.protobuf import timestamp_pb2


class IssuesServicerTest(unittest.TestCase):

  def setUp(self):
    # memcache and datastore needed for generating page tokens.
    self.testbed = testbed.Testbed()
    self.testbed.activate()
    self.testbed.init_memcache_stub()
    self.testbed.init_datastore_v3_stub()

    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        config=fake.ConfigService(),
        issue=fake.IssueService(),
        issue_star=fake.IssueStarService(),
        project=fake.ProjectService(),
        features=fake.FeaturesService(),
        spam=fake.SpamService(),
        user=fake.UserService(),
        usergroup=fake.UserGroupService())
    self.issues_svcr = issues_servicer.IssuesServicer(
        self.services, make_rate_limiter=False)
    self.PAST_TIME = 12345

    self.owner = self.services.user.TestAddUser('owner@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('user_2@example.com', 222)

    self.project_1 = self.services.project.TestAddProject(
        'chicken', project_id=789)
    self.issue_1_resource_name = 'projects/chicken/issues/1234'
    self.issue_1 = fake.MakeTestIssue(
        self.project_1.project_id,
        1234,
        'sum',
        'New',
        self.owner.user_id,
        labels=['find-me', 'pri-3'],
        project_name=self.project_1.project_name)
    self.services.issue.TestAddIssue(self.issue_1)

    self.project_2 = self.services.project.TestAddProject('cow', project_id=788)
    self.issue_2_resource_name = 'projects/cow/issues/1234'
    self.issue_2 = fake.MakeTestIssue(
        self.project_2.project_id,
        1235,
        'sum',
        'New',
        self.user_2.user_id,
        project_name=self.project_2.project_name)
    self.services.issue.TestAddIssue(self.issue_2)
    self.issue_3 = fake.MakeTestIssue(
        self.project_2.project_id,
        1236,
        'sum',
        'New',
        self.user_2.user_id,
        labels=['find-me', 'pri-1'],
        project_name=self.project_2.project_name)
    self.services.issue.TestAddIssue(self.issue_3)

  def CallWrapped(self, wrapped_handler, mc, *args, **kwargs):
    self.issues_svcr.converter = converters.Converter(mc, self.services)
    return wrapped_handler.wrapped(self.issues_svcr, mc, *args, **kwargs)

  def testGetIssue(self):
    """We can get an issue."""
    request = issues_pb2.GetIssueRequest(name=self.issue_1_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    actual_response = self.CallWrapped(self.issues_svcr.GetIssue, mc, request)
    self.assertEqual(
        actual_response, self.issues_svcr.converter.ConvertIssue(self.issue_1))

  def testBatchGetIssues(self):
    """We can batch get issues."""
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    request = issues_pb2.BatchGetIssuesRequest(
        names=['projects/cow/issues/1235', 'projects/cow/issues/1236'])
    actual_response = self.CallWrapped(
        self.issues_svcr.BatchGetIssues, mc, request)
    self.assertEqual(
        [issue.name for issue in actual_response.issues],
        ['projects/cow/issues/1235', 'projects/cow/issues/1236'])

  def testBatchGetIssues_Empty(self):
    """We can return a response if the request has no names."""
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    request = issues_pb2.BatchGetIssuesRequest(names=[])
    actual_response = self.CallWrapped(
        self.issues_svcr.BatchGetIssues, mc, request)
    self.assertEqual(
        actual_response, issues_pb2.BatchGetIssuesResponse(issues=[]))

  def testBatchGetIssues_WithParent(self):
    """We can batch get issues with a given parent."""
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    request = issues_pb2.BatchGetIssuesRequest(
        parent='projects/cow',
        names=['projects/cow/issues/1235', 'projects/cow/issues/1236'])
    actual_response = self.CallWrapped(
        self.issues_svcr.BatchGetIssues, mc, request)
    self.assertEqual(
        [issue.name for issue in actual_response.issues],
        ['projects/cow/issues/1235', 'projects/cow/issues/1236'])

  def testBatchGetIssues_FromMultipleProjects(self):
    """We can batch get issues from multiple projects."""
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    request = issues_pb2.BatchGetIssuesRequest(
        names=[
            'projects/chicken/issues/1234', 'projects/cow/issues/1235',
            'projects/cow/issues/1236'
        ])
    actual_response = self.CallWrapped(
        self.issues_svcr.BatchGetIssues, mc, request)
    self.assertEqual(
        [issue.name for issue in actual_response.issues], [
            'projects/chicken/issues/1234', 'projects/cow/issues/1235',
            'projects/cow/issues/1236'
        ])

  def testBatchGetIssues_WithBadInput(self):
    """We raise an exception with bad input to batch get issues."""
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    request = issues_pb2.BatchGetIssuesRequest(
        parent='projects/cow',
        names=['projects/cow/issues/1235', 'projects/chicken/issues/1234'])
    with self.assertRaisesRegexp(
        exceptions.InputException,
        'projects/chicken/issues/1234 is not a child issue of projects/cow.'):
      self.CallWrapped(self.issues_svcr.BatchGetIssues, mc, request)

    request = issues_pb2.BatchGetIssuesRequest(
        parent='projects/sheep',
        names=['projects/cow/issues/1235', 'projects/chicken/issues/1234'])
    with self.assertRaisesRegexp(
        exceptions.InputException,
        'projects/cow/issues/1235 is not a child issue of projects/sheep.\n' +
        'projects/chicken/issues/1234 is not a child issue of projects/sheep.'):
      self.CallWrapped(self.issues_svcr.BatchGetIssues, mc, request)

    request = issues_pb2.BatchGetIssuesRequest(
        parent='projects/cow',
        names=['projects/cow/badformat/1235', 'projects/chicken/issues/1234'])
    with self.assertRaisesRegexp(
        exceptions.InputException,
        'Invalid resource name: projects/cow/badformat/1235.'):
      self.CallWrapped(self.issues_svcr.BatchGetIssues, mc, request)

  @mock.patch('search.frontendsearchpipeline.FrontendSearchPipeline')
  @mock.patch('tracker.tracker_constants.MAX_ISSUES_PER_PAGE', 2)
  def testSearchIssues(self, mock_pipeline):
    """We can search for issues in some projects."""
    request = issues_pb2.SearchIssuesRequest(
        projects=['projects/chicken', 'projects/cow'],
        query='label:find-me',
        order_by='-pri',
        page_size=3)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_2.email)

    instance = mock.Mock(
        spec=True,
        visible_results=[self.issue_1, self.issue_3],
        allowed_results=[self.issue_1, self.issue_3, self.issue_2])
    mock_pipeline.return_value = instance
    instance.SearchForIIDs = mock.Mock()
    instance.MergeAndSortIssues = mock.Mock()
    instance.Paginate = mock.Mock()

    actual_response = self.CallWrapped(
        self.issues_svcr.SearchIssues, mc, request)
    # start index is 0.
    # number of items is coerced from 3 -> 2
    mock_pipeline.assert_called_once_with(
        self.cnxn,
        self.services,
        mc.auth, [222],
        'label:find-me', ['chicken', 'cow'],
        2,
        0,
        1,
        '',
        '-pri',
        mc.warnings,
        mc.errors,
        True,
        mc.profiler,
        project=None)
    self.assertEqual(
        [issue.name for issue in actual_response.issues],
        ['projects/chicken/issues/1234', 'projects/cow/issues/1236'])

    # Check the `next_page_token` can be used to get the next page of results.
    request.page_token = actual_response.next_page_token
    self.CallWrapped(self.issues_svcr.SearchIssues, mc, request)
    # start index is now 2.
    mock_pipeline.assert_called_with(
        self.cnxn,
        self.services,
        mc.auth, [222],
        'label:find-me', ['chicken', 'cow'],
        2,
        2,
        1,
        '',
        '-pri',
        mc.warnings,
        mc.errors,
        True,
        mc.profiler,
        project=None)

  # Note the 'empty' case doesn't make sense for ListComments, as one is created
  # for every issue.
  def testListComments(self):
    """We can list comments."""
    request = issues_pb2.ListCommentsRequest(parent=self.issue_1_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    actual_response = self.CallWrapped(
        self.issues_svcr.ListComments, mc, request)
    self.assertEqual(1, len(actual_response.comments))

  def testListApprovalValues(self):
    config = fake.MakeTestConfig(self.project_2.project_id, [], [])
    self.services.config.StoreConfig(self.cnxn, config)

    # Make regular field def and value
    fd_1 = fake.MakeTestFieldDef(
        1, self.project_2.project_id, tracker_pb2.FieldTypes.STR_TYPE,
        field_name='field1')
    self.services.config.TestAddFieldDef(fd_1)
    fv_1 = fake.MakeFieldValue(
        field_id=fd_1.field_id, str_value='value1', derived=False)

    # Make testing approval def and its associated field def
    approval_gate = fake.MakeTestFieldDef(
        2, self.project_2.project_id, tracker_pb2.FieldTypes.APPROVAL_TYPE,
        field_name='approval-gate-1')
    self.services.config.TestAddFieldDef(approval_gate)
    ad = fake.MakeTestApprovalDef(2, approver_ids=[self.user_2.user_id])
    self.services.config.TestAddApprovalDef(ad, self.project_2.project_id)

    # Make approval value
    av = fake.MakeApprovalValue(2, set_on=self.PAST_TIME,
          approver_ids=[self.user_2.user_id], setter_id=self.user_2.user_id)

    # Make field def that belongs to above approval_def
    fd_2 = fake.MakeTestFieldDef(
        3, self.project_2.project_id, tracker_pb2.FieldTypes.STR_TYPE,
        field_name='field2', approval_id=2)
    self.services.config.TestAddFieldDef(fd_2)
    fv_2 = fake.MakeFieldValue(
        field_id=fd_2.field_id, str_value='value2', derived=False)

    issue_resource_name = 'projects/cow/issues/1237'
    issue = fake.MakeTestIssue(
        self.project_2.project_id,
        1237,
        'sum',
        'New',
        self.user_2.user_id,
        project_name=self.project_2.project_name,
        field_values=[fv_1, fv_2],
        approval_values=[av])
    self.services.issue.TestAddIssue(issue)

    request = issues_pb2.ListApprovalValuesRequest(parent=issue_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    actual_response = self.CallWrapped(
        self.issues_svcr.ListApprovalValues, mc, request)

    self.assertEqual(len(actual_response.approval_values), 1)
    expected_fv = issue_objects_pb2.FieldValue(
        field='projects/cow/fieldDefs/3',
        value='value2',
        derivation=issue_objects_pb2.Derivation.Value('EXPLICIT'))
    expected = issue_objects_pb2.ApprovalValue(
        name='projects/cow/issues/1237/approvalValues/2',
        status=issue_objects_pb2.ApprovalValue.ApprovalStatus.Value('NOT_SET'),
        approvers=['users/222'],
        approval_def='projects/cow/approvalDefs/2',
        set_time=timestamp_pb2.Timestamp(seconds=self.PAST_TIME),
        setter='users/222',
        field_values=[expected_fv])
    self.assertEqual(actual_response.approval_values[0], expected)

  def testListApprovalValues_Empty(self):
    request = issues_pb2.ListApprovalValuesRequest(
        parent=self.issue_1_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    actual_response = self.CallWrapped(
        self.issues_svcr.ListApprovalValues, mc, request)
    self.assertEqual(len(actual_response.approval_values), 0)

  @mock.patch(
      'features.send_notifications.PrepareAndSendIssueChangeNotification')
  def testMakeIssue(self, _fake_pasicn):
    request_issue = issue_objects_pb2.Issue(
        summary='sum',
        status=issue_objects_pb2.Issue.StatusValue(status='New'),
        cc_users=[issue_objects_pb2.Issue.UserValue(user='users/222')],
        labels=[issue_objects_pb2.Issue.LabelValue(label='foo-bar')]
    )
    request = issues_pb2.MakeIssueRequest(
        parent='projects/chicken',
        issue=request_issue,
        description='description'
    )
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    response = self.CallWrapped(
        self.issues_svcr.MakeIssue, mc, request)
    self.assertEqual(response.summary, 'sum')
    self.assertEqual(response.status.status, 'New')
    self.assertEqual(response.cc_users[0].user, 'users/222')
    self.assertEqual(response.labels[0].label, 'foo-bar')
    self.assertEqual(response.star_count, 1)
