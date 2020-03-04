# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import logging

from google.protobuf import timestamp_pb2

from api import resource_name_converters as rnc
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import user_objects_pb2

from framework import framework_bizobj
from framework import framework_helpers


def ConvertHotlist(hotlist):
  # proto.features_pb2.Hotlist -> api_proto.feature_objects_pb2.Hotlist
  """Convert a protorpc Hotlist into a protoc Hotlist.

  Args:
    hotlist: Hotlist protorpc object.

  Returns:
    The equivalent Hotlist protoc Hotlist.

  """
  hotlist_resource_name = rnc.ConvertHotlistName(hotlist.hotlist_id)
  user_resource_names_dict = rnc.ConvertUserNames(
      hotlist.owner_ids + hotlist.editor_ids)
  default_columns  = [issue_objects_pb2.IssuesListColumn(column=col)
                      for col in hotlist.default_col_spec.split()]
  api_hotlist = feature_objects_pb2.Hotlist(
      name=hotlist_resource_name,
      display_name=hotlist.name,
      owner=user_resource_names_dict.get(hotlist.owner_ids[0]),
      editors=[
          user_resource_names_dict.get(user_id)
          for user_id in hotlist.editor_ids
      ],
      summary=hotlist.summary,
      description=hotlist.description,
      default_columns=default_columns)
  if not hotlist.is_private:
    api_hotlist.hotlist_privacy = (
        feature_objects_pb2.Hotlist.HotlistPrivacy.Value('PUBLIC'))
  return api_hotlist


def ConvertHotlistItems(cnxn, hotlist_id, items, services):
  # MonorailConnection, int, Sequence[proto.features_pb2.HotlistItem],
  #     Services -> Sequence[api_proto.feature_objects_pb2.Hotlist]
  """Convert a Sequence of protorpc HotlistItems into a Sequence of protoc
     HotlistItems.

  Args:
    cnxn: MonorailConnection object.
    hotlist_id: ID of the Hotlist the items belong to.
    items: Sequence of HotlistItem protorpc objects.
    services: Services object for connections to backend services.

  Returns:
    Sequence of protoc HotlistItems in the same order they are given in `items`.
    In the rare event that any issues in `items` are not found, they will be
    omitted from the result.
  """
  issue_ids = [item.issue_id for item in items]
  # Converting HotlistItemNames and IssueNames both require looking up the
  # issues in the hotlist. However, we want to keep the code clean and readable
  # so we keep the two processes separate.
  resource_names_dict = rnc.ConvertHotlistItemNames(
      cnxn, hotlist_id, issue_ids, services)
  issue_names_dict = rnc.ConvertIssueNames(cnxn, issue_ids, services)
  adder_names_dict = rnc.ConvertUserNames([item.adder_id for item in items])

  # Filter out items whose issues were not found.
  found_items = [
      item for item in items if resource_names_dict.get(item.issue_id) and
      issue_names_dict.get(item.issue_id)
  ]
  if len(items) != len(found_items):
    found_ids = [item.issue_id for item in found_items]
    missing_ids = [iid for iid in issue_ids if iid not in found_ids]
    logging.info('HotlistItem issues %r not found' % missing_ids)

  # Generate user friendly ranks (0, 1, 2, 3,...) that are exposed to API
  # clients, instead of using padded ranks (1, 11, 21, 31,...).
  sorted_ranks = sorted(item.rank for item in found_items)
  friendly_ranks_dict = {
      rank: friendly_rank for friendly_rank, rank in enumerate(sorted_ranks)
  }

  api_items = []
  for item in found_items:
    api_item = feature_objects_pb2.HotlistItem(
        name=resource_names_dict.get(item.issue_id),
        issue=issue_names_dict.get(item.issue_id),
        rank=friendly_ranks_dict[item.rank],
        adder=adder_names_dict.get(item.adder_id),
        note=item.note)
    if item.date_added:
      api_item.create_time.FromSeconds(item.date_added)
    api_items.append(api_item)

  return api_items

# Users


# Because Monorail obscures emails of Users on the site, wherever
# in the API we would normally use User resource names, we use
# full User objects instead. For this reason, ConvertUsers is called
# where we would normally call some ConvertUserResourceNames function.
# So ConvertUsers follows the patterns in resource_name_converters.py
# by taking in User IDs and and returning a dict rather than a list.
# TODO(crbug/monorail/7238): take a list of projects when
# CreateUserDisplayNames() can take in a list of projects.
def ConvertUsers(cnxn, user_ids, user_auth, project, services):
  # type: (MonorailConnection, List(int), AuthData, protorpc.Project,
  #   Services) -> Map(int, api_proto.user_objects_pb2.User)
  """Convert list of protorpc_users into list of protoc Users.

  Args:
    cnxn: MonorailConnection object.
    user_ids: List of User IDs.
    user_auth: AuthData of the logged-in user.
    project: currently viewed project.
    services: Services object for connections to backend services.

  Returns:
    Dict of User IDs to User resource names for all given users.
  """
  user_ids_to_names = {}

  # Get display names
  users_by_id = services.user.GetUsersByIDs(cnxn, user_ids)
  display_names_by_id = framework_bizobj.CreateUserDisplayNames(
      user_auth, users_by_id.values(), project)


  for user_id, user in users_by_id.items():
    name = rnc.ConvertUserNames([user_id]).get(user_id)

    display_name = display_names_by_id.get(user_id)
    availability = framework_helpers.GetUserAvailability(user)
    availability_message, _availability_status = availability

    user_ids_to_names[user_id] = user_objects_pb2.User(
        name=name,
        display_name=display_name,
        availability_message=availability_message)

  return user_ids_to_names
