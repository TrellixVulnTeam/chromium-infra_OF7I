# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api import resource_name_converters as rnc
from api.v1 import monorail_servicer
from api.v1 import converters
from api.v1.api_proto import issues_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import issues_prpc_pb2
from businesslogic import work_env
from framework import exceptions

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
      InputException if the given name does not have a valid format.
      NoSuchIssueException if the issue is not found.
      PermissionException if the user is not allowed to view the issue.
    """
    issue_id = rnc.IngestIssueName(mc.cnxn, request.name, self.services)
    with work_env.WorkEnv(mc, self.services) as we:
      issue = we.GetIssue(issue_id, allow_viewing_deleted=True)
    return self.converter.ConvertIssue(issue)
