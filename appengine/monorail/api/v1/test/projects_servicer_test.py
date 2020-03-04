# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd
"""Tests for the hotlists servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from api import resource_name_converters as rnc
from api.v1 import projects_servicer
from api.v1 import converters
from api.v1.api_proto import projects_pb2
from framework import exceptions
from framework import monorailcontext
from framework import permissions
from testing import fake
from services import service_manager


class ProjectsServicerTest(unittest.TestCase):

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
    self.projects_svcr = projects_servicer.ProjectsServicer(
        self.services, make_rate_limiter=False)

    self.user_1 = self.services.user.TestAddUser('user_111@example.com', 111)

    self.project_1 = self.services.project.TestAddProject(
        'proj', project_id=789)
    self.template_1 = self.services.template.TestAddIssueTemplateDef(
        123, 789, 'template_1_name')
    self.project_1_resource_name = 'projects/proj'

  def CallWrapped(self, wrapped_handler, *args, **kwargs):
    return wrapped_handler.wrapped(self.projects_svcr, *args, **kwargs)

  def testListIssueTemplates(self):
    """We can list a project's IssueTemplates."""
    request = projects_pb2.ListIssueTemplatesRequest(
        parent=self.project_1_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    response = self.CallWrapped(
        self.projects_svcr.ListIssueTemplates, mc, request)

    # TODO(crbug.com/monorail/7216): Replace after implementation
    expected = []
    self.assertEqual(
        response, projects_pb2.ListIssueTemplatesResponse(templates=expected))
