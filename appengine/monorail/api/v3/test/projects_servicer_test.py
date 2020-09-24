# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd
"""Tests for the hotlists servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest
import mock
import logging

from google.protobuf import timestamp_pb2
from google.protobuf import empty_pb2

from api import resource_name_converters as rnc
from api.v3 import projects_servicer
from api.v3 import converters
from api.v3.api_proto import projects_pb2
from api.v3.api_proto import project_objects_pb2
from api.v3.api_proto import issue_objects_pb2
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
        123, 789, 'template_1_name', content='foo bar', summary='foo')
    self.project_1_resource_name = 'projects/proj'
    self.converter = None

  def CallWrapped(self, wrapped_handler, mc, *args, **kwargs):
    self.converter = converters.Converter(mc, self.services)
    self.projects_svcr.converter = self.converter
    return wrapped_handler.wrapped(self.projects_svcr, mc, *args, **kwargs)

  def testListIssueTemplates(self):
    request = projects_pb2.ListIssueTemplatesRequest(
        parent=self.project_1_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    response = self.CallWrapped(
        self.projects_svcr.ListIssueTemplates, mc, request)

    expected_issue = issue_objects_pb2.Issue(
        summary=self.template_1.summary,
        state=issue_objects_pb2.IssueContentState.Value('ACTIVE'),
        status=issue_objects_pb2.Issue.StatusValue(
            status=self.template_1.status,
            derivation=issue_objects_pb2.Derivation.Value('EXPLICIT')))
    expected_template = project_objects_pb2.IssueTemplate(
        name='projects/{}/templates/{}'.format(
            self.project_1.project_name, self.template_1.template_id),
        display_name=self.template_1.name,
        issue=expected_issue,
        summary_must_be_edited=False,
        template_privacy=project_objects_pb2.IssueTemplate.TemplatePrivacy
        .Value('PUBLIC'),
        default_owner=project_objects_pb2.IssueTemplate.DefaultOwner.Value(
            'DEFAULT_OWNER_UNSPECIFIED'),
        component_required=False)

    self.assertEqual(
        response,
        projects_pb2.ListIssueTemplatesResponse(templates=[expected_template]))

  @mock.patch('time.time')
  def testCreateComponentDef(self, mockTime):
    now = 123
    mockTime.return_value = now

    user_1 = self.services.user.TestAddUser('achilles@test.com', 981)
    self.services.user.TestAddUser('patroclus@test.com', 982)
    self.services.user.TestAddUser('circe@test.com', 983)

    project = self.services.project.TestAddProject(
        'chicken', project_id=987, owner_ids=[user_1.user_id])
    config = fake.MakeTestConfig(project.project_id, [], [])
    self.services.config.StoreConfig(self.cnxn, config)

    expected = project_objects_pb2.ComponentDef(
        value='circe',
        docstring='You threw me to the crows',
        admins=['users/983'],
        ccs=['users/981', 'users/982'],
        labels=['more-soup', 'beach-day'],
    )
    request = projects_pb2.CreateComponentDefRequest(
        parent='projects/chicken', component_def=expected)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=user_1.email)
    response = self.CallWrapped(
        self.projects_svcr.CreateComponentDef, mc, request)

    self.assertEqual(1, len(config.component_defs))
    expected.name = 'projects/chicken/componentDefs/%d' % config.component_defs[
        0].component_id
    expected.state = project_objects_pb2.ComponentDef.ComponentDefState.Value(
        'ACTIVE')
    expected.creator = 'users/981'
    expected.create_time.FromSeconds(now)
    expected.modify_time.FromSeconds(0)
    self.assertEqual(response, expected)

  def testDeleteComponentDef(self):
    user_1 = self.services.user.TestAddUser('achilles@test.com', 981)
    project = self.services.project.TestAddProject(
        'chicken', project_id=987, owner_ids=[user_1.user_id])
    config = fake.MakeTestConfig(project.project_id, [], [])
    component_def = fake.MakeTestComponentDef(
        project.project_id, 1, path='Chickens>Dickens')
    config.component_defs = [component_def]
    self.services.config.StoreConfig(self.cnxn, config)

    request = projects_pb2.DeleteComponentDefRequest(
        name='projects/chicken/componentDefs/1')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=user_1.email)
    actual = self.CallWrapped(
        self.projects_svcr.DeleteComponentDef, mc, request)
    self.assertEqual(actual, empty_pb2.Empty())

    self.assertEqual(config.component_defs, [])

  @mock.patch('project.project_helpers.GetThumbnailUrl')
  def testListProjects(self, mock_GetThumbnailUrl):
    mock_GetThumbnailUrl.return_value = 'xyz'

    request = projects_pb2.ListProjectsRequest()

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    response = self.CallWrapped(self.projects_svcr.ListProjects, mc, request)

    expected_project = project_objects_pb2.Project(
        name=self.project_1_resource_name,
        display_name=self.project_1.project_name,
        summary=self.project_1.summary,
        thumbnail_url='xyz')

    self.assertEqual(
        response,
        projects_pb2.ListProjectsResponse(projects=[expected_project]))
