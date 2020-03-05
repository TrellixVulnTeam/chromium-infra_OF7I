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


def ConvertHotlist(cnxn, user_auth, hotlist, services):
  # type: (MonorailConnection, AuthData, proto.feature_objects_pb2.Hotlist,
  #   Services) -> api_proto.feature_objects_pb2.Hotlist
  """Convert a protorpc Hotlist into a protoc Hotlist.

  Args:
    cnxn: MonorailConnection object.
    user_auth: AuthData of the logged-in user.
    hotlist: Hotlist protorpc object.
    services: Services object for connections to backend services.

  Returns:
    The equivalent Hotlist protoc Hotlist.

  """
  hotlist_resource_name = rnc.ConvertHotlistName(hotlist.hotlist_id)
  # TODO(crbug/monorail/7238): Get the list of projects that have issues
  # in hotlist.
  members_by_id = ConvertUsers(
      cnxn, hotlist.owner_ids + hotlist.editor_ids, user_auth, None, services)
  default_columns  = [issue_objects_pb2.IssuesListColumn(column=col)
                      for col in hotlist.default_col_spec.split()]
  api_hotlist = feature_objects_pb2.Hotlist(
      name=hotlist_resource_name,
      display_name=hotlist.name,
      owner=members_by_id.get(hotlist.owner_ids[0]),
      editors=[members_by_id.get(editor_id) for editor_id in
               hotlist.editor_ids],
      summary=hotlist.summary,
      description=hotlist.description,
      default_columns=default_columns)
  if not hotlist.is_private:
    api_hotlist.hotlist_privacy = (
        feature_objects_pb2.Hotlist.HotlistPrivacy.Value('PUBLIC'))
  return api_hotlist


def ConvertHotlistItems(cnxn, user_auth, hotlist_id, items, services):
  # type: (MonorailConnection, int, Sequence[proto.features_pb2.HotlistItem],
  #     Services) -> Sequence[api_proto.feature_objects_pb2.Hotlist]
  """Convert a Sequence of protorpc HotlistItems into a Sequence of protoc
     HotlistItems.

  Args:
    cnxn: MonorailConnection object.
    user_auth: AuthData object of the logged-in user.
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
  # TODO(crbug/monorail/7238): Get the list of projects that have issues
  # in hotlist.
  adders_by_id = ConvertUsers(
      cnxn, [item.adder_id for item in items], user_auth, None, services)

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
        adder=adders_by_id.get(item.adder_id),
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
