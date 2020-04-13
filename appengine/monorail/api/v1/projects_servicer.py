# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api import resource_name_converters as rnc
from api.v1 import monorail_servicer
from api.v1.api_proto import projects_pb2
from api.v1.api_proto import projects_prpc_pb2
from businesslogic import work_env


class ProjectsServicer(monorail_servicer.MonorailServicer):
  """Handle API requests related to Project objects.
  Each API request is implemented with a method as defined in the
  .proto file. Each method does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = projects_prpc_pb2.ProjectsServiceDescription

  @monorail_servicer.PRPCMethod
  def ListIssueTemplates(self, mc, request):
    # type: (MonorailConnection, ListIssueTemplatesRequest) ->
    #   ListIssueTemplatesResponse
    """pRPC API method that implements ListIssueTemplates.

      Raises:
        InputException if the request.parent is invalid.
    """
    project_id = rnc.IngestProjectName(mc.cnxn, request.parent, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      templates = we.ListProjectTemplates(project_id)

    return projects_pb2.ListIssueTemplatesResponse(
        templates=self.converter.ConvertIssueTemplates(project_id, templates))

  @monorail_servicer.PRPCMethod
  def ListProjects(self, mc, _):
    # type: (MonorailConnection, ListProjectsRequest) -> ListProjectsResponse
    """pRPC API method that implements ListProjects.

      Raises:
        InputException if the request.page_token is invalid or the request does
          not match the previous request that provided the given page_token.
    """
    with work_env.WorkEnv(mc, self.services) as we:
      allowed_project_ids = we.ListProjects()
      projects_dict = we.GetProjects(allowed_project_ids)
      projects = [projects_dict[proj_id] for proj_id in allowed_project_ids]

    # TODO(crbug.com/monorail/7505): Add pagination logic.
    return projects_pb2.ListProjectsResponse(
        projects=self.converter.ConvertProjects(projects))