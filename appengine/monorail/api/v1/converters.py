# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api import resource_name_converters as rnc
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2


def ConvertHotlist(hotlist):
  # proto.features_pb2.Hotlist -> api_proto.feature_objects_pb2.Hotlist
  """Convert a protorpc Hotlist into a protoc Hotlist.

  Args:
    hotlist: Hotlist protorpc object.

  Returns:
    The equivalent Hotlist protoc Hotlist.

  """
  hotlist_resource_name = rnc.ConvertHotlistName(hotlist.hotlist_id)
  owner_resource_name = rnc.ConvertUserNames(hotlist.owner_ids)[0]
  editor_resource_names = rnc.ConvertUserNames(hotlist.editor_ids)
  default_columns  = [issue_objects_pb2.IssuesListColumn(column=col)
                      for col in hotlist.default_col_spec.split()]
  api_hotlist = feature_objects_pb2.Hotlist(
      name=hotlist_resource_name,
      display_name=hotlist.name,
      owner=owner_resource_name,
      editors=editor_resource_names,
      summary=hotlist.summary,
      description=hotlist.description,
      default_columns=default_columns)
  if not hotlist.is_private:
    api_hotlist.hotlist_privacy = (
        feature_objects_pb2.Hotlist.HotlistPrivacy.Value('PUBLIC'))
  return api_hotlist
