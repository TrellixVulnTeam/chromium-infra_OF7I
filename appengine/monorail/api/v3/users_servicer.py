# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from google.protobuf import empty_pb2

from api import resource_name_converters as rnc
from api.v3 import monorail_servicer
from api.v3.api_proto import users_pb2
from api.v3.api_proto import user_objects_pb2
from api.v3.api_proto import users_prpc_pb2
from businesslogic import work_env


class UsersServicer(monorail_servicer.MonorailServicer):
  """Handle API requests related to User objects.
  Each API request is implemented with a method as defined in the
  .proto file. Each method does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = users_prpc_pb2.UsersServiceDescription

  @monorail_servicer.PRPCMethod
  def GetUser(self, mc, request):
    # type: (MonorailContext, GetUserRequest) ->
    # GetUserResponse
    """pRPC API method that implements GetUser.

      Raises:
        InputException if a name in request.name is invalid.
        NoSuchUserException if a User is not found.
    """
    user_id = rnc.IngestUserName(mc.cnxn, request.name, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      user = we.GetUser(user_id)

    return self.converter.ConvertUser(user, None)

  @monorail_servicer.PRPCMethod
  def BatchGetUsers(self, mc, request):
    # type: (MonorailContext, BatchGetUsersRequest) ->
    # BatchGetUsersResponse
    """pRPC API method that implements BatchGetUsers.

      Raises:
        InputException if a name in request.names is invalid.
        NoSuchUserException if a User is not found.
    """
    user_ids = rnc.IngestUserNames(mc.cnxn, request.names, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      users = we.BatchGetUsers(user_ids)

    api_users_by_id = self.converter.ConvertUsers(
        [user.user_id for user in users], None)
    api_users = [api_users_by_id[user_id] for user_id in user_ids]

    return users_pb2.BatchGetUsersResponse(users=api_users)

  @monorail_servicer.PRPCMethod
  def StarProject(self, mc, request):
    # type: (MonorailContext, StarProjectRequest) ->
    # ProjectStar
    """pRPC API method that implements StarProject.

      Raises:
        InputException if the project name in request.project is invalid.
        NoSuchProjectException if no project exists with the given name.
    """
    project_id = rnc.IngestProjectName(mc.cnxn, request.project, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      we.StarProject(project_id, True)

    user_id = mc.auth.user_id
    star_name = rnc.ConvertProjectStarName(
        mc.cnxn, user_id, project_id, self.services)

    return user_objects_pb2.ProjectStar(name=star_name)

  @monorail_servicer.PRPCMethod
  def UnStarProject(self, mc, request):
    # type: (MonorailContext, UnStarProjectRequest) ->
    # Empty
    """pRPC API method that implements UnStarProject.

      Raises:
        InputException if the project name in request.project is invalid.
        NoSuchProjectException if no project exists with the given name.
    """
    project_id = rnc.IngestProjectName(mc.cnxn, request.project, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      we.StarProject(project_id, False)

    return empty_pb2.Empty()

  @monorail_servicer.PRPCMethod
  def ListProjectStars(self, mc, request):
    # type: (MonorailContext, ListProjectStarsRequest) ->
    #   ListProjectStarsResponse
    """pRPC API method that implements ListProjectStars.

      Raises:
        InputException: if the `page_token` or `parent` is invalid.
        NoSuchUserException: if the User is not found.
    """
    user_id = rnc.IngestUserName(mc.cnxn, request.parent, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      projects = we.ListStarredProjects(user_id)

    # TODO(crbug.com/monorail/7175): Add pagination logic.
    return users_pb2.ListProjectStarsResponse(
        project_stars=self.converter.ConvertProjectStars(user_id, projects))
