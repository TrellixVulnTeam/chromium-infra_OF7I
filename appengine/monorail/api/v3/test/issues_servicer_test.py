# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the issues servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from api.v3 import converters
from api.v3 import issues_servicer
from api.v3.api_proto import issues_pb2
from api.v3.api_proto import issue_objects_pb2
from framework import monorailcontext
from testing import fake
from services import service_manager


class IssuesServicerTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        config=fake.ConfigService(),
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        spam=fake.SpamService(),
        user=fake.UserService(),
        usergroup=fake.UserGroupService())
    self.issues_svcr = issues_servicer.IssuesServicer(
        self.services, make_rate_limiter=False)
    self.PAST_TIME = 12345
    self.owner = self.services.user.TestAddUser('owner@example.com', 111)
    self.project_1 = self.services.project.TestAddProject(
        'chicken', project_id=789)
    self.issue_resource_name = 'projects/chicken/issues/1234'
    self.issue = fake.MakeTestIssue(
        self.project_1.project_id,
        1234,
        'sum',
        'New',
        self.owner.user_id,
        project_name=self.project_1.project_name)
    self.services.issue.TestAddIssue(self.issue)

  def CallWrapped(self, wrapped_handler, mc, *args, **kwargs):
    self.issues_svcr.converter = converters.Converter(mc, self.services)
    return wrapped_handler.wrapped(self.issues_svcr, mc, *args, **kwargs)

  def testGetIssue(self):
    """We can get an issue."""
    request = issues_pb2.GetIssueRequest(name=self.issue_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    actual_response = self.CallWrapped(self.issues_svcr.GetIssue, mc, request)
    self.assertEqual(
        actual_response, self.issues_svcr.converter.ConvertIssue(self.issue))

  # Note the 'empty' case doesn't make sense for ListComments, as one is created
  # for every issue.
  def testListComments(self):
    """We can list comments."""
    request = issues_pb2.ListCommentsRequest(parent=self.issue_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    actual_response = self.CallWrapped(
        self.issues_svcr.ListComments, mc, request)
    self.assertEqual(1, len(actual_response.comments))
