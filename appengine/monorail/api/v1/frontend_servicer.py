# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api import resource_name_converters as rnc
from api.v1 import monorail_servicer
from api.v1.api_proto import frontend_pb2
from api.v1.api_proto import project_objects_pb2
from api.v1.api_proto import frontend_prpc_pb2
from businesslogic import work_env


class FrontendServicer(monorail_servicer.MonorailServicer):
  """Handle frontend specific API requests.
  Each API request is implemented with a method as defined in the
  .proto file. Each method does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = frontend_prpc_pb2.FrontendServiceDescription

  @monorail_servicer.PRPCMethod
  def GatherProjectEnvironment(self, _mc, _request):
    # type: (MonorailConnection, GatherProjectEnvironmentRequest) ->
    #     GatherProjectEnvironmentResponse
    """pRPC API method that implements GatherProjectEnvironment."""
    return frontend_pb2.GatherProjectEnvironmentResponse()

  @monorail_servicer.PRPCMethod
  def GatherProjectMembersForUser(self, mc, request):
    # type: (MonorailConnection, GatherProjectMembersForUserRequest) ->
    #     GatherProjectMembersForUserResponse
    """pRPC API method that implements GatherProjectMembersForUser.

    Raises:
      NoSuchUserException if the user is not found.
      InputException if the user resource name is invalid.
    """

    user_id = rnc.IngestUserName(mc, request.user, self.services)

    project_memberships = []

    with work_env.WorkEnv(mc, self.services) as we:
      owner, committer, contributor = we.GatherProjectMembersForUser(user_id)

    for project_id in owner:
      project_member = self.converter.CreateProjectMember(
          mc.cnxn, project_id, user_id, 'OWNER')
      project_memberships.append(project_member)

    for project_id in committer:
      project_member = self.converter.CreateProjectMember(
          mc.cnxn, project_id, user_id, 'COMMITTER')
      project_memberships.append(project_member)

    for project_id in contributor:
      project_member = self.converter.CreateProjectMember(
          mc.cnxn, project_id, user_id, 'CONTRIBUTOR')
      project_memberships.append(project_member)

    return frontend_pb2.GatherProjectMembersForUserResponse(
        project_memberships=project_memberships)
