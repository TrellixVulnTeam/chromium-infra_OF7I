# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from google.protobuf import empty_pb2

from api import resource_name_converters as rnc
from api.v3 import permission_converters as pc
from api.v3 import monorail_servicer
from api.v3.api_proto import permission_objects_pb2
from api.v3.api_proto import permissions_pb2
from api.v3.api_proto import permissions_prpc_pb2
from businesslogic import work_env
from framework import exceptions


class PermissionsServicer(monorail_servicer.MonorailServicer):
  """Handle API requests related to Permissions.
  Each API request is implemented with a method as defined in the
  .proto file. Each method does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = permissions_prpc_pb2.PermissionsServiceDescription

  @monorail_servicer.PRPCMethod
  def BatchGetPermissionSets(self, mc, request):
    # type: (MonorailConnection, BatchGetPermissionSetsRequest) ->
    # BatchGetPermissionSetsResponse
    """pRPC API method that implements BatchGetPermissionSets.

    Raises:
      InputException: if any name in request.names is not a valid resource name
          or a permission string is not recognized.
      PermissionException: if the requester does not have permission to
          view one of the resources.
    """
    api_permission_sets = []
    with work_env.WorkEnv(mc, self.services) as we:
      for name in request.names:
        api_permission_sets.append(self._GetPermissionSet(mc.cnxn, we, name))

    return permissions_pb2.BatchGetPermissionSetsResponse(
        permission_sets=api_permission_sets)

  def _GetPermissionSet(self, cnxn, we, name):
    # type: (sql.MonorailConnection, businesslogic.WorkEnv, str) ->
    # permission_objects_pb2.PermissionSet
    """Takes a resource name and returns the PermissionSet for the resource.

      Args:
        cnxn: MonorailConnection object to the database.
        we: WorkEnv object to get the permission strings.
        name: resource name of a resource we want a PermissionSet for.

      Returns:
        PermissionSet object.

      Raises:
      InputException: if request.name is not a valid resource name or a
          permission string is not recognized.
      PermissionException: if the requester does not have permission to
          view the resource.
    """
    try:
      hotlist_id = rnc.IngestHotlistName(name)
      permissions = we.ListHotlistPermissions(hotlist_id)
      api_permissions = pc.ConvertHotlistPermissions(permissions)
      return permission_objects_pb2.PermissionSet(
          resource=name, permissions=api_permissions)
    except exceptions.InputException:
      pass
    try:
      project_id, field_id = rnc.IngestFieldDefName(cnxn, name, self.services)
      permissions = we.ListFieldDefPermissions(field_id, project_id)
      api_permissions = pc.ConvertFieldDefPermissions(permissions)
      return permission_objects_pb2.PermissionSet(
          resource=name, permissions=api_permissions)
    except exceptions.InputException:
      pass
    # TODO(crbug/monorail/7339): Add more try-except blocks for other
    # resource types.
    raise exceptions.InputException('invalid resource name')
