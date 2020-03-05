# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from google.protobuf import empty_pb2

from api import resource_name_converters as rnc
from api.v1 import converters
from api.v1 import monorail_servicer
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import hotlists_pb2
from api.v1.api_proto import hotlists_prpc_pb2
from businesslogic import work_env
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
        items=converters.ConvertHotlistItems(
            mc.cnxn, mc.auth, hotlist_id, visible_hotlist_items, self.services),
        next_page_token=next_page_token)


  @monorail_servicer.PRPCMethod
  def RerankHotlistItems(self, mc, request):
    # MonorailConnection, RerankHotlistItemsRequest ->
    # RerankHotlistItemsResponse
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
  def GetHotlist(self, mc, request):
    # MonorailConnection, GetHotlistRequest -> GetHotlistResponse
    """pRPC API method that implements GetHotlist.

    Raises:
      NoSuchHotlistException if the hotlist is not found.
      PermissionException if the user is not allowed to view the hotlist.
    """

    hotlist_id = rnc.IngestHotlistName(request.name)

    with work_env.WorkEnv(mc, self.services) as we:
      hotlist = we.GetHotlist(hotlist_id)

    return converters.ConvertHotlist(mc.cnxn, mc.auth, hotlist, self.services)
