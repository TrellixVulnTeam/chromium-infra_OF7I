# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from google.protobuf import empty_pb2

from api import resource_name_converters as rnc
from api.v1 import monorail_servicer
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import hotlists_pb2
from api.v1.api_proto import hotlists_prpc_pb2
from businesslogic import work_env
from framework import exceptions
from tracker import tracker_constants


class HotlistsServicer(monorail_servicer.MonorailServicer):
  """Handle API requests related to Hotlist objects.
  Each API request is implemented with a method as defined in the
  .proto file. Each method does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = hotlists_prpc_pb2.HotlistsServiceDescription

  @monorail_servicer.PRPCMethod
  def ListHotlistItems(self, mc, request):
    # MonorailConnection, ListHotlistItemsRequest -> ListHotlistItemsResponse
    """pRPC API method that implements ListHotlistItems.

      Raises:
        NoSuchHotlistException if the hotlist is not found.
        PermissionException if the user is not allowed to view the hotlist.
        InputException if the request.page_token is invalid or the request does
          not match the previous request that provided the given page_token.
    """
    hotlist_id = rnc.IngestHotlistName(request.parent)

    # TODO(crbug/monorail/7104): take start from request.page_token
    start = 0
    sort_spec = request.order_by.replace(',', ' ')

    with work_env.WorkEnv(mc, self.services) as we:
      visible_hotlist_items, _harmonized_config = we.ListHotlistItems(
          hotlist_id, request.page_size, start,
          tracker_constants.ALL_ISSUES_CAN, sort_spec, '')

    # TODO(crbug/monorail/7104): plug in next_page_token when it's been
    # implemented.
    next_page_token = ''
    return hotlists_pb2.ListHotlistItemsResponse(
        items=self.converter.ConvertHotlistItems(
            hotlist_id, visible_hotlist_items),
        next_page_token=next_page_token)


  @monorail_servicer.PRPCMethod
  def RerankHotlistItems(self, mc, request):
    # MonorailConnection, RerankHotlistItemsRequest -> Empty
    """pRPC API method that implements RerankHotlistItems.

    Raises:
      NoSuchHotlistException if the hotlist is not found.
      PermissionException if the user is not allowed to rerank the hotlist.
      InputException if request.target_position is invalid or
        request.hotlist_items is empty or contains invalid items.
    """

    hotlist_id = rnc.IngestHotlistName(request.name)
    moved_issue_ids = rnc.IngestHotlistItemNames(
        mc.cnxn, request.hotlist_items, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      we.RerankHotlistItems(
          hotlist_id, moved_issue_ids, request.target_position)

    return empty_pb2.Empty()


  @monorail_servicer.PRPCMethod
  def RemoveHotlistItems(self, mc, request):
    # type: (MonorailConnection, RemoveHotlistItemsRequest) -> Empty
    """pPRC API method that implements RemoveHotlistItems.

    Raises:
      NoSuchHotlistException if the hotlist is not found.
      PermissionException if the user is not allowed to edit the hotlist.
      InputException if the items to be removed are not found in the hotlist.
    """

    hotlist_id = rnc.IngestHotlistName(request.parent)
    remove_issue_ids = rnc.IngestIssueNames(
        mc.cnxn, request.issues, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      we.RemoveHotlistItems(hotlist_id, remove_issue_ids)

    return empty_pb2.Empty()


  @monorail_servicer.PRPCMethod
  def AddHotlistItems(self, mc, request):
    # type: (MonorailConnection, AddHotlistItemsRequest) -> Empty
    """pRPC API method that implements AddHotlistItems.

    Raises:
      NoSuchHotlistException if the hotlist is not found.
      PermissionException if the user is not allowed to edit the hotlist.
      InputException if the request.target_position is invalid or the given
        list of issues to add is empty or invalid.
    """
    hotlist_id = rnc.IngestHotlistName(request.parent)
    new_issue_ids = rnc.IngestIssueNames(mc.cnxn, request.issues, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      we.AddHotlistItems(hotlist_id, new_issue_ids, request.target_position)

    return empty_pb2.Empty()


  @monorail_servicer.PRPCMethod
  def GetHotlist(self, mc, request):
    # MonorailConnection, GetHotlistRequest -> Hotlist
    """pRPC API method that implements GetHotlist.

    Raises:
      NoSuchHotlistException if the hotlist is not found.
      PermissionException if the user is not allowed to view the hotlist.
    """

    hotlist_id = rnc.IngestHotlistName(request.name)

    with work_env.WorkEnv(mc, self.services) as we:
      hotlist = we.GetHotlist(hotlist_id)

    return self.converter.ConvertHotlist(hotlist)

  @monorail_servicer.PRPCMethod
  def UpdateHotlist(self, mc, request):
    # type: (MonorailConnection, UpdateHotlistRequest) -> UpdateHotlistResponse
    """pRPC API method that implements UpdateHotlist.

    Raises:
      NoSuchHotlistException if the hotlist is not found.
      PermissionException if the user is not allowed to make this update.
      InputException if some request parameters are required and missing or
        invalid.
    """
    if not request.update_mask:
      raise exceptions.InputException('No paths given in `update_mask`.')
    if not request.hotlist:
      raise exceptions.InputException('No `hotlist` param given.')

    if not request.update_mask.IsValidForDescriptor(
        feature_objects_pb2.Hotlist.DESCRIPTOR):
      raise exceptions.InputException('Invalid `update_mask` for `hotlist`')

    hotlist_id = rnc.IngestHotlistName(request.hotlist.name)

    update_args = {}
    hotlist = request.hotlist
    for path in request.update_mask.paths:
      if path == 'display_name':
        update_args['hotlist_name'] = hotlist.display_name
      elif path == 'summary':
        update_args['summary'] = hotlist.summary
      elif path == 'description':
        update_args['description'] = hotlist.description
      elif path == 'hotlist_privacy':
        update_args['is_private'] = (
            hotlist.hotlist_privacy == feature_objects_pb2.Hotlist
            .HotlistPrivacy.Value('PRIVATE'))
      elif path == 'default_columns':
        update_args[
            'default_col_spec'] = self.converter.IngestIssuesListColumns(
                hotlist.default_columns)
      # TODO(crbug/monorail/7104): Add hotlist owner and editors.
    with work_env.WorkEnv(mc, self.services) as we:
      we.UpdateHotlist(hotlist_id, **update_args)
      hotlist = we.GetHotlist(hotlist_id, use_cache=False)

    return self.converter.ConvertHotlist(hotlist)

  @monorail_servicer.PRPCMethod
  def DeleteHotlist(self, mc, request):
    # MonorailConnection, GetHotlistRequest -> Empty
    """pRPC API method that implements DeleteHotlist.

    Raises:
      NoSuchHotlistException if the hotlist is not found.
      PermissionException if the user is not allowed to delete the hotlist.
    """

    hotlist_id = rnc.IngestHotlistName(request.name)

    with work_env.WorkEnv(mc, self.services) as we:
      we.DeleteHotlist(hotlist_id)

    return empty_pb2.Empty()
