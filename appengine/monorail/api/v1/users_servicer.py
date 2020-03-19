# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api import resource_name_converters as rnc
from api.v1 import monorail_servicer
from api.v1.api_proto import users_pb2
from api.v1.api_proto import user_objects_pb2
from api.v1.api_proto import users_prpc_pb2
from businesslogic import work_env


class UsersServicer(monorail_servicer.MonorailServicer):
  """Handle API requests related to User objects.
  Each API request is implemented with a method as defined in the
  .proto file. Each method does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = users_prpc_pb2.UsersServiceDescription

  @monorail_servicer.PRPCMethod
  def BatchGetUsers(self, mc, request):
    # type: (MonorailConnection, BatchGetUsersRequest) ->
    # BatchGetUsersResponse
    """pRPC API method that implements BatchGetUsers.

      Raises:
        InputException: if a name in request.names is invalid.
        NoSuchUserException: if a User is not found.
    """
    user_ids = rnc.IngestUserNames(request.names)

    with work_env.WorkEnv(mc, self.services) as we:
      users = we.BatchGetUsers(user_ids)

    api_users_by_id = self.converter.ConvertUsers(
        [user.user_id for user in users], None)
    api_users = [api_users_by_id[user_id] for user_id in user_ids]

    return users_pb2.BatchGetUsersResponse(users=api_users)
