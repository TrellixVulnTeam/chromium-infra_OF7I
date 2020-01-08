# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api.v1 import monorail_servicer
from api.v1.api_proto import issues_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import issues_prpc_pb2


class IssuesServicer(monorail_servicer.MonorailServicer):
   """Handle API requests related to Issue objects.
  Each API request is implemented with a method as defined in the
  .proto file that does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

   DESCRIPTION = issues_prpc_pb2.IssuesServiceDescription

   @monorail_servicer.PRPCMethod
   def GetIssue(self, _mc, request):
     """Return the specified issue in a response proto."""
     return issue_objects_pb2.Issue(
         name=request.name, summary="sum summary")
