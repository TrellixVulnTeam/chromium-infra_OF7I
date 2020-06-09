# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd
"""Tests for the hotlists servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from google.protobuf import timestamp_pb2
from mock import patch

from api import resource_name_converters as rnc
from api.v3 import converters
from api.v3 import frontend_servicer
from api.v3.api_proto import frontend_pb2
from api.v3.api_proto import project_objects_pb2
from framework import exceptions
from framework import monorailcontext
from proto import tracker_pb2
from services import service_manager
from testing import fake
from tracker import tracker_constants


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
    self.template_0 = self.services.template.TestAddIssueTemplateDef(
        11110, self.project_1.project_id, 'template0')
    self.PAST_TIME = 12345
    self.component_def_1_path = 'foo'
    self.component_def_1_id = self.services.config.CreateComponentDef(
        self.cnxn, self.project_1.project_id, self.component_def_1_path,
        'cd1_docstring', False, [self.user_1.user_id], [], self.PAST_TIME,
        self.user_1.user_id, [])
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
    self.approval_def_1_name = 'approval_field_1'
    self.approval_def_1_id = self._CreateFieldDef(
        self.project_1.project_id,
        self.approval_def_1_name,
        'APPROVAL_TYPE',
        docstring='ad_1_docstring',
        admin_ids=[self.user_1.user_id])
    self.approval_def_1 = tracker_pb2.ApprovalDef(
        approval_id=self.approval_def_1_id,
        approver_ids=[self.user_1.user_id],
        survey='approval_def_1 survey')
    self.services.config.UpdateConfig(
        self.cnxn,
        self.project_1,
        # UpdateConfig accepts tuples rather than protorpc *Defs
        approval_defs=[
            (ad.approval_id, ad.approver_ids, ad.survey)
            for ad in [self.approval_def_1]
        ])

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

  def CallWrapped(self, wrapped_handler, mc, *args, **kwargs):
    self.frontend_svcr.converter = converters.Converter(mc, self.services)
    return wrapped_handler.wrapped(self.frontend_svcr, mc, *args, **kwargs)

  @patch('project.project_helpers.GetThumbnailUrl')
  def testGatherProjectEnvironment(self, mock_GetThumbnailUrl):
    """We can fetch all project related parameters for web frontend."""
    mock_GetThumbnailUrl.return_value = 'xyz'

    request = frontend_pb2.GatherProjectEnvironmentRequest(
        parent=self.project_1_resource_name)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    response = self.CallWrapped(
        self.frontend_svcr.GatherProjectEnvironment, mc, request)
    project_config = self.services.config.GetProjectConfig(
        self.cnxn, self.project_1.project_id)

    self.assertEqual(
        response.project,
        self.frontend_svcr.converter.ConvertProject(self.project_1))
    self.assertEqual(
        response.project_config,
        self.frontend_svcr.converter.ConvertProjectConfig(project_config))

    self.assertEqual(
        len(response.statuses),
        len(tracker_constants.DEFAULT_WELL_KNOWN_STATUSES))
    self.assertEqual(
        response.statuses[0],
        project_objects_pb2.StatusDef(
            name='projects/{project_name}/statusDefs/{status}'.format(
                project_name=self.project_1.project_name,
                status=tracker_constants.DEFAULT_WELL_KNOWN_STATUSES[0][0]),
            value=tracker_constants.DEFAULT_WELL_KNOWN_STATUSES[0][0],
            type=project_objects_pb2.StatusDef.StatusDefType.Value('OPEN'),
            rank=0,
            docstring=tracker_constants.DEFAULT_WELL_KNOWN_STATUSES[0][1],
            state=project_objects_pb2.StatusDef.StatusDefState.Value('ACTIVE'),
        ))

    self.assertEqual(
        len(response.well_known_labels),
        len(tracker_constants.DEFAULT_WELL_KNOWN_LABELS))
    self.assertEqual(
        response.well_known_labels[0],
        project_objects_pb2.LabelDef(
            name='projects/{project_name}/labelDefs/{label}'.format(
                project_name=self.project_1.project_name,
                label=tracker_constants.DEFAULT_WELL_KNOWN_LABELS[0][0]),
            value=tracker_constants.DEFAULT_WELL_KNOWN_LABELS[0][0],
            docstring=tracker_constants.DEFAULT_WELL_KNOWN_LABELS[0][1],
            state=project_objects_pb2.LabelDef.LabelDefState.Value('ACTIVE'),
        ))

    expected = self.frontend_svcr.converter.ConvertComponentDefs(
        project_config.component_defs, self.project_1.project_id)
    # Have to use list comprehension to break response sub field into list
    self.assertEqual([api_cd for api_cd in response.components], expected)

    expected = self.frontend_svcr.converter.ConvertFieldDefs(
        project_config.field_defs, self.project_1.project_id)
    self.assertEqual([api_fd for api_fd in response.fields], expected)

    expected = self.frontend_svcr.converter.ConvertApprovalDefs(
        project_config.approval_defs, self.project_1.project_id)
    self.assertEqual([api_ad for api_ad in response.approval_fields], expected)

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
