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


class HotlistsServicer(monorail_servicer.MonorailServicer):
  """Handle API requests related to Hotlist objects.
  Each API request is implemented with a method as defined in the
  .proto file. Each method does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = hotlists_prpc_pb2.HotlistsServiceDescription

  @monorail_servicer.PRPCMethod
  def RerankHotlistItems(self, mc, request):
    """Rerank items of a Hotlist."""

    hotlist_id = rnc.IngestHotlistName(request.name)
    moved_issue_ids = rnc.IngestHotlistItemNames(
        mc, request.hotlist_items, self.services)

    with work_env.WorkEnv(mc, self.services) as we:
      we.RerankHotlistItems(
          hotlist_id, moved_issue_ids, request.target_position)

    return empty_pb2.Empty()
