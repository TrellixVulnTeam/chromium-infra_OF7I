# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Business objects for Monorail's framework.

These are classes and functions that operate on the objects that
users care about in Monorail but that are not part of just one specific
component: e.g., projects, users, and labels.
"""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import functools
import itertools
import re

import six

import settings
from framework import exceptions
from framework import framework_constants
from proto import tracker_pb2
from services import client_config_svc


# Pattern to match a valid column header name.
RE_COLUMN_NAME = r'\w+[\w+-.]*\w+'

# Compiled regexp to match a valid column specification.
RE_COLUMN_SPEC = re.compile('(%s(\s%s)*)*$' % (RE_COLUMN_NAME, RE_COLUMN_NAME))


def WhichUsersShareAProject(cnxn, services, user_effective_ids, other_users):
  # type: (MonorailConnection, Services, Sequence[int],
  #     Collection[user_pb2.User]) -> Collection[user_pb2.User]
  """Returns a list of users that share a project with given user_effective_ids.

  Args:
    cnxn: MonorailConnection to the database.
    services: Services object for connections to backend services.
    user_effective_ids: The user's set of effective_ids.
    other_users: The list of users to be filtered for email visibility.

  Returns:
    Collection of users that share a project with at least one effective_id.
  """

  projects_by_user_effective_id = services.project.GetProjectMemberships(
      cnxn, user_effective_ids)
  authed_user_projects = set(
      itertools.chain.from_iterable(projects_by_user_effective_id.values()))

  other_user_ids = [other_user.user_id for other_user in other_users]
  all_other_user_effective_ids = GetEffectiveIds(cnxn, services, other_user_ids)
  users_that_share_project = []
  for other_user in other_users:
    other_user_effective_ids = all_other_user_effective_ids[other_user.user_id]

    # Do not filter yourself.
    if any(uid in user_effective_ids for uid in other_user_effective_ids):
      users_that_share_project.append(other_user)
      continue

    other_user_proj_by_effective_ids = services.project.GetProjectMemberships(
        cnxn, other_user_effective_ids)
    other_user_projects = itertools.chain.from_iterable(
        other_user_proj_by_effective_ids.values())
    if any(project in authed_user_projects for project in other_user_projects):
      users_that_share_project.append(other_user)
  return users_that_share_project


def FilterViewableEmails(cnxn, services, user_auth, other_users):
  # type: (MonorailConnection, Services, AuthData,
  #     Collection[user_pb2.User]) -> Collection[user_pb2.User]
  """Returns a list of users with emails visible to `user_auth`.

  Args:
    cnxn: MonorailConnection to the database.
    services: Services object for connections to backend services.
    user_auth: The AuthData of the user viewing the email addresses.
    other_users: The list of users to be filtered for email visibility.

  Returns:
    Collection of user that should reveal their emails.
  """
  # Case 1: Anon users don't see anything revealed.
  if user_auth.user_pb is None:
    return []

  # Case 2: site admins always see unobscured email addresses.
  if user_auth.user_pb.is_site_admin:
    return other_users

  # Case 3: Members of any groups in settings.full_emails_perm_groups
  # can view unobscured email addresses.
  for group_email in settings.full_emails_perm_groups:
    if services.usergroup.LookupUserGroupID(
        cnxn, group_email) in user_auth.effective_ids:
      return other_users

  # Case 4: Users see unobscured emails as long as they share a common Project.
  return WhichUsersShareAProject(
      cnxn, services, user_auth.effective_ids, other_users)


def DoUsersShareAProject(cnxn, services, user_effective_ids, other_user_id):
  # type: (MonorailConnection, Services, Sequence[int], int) -> bool
  """Determine whether two users share at least one Project.

  The user_effective_ids may include group ids or the other_user_id may be a
  member of a group that results in transitive Project ownership.

  Args:
    cnxn: MonorailConnection to the database.
    services: Services object for connections to backend services.
    user_effective_ids: The effective ids of the authorized User.
    other_user_id: The other user's user_id to compare against.

  Returns:
    True if one or more Projects are shared between the Users.
  """
  projects_by_user_effective_id = services.project.GetProjectMemberships(
      cnxn, user_effective_ids)
  authed_user_projects = itertools.chain.from_iterable(
      projects_by_user_effective_id.values())

  # Get effective ids for other user to handle transitive Project membership.
  other_user_effective_ids = GetEffectiveIds(cnxn, services, other_user_id)
  projects_by_other_user_effective_ids = services.project.GetProjectMemberships(
      cnxn, other_user_effective_ids)
  other_user_projects = itertools.chain.from_iterable(
      projects_by_other_user_effective_ids.values())

  return any(project in authed_user_projects for project in other_user_projects)


# TODO(https://crbug.com/monorail/8192): Remove this method.
def DeprecatedShouldRevealEmail(user_auth, project, viewed_email):
  # type: (AuthData, Project, str) -> bool
  """
  Deprecated V1 API logic to decide whether to publish a user's email
  address. Avoid updating this method.

  Args:
    user_auth: The AuthData of the user viewing the email addresses.
    project: The Project PB to which the viewed user belongs.
    viewed_email: The email of the viewed user.

  Returns:
    True if email addresses should be published to the logged-in user.
  """
  # Case 1: Anon users don't see anything revealed.
  if user_auth.user_pb is None:
    return False

  # Case 2: site admins always see unobscured email addresses.
  if user_auth.user_pb.is_site_admin:
    return True

  # Case 3: Project members see the unobscured email of everyone in a project.
  if project and UserIsInProject(project, user_auth.effective_ids):
    return True

  # Case 4: Do not obscure your own email.
  if viewed_email and user_auth.user_pb.email == viewed_email:
    return True

  return False


def ParseAndObscureAddress(email):
  # type: str -> str
  """Break the given email into username and domain, and obscure.

  Args:
    email: string email address to process

  Returns:
    A 4-tuple (username, domain, obscured_username, obscured_email).
    The obscured_username is truncated more aggressively than how Google Groups
    does it: it truncates at 5 characters or truncates OFF 3 characters,
    whichever results in a shorter obscured_username.
  """
  if '@' in email:
    username, user_domain = email.split('@', 1)
  else:  # don't fail if User table has unexpected email address format.
    username, user_domain = email, ''

  base_username = username.split('+')[0]
  cutoff_point = min(5, max(1, len(base_username) - 3))
  obscured_username = base_username[:cutoff_point]
  obscured_email = '%s...@%s' %(obscured_username, user_domain)

  return username, user_domain, obscured_username, obscured_email


def CreateUserDisplayNamesAndEmails(cnxn, services, user_auth, users):
  # type: (MonorailConnection, Services, AuthData,
  #     Collection[user_pb2.User]) ->
  #     Tuple[Mapping[int, str], Mapping[int, str]]
  """Create the display names and emails of the given users based on the
     current user.

  Args:
    cnxn: MonorailConnection to the database.
    services: Services object for connections to backend services.
    user_auth: AuthData object that identifies the logged in user.
    users: Collection of User PB objects.

  Returns:
    A Tuple containing two Dicts of user_ids to display names and user_ids to
        emails. If a given User does not have an email, there will be an empty
        string in both.
  """
  # NOTE: Currently only service accounts can have display_names set. For all
  # other users and service accounts with no display_names specified, we use the
  # obscured or unobscured emails for both `display_names` and `emails`.
  # See crbug.com/monorail/8510.
  display_names = {}
  emails = {}

  # Do a pass on simple display cases.
  maybe_revealed_users = []
  for user in users:
    if user.user_id == framework_constants.DELETED_USER_ID:
      display_names[user.user_id] = framework_constants.DELETED_USER_NAME
      emails[user.user_id] = ''
    elif not user.email:
      display_names[user.user_id] = ''
      emails[user.user_id] = ''
    elif not user.obscure_email:
      display_names[user.user_id] = user.email
      emails[user.user_id] = user.email
    else:
      # Default to hiding user email.
      (_username, _domain, _obs_username,
       obs_email) = ParseAndObscureAddress(user.email)
      display_names[user.user_id] = obs_email
      emails[user.user_id] = obs_email
      maybe_revealed_users.append(user)

  # Reveal viewable emails.
  viewable_users = FilterViewableEmails(
      cnxn, services, user_auth, maybe_revealed_users)
  for user in viewable_users:
    display_names[user.user_id] = user.email
    emails[user.user_id] = user.email

  # Use Client.display_names for service accounts that have one specified.
  for user in users:
    if user.email in client_config_svc.GetServiceAccountMap():
      display_names[user.user_id] = client_config_svc.GetServiceAccountMap()[
          user.email]

  return display_names, emails


def UserOwnsProject(project, effective_ids):
  """Return True if any of the effective_ids is a project owner."""
  return not effective_ids.isdisjoint(project.owner_ids or set())


def UserIsInProject(project, effective_ids):
  """Return True if any of the effective_ids is a project member.

  Args:
    project: Project PB for the current project.
    effective_ids: set of int user IDs for the current user (including all
        user groups).  This will be an empty set for anonymous users.

  Returns:
    True if the user has any direct or indirect role in the project.  The value
    will actually be a set(), but it will have an ID in it if the user is in
    the project, or it will be an empty set which is considered False.
  """
  return (UserOwnsProject(project, effective_ids) or
          not effective_ids.isdisjoint(project.committer_ids or set()) or
          not effective_ids.isdisjoint(project.contributor_ids or set()))


def IsPriviledgedDomainUser(email):
  """Return True if the user's account is from a priviledged domain."""
  if email and '@' in email:
    _, user_domain = email.split('@', 1)
    return user_domain in settings.priviledged_user_domains

  return False


def IsValidColumnSpec(col_spec):
  # type: str -> bool
  """Return true if the given column specification is valid."""
  return re.match(RE_COLUMN_SPEC, col_spec)


# String translation table to catch a common typos in label names.
_CANONICALIZATION_TRANSLATION_TABLE = {
    ord(delete_u_char): None
    for delete_u_char in u'!"#$%&\'()*+,/:;<>?@[\\]^`{|}~\t\n\x0b\x0c\r '
    }
_CANONICALIZATION_TRANSLATION_TABLE.update({ord(u'='): ord(u'-')})


def CanonicalizeLabel(user_input):
  """Canonicalize a given label or status value.

  When the user enters a string that represents a label or an enum,
  convert it a canonical form that makes it more likely to match
  existing values.

  Args:
    user_input: string that the user typed for a label.

  Returns:
    Canonical form of that label as a unicode string.
  """
  if user_input is None:
    return user_input

  if not isinstance(user_input, six.text_type):
    user_input = user_input.decode('utf-8')

  canon_str = user_input.translate(_CANONICALIZATION_TRANSLATION_TABLE)
  return canon_str


def MergeLabels(labels_list, labels_add, labels_remove, config):
  """Update a list of labels with the given add and remove label lists.

  Args:
    labels_list: list of current labels.
    labels_add: labels that the user wants to add.
    labels_remove: labels that the user wants to remove.
    config: ProjectIssueConfig with info about exclusive prefixes and
        enum fields.

  Returns:
    (merged_labels, update_labels_add, update_labels_remove):
    A new list of labels with the given labels added and removed, and
    any exclusive label prefixes taken into account.  Then two
    lists of update strings to explain the changes that were actually
    made.
  """
  old_lower_labels = [lab.lower() for lab in labels_list]
  labels_add = [lab for lab in labels_add
                if lab.lower() not in old_lower_labels]
  labels_remove = [lab for lab in labels_remove
                   if lab.lower() in old_lower_labels]
  labels_remove_lower = [lab.lower() for lab in labels_remove]
  exclusive_prefixes = [
      lab.lower() + '-' for lab in config.exclusive_label_prefixes]
  for fd in config.field_defs:
    if (fd.field_type == tracker_pb2.FieldTypes.ENUM_TYPE and
        not fd.is_multivalued):
      exclusive_prefixes.append(fd.field_name.lower() + '-')

  # We match prefix strings rather than splitting on dash because
  # an exclusive-prefix or field name may contain dashes.
  def MatchPrefix(lab, prefixes):
    for prefix_dash in prefixes:
      if lab.lower().startswith(prefix_dash):
        return prefix_dash
    return False

  # Dedup any added labels.  E.g., ignore attempts to add Priority twice.
  excl_add = []
  dedupped_labels_add = []
  for lab in labels_add:
    matched_prefix_dash = MatchPrefix(lab, exclusive_prefixes)
    if matched_prefix_dash:
      if matched_prefix_dash not in excl_add:
        excl_add.append(matched_prefix_dash)
        dedupped_labels_add.append(lab)
    else:
      dedupped_labels_add.append(lab)

  # "Old minus exclusive" is the set of old label values minus any
  # that should be overwritten by newly set exclusive labels.
  old_minus_excl = []
  for lab in labels_list:
    matched_prefix_dash = MatchPrefix(lab, excl_add)
    if not matched_prefix_dash:
      old_minus_excl.append(lab)

  merged_labels = [lab for lab in old_minus_excl + dedupped_labels_add
                   if lab.lower() not in labels_remove_lower]

  return merged_labels, dedupped_labels_add, labels_remove


# Pattern to match a valid hotlist name.
RE_HOTLIST_NAME_PATTERN = r"[a-zA-Z][-0-9a-zA-Z\.]*"

# Compiled regexp to match the hotlist name and nothing more before or after.
RE_HOTLIST_NAME = re.compile(
    '^%s$' % RE_HOTLIST_NAME_PATTERN, re.VERBOSE)


def IsValidHotlistName(s):
  """Return true if the given string is a valid hotlist name."""
  return (RE_HOTLIST_NAME.match(s) and
          len(s) <= framework_constants.MAX_HOTLIST_NAME_LENGTH)


USER_PREF_DEFS = {
  'code_font': re.compile('(true|false)'),
  'render_markdown': re.compile('(true|false)'),

  # The are for dismissible cues.  True means the user has dismissed them.
  'privacy_click_through': re.compile('(true|false)'),
  'corp_mode_click_through': re.compile('(true|false)'),
  'code_of_conduct': re.compile('(true|false)'),
  'dit_keystrokes': re.compile('(true|false)'),
  'italics_mean_derived': re.compile('(true|false)'),
  'availability_msgs': re.compile('(true|false)'),
  'your_email_bounced': re.compile('(true|false)'),
  'search_for_numbers': re.compile('(true|false)'),
  'restrict_new_issues': re.compile('(true|false)'),
  'public_issue_notice': re.compile('(true|false)'),
  'you_are_on_vacation': re.compile('(true|false)'),
  'how_to_join_project': re.compile('(true|false)'),
  'document_team_duties': re.compile('(true|false)'),
  'showing_ids_instead_of_tiles': re.compile('(true|false)'),
  'issue_timestamps': re.compile('(true|false)'),
  'stale_fulltext': re.compile('(true|false)'),
  }
MAX_PREF_VALUE_LENGTH = 80


def ValidatePref(name, value):
  """Return an error message if the server does not support a pref value."""
  if name not in USER_PREF_DEFS:
    return 'Unknown pref name: %r' % name
  if len(value) > MAX_PREF_VALUE_LENGTH:
    return 'Value for pref name %r is too long' % name
  if not USER_PREF_DEFS[name].match(value):
    return 'Invalid pref value %r for %r' % (value, name)
  return None


def IsRestrictNewIssuesUser(cnxn, services, user_id):
  # type: (MonorailConnection, Services, int) -> bool)
  """Returns true iff user's new issues should be restricted by default."""
  user_group_ids = services.usergroup.LookupMemberships(cnxn, user_id)
  restrict_new_issues_groups_dict = services.user.LookupUserIDs(
      cnxn, settings.restrict_new_issues_user_groups, autocreate=True)
  restrict_new_issues_group_ids = set(restrict_new_issues_groups_dict.values())
  return any(gid in restrict_new_issues_group_ids for gid in user_group_ids)


def IsPublicIssueNoticeUser(cnxn, services, user_id):
  # type: (MonorailConnection, Services, int) -> bool)
  """Returns true iff user should see a public issue notice by default."""
  user_group_ids = services.usergroup.LookupMemberships(cnxn, user_id)
  public_issue_notice_groups_dict = services.user.LookupUserIDs(
      cnxn, settings.public_issue_notice_user_groups, autocreate=True)
  public_issue_notice_group_ids = set(public_issue_notice_groups_dict.values())
  return any(gid in public_issue_notice_group_ids for gid in user_group_ids)


def GetEffectiveIds(cnxn, services, user_ids):
  # type: (MonorailConnection, Services, Collection[int]) ->
  #   Mapping[int, Collection[int]]
  """
  Given a set of user IDs, it returns a mapping of user_id to a set of effective
  IDs that include the user's ID and all of their user groups. This mapping
  will be contain only the user_id anonymous users.
  """
  # Get direct memberships for user_ids.
  effective_ids_by_user_id = services.usergroup.LookupAllMemberships(
      cnxn, user_ids)
  # Add user_id to list of effective_ids.
  for user_id, effective_ids in effective_ids_by_user_id.items():
    effective_ids.add(user_id)
  # Get User objects for user_ids.
  users_by_id = services.user.GetUsersByIDs(cnxn, user_ids)
  for user_id, user in users_by_id.items():
    if user and user.email:
      effective_ids_by_user_id[user_id].update(
          _ComputeMembershipsByEmail(cnxn, services, user.email))

      # Add related parent and child ids.
      related_ids = []
      if user.linked_parent_id:
        related_ids.append(user.linked_parent_id)
      if user.linked_child_ids:
        related_ids.extend(user.linked_child_ids)

      # Add any related efective_ids.
      if related_ids:
        effective_ids_by_user_id[user_id].update(related_ids)
        effective_ids_by_related_id = services.usergroup.LookupAllMemberships(
            cnxn, related_ids)
        related_effective_ids = functools.reduce(
            set.union, effective_ids_by_related_id.values(), set())
        effective_ids_by_user_id[user_id].update(related_effective_ids)
  return effective_ids_by_user_id


def _ComputeMembershipsByEmail(cnxn, services, email):
  # type: (MonorailConnection, Services, str) -> Collection[int]
  """
  Given an user email, it returns a list [group_id] of computed user groups.
  """
  # Get the user email domain to compute memberships of the user.
  (_username, user_email_domain, _obs_username,
   _obs_email) = ParseAndObscureAddress(email)
  return services.usergroup.LookupComputedMemberships(cnxn, user_email_domain)
