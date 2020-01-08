# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the issues servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from api.v1 import issues_servicer
from api.v1.api_proto import issues_pb2
from api.v1.api_proto import issue_objects_pb2
from framework import monorailcontext
from testing import fake
from services import service_manager


class IssuesServicerTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        user=fake.UserService(),
        usergroup=fake.UserGroupService())
    self.issues_svcr = issues_servicer.IssuesServicer(
        self.services, make_rate_limiter=False)
    self.owner = self.services.user.TestAddUser('owner@example.com', 111)

  def CallWrapped(self, wrapped_handler, *args, **kwargs):
    return wrapped_handler.wrapped(self.issues_svcr, *args, **kwargs)

  def testGetIssue(self):
    """We can get an issue."""
    request = issues_pb2.GetIssueRequest()
    request.name = "projects/chicken/issues/1234"
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.owner.email)
    actual_response = self.CallWrapped(self.issues_svcr.GetIssue, mc, request)
    self.assertEqual(
        actual_response, issue_objects_pb2.Issue(
            name=request.name, summary="sum summary"))
