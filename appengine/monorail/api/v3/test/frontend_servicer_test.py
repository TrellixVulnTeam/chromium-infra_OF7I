# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd
"""Tests for the hotlists servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from api.v3 import frontend_servicer
from api.v3 import converters
from api.v3.api_proto import frontend_pb2
from api.v3.api_proto import project_objects_pb2
from framework import exceptions
from framework import monorailcontext
from testing import fake
from services import service_manager


class FrontendServicerTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        features=fake.FeaturesService(),
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        config=fake.ConfigService(),
        user=fake.UserService(),
        template=fake.TemplateService(),
        usergroup=fake.UserGroupService())
    self.frontend_svcr = frontend_servicer.FrontendServicer(
        self.services, make_rate_limiter=False)

    self.user_1 = self.services.user.TestAddUser('user_111@example.com', 111)
    self.user_1_resource_name = 'users/111'
    self.project_1_resource_name = 'projects/proj'
    self.project_1 = self.services.project.TestAddProject(
        'proj', project_id=789)

  def CallWrapped(self, wrapped_handler, mc, *args, **kwargs):
    self.frontend_svcr.converter = converters.Converter(mc, self.services)
    return wrapped_handler.wrapped(self.frontend_svcr, mc, *args, **kwargs)

  def testGatherProjectEnvironment(self):
    """We can list a project's IssueTemplates."""
    request = frontend_pb2.GatherProjectEnvironmentRequest(
        parent=self.project_1_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    response = self.CallWrapped(
        self.frontend_svcr.GatherProjectEnvironment, mc, request)

    self.assertEqual(response, frontend_pb2.GatherProjectEnvironmentResponse())

  def testGatherProjectMembershipsForUser(self):
    """We can list a user's project memberships."""
    self.services.project.TestAddProject(
        'owner_proj', project_id=777, owner_ids=[111])
    self.services.project.TestAddProject(
        'committer_proj', project_id=888, committer_ids=[111])
    contributor_proj = self.services.project.TestAddProject(
        'contributor_proj', project_id=999)
    contributor_proj.contributor_ids = [111]

    request = frontend_pb2.GatherProjectMembershipsForUserRequest(
        user=self.user_1_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    response = self.CallWrapped(
        self.frontend_svcr.GatherProjectMembershipsForUser, mc, request)

    owner_membership = project_objects_pb2.ProjectMember(
        name='projects/{}/members/{}'.format('owner_proj', '111'),
        role=project_objects_pb2.ProjectMember.ProjectRole.Value('OWNER'))
    committer_membership = project_objects_pb2.ProjectMember(
        name='projects/{}/members/{}'.format('committer_proj', '111'),
        role=project_objects_pb2.ProjectMember.ProjectRole.Value('COMMITTER'))
    contributor_membership = project_objects_pb2.ProjectMember(
        name='projects/{}/members/{}'.format('contributor_proj', '111'),
        role=project_objects_pb2.ProjectMember.ProjectRole.Value('CONTRIBUTOR'))
    self.assertEqual(
        response,
        frontend_pb2.GatherProjectMembershipsForUserResponse(
            project_memberships=[
                owner_membership, committer_membership, contributor_membership
            ]))
