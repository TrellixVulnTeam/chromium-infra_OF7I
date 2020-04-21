# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api import resource_name_converters as rnc
from api.v1 import converters
from api.v1 import monorail_servicer
from api.v1 import paginator
from api.v1.api_proto import issues_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import issues_prpc_pb2
from businesslogic import work_env
from framework import exceptions
from tracker import tracker_constants


class IssuesServicer(monorail_servicer.MonorailServicer):
  """Handle API requests related to Issue objects.
  Each API request is implemented with a method as defined in the
  .proto file that does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = issues_prpc_pb2.IssuesServiceDescription

  @monorail_servicer.PRPCMethod
  def GetIssue(self, mc, request):
    # type: (MonorailConnection, GetIssueRequest) -> Issue
    """pRPC API method that implements GetIssue.

    Raises:
      InputException: the given name does not have a valid format.
      NoSuchIssueException: the issue is not found.
      PermissionException the user is not allowed to view the issue.
    """
    issue_id = rnc.IngestIssueName(mc.cnxn, request.name, self.services)
    with work_env.WorkEnv(mc, self.services) as we:
      issue = we.GetIssue(issue_id, allow_viewing_deleted=True)
    return self.converter.ConvertIssue(issue)


  @monorail_servicer.PRPCMethod
  def ListComments(self, mc, request):
    # type: (MonorailConnection, ListCommentsRequest) -> ListCommentsResponse
    """pRPC API method that implements ListComments.

    Raises:
      InputException: the given name format or page_size are not valid.
      NoSuchIssue: the parent is not found.
      PermissionException: the user is not allowed to view the parent.
    """
    issue_id = rnc.IngestIssueName(mc.cnxn, request.parent, self.services)
    page_size = paginator.CoercePageSize(
        request.page_size, tracker_constants.MAX_COMMENTS_PER_PAGE)
    pager = paginator.Paginator(parent=request.parent, page_size=page_size)

    with work_env.WorkEnv(mc, self.services) as we:
      list_result = we.SafeListIssueComments(
          issue_id, page_size, pager.GetStart(request.page_token))
      # TODO(crbug.com/monorail/7143): Rewrite ConvertComments to take issue_id.
      issue = we.GetIssue(issue_id, allow_viewing_deleted=True)
    return issues_pb2.ListCommentsResponse(
        comments=self.converter.ConvertComments(issue, list_result.items),
        next_page_token=pager.GenerateNextPageToken(list_result.next_start))
