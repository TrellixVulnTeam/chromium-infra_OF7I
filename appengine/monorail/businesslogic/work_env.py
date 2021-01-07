# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""WorkEnv is a context manager and API for high-level operations.

A work environment is used by request handlers for the legacy UI, v1
API, and v2 API.  The WorkEnvironment operations are a common code
path that does permission checking, input validation, coordination of
service-level calls, follow-up tasks (e.g., triggering
notifications after certain operations) and other systemic
functionality so that that code is not duplicated in multiple request
handlers.

Responsibilities of request handers (legacy UI and external API) and associated
frameworks:
+ API: check oauth client allowlist or XSRF token
+ Rate-limiting
+ Create a MonorailContext (or MonorailRequest) object:
  - Parse the request, including syntactic validation, e.g, non-negative ints
  - Authenticate the requesting user
+ Call the WorkEnvironment to perform the requested action
  - Catch exceptions and generate error messages
+ UI: Decide screen flow, and on-page online-help
+ Render the result business objects as UI HTML or API response protobufs

Responsibilities of WorkEnv:
+ Most monitoring, profiling, and logging
+ Apply business rules:
  - Check permissions
    - Every GetFoo/GetFoosDict method will assert that the user can view Foo(s)
  - Detailed validation of request parameters
  - Raise exceptions to indicate problems
+ Make coordinated calls to the services layer to make DB changes
  - E.g., calls may need to be made in a specific order
+ Enqueue tasks for background follow-up work:
  - E.g., email notifications

Responsibilities of the Services layer:
+ Individual CRUD operations on objects in the database
  - Each services class should be independent of others
+ App-specific interface around external services:
  - E.g., GAE search, GCS, monorail-predict
+ Business object caches
+ Breaking large operations into batches as appropriate for the underlying
  data storage service, e.g., DB shards and search engine indexing.
"""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import collections
import itertools
import logging
import time

import settings
from features import features_constants
from features import filterrules_helpers
from features import send_notifications
from features import features_bizobj
from features import hotlist_helpers
from framework import authdata
from framework import exceptions
from framework import framework_bizobj
from framework import framework_constants
from framework import framework_helpers
from framework import framework_views
from framework import permissions
from search import frontendsearchpipeline
from services import features_svc
from services import tracker_fulltext
from sitewide import sitewide_helpers
from tracker import field_helpers
from tracker import rerank_helpers
from tracker import field_helpers
from tracker import tracker_bizobj
from tracker import tracker_constants
from tracker import tracker_helpers
from project import project_helpers
from proto import features_pb2
from proto import project_pb2
from proto import tracker_pb2
from proto import user_pb2


# TODO(jrobbins): break this file into one facade plus ~5
# implementation parts that roughly correspond to services files.

# ListResult is returned in List/Search methods to bundle the requested
# items and the next start index for a subsequent request. If there are
# no more items to be fetched, `next_start` should be None.
ListResult = collections.namedtuple('ListResult', ['items', 'next_start'])
# type: (Sequence[Object], Optional[int]) -> None

# AttachmentUpload holds the information of an incoming uploaded
# attachment before it gets saved as a gcs file and saved to the DB.
AttachmentUpload = collections.namedtuple(
    'AttachmentUpload', ['filename', 'contents', 'mimetype'])
# type: (str, str, str) -> None

# Comments added to issues impacted by another issue's mergedInto change.
UNMERGE_COMMENT = 'Issue %s has been un-merged from this issue.\n'
MERGE_COMMENT = 'Issue %s has been merged into this issue.\n'


class WorkEnv(object):

  def __init__(self, mc, services, phase=None):
    self.mc = mc
    self.services = services
    self.phase = phase

  def __enter__(self):
    if self.mc.profiler and self.phase:
      self.mc.profiler.StartPhase(name=self.phase)
    return self  # The instance of this class is the context object.

  def __exit__(self, exception_type, value, traceback):
    if self.mc.profiler and self.phase:
      self.mc.profiler.EndPhase()
    return False  # Re-raise any exception in the with-block.

  def _UserCanViewProject(self, project):
    """Test if the user may view the given project."""
    return permissions.UserCanViewProject(
        self.mc.auth.user_pb, self.mc.auth.effective_ids, project)

  def _FilterVisibleProjectsDict(self, projects):
    """Filter out projects the user doesn't have permission to view."""
    return {
        key: proj
        for key, proj in projects.items()
        if self._UserCanViewProject(proj)}

  def _AssertPermInProject(self, perm, project):
    """Make sure the user may use perm in the given project."""
    project_perms = permissions.GetPermissions(
        self.mc.auth.user_pb, self.mc.auth.effective_ids, project)
    permitted = project_perms.CanUsePerm(
        perm, self.mc.auth.effective_ids, project, [])
    if not permitted:
      raise permissions.PermissionException(
        'User lacks permission %r in project %s' % (perm, project.project_name))

  def _UserCanViewIssue(self, issue, allow_viewing_deleted=False):
    """Test if user may view an issue according to perms in issue's project."""
    project = self.GetProject(issue.project_id)
    config = self.GetProjectConfig(issue.project_id)
    granted_perms = tracker_bizobj.GetGrantedPerms(
        issue, self.mc.auth.effective_ids, config)
    project_perms = permissions.GetPermissions(
        self.mc.auth.user_pb, self.mc.auth.effective_ids, project)
    issue_perms = permissions.UpdateIssuePermissions(
        project_perms, project, issue, self.mc.auth.effective_ids,
        granted_perms=granted_perms)
    permit_view = permissions.CanViewIssue(
        self.mc.auth.effective_ids, issue_perms, project, issue,
        allow_viewing_deleted=allow_viewing_deleted,
        granted_perms=granted_perms)
    return issue_perms, permit_view

  def _AssertUserCanViewIssue(self, issue, allow_viewing_deleted=False):
    """Make sure the user may view the issue."""
    issue_perms, permit_view = self._UserCanViewIssue(
        issue, allow_viewing_deleted)
    if not permit_view:
      raise permissions.PermissionException(
          'User is not allowed to view issue: %s:%d.' %
          (issue.project_name, issue.local_id))
    return issue_perms

  def _UserCanUsePermInIssue(self, issue, perm):
    """Test if the user may use perm on the given issue."""
    issue_perms = self._AssertUserCanViewIssue(
        issue, allow_viewing_deleted=True)
    return issue_perms.HasPerm(perm, None, None, [])

  def _AssertPermInIssue(self, issue, perm):
    """Make sure the user may use perm on the given issue."""
    permitted = self._UserCanUsePermInIssue(issue, perm)
    if not permitted:
      raise permissions.PermissionException(
        'User lacks permission %r in issue' % perm)

  def _AssertUserCanModifyIssues(
      self, issue_delta_pairs, is_description_change, comment_content=None):
    # type: (Tuple[Issue, IssueDelta], Boolean, Optional[str]) -> None
    """Make sure the user may make the delta changes for each paired issue."""
    # We assume that view permission for each issue, and therefore project,
    # was checked by the caller.
    project_ids = list(
        {issue.project_id for (issue, _delta) in issue_delta_pairs})
    projects_by_id = self.services.project.GetProjects(
        self.mc.cnxn, project_ids)
    configs_by_id = self.services.config.GetProjectConfigs(
        self.mc.cnxn, project_ids)

    project_perms_by_ids = {}
    for project_id, project in projects_by_id.items():
      project_perms_by_ids[project_id] = permissions.GetPermissions(
          self.mc.auth.user_pb, self.mc.auth.effective_ids, project)

    with exceptions.ErrorAggregator(permissions.PermissionException) as err_agg:
      for issue, delta in issue_delta_pairs:
        project_perms = project_perms_by_ids.get(issue.project_id)
        config = configs_by_id.get(issue.project_id)
        project = projects_by_id.get(issue.project_id)
        granted_perms = tracker_bizobj.GetGrantedPerms(
            issue, self.mc.auth.effective_ids, config)
        issue_perms = permissions.UpdateIssuePermissions(
            project_perms,
            project,
            issue,
            self.mc.auth.effective_ids,
            granted_perms=granted_perms)

        # User cannot merge any issue into an issue they cannot edit.
        if delta.merged_into:
          merged_into_issue = self.GetIssue(
              delta.merged_into, use_cache=False, allow_viewing_deleted=True)
          self._AssertPermInIssue(merged_into_issue, permissions.EDIT_ISSUE)

        # User cannot change values for restricted fields they cannot edit.
        field_ids = [fv.field_id for fv in delta.field_vals_add]
        field_ids.extend([fv.field_id for fv in delta.field_vals_remove])
        field_ids.extend(delta.fields_clear)
        labels = itertools.chain(delta.labels_add, delta.labels_remove)
        try:
          self._AssertUserCanEditFieldsAndEnumMaskedLabels(
              project, config, field_ids, labels)
        except permissions.PermissionException as e:
          err_agg.AddErrorMessage(e.message)

        if issue_perms.HasPerm(permissions.EDIT_ISSUE, self.mc.auth.user_id,
                               project):
          continue

        # The user does not have general EDIT_ISSUE permissions, but may
        # have perms to modify certain issue parts/fields.

        # Description changes can only be made by users with EDIT_ISSUE.
        if is_description_change:
          err_agg.AddErrorMessage(
              'User not allowed to edit description in issue %s:%d' %
              (issue.project_name, issue.local_id))

        if comment_content and not issue_perms.HasPerm(
            permissions.ADD_ISSUE_COMMENT, self.mc.auth.user_id, project):
          err_agg.AddErrorMessage(
              'User not allowed to add comment in issue %s:%d' %
              (issue.project_name, issue.local_id))

        if delta == tracker_pb2.IssueDelta():
          continue

        allowed_delta = tracker_pb2.IssueDelta()
        if issue_perms.HasPerm(permissions.EDIT_ISSUE_STATUS,
                               self.mc.auth.user_id, project):
          allowed_delta.status = delta.status
        if issue_perms.HasPerm(permissions.EDIT_ISSUE_SUMMARY,
                               self.mc.auth.user_id, project):
          allowed_delta.summary = delta.summary
        if issue_perms.HasPerm(permissions.EDIT_ISSUE_OWNER,
                               self.mc.auth.user_id, project):
          allowed_delta.owner_id = delta.owner_id
        if issue_perms.HasPerm(permissions.EDIT_ISSUE_CC, self.mc.auth.user_id,
                               project):
          allowed_delta.cc_ids_add = delta.cc_ids_add
          allowed_delta.cc_ids_remove = delta.cc_ids_remove
        # We do not check for or add other fields (e.g. comps, labels, fields)
        # of `delta` to `allowed_delta` because they are only allowed
        # with EDIT_ISSUE perms.
        if delta != allowed_delta:
          err_agg.AddErrorMessage(
              'User lack permission to make these changes to issue %s:%d' %
              (issue.project_name, issue.local_id))

  # end of `with` block.

  def _AssertUserCanDeleteComment(self, issue, comment):
    issue_perms = self._AssertUserCanViewIssue(
       issue, allow_viewing_deleted=True)
    commenter = self.services.user.GetUser(self.mc.cnxn, comment.user_id)
    permitted = permissions.CanDeleteComment(
        comment, commenter, self.mc.auth.user_id, issue_perms)
    if not permitted:
      raise permissions.PermissionException('Cannot delete comment')

  def _AssertUserCanViewHotlist(self, hotlist):
    """Make sure the user may view the hotlist."""
    if not permissions.CanViewHotlist(
        self.mc.auth.effective_ids, self.mc.perms, hotlist):
      raise permissions.PermissionException(
          'User is not allowed to view this hotlist')

  def _AssertUserCanEditHotlist(self, hotlist):
    if not permissions.CanEditHotlist(
        self.mc.auth.effective_ids, self.mc.perms, hotlist):
      raise permissions.PermissionException(
          'User is not allowed to edit this hotlist')

  def _AssertUserCanEditValueForFieldDef(self, project, fielddef):
    if not permissions.CanEditValueForFieldDef(
        self.mc.auth.effective_ids, self.mc.perms, project, fielddef):
      raise permissions.PermissionException(
          'User is not allowed to edit custom field %s' % fielddef.field_name)

  def _AssertUserCanEditFieldsAndEnumMaskedLabels(
      self, project, config, field_ids, labels):
    field_ids = set(field_ids)

    enum_fds_by_name = {
        f.field_name.lower(): f.field_id
        for f in config.field_defs
        if f.field_type is tracker_pb2.FieldTypes.ENUM_TYPE and not f.is_deleted
    }
    for label in labels:
      enum_field_name = tracker_bizobj.LabelIsMaskedByField(
          label, enum_fds_by_name.keys())
      if enum_field_name:
        field_ids.add(enum_fds_by_name.get(enum_field_name))

    fds_by_id = {fd.field_id: fd for fd in config.field_defs}
    with exceptions.ErrorAggregator(permissions.PermissionException) as err_agg:
      for field_id in field_ids:
        fd = fds_by_id.get(field_id)
        if fd:
          try:
            self._AssertUserCanEditValueForFieldDef(project, fd)
          except permissions.PermissionException as e:
            err_agg.AddErrorMessage(e.message)

  def _AssertUserCanViewFieldDef(self, project, field):
    """Make sure the user may view the field."""
    if not permissions.CanViewFieldDef(self.mc.auth.effective_ids,
                                       self.mc.perms, project, field):
      raise permissions.PermissionException(
          'User is not allowed to view this field')

  ### Site methods

  # FUTURE: GetSiteReadOnlyState()
  # FUTURE: SetSiteReadOnlyState()
  # FUTURE: GetSiteBannerMessage()
  # FUTURE: SetSiteBannerMessage()

  ### Project methods

  def CreateProject(
      self, project_name, owner_ids, committer_ids, contributor_ids,
      summary, description, state=project_pb2.ProjectState.LIVE,
      access=None, read_only_reason=None, home_page=None, docs_url=None,
      source_url=None, logo_gcs_id=None, logo_file_name=None):
    """Create and store a Project with the given attributes.

    Args:
      cnxn: connection to SQL database.
      project_name: a valid project name, all lower case.
      owner_ids: a list of user IDs for the project owners.
      committer_ids: a list of user IDs for the project members.
      contributor_ids: a list of user IDs for the project contributors.
      summary: one-line explanation of the project.
      description: one-page explanation of the project.
      state: a project state enum defined in project_pb2.
      access: optional project access enum defined in project.proto.
      read_only_reason: if given, provides a status message and marks
        the project as read-only.
      home_page: home page of the project
      docs_url: url to redirect to for wiki/documentation links
      source_url: url to redirect to for source browser links
      logo_gcs_id: google storage object id of the project's logo
      logo_file_name: uploaded file name of the project's logo

    Returns:
      The int project_id of the new project.

    Raises:
      ProjectAlreadyExists: A project with that name already exists.
    """
    if not permissions.CanCreateProject(self.mc.perms):
      raise permissions.PermissionException(
          'User is not allowed to create a project')

    with self.mc.profiler.Phase('creating project %r' % project_name):
      project_id = self.services.project.CreateProject(
          self.mc.cnxn, project_name, owner_ids, committer_ids, contributor_ids,
          summary, description, state=state, access=access,
          read_only_reason=read_only_reason, home_page=home_page,
          docs_url=docs_url, source_url=source_url, logo_gcs_id=logo_gcs_id,
          logo_file_name=logo_file_name)
      self.services.template.CreateDefaultProjectTemplates(self.mc.cnxn,
          project_id)
    return project_id

  def ListProjects(self, domain=None, use_cache=True):
    """Return a list of project IDs that the current user may view."""
    # TODO(crbug.com/monorail/7508): Add permission checking in ListProjects.
    # Note: No permission checks because anyone can list projects, but
    # the results are filtered by permission to view each project.

    with self.mc.profiler.Phase('list projects for %r' % self.mc.auth.user_id):
      project_ids = self.services.project.GetVisibleLiveProjects(
          self.mc.cnxn, self.mc.auth.user_pb, self.mc.auth.effective_ids,
          domain=domain, use_cache=use_cache)

    return project_ids

  def CheckProjectName(self, project_name):
    """Check that a project name is valid and not already in use.

    Args:
      project_name: str the project name to check.

    Returns:
      None if the user can create a project with that name, or a string with the
      reason the name can't be used.

    Raises:
      PermissionException: The user is not allowed to create a project.
    """
    # We check that the user can create a project so we don't leak information
    # about project names.
    if not permissions.CanCreateProject(self.mc.perms):
      raise permissions.PermissionException(
          'User is not allowed to create a project')

    with self.mc.profiler.Phase('checking project name %s' % project_name):
      if not project_helpers.IsValidProjectName(project_name):
        return '"%s" is not a valid project name.' % project_name
      if self.services.project.LookupProjectIDs(self.mc.cnxn, [project_name]):
        return 'There is already a project with that name.'
    return None

  def CheckComponentName(self, project_id, parent_path, component_name):
    """Check that the component name is valid and not already in use.

    Args:
      project_id: int with the id of the project where we want to create the
          component.
      parent_path: optional str with the path of the parent component.
      component_name: str with the name of the proposed component.

    Returns:
      None if the user can create a component with that name, or a string with
      the reason the name can't be used.
    """
    # Check that the project exists and the user can view it.
    self.GetProject(project_id)
    # If a parent component is given, make sure it exists.
    config = self.GetProjectConfig(project_id)
    if parent_path and not tracker_bizobj.FindComponentDef(parent_path, config):
      raise exceptions.NoSuchComponentException(
          'Component %r not found' % parent_path)
    with self.mc.profiler.Phase(
        'checking component name %r %r' % (parent_path, component_name)):
      if not tracker_constants.COMPONENT_NAME_RE.match(component_name):
        return '"%s" is not a valid component name.' % component_name
      if parent_path:
        component_name = '%s>%s' % (parent_path, component_name)
      if tracker_bizobj.FindComponentDef(component_name, config):
        return 'There is already a component with that name.'
    return None

  def CheckFieldName(self, project_id, field_name):
    """Check that the field name is valid and not already in use.

    Args:
      project_id: int with the id of the project where we want to create the
          field.
      field_name: str with the name of the proposed field.

    Returns:
      None if the user can create a field with that name, or a string with
      the reason the name can't be used.
    """
    # Check that the project exists and the user can view it.
    self.GetProject(project_id)
    config = self.GetProjectConfig(project_id)

    field_name = field_name.lower()
    with self.mc.profiler.Phase('checking field name %r' % field_name):
      if not tracker_constants.FIELD_NAME_RE.match(field_name):
        return '"%s" is not a valid field name.' % field_name
      if field_name in tracker_constants.RESERVED_PREFIXES:
        return 'That name is reserved'
      if field_name.endswith(
          tuple(tracker_constants.RESERVED_COL_NAME_SUFFIXES)):
        return 'That suffix is reserved'
      for fd in config.field_defs:
        fn = fd.field_name.lower()
        if field_name == fn:
          return 'There is already a field with that name.'
        if field_name.startswith(fn + '-'):
          return 'An existing field is a prefix of that name.'
        if fn.startswith(field_name + '-'):
          return 'That name is a prefix of an existing field name.'

    return None

  def GetProjects(self, project_ids, use_cache=True):
    """Return the specified projects.

    Args:
      project_ids: int project_ids of the projects to retrieve.
      use_cache: set to false when doing read-modify-write.

    Returns:
      The specified projects.

    Raises:
      NoSuchProjectException: There is no project with that ID.
    """
    with self.mc.profiler.Phase('getting projects %r' % project_ids):
      projects = self.services.project.GetProjects(
          self.mc.cnxn, project_ids, use_cache=use_cache)

    projects = self._FilterVisibleProjectsDict(projects)
    return projects

  def GetProject(self, project_id, use_cache=True):
    """Return the specified project.

    Args:
      project_id: int project_id of the project to retrieve.
      use_cache: set to false when doing read-modify-write.

    Returns:
      The specified project.

    Raises:
      NoSuchProjectException: There is no project with that ID.
    """
    projects = self.GetProjects([project_id], use_cache=use_cache)
    if project_id not in projects:
      raise permissions.PermissionException(
          'User is not allowed to view this project')
    return projects[project_id]

  def GetProjectsByName(self, project_names, use_cache=True):
    """Return the named project.

    Args:
      project_names: string names of the projects to retrieve.
      use_cache: set to false when doing read-modify-write.

    Returns:
      The specified projects.
    """
    with self.mc.profiler.Phase('getting projects %r' % project_names):
      projects = self.services.project.GetProjectsByName(
          self.mc.cnxn, project_names, use_cache=use_cache)

    for pn in project_names:
      if pn not in projects:
        raise exceptions.NoSuchProjectException('Project %r not found.' % pn)

    projects = self._FilterVisibleProjectsDict(projects)
    return projects

  def GetProjectByName(self, project_name, use_cache=True):
    """Return the named project.

    Args:
      project_name: string name of the project to retrieve.
      use_cache: set to false when doing read-modify-write.

    Returns:
      The specified project.

    Raises:
      NoSuchProjectException: There is no project with that name.
    """
    projects = self.GetProjectsByName([project_name], use_cache)
    if not projects:
      raise permissions.PermissionException(
          'User is not allowed to view this project')

    return projects[project_name]

  def GatherProjectMembershipsForUser(self, user_id):
    """Return the projects where the user has a role.

    Args:
      user_id: ID of the user we are requesting project memberships for.

    Returns:
      A triple with project IDs where the user is an owner, a committer, or a
      contributor.
    """
    viewed_user_effective_ids = authdata.AuthData.FromUserID(
        self.mc.cnxn, user_id, self.services).effective_ids

    owner_projects, _archived, committer_projects, contrib_projects = (
        self.GetUserProjects(viewed_user_effective_ids))

    owner_proj_ids = [proj.project_id for proj in owner_projects]
    committer_proj_ids = [proj.project_id for proj in committer_projects]
    contrib_proj_ids = [proj.project_id for proj in contrib_projects]
    return owner_proj_ids, committer_proj_ids, contrib_proj_ids

  def GetUserRolesInAllProjects(self, viewed_user_effective_ids):
    """Return the projects where the user has a role.

    Args:
      viewed_user_effective_ids: list of IDs of the user whose projects we want
          to see.

    Returns:
      A triple with projects where the user is an owner, a member or a
      contributor.
    """
    with self.mc.profiler.Phase(
        'Finding roles in all projects for %r' % viewed_user_effective_ids):
      project_ids = self.services.project.GetUserRolesInAllProjects(
          self.mc.cnxn, viewed_user_effective_ids)

    owner_projects = self.GetProjects(project_ids[0])
    member_projects = self.GetProjects(project_ids[1])
    contrib_projects = self.GetProjects(project_ids[2])

    return owner_projects, member_projects, contrib_projects

  def GetUserProjects(self, viewed_user_effective_ids):
    # TODO(crbug.com/monorail/7398): Combine this function with
    # GatherProjectMembershipsForUser after removing the legacy
    # project list page and the v0 GetUsersProjects RPC.
    """Get the projects to display in the user's profile.

    Args:
      viewed_user_effective_ids: set of int user IDs of the user being viewed.

    Returns:
      A 4-tuple of lists of PBs:
        - live projects the viewed user owns
        - archived projects the viewed user owns
        - live projects the viewed user is a member of
        - live projects the viewed user is a contributor to

      Any projects the viewing user should not be able to see are filtered out.
      Admins can see everything, while other users can see all non-locked
      projects they own or are a member of, as well as all live projects.
    """
    # Permissions are checked in we.GetUserRolesInAllProjects()
    owner_projects, member_projects, contrib_projects = (
        self.GetUserRolesInAllProjects(viewed_user_effective_ids))

    # We filter out DELETABLE projects, and keep a project where the user has a
    # highest role, e.g. if the user is both an owner and a member, the project
    # is listed under owner projects, not under member_projects.
    archived_projects = [
        project
        for project in owner_projects.values()
        if project.state == project_pb2.ProjectState.ARCHIVED]

    contrib_projects = [
        project
        for pid, project in contrib_projects.items()
        if pid not in owner_projects
        and pid not in member_projects
        and project.state != project_pb2.ProjectState.DELETABLE
        and project.state != project_pb2.ProjectState.ARCHIVED]

    member_projects = [
        project
        for pid, project in member_projects.items()
        if pid not in owner_projects
        and project.state != project_pb2.ProjectState.DELETABLE
        and project.state != project_pb2.ProjectState.ARCHIVED]

    owner_projects = [
        project
        for pid, project in owner_projects.items()
        if project.state != project_pb2.ProjectState.DELETABLE
        and project.state != project_pb2.ProjectState.ARCHIVED]

    by_name = lambda project: project.project_name
    owner_projects = sorted(owner_projects, key=by_name)
    archived_projects = sorted(archived_projects, key=by_name)
    member_projects = sorted(member_projects, key=by_name)
    contrib_projects = sorted(contrib_projects, key=by_name)

    return owner_projects, archived_projects, member_projects, contrib_projects

  def UpdateProject(
      self,
      project_id,
      summary=None,
      description=None,
      state=None,
      state_reason=None,
      access=None,
      issue_notify_address=None,
      attachment_bytes_used=None,
      attachment_quota=None,
      moved_to=None,
      process_inbound_email=None,
      only_owners_remove_restrictions=None,
      read_only_reason=None,
      cached_content_timestamp=None,
      only_owners_see_contributors=None,
      delete_time=None,
      recent_activity=None,
      revision_url_format=None,
      home_page=None,
      docs_url=None,
      source_url=None,
      logo_gcs_id=None,
      logo_file_name=None,
      issue_notify_always_detailed=None):
    """Update the DB with the given project information."""
    project = self.GetProject(project_id)
    self._AssertPermInProject(permissions.EDIT_PROJECT, project)

    with self.mc.profiler.Phase('updating project %r' % project_id):
      self.services.project.UpdateProject(
          self.mc.cnxn,
          project_id,
          summary=summary,
          description=description,
          state=state,
          state_reason=state_reason,
          access=access,
          issue_notify_address=issue_notify_address,
          attachment_bytes_used=attachment_bytes_used,
          attachment_quota=attachment_quota,
          moved_to=moved_to,
          process_inbound_email=process_inbound_email,
          only_owners_remove_restrictions=only_owners_remove_restrictions,
          read_only_reason=read_only_reason,
          cached_content_timestamp=cached_content_timestamp,
          only_owners_see_contributors=only_owners_see_contributors,
          delete_time=delete_time,
          recent_activity=recent_activity,
          revision_url_format=revision_url_format,
          home_page=home_page,
          docs_url=docs_url,
          source_url=source_url,
          logo_gcs_id=logo_gcs_id,
          logo_file_name=logo_file_name,
          issue_notify_always_detailed=issue_notify_always_detailed)

  def DeleteProject(self, project_id):
    """Mark the project as deletable.  It will be reaped by a cron job.

    Args:
      project_id: int ID of the project to delete.

    Returns:
      Nothing.

    Raises:
      NoSuchProjectException: There is no project with that ID.
    """
    project = self.GetProject(project_id)
    self._AssertPermInProject(permissions.EDIT_PROJECT, project)

    with self.mc.profiler.Phase('marking deletable %r' % project_id):
      _project = self.GetProject(project_id)
      self.services.project.MarkProjectDeletable(
          self.mc.cnxn, project_id, self.services.config)

  def StarProject(self, project_id, starred):
    """Star or unstar the specified project.

    Args:
      project_id: int ID of the project to star/unstar.
      starred: true to add a star, false to remove it.

    Returns:
      Nothing.

    Raises:
      NoSuchProjectException: There is no project with that ID.
    """
    project = self.GetProject(project_id)
    self._AssertPermInProject(permissions.SET_STAR, project)

    with self.mc.profiler.Phase('(un)starring project %r' % project_id):
      self.services.project_star.SetStar(
          self.mc.cnxn, project_id, self.mc.auth.user_id, starred)

  def IsProjectStarred(self, project_id):
    """Return True if the current user has starred the given project.

    Args:
      project_id: int ID of the project to check.

    Returns:
      True if starred.

    Raises:
      NoSuchProjectException: There is no project with that ID.
    """
    if project_id is None:
      raise exceptions.InputException('No project specified')

    if not self.mc.auth.user_id:
      return False

    with self.mc.profiler.Phase('checking project star %r' % project_id):
      # Make sure the project exists and user has permission to see it.
      _project = self.GetProject(project_id)
      return self.services.project_star.IsItemStarredBy(
        self.mc.cnxn, project_id, self.mc.auth.user_id)

  def GetProjectStarCount(self, project_id):
    """Return the number of times the project has been starred.

    Args:
      project_id: int ID of the project to check.

    Returns:
      The number of times the project has been starred.

    Raises:
      NoSuchProjectException: There is no project with that ID.
    """
    if project_id is None:
      raise exceptions.InputException('No project specified')

    with self.mc.profiler.Phase('counting stars for project %r' % project_id):
      # Make sure the project exists and user has permission to see it.
      _project = self.GetProject(project_id)
      return self.services.project_star.CountItemStars(self.mc.cnxn, project_id)

  def ListStarredProjects(self, viewed_user_id=None):
    """Return a list of projects starred by the current or viewed user.

    Args:
      viewed_user_id: optional user ID for another user's profile page, if
          not supplied, the signed in user is used.

    Returns:
      A list of projects that were starred by current user and that they
      are currently allowed to view.
    """
    # Note: No permission checks for this call, but the list of starred
    # projects is filtered based on permission to view.

    if viewed_user_id is None:
      if self.mc.auth.user_id:
        viewed_user_id = self.mc.auth.user_id
      else:
        return []  # Anon user and no viewed user specified.
    with self.mc.profiler.Phase('ListStarredProjects for %r' % viewed_user_id):
      viewable_projects = sitewide_helpers.GetViewableStarredProjects(
          self.mc.cnxn, self.services, viewed_user_id,
          self.mc.auth.effective_ids, self.mc.auth.user_pb)
    return viewable_projects

  def GetProjectConfigs(self, project_ids, use_cache=True):
    """Return the specifed configs.

    Args:
      project_ids: int IDs of the projects to retrieve.
      use_cache: set to false when doing read-modify-write.

    Returns:
      The specified configs.
    """
    with self.mc.profiler.Phase('getting configs for %r' % project_ids):
      configs = self.services.config.GetProjectConfigs(
          self.mc.cnxn, project_ids, use_cache=use_cache)

    projects = self._FilterVisibleProjectsDict(
        self.GetProjects(list(configs.keys())))
    configs = {project_id: configs[project_id] for project_id in projects}

    return configs

  def GetProjectConfig(self, project_id, use_cache=True):
    """Return the specifed config.

    Args:
      project_id: int ID of the project to retrieve.
      use_cache: set to false when doing read-modify-write.

    Returns:
      The specified config.

    Raises:
      NoSuchProjectException: There is no matching config.
    """
    configs = self.GetProjectConfigs([project_id], use_cache)
    if not configs:
      raise exceptions.NoSuchProjectException()
    return configs[project_id]

  def ListProjectTemplates(self, project_id):
    templates = self.services.template.GetProjectTemplates(
        self.mc.cnxn, project_id)
    project = self.GetProject(project_id)
    # Filter non-viewable templates
    if framework_bizobj.UserIsInProject(project, self.mc.auth.effective_ids):
      return templates
    return [template for template in templates if not template.members_only]

  def ListComponentDefs(self, project_id, page_size, start):
    # type: (int, int, int) -> ListResult
    """Returns component defs that belong to the project."""
    if start < 0:
      raise exceptions.InputException('Invalid `start`: %d' % start)
    if page_size < 0:
      raise exceptions.InputException('Invalid `page_size`: %d' % page_size)

    config = self.GetProjectConfig(project_id)
    end = start + page_size
    next_start = None
    if end < len(config.component_defs):
      next_start = end
    return ListResult(config.component_defs[start:end], next_start)

  def CreateComponentDef(
      self, project_id, path, description, admin_ids, cc_ids, labels):
    # type: (int, str, str, Collection[int], Collection[int], Collection[str])
    #     -> ComponentDef
    """Creates a ComponentDef with the given information."""
    project = self.GetProject(project_id)
    config = self.GetProjectConfig(project_id)

    # Validate new ComponentDef and check permissions.
    ancestor_path, leaf_name = None, path
    if '>' in path:
      ancestor_path, leaf_name = path.rsplit('>', 1)
      ancestor_def = tracker_bizobj.FindComponentDef(ancestor_path, config)
      if not ancestor_def:
        raise exceptions.InputException(
            'Ancestor path %s is invalid.' % ancestor_path)
      project_perms = permissions.GetPermissions(
          self.mc.auth.user_pb, self.mc.auth.effective_ids, project)
      if not permissions.CanEditComponentDef(
          self.mc.auth.effective_ids, project_perms, project, ancestor_def,
          config):
        raise permissions.PermissionException(
            'User is not allowed to create a subcomponent under %s.' %
            ancestor_path)
    else:
      # A brand new top level component is being created.
      self._AssertPermInProject(permissions.EDIT_PROJECT, project)

    if not tracker_constants.COMPONENT_NAME_RE.match(leaf_name):
      raise exceptions.InputException('Invalid component path: %s.' % leaf_name)

    if tracker_bizobj.FindComponentDef(path, config):
      raise exceptions.ComponentDefAlreadyExists(
          'Component path %s already exists.' % path)

    with exceptions.ErrorAggregator(exceptions.InputException) as err_agg:
      tracker_helpers.AssertUsersExist(
          self.mc.cnxn, self.services, cc_ids + admin_ids, err_agg)

    label_ids = self.services.config.LookupLabelIDs(
        self.mc.cnxn, project_id, labels, autocreate=True)
    self.services.config.CreateComponentDef(
        self.mc.cnxn, project_id, path, description, False, admin_ids, cc_ids,
        int(time.time()), self.mc.auth.user_id, label_ids)
    updated_config = self.GetProjectConfig(project_id, use_cache=False)
    return tracker_bizobj.FindComponentDef(path, updated_config)

  def DeleteComponentDef(self, project_id, component_id):
    # type: (MonorailConnection, int, int) -> None
    """Deletes the given ComponentDef."""
    project = self.GetProject(project_id)
    config = self.GetProjectConfig(project_id)

    component_def = tracker_bizobj.FindComponentDefByID(component_id, config)
    if not component_def:
      raise exceptions.NoSuchComponentException('The component does not exist.')

    project_perms = permissions.GetPermissions(
        self.mc.auth.user_pb, self.mc.auth.effective_ids, project)
    if not permissions.CanEditComponentDef(
        self.mc.auth.effective_ids, project_perms, project, component_def,
        config):
      raise permissions.PermissionException(
          'User is not allowed to delete this component.')

    if tracker_bizobj.FindDescendantComponents(config, component_def):
      raise exceptions.InputException(
          'Components with subcomponents cannot be deleted.')

    self.services.config.DeleteComponentDef(
        self.mc.cnxn, project_id, component_id)

  # FUTURE: labels, statuses, components, rules, templates, and views.
  # FUTURE: project saved queries.
  # FUTURE: GetProjectPermissionsForUser()

  ### Field methods

  # FUTURE: All other field methods.

  def GetFieldDef(self, field_id, project):
    # type: (int, Project) -> FieldDef
    """Return the specified hotlist.

    Args:
      field_id: int field_id of the field to retrieve.
      project: Project object that the field belongs to.

    Returns:
      The specified field.

    Raises:
      InputException: No field was specified.
      NoSuchFieldDefException: There is no field with that ID.
      PermissionException: The user is not allowed to view the field.
    """
    with self.mc.profiler.Phase('getting fielddef %r' % field_id):
      config = self.GetProjectConfig(project.project_id)
      field = tracker_bizobj.FindFieldDefByID(field_id, config)
      if field is None:
        raise exceptions.NoSuchFieldDefException('Field not found.')
    self._AssertUserCanViewFieldDef(project, field)
    return field

  ### Issue methods

  def CreateIssue(
      self,
      project_id,  # type: int
      summary,  # type: str
      status,  # type: str
      owner_id,  # type: int
      cc_ids,  # type: Sequence[int]
      labels,  # type: Sequence[str]
      field_values,  # type: Sequence[proto.tracker_pb2.FieldValue]
      component_ids,  # type: Sequence[int]
      marked_description,  # type: str
      blocked_on=None,  # type: Sequence[int]
      blocking=None,  # type: Sequence[int]
      attachments=None,  # type: Sequence[Tuple[str, str, str]]
      phases=None,  # type: Sequence[proto.tracker_pb2.Phase]
      approval_values=None,  # type: Sequence[proto.tracker_pb2.ApprovalValue]
      send_email=True,  # type: bool
      reporter_id=None,  # type: int
      timestamp=None,  # type: int
      dangling_blocked_on=None,  # type: Sequence[DanglingIssueRef]
      dangling_blocking=None,  # type: Sequence[DanglingIssueRef]
      raise_filter_errors=True,  # type: bool
  ):
    # type: (...) -> (proto.tracker_pb2.Issue, proto.tracker_pb2.IssueComment)
    """Create and store a new issue with all the given information.

    Args:
      project_id: int ID for the current project.
      summary: one-line summary string summarizing this issue.
      status: string issue status value.  E.g., 'New'.
      owner_id: user ID of the issue owner.
      cc_ids: list of user IDs for users to be CC'd on changes.
      labels: list of label strings.  E.g., 'Priority-High'.
      field_values: list of FieldValue PBs.
      component_ids: list of int component IDs.
      marked_description: issue description with initial HTML markup.
      blocked_on: list of issue_ids that this issue is blocked on.
      blocking: list of issue_ids that this issue blocks.
      attachments: [(filename, contents, mimetype),...] attachments uploaded at
          the time the comment was made.
      phases: list of Phase PBs.
      approval_values: list of ApprovalValue PBs.
      send_email: set to False to avoid email notifications.
      reporter_id: optional user ID of a different user to attribute this
          issue report to.  The requester must have the ImportComment perm.
      timestamp: optional int timestamp of an imported issue.
      dangling_blocked_on: a list of DanglingIssueRefs this issue is blocked on.
      dangling_blocking: a list of DanglingIssueRefs that this issue blocks.
      raise_filter_errors: whether to raise when filter rules produce errors.

    Returns:
      A tuple (newly created Issue, Comment PB for the description).

    Raises:
      FilterRuleException if creation violates any filter rule that shows error.
      InputException: The issue has invalid input, see validation below.
      PermissionException if user lacks sufficient permissions.
    """
    project = self.GetProject(project_id)
    self._AssertPermInProject(permissions.CREATE_ISSUE, project)

    # TODO(crbug/monorail/7197): The following are needed for v3 API
    # Phase 5.2 Validate sufficient attachment quota and update

    if reporter_id and reporter_id != self.mc.auth.user_id:
      self._AssertPermInProject(permissions.IMPORT_COMMENT, project)
      importer_id = self.mc.auth.user_id
    else:
      reporter_id = self.mc.auth.user_id
      importer_id = None

    with self.mc.profiler.Phase('creating issue in project %r' % project_id):
      # TODO(crbug/monorail/8000): Refactor issue proto construction
      # to the caller.
      status = framework_bizobj.CanonicalizeLabel(status)
      labels = [framework_bizobj.CanonicalizeLabel(l) for l in labels]
      labels = [l for l in labels if l]

      issue = tracker_pb2.Issue()
      issue.project_id = project_id
      issue.project_name = self.services.project.LookupProjectNames(
          self.mc.cnxn, [project_id]).get(project_id)
      issue.summary = summary
      issue.status = status
      issue.owner_id = owner_id
      issue.cc_ids.extend(cc_ids)
      issue.labels.extend(labels)
      issue.field_values.extend(field_values)
      issue.component_ids.extend(component_ids)
      issue.reporter_id = reporter_id
      if blocked_on is not None:
        issue.blocked_on_iids = blocked_on
        issue.blocked_on_ranks = [0] * len(blocked_on)
      if blocking is not None:
        issue.blocking_iids = blocking
      if dangling_blocked_on is not None:
        issue.dangling_blocked_on_refs = dangling_blocked_on
      if dangling_blocking is not None:
        issue.dangling_blocking_refs = dangling_blocking
      if attachments:
        issue.attachment_count = len(attachments)
      if phases:
        issue.phases = phases
      if approval_values:
        issue.approval_values = approval_values
      timestamp = timestamp or int(time.time())
      issue.opened_timestamp = timestamp
      issue.modified_timestamp = timestamp
      issue.owner_modified_timestamp = timestamp
      issue.status_modified_timestamp = timestamp
      issue.component_modified_timestamp = timestamp

      # Validate the issue
      tracker_helpers.AssertValidIssueForCreate(
          self.mc.cnxn, self.services, issue, marked_description)

      # Apply filter rules.
      # Set the closed_timestamp both before and after filter rules.
      config = self.GetProjectConfig(issue.project_id)
      if not tracker_helpers.MeansOpenInProject(
          tracker_bizobj.GetStatus(issue), config):
        issue.closed_timestamp = issue.opened_timestamp
      filterrules_helpers.ApplyFilterRules(
          self.mc.cnxn, self.services, issue, config)
      if issue.derived_errors and raise_filter_errors:
        raise exceptions.FilterRuleException(issue.derived_errors)
      if not tracker_helpers.MeansOpenInProject(
          tracker_bizobj.GetStatus(issue), config):
        issue.closed_timestamp = issue.opened_timestamp

      new_issue, comment = self.services.issue.CreateIssue(
          self.mc.cnxn,
          self.services,
          issue,
          marked_description,
          attachments=attachments,
          index_now=False,
          importer_id=importer_id)
      logging.info(
          'created issue %r in project %r', new_issue.local_id, project_id)

    with self.mc.profiler.Phase('following up after issue creation'):
      self.services.project.UpdateRecentActivity(self.mc.cnxn, project_id)

    if send_email:
      with self.mc.profiler.Phase('queueing notification tasks'):
        hostport = framework_helpers.GetHostPort(
            project_name=project.project_name)
        send_notifications.PrepareAndSendIssueChangeNotification(
            new_issue.issue_id, hostport, reporter_id, comment_id=comment.id)
        send_notifications.PrepareAndSendIssueBlockingNotification(
            new_issue.issue_id, hostport, new_issue.blocked_on_iids,
            reporter_id)

    return new_issue, comment

  def MakeIssueFromTemplate(self, _template, _description, _issue_delta):
    # type: (tracker_pb2.TemplateDef, str, tracker_pb2.IssueDelta) ->
    #     tracker_pb2.Issue
    """Creates issue from template, issue description, and delta.

    Args:
      template: Template that issue creation is based on.
      description: Issue description string.
      issue_delta: Difference between desired issue and base issue.

    Returns:
      Newly created issue, as protorpc Issue.

    Raises:
      TODO(crbug/monorail/7197): Document errors when implemented
    """
    # Phase 2: Build Issue from TemplateDef
    # Use helper method, likely from template_helpers

    # Phase 3: Validate proposed deltas and check permissions
    # Check summary has been edited if required, else throw
    # Check description is different from template default, else throw
    # Check edit permission on field values of issue deltas, else throw

    # Phase 4: Merge template, delta, and defaults
    # Merge delta into issue
    # Apply approval def defaults to approval values
    # Capitalize every line of description

    # Phase 5: Create issue by calling work_env.CreateIssue

    return tracker_pb2.Issue()

  def MakeIssue(self, issue, description, send_email):
    # type: (tracker_pb2.Issue, str, bool) -> tracker_pb2.Issue
    """Check restricted field permissions and create issue.

    Args:
      issue: Data for the created issue in a Protocol Bugger.
      description: Description for the initial description comment created.
      send_email: Whether this issue creation should email people.

    Returns:
      The created Issue PB.

    Raises:
      FilterRuleException if creation violates any filter rule that shows error.
      InputException: The issue has invalid input, see validation below.
      PermissionException if user lacks sufficient permissions.
    """
    config = self.GetProjectConfig(issue.project_id)
    project = self.GetProject(issue.project_id)
    self._AssertUserCanEditFieldsAndEnumMaskedLabels(
        project, config, [fv.field_id for fv in issue.field_values],
        issue.labels)
    issue, _comment = self.CreateIssue(
        issue.project_id,
        issue.summary,
        issue.status,
        issue.owner_id,
        issue.cc_ids,
        issue.labels,
        issue.field_values,
        issue.component_ids,
        description,
        blocked_on=issue.blocked_on_iids,
        blocking=issue.blocking_iids,
        dangling_blocked_on=issue.dangling_blocked_on_refs,
        dangling_blocking=issue.dangling_blocking_refs,
        send_email=send_email)
    return issue

  def MoveIssue(self, issue, target_project):
    """Move issue to the target_project.

    The current user needs to have permission to delete the current issue, and
    to edit issues on the target project.

    Args:
      issue: the issue PB.
      target_project: the project PB where the issue should be moved to.
    Returns:
      The issue PB of the new issue on the target project.
    """
    self._AssertPermInIssue(issue, permissions.DELETE_ISSUE)
    self._AssertPermInProject(permissions.EDIT_ISSUE, target_project)

    if permissions.GetRestrictions(issue):
      raise exceptions.InputException(
          'Issues with Restrict labels are not allowed to be moved')

    with self.mc.profiler.Phase('Moving Issue'):
      tracker_fulltext.UnindexIssues([issue.issue_id])

      # issue is modified by MoveIssues
      old_text_ref = 'issue %s:%s' % (issue.project_name, issue.local_id)
      moved_back_iids = self.services.issue.MoveIssues(
          self.mc.cnxn, target_project, [issue], self.services.user)
      new_text_ref = 'issue %s:%s' % (issue.project_name, issue.local_id)

      if issue.issue_id in moved_back_iids:
        content = 'Moved %s back to %s again.' % (old_text_ref, new_text_ref)
      else:
        content = 'Moved %s to now be %s.' % (old_text_ref, new_text_ref)
      self.services.issue.CreateIssueComment(
          self.mc.cnxn, issue, self.mc.auth.user_id, content,
          amendments=[
              tracker_bizobj.MakeProjectAmendment(target_project.project_name)])

      tracker_fulltext.IndexIssues(
          self.mc.cnxn, [issue], self.services.user, self.services.issue,
          self.services.config)

    return issue

  def CopyIssue(self, issue, target_project):
    """Copy issue to the target_project.

    The current user needs to have permission to delete the current issue, and
    to edit issues on the target project.

    Args:
      issue: the issue PB.
      target_project: the project PB where the issue should be copied to.
    Returns:
      The issue PB of the new issue on the target project.
    """
    self._AssertPermInIssue(issue, permissions.DELETE_ISSUE)
    self._AssertPermInProject(permissions.EDIT_ISSUE, target_project)

    if permissions.GetRestrictions(issue):
      raise exceptions.InputException(
          'Issues with Restrict labels are not allowed to be copied')

    with self.mc.profiler.Phase('Copying Issue'):
      copied_issue = self.services.issue.CopyIssues(
          self.mc.cnxn, target_project, [issue], self.services.user,
          self.mc.auth.user_id)[0]

      issue_ref = 'issue %s:%s' % (issue.project_name, issue.local_id)
      copied_issue_ref = 'issue %s:%s' % (
          copied_issue.project_name, copied_issue.local_id)

      # Add comment to the original issue.
      content = 'Copied %s to %s' % (issue_ref, copied_issue_ref)
      self.services.issue.CreateIssueComment(
          self.mc.cnxn, issue, self.mc.auth.user_id, content)

      # Add comment to the newly created issue.
      # Add project amendment only if the project changed.
      amendments = []
      if issue.project_id != copied_issue.project_id:
        amendments.append(
            tracker_bizobj.MakeProjectAmendment(target_project.project_name))
      new_issue_content = 'Copied %s from %s' % (copied_issue_ref, issue_ref)
      self.services.issue.CreateIssueComment(
          self.mc.cnxn, copied_issue, self.mc.auth.user_id, new_issue_content,
          amendments=amendments)

      tracker_fulltext.IndexIssues(
          self.mc.cnxn, [copied_issue], self.services.user, self.services.issue,
          self.services.config)

    return copied_issue

  def _MergeLinkedAccounts(self, me_user_id):
    """Return a list of the given user ID and any linked accounts."""
    if not me_user_id:
      return []

    result = [me_user_id]
    me_user = self.services.user.GetUser(self.mc.cnxn, me_user_id)
    if me_user:
      if me_user.linked_parent_id:
        result.append(me_user.linked_parent_id)
      result.extend(me_user.linked_child_ids)
    return result

  def SearchIssues(
      self, query_string, query_project_names, me_user_id, items_per_page,
      paginate_start, sort_spec):
    # type: (str, Sequence[str], int, int, int, str) -> ListResult
    """Search for issues in the given projects."""
    # TODO(crbug.com/monorail/7678): Remove can. Replace
    # project_names with project_ids.
    # TODO(crbug.com/monorail/6988): Delete ListIssues when endpoints and v1
    # are deprecated. Move pipeline call to SearchIssues.
    use_cached_searches = not settings.local_mode
    pipeline = self.ListIssues(
        query_string, query_project_names, me_user_id, items_per_page,
        paginate_start, 1, '', sort_spec, use_cached_searches)

    end = paginate_start + items_per_page
    next_start = None
    if end < pipeline.total_count:
      next_start = end
    return ListResult(pipeline.visible_results, next_start)

  def ListIssues(
      self,
      query_string,  # type: str
      query_project_names,  # type: Sequence[str]
      me_user_id,  # type: int
      items_per_page,  # type: int
      paginate_start,  # type: int
      can,  # type: int
      group_by_spec,  # type: str
      sort_spec,  # type: str
      use_cached_searches,  # type: bool
      project=None  # type: proto.Project
  ):
    # type: (...) -> search.frontendsearchpipeline.FrontendSearchPipeline
    """Do an issue search w/ mc + passed in args to return a pipeline object.

    Args:
      query_string: str with the query the user is searching for.
      query_project_names: List of project names to query for.
      me_user_id: Relevant user id. Usually the logged in user.
      items_per_page: Max number of issues to include in the results.
      paginate_start: Offset of issues to skip for pagination.
      can: id of canned query to use.
      group_by_spec: str used to specify how issues should be grouped.
      sort_spec: str used to specify how issues should be sorted.
      use_cached_searches: Whether to use the cache or not.
      project: Project object for the current project the user is viewing.

    Returns:
      A FrontendSearchPipeline instance with data on issues found.
    """
    # Permission to view a project is checked in FrontendSearchPipeline().
    # Individual results are filtered by permissions in SearchForIIDs().

    with self.mc.profiler.Phase('searching issues'):
      me_user_ids = self._MergeLinkedAccounts(me_user_id)
      pipeline = frontendsearchpipeline.FrontendSearchPipeline(
          self.mc.cnxn,
          self.services,
          self.mc.auth,
          me_user_ids,
          query_string,
          query_project_names,
          items_per_page,
          paginate_start,
          can,
          group_by_spec,
          sort_spec,
          self.mc.warnings,
          self.mc.errors,
          use_cached_searches,
          self.mc.profiler,
          project=project)
      if not self.mc.errors.AnyErrors():
        pipeline.SearchForIIDs()
        pipeline.MergeAndSortIssues()
        pipeline.Paginate()
      # TODO(jojwang): raise InvalidQueryException.
      return pipeline

  # TODO(jrobbins): This method also requires self.mc to be a MonorailRequest.
  def FindIssuePositionInSearch(self, issue):
    """Do an issue search and return flipper info for the given issue.

    Args:
      issue: issue that the user is currently viewing.

    Returns:
      A 4-tuple of flipper info: (prev_iid, cur_index, next_iid, total_count).
    """
    # Permission to view a project is checked in FrontendSearchPipeline().
    # Individual results are filtered by permissions in SearchForIIDs().

    with self.mc.profiler.Phase('finding issue position in search'):
      me_user_ids = self._MergeLinkedAccounts(self.mc.me_user_id)
      pipeline = frontendsearchpipeline.FrontendSearchPipeline(
          self.mc.cnxn,
          self.services,
          self.mc.auth,
          me_user_ids,
          self.mc.query,
          self.mc.query_project_names,
          self.mc.num,
          self.mc.start,
          self.mc.can,
          self.mc.group_by_spec,
          self.mc.sort_spec,
          self.mc.warnings,
          self.mc.errors,
          self.mc.use_cached_searches,
          self.mc.profiler,
          project=self.mc.project)
      if not self.mc.errors.AnyErrors():
        # Only do the search if the user's query parsed OK.
        pipeline.SearchForIIDs()

      # Note: we never call MergeAndSortIssues() because we don't need a unified
      # sorted list, we only need to know the position on such a list of the
      # current issue.
      prev_iid, cur_index, next_iid = pipeline.DetermineIssuePosition(issue)

      return prev_iid, cur_index, next_iid, pipeline.total_count

  # TODO(crbug/monorail/6988): add boolean to ignore_private_issues
  def GetIssuesDict(self, issue_ids, use_cache=True,
                    allow_viewing_deleted=False):
    # type: (Collection[int], Optional[Boolean], Optional[Boolean]) ->
    #     Mapping[int, Issue]
    """Return a dict {iid: issue} with the specified issues, if allowed.

    Args:
      issue_ids: int global issue IDs.
      use_cache: set to false to ensure fresh issues.
      allow_viewing_deleted: set to true to allow user to view deleted issues.

    Returns:
      A dict {issue_id: issue} for only those issues that the user is allowed
      to view.

    Raises:
      NoSuchIssueException if an issue is not found.
      PermissionException if the user cannot view all issues.
    """
    with self.mc.profiler.Phase('getting issues %r' % issue_ids):
      issues_by_id, missing_ids = self.services.issue.GetIssuesDict(
          self.mc.cnxn, issue_ids, use_cache=use_cache)

    if missing_ids:
      with exceptions.ErrorAggregator(
          exceptions.NoSuchIssueException) as missing_err_agg:
        for missing_id in missing_ids:
          missing_err_agg.AddErrorMessage('No such issue: %s' % missing_id)

    with exceptions.ErrorAggregator(
        permissions.PermissionException) as permission_err_agg:
      for issue in issues_by_id.values():
        try:
          self._AssertUserCanViewIssue(
              issue, allow_viewing_deleted=allow_viewing_deleted)
        except permissions.PermissionException as e:
          permission_err_agg.AddErrorMessage(e.message)

    return issues_by_id

  def GetIssue(self, issue_id, use_cache=True, allow_viewing_deleted=False):
    """Return the specified issue.

    Args:
      issue_id: int global issue ID.
      use_cache: set to false to ensure fresh issue.
      allow_viewing_deleted: set to true to allow user to view a deleted issue.

    Returns:
      The requested Issue PB.
    """
    if issue_id is None:
      raise exceptions.InputException('No issue issue_id specified')

    with self.mc.profiler.Phase('getting issue %r' % issue_id):
      issue = self.services.issue.GetIssue(
          self.mc.cnxn, issue_id, use_cache=use_cache)

    self._AssertUserCanViewIssue(
        issue, allow_viewing_deleted=allow_viewing_deleted)
    return issue

  def ListReferencedIssues(self, ref_tuples, default_project_name):
    """Return the specified issues."""
    # Make sure ref_tuples are unique, preserving order.
    ref_tuples = list(collections.OrderedDict(
        list(zip(ref_tuples, ref_tuples))))
    ref_projects = self.services.project.GetProjectsByName(
        self.mc.cnxn,
        [(ref_pn or default_project_name) for ref_pn, _ in ref_tuples])
    issue_ids, _misses = self.services.issue.ResolveIssueRefs(
        self.mc.cnxn, ref_projects, default_project_name, ref_tuples)
    open_issues, closed_issues = (
        tracker_helpers.GetAllowedOpenedAndClosedIssues(
            self.mc, issue_ids, self.services))
    return open_issues, closed_issues

  def GetIssueByLocalID(
      self, project_id, local_id, use_cache=True,
      allow_viewing_deleted=False):
    """Return the specified issue, TODO: iff the signed in user may view it.

    Args:
      project_id: int project ID of the project that contains the issue.
      local_id: int issue local id number.
      use_cache: set to False when doing read-modify-write operations.
      allow_viewing_deleted: set to True to return a deleted issue so that
          an authorized user may undelete it.

    Returns:
      The specified Issue PB.

    Raises:
      exceptions.InputException: Something was not specified properly.
      exceptions.NoSuchIssueException: The issue does not exist.
    """
    if project_id is None:
      raise exceptions.InputException('No project specified')
    if local_id is None:
      raise exceptions.InputException('No issue local_id specified')

    with self.mc.profiler.Phase('getting issue %r:%r' % (project_id, local_id)):
      issue = self.services.issue.GetIssueByLocalID(
          self.mc.cnxn, project_id, local_id, use_cache=use_cache)

    self._AssertUserCanViewIssue(
        issue, allow_viewing_deleted=allow_viewing_deleted)
    return issue

  def GetRelatedIssueRefs(self, issues):
    """Return a dict {iid: (project_name, local_id)} for all related issues."""
    related_iids = set()
    with self.mc.profiler.Phase('getting related issue refs'):
      for issue in issues:
        related_iids.update(issue.blocked_on_iids)
        related_iids.update(issue.blocking_iids)
        if issue.merged_into:
          related_iids.add(issue.merged_into)
      logging.info('related_iids is %r', related_iids)
      return self.services.issue.LookupIssueRefs(self.mc.cnxn, related_iids)

  def GetIssueRefs(self, issue_ids):
    """Return a dict {iid: (project_name, local_id)} for all issue_ids."""
    return self.services.issue.LookupIssueRefs(self.mc.cnxn, issue_ids)

  def BulkUpdateIssueApprovals(self, issue_ids, approval_id, project,
                               approval_delta, comment_content,
                               send_email):
    """Update all given issues' specified approval."""
    # Anon users and users with no permission to view the project
    # will get permission denied. Missing permissions to update
    # individual issues will not throw exceptions. Issues will just not be
    # updated.
    if not self.mc.auth.user_id:
      raise permissions.PermissionException('Anon cannot make changes')
    if not self._UserCanViewProject(project):
      raise permissions.PermissionException('User cannot view project')
    updated_issue_ids = []
    for issue_id in issue_ids:
      try:
        self.UpdateIssueApproval(
            issue_id, approval_id, approval_delta, comment_content, False,
            send_email=False)
        updated_issue_ids.append(issue_id)
      except exceptions.NoSuchIssueApprovalException as e:
        logging.info('Skipping issue %s, no approval: %s', issue_id, e)
      except permissions.PermissionException as e:
        logging.info('Skipping issue %s, update not allowed: %s', issue_id, e)
    # TODO(crbug/monorail/8122): send bulk approval update email if send_email.
    if send_email:
      pass
    return updated_issue_ids

  def BulkUpdateIssueApprovalsV3(
      self, delta_specifications, comment_content, send_email):
    # type: (Sequence[Tuple[int, int, tracker_pb2.ApprovalDelta]]], str,
    #     Boolean -> Sequence[proto.tracker_pb2.ApprovalValue]
    """Executes the ApprovalDeltas.

    Args:
      delta_specifications: List of (issue_id, approval_id, ApprovalDelta).
      comment_content: The content of the comment to be posted with each delta.
      send_email: Whether to send an email on each change.
          TODO(crbug/monorail/8122): send bulk approval update email instead.

    Returns:
      A list of (Issue, ApprovalValue) pairs corresponding to each
      specification provided in `delta_specifications`.

    Raises:
      InputException: If a comment is too long.
      NoSuchIssueApprovalException: If any of the approvals specified
          does not exist.
      PermissionException: If the current user lacks permissions to execute
          any of the deltas provided.
    """
    updated_approval_values = []
    for (issue_id, approval_id, approval_delta) in delta_specifications:
      updated_av, _comment, issue = self.UpdateIssueApproval(
          issue_id,
          approval_id,
          approval_delta,
          comment_content,
          False,
          send_email=send_email,
          update_perms=True)
      updated_approval_values.append((issue, updated_av))
    return updated_approval_values

  def UpdateIssueApproval(
      self,
      issue_id,
      approval_id,
      approval_delta,
      comment_content,
      is_description,
      attachments=None,
      send_email=True,
      kept_attachments=None,
      update_perms=False):
    # type: (int, int, proto.tracker_pb2.ApprovalDelta, str, Boolean,
    #     Optional[Sequence[proto.tracker_pb2.Attachment]], Optional[Boolean],
    #     Optional[Sequence[int]], Optional[Boolean]) ->
    #     (proto.tracker_pb2.ApprovalValue, proto.tracker_pb2.IssueComment)
    """Update an issue's approval.

    Raises:
      InputException: The comment content is too long or additional approvers do
      not exist.
      PermissionException: The user is lacking one of the permissions needed
      for the given delta.
      NoSuchIssueApprovalException: The issue/approval combo does not exist.
    """

    issue, approval_value = self.services.issue.GetIssueApproval(
        self.mc.cnxn, issue_id, approval_id, use_cache=False)

    self._AssertPermInIssue(issue, permissions.EDIT_ISSUE)

    if len(comment_content) > tracker_constants.MAX_COMMENT_CHARS:
      raise exceptions.InputException('Comment is too long')

    project = self.GetProject(issue.project_id)
    config = self.GetProjectConfig(issue.project_id)
    # TODO(crbug/monorail/7614): Remove the need for this hack to update perms.
    if update_perms:
      self.mc.LookupLoggedInUserPerms(project)

    if attachments:
      with self.mc.profiler.Phase('Accounting for quota'):
        new_bytes_used = tracker_helpers.ComputeNewQuotaBytesUsed(
          project, attachments)
        self.services.project.UpdateProject(
          self.mc.cnxn, issue.project_id, attachment_bytes_used=new_bytes_used)

    if kept_attachments:
      with self.mc.profiler.Phase('Filtering kept attachments'):
        kept_attachments = tracker_helpers.FilterKeptAttachments(
            is_description, kept_attachments, self.ListIssueComments(issue),
            approval_id)

    if approval_delta.status:
      if not permissions.CanUpdateApprovalStatus(
          self.mc.auth.effective_ids, self.mc.perms, project,
          approval_value.approver_ids, approval_delta.status):
        raise permissions.PermissionException(
            'User not allowed to make this status update.')

    if approval_delta.approver_ids_remove or approval_delta.approver_ids_add:
      if not permissions.CanUpdateApprovers(
          self.mc.auth.effective_ids, self.mc.perms, project,
          approval_value.approver_ids):
        raise permissions.PermissionException(
            'User not allowed to modify approvers of this approval.')

    # Check additional approvers exist.
    with exceptions.ErrorAggregator(exceptions.InputException) as err_agg:
      tracker_helpers.AssertUsersExist(
          self.mc.cnxn, self.services, approval_delta.approver_ids_add, err_agg)

    with self.mc.profiler.Phase(
        'updating approval for issue %r, aprpoval %r' % (
            issue_id, approval_id)):
      comment_pb = self.services.issue.DeltaUpdateIssueApproval(
          self.mc.cnxn, self.mc.auth.user_id, config, issue, approval_value,
          approval_delta, comment_content=comment_content,
          is_description=is_description, attachments=attachments,
          kept_attachments=kept_attachments)
      hostport = framework_helpers.GetHostPort(
          project_name=project.project_name)
      send_notifications.PrepareAndSendApprovalChangeNotification(
          issue_id, approval_id, hostport, comment_pb.id,
          send_email=send_email)

    return approval_value, comment_pb, issue

  def ConvertIssueApprovalsTemplate(
      self, config, issue, template_name, comment_content, send_email=True):
    # type: (proto.tracker_pb2.ProjectIssueConfig, proto.tracker_pb2.Issue,
    #     str, str, Optional[Boolean] )
    """Convert an issue's existing approvals structure to match the one of
       the given template.

    Raises:
      InputException: The comment content is too long.
    """
    self._AssertPermInIssue(issue, permissions.EDIT_ISSUE)

    template = self.services.template.GetTemplateByName(
        self.mc.cnxn, template_name, issue.project_id)
    if not template:
      raise exceptions.NoSuchTemplateException(
          'Template %s is not found' % template_name)

    if len(comment_content) > tracker_constants.MAX_COMMENT_CHARS:
      raise exceptions.InputException('Comment is too long')

    with self.mc.profiler.Phase('updating issue %r' % issue):
      comment_pb = self.services.issue.UpdateIssueStructure(
          self.mc.cnxn, config, issue, template, self.mc.auth.user_id,
          comment_content)
      hostport = framework_helpers.GetHostPort(project_name=issue.project_name)
      send_notifications.PrepareAndSendIssueChangeNotification(
          issue.issue_id, hostport, self.mc.auth.user_id,
          send_email=send_email, comment_id=comment_pb.id)

  def UpdateIssue(
      self, issue, delta, comment_content, attachments=None, send_email=True,
      is_description=False, kept_attachments=None, inbound_message=None):
    # type: (...) => None
    """Update an issue with a set of changes and add a comment.

    Args:
      issue: Existing Issue PB for the issue to be modified.
      delta: IssueDelta object containing all the changes to be made.
      comment_content: string content of the user's comment.
      attachments: List [(filename, contents, mimetype),...] of attachments.
      send_email: set to False to suppress email notifications.
      is_description: True if this adds a new issue description.
      kept_attachments: This should be a list of int attachment ids for
          attachments kept from previous descriptions, if the comment is
          a change to the issue description.
      inbound_message: optional string full text of an email that caused
          this comment to be added.

    Returns:
      Nothing.

    Raises:
      InputException: The comment content is too long.
    """
    if not self._UserCanUsePermInIssue(issue, permissions.EDIT_ISSUE):
      # We're editing the issue description. Only users with EditIssue
      # permission can edit the description.
      if is_description:
        raise permissions.PermissionException(
            'Users lack permission EditIssue in issue')
      # If we're adding a comment, we must have AddIssueComment permission and
      # verify it's size.
      if comment_content:
        self._AssertPermInIssue(issue, permissions.ADD_ISSUE_COMMENT)
      # If we're modifying the issue, check that we only modify the fields we're
      # allowed to edit.
      if delta != tracker_pb2.IssueDelta():
        allowed_delta = tracker_pb2.IssueDelta()
        if self._UserCanUsePermInIssue(issue, permissions.EDIT_ISSUE_STATUS):
          allowed_delta.status = delta.status
        if self._UserCanUsePermInIssue(issue, permissions.EDIT_ISSUE_SUMMARY):
          allowed_delta.summary = delta.summary
        if self._UserCanUsePermInIssue(issue, permissions.EDIT_ISSUE_OWNER):
          allowed_delta.owner_id = delta.owner_id
        if self._UserCanUsePermInIssue(issue, permissions.EDIT_ISSUE_CC):
          allowed_delta.cc_ids_add = delta.cc_ids_add
          allowed_delta.cc_ids_remove = delta.cc_ids_remove
        if delta != allowed_delta:
          raise permissions.PermissionException(
              'Users lack permission EditIssue in issue')

    if delta.merged_into:
      # Reject attempts to merge an issue into an issue we cannot view and edit.
      merged_into_issue = self.GetIssue(
          delta.merged_into, use_cache=False, allow_viewing_deleted=True)
      self._AssertPermInIssue(issue, permissions.EDIT_ISSUE)
      # Reject attempts to merge an issue into itself.
      if issue.issue_id == delta.merged_into:
        raise exceptions.InputException(
          'Cannot merge an issue into itself.')

    # Reject comments that are too long.
    if comment_content and len(
        comment_content) > tracker_constants.MAX_COMMENT_CHARS:
      raise exceptions.InputException('Comment is too long')

    # Reject attempts to block on issue on itself.
    if (issue.issue_id in delta.blocked_on_add
        or issue.issue_id in delta.blocking_add):
      raise exceptions.InputException(
        'Cannot block an issue on itself.')

    project = self.GetProject(issue.project_id)
    config = self.GetProjectConfig(issue.project_id)

    # Reject attempts to edit restricted fields that the user cannot change.
    field_ids = [fv.field_id for fv in delta.field_vals_add]
    field_ids.extend([fvr.field_id for fvr in delta.field_vals_remove])
    field_ids.extend(delta.fields_clear)
    labels = itertools.chain(delta.labels_add, delta.labels_remove)
    self._AssertUserCanEditFieldsAndEnumMaskedLabels(
        project, config, field_ids, labels)

    old_owner_id = tracker_bizobj.GetOwnerId(issue)

    if attachments:
      with self.mc.profiler.Phase('Accounting for quota'):
        new_bytes_used = tracker_helpers.ComputeNewQuotaBytesUsed(
            project, attachments)
        self.services.project.UpdateProject(
            self.mc.cnxn, issue.project_id,
            attachment_bytes_used=new_bytes_used)

    with self.mc.profiler.Phase('Validating the issue change'):
      # If the owner changed, it must be a project member.
      if (delta.owner_id is not None and delta.owner_id != issue.owner_id):
        parsed_owner_valid, msg = tracker_helpers.IsValidIssueOwner(
          self.mc.cnxn, project, delta.owner_id, self.services)
        if not parsed_owner_valid:
          raise exceptions.InputException(msg)

    if kept_attachments:
      with self.mc.profiler.Phase('Filtering kept attachments'):
        kept_attachments = tracker_helpers.FilterKeptAttachments(
            is_description, kept_attachments, self.ListIssueComments(issue),
            None)

    with self.mc.profiler.Phase('Updating issue %r' % (issue.issue_id)):
      _amendments, comment_pb = self.services.issue.DeltaUpdateIssue(
          self.mc.cnxn, self.services, self.mc.auth.user_id, issue.project_id,
          config, issue, delta, comment=comment_content,
          attachments=attachments, is_description=is_description,
          kept_attachments=kept_attachments, inbound_message=inbound_message)

    with self.mc.profiler.Phase('Following up after issue update'):
      if delta.merged_into:
        new_starrers = tracker_helpers.GetNewIssueStarrers(
            self.mc.cnxn, self.services, [issue.issue_id],
            delta.merged_into)
        merged_into_project = self.GetProject(merged_into_issue.project_id)
        tracker_helpers.AddIssueStarrers(
            self.mc.cnxn, self.services, self.mc,
            delta.merged_into, merged_into_project, new_starrers)
        # Load target issue again to get the updated star count.
        merged_into_issue = self.GetIssue(
            merged_into_issue.issue_id, use_cache=False)
        merge_comment_pb = tracker_helpers.MergeCCsAndAddComment(
            self.services, self.mc, issue, merged_into_issue)
        # Send notification emails.
        hostport = framework_helpers.GetHostPort(
            project_name=merged_into_project.project_name)
        reporter_id = self.mc.auth.user_id
        send_notifications.PrepareAndSendIssueChangeNotification(
            merged_into_issue.issue_id,
            hostport,
            reporter_id,
            send_email=send_email,
            comment_id=merge_comment_pb.id)
      self.services.project.UpdateRecentActivity(
          self.mc.cnxn, issue.project_id)

    with self.mc.profiler.Phase('Generating notifications'):
      if comment_pb:
        hostport = framework_helpers.GetHostPort(
            project_name=project.project_name)
        reporter_id = self.mc.auth.user_id
        send_notifications.PrepareAndSendIssueChangeNotification(
            issue.issue_id, hostport, reporter_id,
            send_email=send_email, old_owner_id=old_owner_id,
            comment_id=comment_pb.id)
        delta_blocked_on_iids = delta.blocked_on_add + delta.blocked_on_remove
        send_notifications.PrepareAndSendIssueBlockingNotification(
            issue.issue_id, hostport, delta_blocked_on_iids,
            reporter_id, send_email=send_email)

  def ModifyIssues(
      self,
      issue_id_delta_pairs,
      attachment_uploads=None,
      comment_content=None,
      send_email=True):
    # type: (Sequence[Tuple[int, IssueDelta]], Boolean, Optional[str],
    #     Optional[bool]) -> Sequence[Issue]
    """Modify issues by the given deltas and returns all issues post-update.

    Note: Issues with NOOP deltas and no comment_content to add will not be
        updated and will not be returned.

    Args:
      issue_id_delta_pairs: List of Tuples containing IDs and IssueDeltas, one
        for each issue to modify.
      attachment_uploads: List of AttachmentUpload tuples to be attached to the
        new comments created for all modified issues in issue_id_delta_pairs.
      comment_content: The text for the comment this issue change will use.
      send_email: Whether this change sends an email or not.

    Returns:
      List of modified issues.
    """

    main_issue_ids = {issue_id for issue_id, _delta in issue_id_delta_pairs}
    issues_by_id = self.GetIssuesDict(main_issue_ids, use_cache=False)
    issue_delta_pairs = [
        (issues_by_id[issue_id], delta)
        for (issue_id, delta) in issue_id_delta_pairs
    ]

    # PHASE 1: Prepare these changes and assert they can be made.
    self._AssertUserCanModifyIssues(
        issue_delta_pairs, False, comment_content=comment_content)
    new_bytes_by_pid = tracker_helpers.PrepareIssueChanges(
        self.mc.cnxn,
        issue_delta_pairs,
        self.services,
        attachment_uploads=attachment_uploads,
        comment_content=comment_content)
    # TODO(crbug.com/monorail/8074): Assert we do not update more than 100
    # issues at once.

    # PHASE 2: Organize data. tracker_helpers.GroupUniqueDeltaIssues()
    (_unique_deltas, issues_for_unique_deltas
    ) = tracker_helpers.GroupUniqueDeltaIssues(issue_delta_pairs)

    # PHASE 3-4: Modify issues in RAM.
    changes = tracker_helpers.ApplyAllIssueChanges(
        self.mc.cnxn, issue_delta_pairs, self.services)

    # PHASE 5: Apply filter rules.
    inflight_issues = changes.issues_to_update_dict.values()
    project_ids = list(
        {issue.project_id for issue in inflight_issues})
    configs_by_id = self.services.config.GetProjectConfigs(
        self.mc.cnxn, project_ids)
    with exceptions.ErrorAggregator(exceptions.FilterRuleException) as err_agg:
      for issue in inflight_issues:
        config = configs_by_id[issue.project_id]

        # Update closed timestamp before filter rules because filter rules
        # may affect them.
        old_effective_status = changes.old_statuses_by_iid.get(issue.issue_id)
        tracker_helpers.UpdateClosedTimestamp(
            config, issue, old_effective_status)

        filterrules_helpers.ApplyFilterRules(
              self.mc.cnxn, self.services, issue, config)
        if issue.derived_errors:
          err_agg.AddErrorMessage('/n'.join(issue.derived_errors))

        # Update closed timestamp after filter rules because filter rules
        # could change effective status.
        tracker_helpers.UpdateClosedTimestamp(
            config, issue, old_effective_status)

    # PHASE 6: Update modified timestamps for issues in RAM.
    all_involved_iids = main_issue_ids.union(
        changes.issues_to_update_dict.keys())

    now_timestamp = int(time.time())
    # Add modified timestamps for issues with amendments.
    for iid in all_involved_iids:
      issue = changes.issues_to_update_dict.get(iid, issues_by_id.get(iid))
      issue_modified = iid in changes.issues_to_update_dict

      if not (issue_modified or comment_content or attachment_uploads):
        # Skip issues that have neither amendments or comment changes.
        continue

      old_owner = changes.old_owners_by_iid.get(issue.issue_id)
      old_status = changes.old_statuses_by_iid.get(issue.issue_id)
      old_components = changes.old_components_by_iid.get(issue.issue_id)

      # Adding this issue to issues_to_update, so its modified_timestamp gets
      # updated in PHASE 7's UpdateIssues() call. Issues with NOOP changes
      # but still need a new comment added for `comment_content` or
      # `attachments` are added back here.
      changes.issues_to_update_dict[issue.issue_id] = issue

      issue.modified_timestamp = now_timestamp

      if (iid in changes.old_owners_by_iid and
          old_owner != tracker_bizobj.GetOwnerId(issue)):
        issue.owner_modified_timestamp = now_timestamp

      if (iid in changes.old_statuses_by_iid and
          old_status != tracker_bizobj.GetStatus(issue)):
        issue.status_modified_timestamp = now_timestamp

      if (iid in changes.old_components_by_iid and
          set(old_components) != set(issue.component_ids)):
        issue.component_modified_timestamp = now_timestamp

    # PHASE 7: Apply changes to DB: update issues, combine starrers
    # for merged issues, create issue comments, enqueue issues for
    # re-indexing.
    if changes.issues_to_update_dict:
      self.services.issue.UpdateIssues(
          self.mc.cnxn, changes.issues_to_update_dict.values(), commit=False)
    comments_by_iid = {}
    impacted_comments_by_iid = {}

    # changes.issues_to_update includes all main issues or impacted
    # issues with updated fields and main issues that had noop changes
    # but still need a comment created for `comment_content` or `attachments`.
    for iid, issue in changes.issues_to_update_dict.items():
      # Update starrers for merged issues.
      new_starrers = changes.new_starrers_by_iid.get(iid)
      if new_starrers:
        self.services.issue_star.SetStarsBatch_SkipIssueUpdate(
            self.mc.cnxn, iid, new_starrers, True, commit=False)

      # Create new issue comment for main issue changes.
      amendments = changes.amendments_by_iid.get(iid)
      if (amendments or comment_content or
          attachment_uploads) and iid in main_issue_ids:
        comments_by_iid[iid] = self.services.issue.CreateIssueComment(
            self.mc.cnxn,
            issue,
            self.mc.auth.user_id,
            comment_content,
            amendments=amendments,
            attachments=attachment_uploads,
            commit=False)

      # Create new issue comment for impacted issue changes.
      # ie: when an issue is marked as blockedOn another or similar.
      imp_amendments = changes.imp_amendments_by_iid.get(iid)
      if imp_amendments:
        filtered_imp_amendments = []
        content = ''
        # Represent MERGEDINTO Amendments for impacted issues with
        # comment content instead to be consistent with previous behavior
        # and so users can tell whether a merged change comment on an issue
        # is a change in the issue's merged_into or a change in another
        # issue's merged_into.
        for am in imp_amendments:
          if am.field is tracker_pb2.FieldID.MERGEDINTO and am.newvalue:
            for value in am.newvalue.split():
              if value.startswith('-'):
                content += UNMERGE_COMMENT % value.strip('-')
              else:
                content += MERGE_COMMENT % value
          else:
            filtered_imp_amendments.append(am)

        impacted_comments_by_iid[iid] = self.services.issue.CreateIssueComment(
            self.mc.cnxn,
            issue,
            self.mc.auth.user_id,
            content,
            amendments=filtered_imp_amendments,
            commit=False)

    # Update used bytes for each impacted project.
    for pid, new_bytes_used in new_bytes_by_pid.items():
      self.services.project.UpdateProject(
          self.mc.cnxn, pid, attachment_bytes_used=new_bytes_used, commit=False)

    # Reindex issues and commit all DB changes.
    issues_to_reindex = set(
        comments_by_iid.keys() + impacted_comments_by_iid.keys())
    if issues_to_reindex:
      self.services.issue.EnqueueIssuesForIndexing(
          self.mc.cnxn, issues_to_reindex, commit=False)
      # We only commit if there are issues to reindex. No issues to reindex
      # means there were no updates that need a commit.
      self.mc.cnxn.Commit()

    # PHASE 8: Send notifications for each group of issues from Phase 2.
    # Fetch hostports.
    hostports_by_pid = {}
    for iid, issue in changes.issues_to_update_dict.items():
      # Note: issues_to_update only include issues with changes in metadata.
      # If iid is not in issues_to_update, the issue may still have a new
      # comment that we want to send notifications for.
      issue = changes.issues_to_update_dict.get(iid, issues_by_id.get(iid))

      if issue.project_id not in hostports_by_pid:
        hostports_by_pid[issue.project_id] = framework_helpers.GetHostPort(
            project_name=issue.project_name)
    # Send emails for main changes in issues by unique delta.
    for issues in issues_for_unique_deltas:
      # Group issues for each unique delta by project because
      # SendIssueBulkChangeNotification cannot handle cross-project
      # notifications and hostports are specific to each project.
      issues_by_pid = collections.defaultdict(set)
      for issue in issues:
        issues_by_pid[issue.project_id].add(issue)
      for project_issues in issues_by_pid.values():
        # Send one email to involved users for the issue.
        if len(project_issues) == 1:
          (project_issue,) = project_issues
          self._ModifyIssuesNotifyForDelta(
              project_issue, changes, comments_by_iid, hostports_by_pid,
              send_email)
        # Send one bulk email for users involved in all updated issues.
        else:
          self._ModifyIssuesBulkNotifyForDelta(
              project_issues,
              changes,
              hostports_by_pid,
              send_email,
              comment_content=comment_content)

    # Send emails for changes to impacted issues.
    for issue_id, comment_pb in impacted_comments_by_iid.items():
      issue = changes.issues_to_update_dict[issue_id]
      hostport = hostports_by_pid[issue.project_id]
      # We do not need to track old owners because the only owner change
      # that could have happened for impacted issues' changes is a change from
      # no owner to a derived owner.
      send_notifications.PrepareAndSendIssueChangeNotification(
          issue_id, hostport, self.mc.auth.user_id, comment_id=comment_pb.id,
          send_email=send_email)

    return [
        issues_by_id[iid] for iid in main_issue_ids if iid in comments_by_iid
    ]

  def _ModifyIssuesNotifyForDelta(
      self, issue, changes, comments_by_iid, hostports_by_pid, send_email):
    # type: (Issue, tracker_helpers._IssueChangesTuple,
    #     Mapping[int, IssueComment], Mapping[int, str], bool) -> None
    comment_pb = comments_by_iid.get(issue.issue_id)
    # Existence of a comment_pb means there were updates to the issue or
    # comment_content added to the issue that should trigger
    # notifications.
    if comment_pb:
      hostport = hostports_by_pid[issue.project_id]
      old_owner_id = changes.old_owners_by_iid.get(issue.issue_id)
      send_notifications.PrepareAndSendIssueChangeNotification(
          issue.issue_id,
          hostport,
          self.mc.auth.user_id,
          old_owner_id=old_owner_id,
          comment_id=comment_pb.id,
          send_email=send_email)

  def _ModifyIssuesBulkNotifyForDelta(
      self, issues, changes, hostports_by_pid, send_email,
      comment_content=None):
    # type: (Collection[Issue], _IssueChangesTuple, Mapping[int, str], bool,
    #     Optional[str]) -> None
    iids = {issue.issue_id for issue in issues}
    old_owner_ids = [
        changes.old_owners_by_iid.get(iid)
        for iid in iids
        if changes.old_owners_by_iid.get(iid)
    ]
    amendments = []
    for iid in iids:
      ams = changes.amendments_by_iid.get(iid, [])
      amendments.extend(ams)
    # Calling SendBulkChangeNotification does not require the comment_pb
    # objects only the amendments. Checking for existence of amendments
    # and comment_content is equivalent to checking for existence of new
    # comments created for these issues.
    if amendments or comment_content:
      # TODO(crbug.com/monorail/8125): Stop using UserViews for bulk
      # notifications.
      users_by_id = framework_views.MakeAllUserViews(
          self.mc.cnxn, self.services.user, old_owner_ids,
          tracker_bizobj.UsersInvolvedInAmendments(amendments))
      hostport = hostports_by_pid[issues.pop().project_id]
      send_notifications.SendIssueBulkChangeNotification(
          iids, hostport, old_owner_ids, comment_content,
          self.mc.auth.user_id, amendments, send_email, users_by_id)

  def DeleteIssue(self, issue, delete):
    """Mark or unmark the given issue as deleted."""
    self._AssertPermInIssue(issue, permissions.DELETE_ISSUE)

    with self.mc.profiler.Phase('Marking issue %r deleted' % (issue.issue_id)):
      self.services.issue.SoftDeleteIssue(
          self.mc.cnxn, issue.project_id, issue.local_id, delete,
          self.services.user)

  def FlagIssues(self, issues, flag):
    """Flag or unflag the given issues as spam."""
    for issue in issues:
      self._AssertPermInIssue(issue, permissions.FLAG_SPAM)

    issue_ids = [issue.issue_id for issue in issues]
    with self.mc.profiler.Phase('Marking issues %r as spam' % issue_ids):
      self.services.spam.FlagIssues(
          self.mc.cnxn, self.services.issue, issues, self.mc.auth.user_id,
          flag)
      if self._UserCanUsePermInIssue(issue, permissions.VERDICT_SPAM):
        self.services.spam.RecordManualIssueVerdicts(
            self.mc.cnxn, self.services.issue, issues, self.mc.auth.user_id,
            flag)

  def LookupIssuesFlaggers(self, issues):
    """Returns users who've reported the issue or its comments as spam.

    Args:
      issues: the list of issues to query.
    Returns:
      A dictionary
        {issue_id: ([issue_reporters], {comment_id: [comment_reporters]})}
      For each issue id, a tuple with the users who have flagged the issue;
      and a dictionary of users who have flagged a comment for each comment id.
    """
    for issue in issues:
      self._AssertUserCanViewIssue(issue)

    issue_ids = [issue.issue_id for issue in issues]
    with self.mc.profiler.Phase('Looking up flaggers for %s' % issue_ids):
      reporters = self.services.spam.LookupIssuesFlaggers(
          self.mc.cnxn, issue_ids)

    return reporters

  def LookupIssueFlaggers(self, issue):
    """Returns users who've reported the issue or its comments as spam.

    Args:
      issue: the issue to query.
    Returns:
      A tuple
        ([issue_reporters], {comment_id: [comment_reporters]})
      With the users who have flagged the issue; and a dictionary of users who
      have flagged a comment for each comment id.
    """
    return self.LookupIssuesFlaggers([issue])[issue.issue_id]

  def GetIssuePositionInHotlist(
      self, current_issue, hotlist, can, sort_spec, group_by_spec):
    # type: (Issue, Hotlist, int, str, str) -> (int, int, int, int)
    """Get index info of an issue within a hotlist.

    Args:
      current_issue: the currently viewed issue.
      hotlist: the hotlist this flipper is flipping through.
      can: int "canned query" number to scope the visible issues.
      sort_spec: string that lists the sort order.
      group_by_spec: string that lists the grouping order.
    """
    issues_list = self.services.issue.GetIssues(self.mc.cnxn,
        [item.issue_id for item in hotlist.items])
    project_ids = hotlist_helpers.GetAllProjectsOfIssues(issues_list)
    config_list = hotlist_helpers.GetAllConfigsOfProjects(
        self.mc.cnxn, project_ids, self.services)
    harmonized_config = tracker_bizobj.HarmonizeConfigs(config_list)
    (sorted_issues, _hotlist_issues_context,
     _users) = hotlist_helpers.GetSortedHotlistIssues(
         self.mc.cnxn, hotlist.items, issues_list, self.mc.auth,
         can, sort_spec, group_by_spec, harmonized_config, self.services,
         self.mc.profiler)
    (prev_iid, cur_index,
     next_iid) = features_bizobj.DetermineHotlistIssuePosition(
         current_issue, [issue.issue_id for issue in sorted_issues])
    total_count = len(sorted_issues)
    return prev_iid, cur_index, next_iid, total_count

  def RerankBlockedOnIssues(self, issue, moved_id, target_id, split_above):
    """Rerank the blocked on issues for issue_id.

    Args:
      issue: The issue to modify.
      moved_id: The id of the issue to move.
      target_id: The id of the issue to move |moved_issue| to.
      split_above: Whether to move |moved_issue| before or after |target_issue|.
    """
    # Make sure the user has permission to edit the issue.
    self._AssertPermInIssue(issue, permissions.EDIT_ISSUE)
    # Make sure the moved and target issues are in the blocked-on list.
    if moved_id not in issue.blocked_on_iids:
      raise exceptions.InputException(
          'The issue to move is not in the blocked-on list.')
    if target_id not in issue.blocked_on_iids:
      raise exceptions.InputException(
          'The target issue is not in the blocked-on list.')

    phase_name = 'Moving issue %r %s issue %d.' % (
        moved_id, 'above' if split_above else 'below', target_id)
    with self.mc.profiler.Phase(phase_name):
      lower, higher = tracker_bizobj.SplitBlockedOnRanks(
          issue, target_id, split_above,
          [iid for iid in issue.blocked_on_iids if iid != moved_id])
      rank_changes = rerank_helpers.GetInsertRankings(
          lower, higher, [moved_id])
      if rank_changes:
        self.services.issue.ApplyIssueRerank(
            self.mc.cnxn, issue.issue_id, rank_changes)

  # FUTURE: GetIssuePermissionsForUser()

  # FUTURE: CreateComment()


  # TODO(crbug.com/monorail/7520): Delete when usages removed.
  def ListIssueComments(self, issue):
    """Return comments on the specified viewable issue."""
    self._AssertUserCanViewIssue(issue)

    with self.mc.profiler.Phase('getting comments for %r' % issue.issue_id):
      comments = self.services.issue.GetCommentsForIssue(
          self.mc.cnxn, issue.issue_id)

    return comments


  def SafeListIssueComments(
      self, issue_id, max_items, start, approval_id=None):
    # type: (tracker_pb2.Issue, int, int, Optional[int]) -> ListResult
    """Return comments on the issue, filtering non-viewable content.

    TODO(crbug.com/monorail/7520): Rename to ListIssueComments.

    Note: This returns `deleted_by`, but it should only be used for the purposes
    of determining whether the comment is deleted. The viewer may not have
    access to view who deleted the comment.

    Args:
      issue_id: The issue for which we're listing comments.
      max_items: The maximum number of comments to return.
      start: The index of the start position in the list of comments.
      approval_id: Whether to only return comments on this approval.

    Returns:
      A work_env.ListResult namedtuple with the comments for the issue.

    Raises:
      PermissionException: The logged-in user is not allowed to view the issue.
    """
    if start < 0:
      raise exceptions.InputException('Invalid `start`: %d' % start)
    if max_items < 0:
      raise exceptions.InputException('Invalid `max_items`: %d' % max_items)

    with self.mc.profiler.Phase('getting comments for %r' % issue_id):
      issue = self.GetIssue(issue_id)
      comments = self.services.issue.GetCommentsForIssue(self.mc.cnxn, issue_id)
      _, comment_reporters = self.LookupIssueFlaggers(issue)
      users_involved_in_comments = tracker_bizobj.UsersInvolvedInCommentList(
          comments)
      users_by_id = framework_views.MakeAllUserViews(
          self.mc.cnxn, self.services.user, users_involved_in_comments)

    with self.mc.profiler.Phase('getting perms for comments'):
      project = self.GetProjectByName(issue.project_name)
      self.mc.LookupLoggedInUserPerms(project)
      config = self.GetProjectConfig(project.project_id)
      perms = permissions.UpdateIssuePermissions(
          self.mc.perms,
          project,
          issue,
          self.mc.auth.effective_ids,
          config=config)

    # TODO(crbug.com/monorail/7525): Check values, and return next_start.
    end = start + max_items
    filtered_comments = []
    with self.mc.profiler.Phase('converting comments'):
      for comment in comments:
        if approval_id and comment.approval_id != approval_id:
          continue
        commenter = users_by_id[comment.user_id]

        _can_flag, is_flagged = permissions.CanFlagComment(
            comment, commenter, comment_reporters.get(comment.id, []),
            self.mc.auth.user_id, perms)
        can_view = permissions.CanViewComment(
            comment, commenter, self.mc.auth.user_id, perms)
        can_view_inbound_message = permissions.CanViewInboundMessage(
            comment, self.mc.auth.user_id, perms)

        # By default, all fields should get filtered out.
        # i.e. this is an allowlist rather than a denylist to reduce leaking
        # info.
        filtered_comment = tracker_pb2.IssueComment(
            id=comment.id,
            issue_id=comment.issue_id,
            project_id=comment.project_id,
            approval_id=comment.approval_id,
            timestamp=comment.timestamp,
            deleted_by=comment.deleted_by,
            sequence=comment.sequence,
            is_spam=is_flagged,
            is_description=comment.is_description,
            description_num=comment.description_num)
        if can_view:
          filtered_comment.content = comment.content
          filtered_comment.user_id = comment.user_id
          filtered_comment.amendments.extend(comment.amendments)
          filtered_comment.attachments.extend(comment.attachments)
          filtered_comment.importer_id = comment.importer_id
          if can_view_inbound_message:
            filtered_comment.inbound_message = comment.inbound_message
        filtered_comments.append(filtered_comment)
    next_start = None
    if end < len(filtered_comments):
      next_start = end
    return ListResult(filtered_comments[start:end], next_start)

  # FUTURE: UpdateComment()

  def DeleteComment(self, issue, comment, delete):
    """Mark or unmark a comment as deleted by the current user."""
    self._AssertUserCanDeleteComment(issue, comment)
    if comment.is_spam and self.mc.auth.user_id == comment.user_id:
      raise permissions.PermissionException('Cannot delete comment.')

    with self.mc.profiler.Phase(
        'deleting issue %r comment %r' % (issue.issue_id, comment.id)):
      self.services.issue.SoftDeleteComment(
          self.mc.cnxn, issue, comment, self.mc.auth.user_id,
          self.services.user, delete=delete)

  def DeleteAttachment(self, issue, comment, attachment_id, delete):
    """Mark or unmark a comment attachment as deleted by the current user."""
    # A user can delete an attachment iff they can delete a comment.
    self._AssertUserCanDeleteComment(issue, comment)

    phase_message = 'deleting issue %r comment %r attachment %r' % (
        issue.issue_id, comment.id, attachment_id)
    with self.mc.profiler.Phase(phase_message):
      self.services.issue.SoftDeleteAttachment(
          self.mc.cnxn, issue, comment, attachment_id, self.services.user,
          delete=delete)

  def FlagComment(self, issue, comment, flag):
    """Mark or unmark a comment as spam."""
    self._AssertPermInIssue(issue, permissions.FLAG_SPAM)
    with self.mc.profiler.Phase(
        'flagging issue %r comment %r' % (issue.issue_id, comment.id)):
      self.services.spam.FlagComment(
          self.mc.cnxn, issue, comment.id, comment.user_id,
          self.mc.auth.user_id, flag)
      if self._UserCanUsePermInIssue(issue, permissions.VERDICT_SPAM):
        self.services.spam.RecordManualCommentVerdict(
            self.mc.cnxn, self.services.issue, self.services.user, comment.id,
            self.mc.auth.user_id, flag)

  def StarIssue(self, issue, starred):
    # type: (Issue, bool) -> Issue
    """Set or clear a star on the given issue for the signed in user."""
    if not self.mc.auth.user_id:
      raise permissions.PermissionException('Anon cannot star issues')
    self._AssertPermInIssue(issue, permissions.SET_STAR)

    with self.mc.profiler.Phase('starring issue %r' % issue.issue_id):
      config = self.services.config.GetProjectConfig(
          self.mc.cnxn, issue.project_id)
      self.services.issue_star.SetStar(
          self.mc.cnxn, self.services, config, issue.issue_id,
          self.mc.auth.user_id, starred)
    return self.services.issue.GetIssue(self.mc.cnxn, issue.issue_id)

  def IsIssueStarred(self, issue, cnxn=None):
    """Return True if the given issue is starred by the signed in user."""
    self._AssertUserCanViewIssue(issue)

    with self.mc.profiler.Phase('checking star %r' % issue.issue_id):
      return self.services.issue_star.IsItemStarredBy(
          cnxn or self.mc.cnxn, issue.issue_id, self.mc.auth.user_id)

  def ListStarredIssueIDs(self):
    """Return a list of the issue IDs that the current issue has starred."""
    # This returns an unfiltered list of issue_ids.  Permissions will be
    # applied if and when the caller attempts to load each issue.

    with self.mc.profiler.Phase('getting stars %r' % self.mc.auth.user_id):
      return self.services.issue_star.LookupStarredItemIDs(
          self.mc.cnxn, self.mc.auth.user_id)

  def SnapshotCountsQuery(self, project, timestamp, group_by, label_prefix=None,
                          query=None, canned_query=None, hotlist=None):
    """Query IssueSnapshots for daily counts.

    See chart_svc.QueryIssueSnapshots for more detail on arguments.

    Args:
      project (Project): Project to search.
      timestamp (int): Will query for snapshots at this timestamp.
      group_by (str): 2nd dimension, see QueryIssueSnapshots for options.
      label_prefix (str): Required for label queries. Only returns results
        with the supplied prefix.
      query (str, optional): If supplied, will parse & apply query conditions.
      canned_query (str, optional): Parsed canned query.
      hotlist (Hotlist, optional): Hotlist to search under (in lieu of project).

    Returns:
      1. A dict of {name: count} for each item in group_by.
      2. A list of any unsupported query conditions in query.
    """
    # This returns counts of viewable issues.
    with self.mc.profiler.Phase('querying snapshot counts'):
      return self.services.chart.QueryIssueSnapshots(
        self.mc.cnxn, self.services, timestamp, self.mc.auth.effective_ids,
        project, self.mc.perms, group_by=group_by, label_prefix=label_prefix,
        query=query, canned_query=canned_query, hotlist=hotlist)

  ### User methods

  # TODO(crbug/monorail/7238): rewrite this method to call BatchGetUsers.
  def GetUser(self, user_id):
    # type: (int) -> User
    """Return the user with the given ID."""

    return self.BatchGetUsers([user_id])[0]

  def BatchGetUsers(self, user_ids):
    # type: (Sequence[int]) -> Sequence[User]
    """Return all Users for given User IDs.

    Args:
      user_ids: list of User IDs.

    Returns:
      A list of User objects in the same order as the given User IDs.

    Raises:
      NoSuchUserException if a User for a given User ID is not found.
    """
    users_by_id = self.services.user.GetUsersByIDs(
        self.mc.cnxn, user_ids, skip_missed=True)
    users = []
    for user_id in user_ids:
      user = users_by_id.get(user_id)
      if not user:
        raise exceptions.NoSuchUserException(
            'No User with ID %s found' % user_id)
      users.append(user)
    return users

  def GetMemberships(self, user_id):
    """Return the user group ids for the given user visible to the requester."""
    group_ids = self.services.usergroup.LookupMemberships(self.mc.cnxn, user_id)
    if user_id == self.mc.auth.user_id:
      return group_ids
    (member_ids_by_ids, owner_ids_by_ids
    ) = self.services.usergroup.LookupAllMembers(
        self.mc.cnxn, group_ids)
    settings_by_id = self.services.usergroup.GetAllGroupSettings(
        self.mc.cnxn, group_ids)

    (owned_project_ids, membered_project_ids,
     contrib_project_ids) = self.services.project.GetUserRolesInAllProjects(
         self.mc.cnxn, self.mc.auth.effective_ids)
    project_ids = owned_project_ids.union(
        membered_project_ids).union(contrib_project_ids)

    visible_group_ids = []
    for group_id, group_settings in settings_by_id.items():
      member_ids = member_ids_by_ids.get(group_id)
      owner_ids = owner_ids_by_ids.get(group_id)
      if permissions.CanViewGroupMembers(
          self.mc.perms, self.mc.auth.effective_ids, group_settings,
          member_ids, owner_ids, project_ids):
        visible_group_ids.append(group_id)

    return visible_group_ids

  def ListReferencedUsers(self, emails):
    """Return a list of the given emails' User PBs, plus linked account ids.

    Args:
      emails: list of emails of users to look up.

    Returns:
      A pair (users, linked_users_ids) where users is an unsorted list of
      User PBs and linked_user_ids is a list of user IDs of any linked accounts.
    """
    with self.mc.profiler.Phase('getting existing users'):
      user_id_dict = self.services.user.LookupExistingUserIDs(
          self.mc.cnxn, emails)
      users_by_id = self.services.user.GetUsersByIDs(
          self.mc.cnxn, list(user_id_dict.values()))
      user_list = list(users_by_id.values())

      linked_user_ids = []
      for user in user_list:
        if user.linked_parent_id:
          linked_user_ids.append(user.linked_parent_id)
        linked_user_ids.extend(user.linked_child_ids)

    return user_list, linked_user_ids

  def StarUser(self, user_id, starred):
    """Star or unstar the specified user.

    Args:
      user_id: int ID of the user to star/unstar.
      starred: true to add a star, false to remove it.

    Returns:
      Nothing.

    Raises:
      NoSuchUserException: There is no user with that ID.
    """
    if not self.mc.auth.user_id:
      raise exceptions.InputException('No current user specified')

    with self.mc.profiler.Phase('(un)starring user %r' % user_id):
      # Make sure the user exists and user has permission to see it.
      self.services.user.LookupUserEmail(self.mc.cnxn, user_id)
      self.services.user_star.SetStar(
          self.mc.cnxn, user_id, self.mc.auth.user_id, starred)

  def IsUserStarred(self, user_id):
    """Return True if the current user has starred the given user.

    Args:
      user_id: int ID of the user to check.

    Returns:
      True if starred.

    Raises:
      NoSuchUserException: There is no user with that ID.
    """
    if user_id is None:
      raise exceptions.InputException('No user specified')

    if not self.mc.auth.user_id:
      return False

    with self.mc.profiler.Phase('checking user star %r' % user_id):
      # Make sure the user exists.
      self.services.user.LookupUserEmail(self.mc.cnxn, user_id)
      return self.services.user_star.IsItemStarredBy(
        self.mc.cnxn, user_id, self.mc.auth.user_id)

  def GetUserStarCount(self, user_id):
    """Return the number of times the user has been starred.

    Args:
      user_id: int ID of the user to check.

    Returns:
      The number of times the user has been starred.

    Raises:
      NoSuchUserException: There is no user with that ID.
    """
    if user_id is None:
      raise exceptions.InputException('No user specified')

    with self.mc.profiler.Phase('counting stars for user %r' % user_id):
      # Make sure the user exists.
      self.services.user.LookupUserEmail(self.mc.cnxn, user_id)
      return self.services.user_star.CountItemStars(self.mc.cnxn, user_id)

  def GetPendingLinkedInvites(self, user_id=None):
    """Return info about a user's linked account invites."""
    with self.mc.profiler.Phase('checking linked account invites'):
      result = self.services.user.GetPendingLinkedInvites(
          self.mc.cnxn, user_id or self.mc.auth.user_id)
      return result

  def InviteLinkedParent(self, parent_email):
    """Invite a matching account to be my parent."""
    if not parent_email:
      raise exceptions.InputException('No parent account specified')
    if not self.mc.auth.user_id:
      raise permissions.PermissionException('Anon cannot link accounts')
    with self.mc.profiler.Phase('Validating proposed parent'):
      # We only offer self-serve account linking to matching usernames.
      (p_username, p_domain,
       _obs_username, _obs_email) = framework_bizobj.ParseAndObscureAddress(
          parent_email)
      c_view = self.mc.auth.user_view
      if p_username != c_view.username:
        logging.info('Username %r != %r', p_username, c_view.username)
        raise exceptions.InputException('Linked account names must match')
      allowed_domains = settings.linkable_domains.get(c_view.domain, [])
      if p_domain not in allowed_domains:
        logging.info('parent domain %r is not in list for %r: %r',
                     p_domain, c_view.domain, allowed_domains)
        raise exceptions.InputException('Linked account unsupported domain')
      parent_id = self.services.user.LookupUserID(self.mc.cnxn, parent_email)
    with self.mc.profiler.Phase('Creating linked account invite'):
      self.services.user.InviteLinkedParent(
          self.mc.cnxn, parent_id, self.mc.auth.user_id)

  def AcceptLinkedChild(self, child_id):
    """Accept an invitation from a child account."""
    with self.mc.profiler.Phase('Accept linked account invite'):
      self.services.user.AcceptLinkedChild(
          self.mc.cnxn, self.mc.auth.user_id, child_id)

  def UnlinkAccounts(self, parent_id, child_id):
    """Delete a linked-account relationship."""
    if (self.mc.auth.user_id != parent_id and
        self.mc.auth.user_id != child_id):
      permitted = self.mc.perms.CanUsePerm(
        permissions.EDIT_OTHER_USERS, self.mc.auth.effective_ids, None, [])
      if not permitted:
        raise permissions.PermissionException(
          'User lacks permission to unlink accounts')

    with self.mc.profiler.Phase('Unlink accounts'):
      self.services.user.UnlinkAccounts(self.mc.cnxn, parent_id, child_id)

  def UpdateUserSettings(self, user, **kwargs):
    """Update the preferences of the specified user.

    Args:
      user: User PB for the user to update.
      keyword_args: dictionary of setting names mapped to new values.
    """
    if not user or not user.user_id:
      raise exceptions.InputException('Cannot update user settings for anon.')

    with self.mc.profiler.Phase(
        'updating settings for %s with %s' % (self.mc.auth.user_id, kwargs)):
      self.services.user.UpdateUserSettings(
          self.mc.cnxn, user.user_id, user, **kwargs)

  def GetUserPrefs(self, user_id):
    """Get the UserPrefs for the specified user."""
    # Anon user always has default prefs.
    if not user_id:
      return user_pb2.UserPrefs(user_id=0)
    if user_id != self.mc.auth.user_id:
      if not self.mc.perms.HasPerm(permissions.EDIT_OTHER_USERS, None, None):
        raise permissions.PermissionException(
            'Only site admins may see other users\' preferences')
    with self.mc.profiler.Phase('Getting prefs for %s' % user_id):
      userprefs = self.services.user.GetUserPrefs(self.mc.cnxn, user_id)

    # Hard-coded user prefs for at-risk users that should use "corp mode".
    # For some users we mark all of their new issues as Restrict-View-Google.
    # Others see a "public issue" warning when commenting on public issues.
    # TODO(crbug.com/monorail/5462):
    # Remove when user group preferences are implemented.
    if framework_bizobj.IsRestrictNewIssuesUser(self.mc.cnxn, self.services,
                                                user_id):
      # Copy so that cached version is not modified.
      userprefs = user_pb2.UserPrefs(user_id=user_id, prefs=userprefs.prefs)
      if 'restrict_new_issues' not in {pref.name for pref in userprefs.prefs}:
        userprefs.prefs.append(user_pb2.UserPrefValue(
            name='restrict_new_issues', value='true'))
    if framework_bizobj.IsPublicIssueNoticeUser(self.mc.cnxn, self.services,
                                                user_id):
      # Copy so that cached version is not modified.
      userprefs = user_pb2.UserPrefs(user_id=user_id, prefs=userprefs.prefs)
      if 'public_issue_notice' not in {pref.name for pref in userprefs.prefs}:
        userprefs.prefs.append(user_pb2.UserPrefValue(
            name='public_issue_notice', value='true'))

    return userprefs

  def SetUserPrefs(self, user_id, prefs):
    """Set zero or more UserPrefValue for the specified user."""
    # Anon user always has default prefs.
    if not user_id:
      raise exceptions.InputException('Anon cannot have prefs')
    if user_id != self.mc.auth.user_id:
      if not self.mc.perms.HasPerm(permissions.EDIT_OTHER_USERS, None, None):
        raise permissions.PermissionException(
            'Only site admins may set other users\' preferences')
    for pref in prefs:
      error_msg = framework_bizobj.ValidatePref(pref.name, pref.value)
      if error_msg:
        raise exceptions.InputException(error_msg)
    with self.mc.profiler.Phase(
        'setting prefs for %s' % (self.mc.auth.user_id)):
      self.services.user.SetUserPrefs(self.mc.cnxn, user_id, prefs)

  # FUTURE: GetUser()
  # FUTURE: UpdateUser()
  # FUTURE: DeleteUser()
  # FUTURE: ListStarredUsers()

  def ExpungeUsers(self, emails, check_perms=True, commit=True):
    """Permanently deletes user data and removes remaining user references
       for all listed users.

      To avoid any executions that might take too long and make the site hang,
      a limit clause will be added to some operations. If any user references
      are left behind due to the cut-off, the final services.user.ExpungeUsers
      will fail because we cannot delete User rows that are still referenced
      in other tables. work_env.ExpungeUsers can be called again until all user
      references are removed and the final services.user.ExpungeUsers succeeds.
      The limit clause will not be applied in operations for tables that contain
      user_id or email columns but do not officially Reference the User table.
      E.g. SpamVerdict and SpamReport. These user references must all be removed
      before the attempt to delete rows from User is made. The limit will also
      not be applied for sets of operations where values removed in earlier
      operations would have to be known in order for later operations to
      succeed.  E.g. ExpungeUsersIngroups().
    """
    if check_perms:
      if not permissions.CanExpungeUsers(self.mc):
        raise permissions.PermissionException(
            'User is not allowed to delete users.')

    limit = 10000
    user_ids_by_email = self.services.user.LookupExistingUserIDs(
        self.mc.cnxn, emails)
    user_ids = list(user_ids_by_email.values())
    if framework_constants.DELETED_USER_ID in user_ids:
      raise exceptions.InputException(
          'Reserved deleted_user_id found in deletion request and'
          'should not be deleted')
    if not user_ids:
      logging.info('Emails %r not found in DB. No users deleted', emails)
      return

    # The operations made in the methods below can be limited.
    # We can adjust 'limit' as necessary to avoid timing out.
    self.services.issue_star.ExpungeStarsByUsers(
        self.mc.cnxn, user_ids, limit=limit)
    self.services.project_star.ExpungeStarsByUsers(
        self.mc.cnxn, user_ids, limit=limit)
    self.services.hotlist_star.ExpungeStarsByUsers(
        self.mc.cnxn, user_ids, limit=limit)
    self.services.user_star.ExpungeStarsByUsers(
        self.mc.cnxn, user_ids, limit=limit)
    for user_id in user_ids:
      self.services.user_star.ExpungeStars(
          self.mc.cnxn, user_id, commit=False, limit=limit)

    self.services.features.ExpungeQuickEditsByUsers(
        self.mc.cnxn, user_ids, limit=limit)
    self.services.features.ExpungeSavedQueriesByUsers(
        self.mc.cnxn, user_ids, limit=limit)

    self.services.template.ExpungeUsersInTemplates(
        self.mc.cnxn, user_ids, limit=limit)
    self.services.config.ExpungeUsersInConfigs(
        self.mc.cnxn, user_ids, limit=limit)

    self.services.project.ExpungeUsersInProjects(
        self.mc.cnxn, user_ids, limit=limit)

    # The upcoming operations cannot be limited with 'limit'.
    # So it's possible that these operations below may lead to timing out
    # and ExpungeUsers will have to run again to fully delete all users.
    # We commit the above operations here, so if a failure does happen
    # below, the second run of ExpungeUsers will have less work to do.
    if commit:
      self.mc.cnxn.Commit()

    affected_issue_ids = self.services.issue.ExpungeUsersInIssues(
        self.mc.cnxn, user_ids_by_email, limit=limit)
    # Commit ExpungeUsersInIssues here, as it has many operations
    # and at least one operation that cannot be limited.
    if commit:
      self.mc.cnxn.Commit()
      self.services.issue.EnqueueIssuesForIndexing(
          self.mc.cnxn, affected_issue_ids)

    # Spam verdict and report tables have user_id columns that do not
    # reference User. No limit will be applied.
    self.services.spam.ExpungeUsersInSpam(self.mc.cnxn, user_ids)
    if commit:
      self.mc.cnxn.Commit()

    # No limit will be applied for expunging in hotlists.
    self.services.features.ExpungeUsersInHotlists(
        self.mc.cnxn, user_ids, self.services.hotlist_star, self.services.user,
        self.services.chart)
    if commit:
      self.mc.cnxn.Commit()

    # No limit will be applied for expunging in UserGroups.
    self.services.usergroup.ExpungeUsersInGroups(
        self.mc.cnxn, user_ids)
    if commit:
      self.mc.cnxn.Commit()

    # No limit will be applied for expunging in FilterRules.
    deleted_rules_by_project = self.services.features.ExpungeFilterRulesByUser(
        self.mc.cnxn, user_ids_by_email)
    rule_strs_by_project = filterrules_helpers.BuildRedactedFilterRuleStrings(
        self.mc.cnxn, deleted_rules_by_project, self.services.user, emails)
    if commit:
      self.mc.cnxn.Commit()

    # We will attempt to expunge all given users here. Limiting the users we
    # delete should be done before work_env.ExpungeUsers is called.
    self.services.user.ExpungeUsers(self.mc.cnxn, user_ids)
    if commit:
      self.mc.cnxn.Commit()
      self.services.usergroup.group_dag.MarkObsolete()

    for project_id, filter_rule_strs in rule_strs_by_project.items():
      project = self.services.project.GetProject(self.mc.cnxn, project_id)
      hostport = framework_helpers.GetHostPort(
          project_name=project.project_name)
      send_notifications.PrepareAndSendDeletedFilterRulesNotification(
          project_id, hostport, filter_rule_strs)

  def TotalUsersCount(self):
    """Returns the total number of Users in Monorail."""
    return self.services.user.TotalUsersCount(self.mc.cnxn)

  def GetAllUserEmailsBatch(self, limit=1000, offset=0):
    """Returns a list emails that belong to Users in Monorail.

    Returns:
      A list of emails for Users within Monorail ordered by the user.user_ids.
      The list will hold at most [limit] emails and will start at the given
      [offset].
    """
    return self.services.user.GetAllUserEmailsBatch(
        self.mc.cnxn, limit=limit, offset=offset)

  ### Group methods

  # FUTURE: CreateGroup()
  # FUTURE: ListGroups()
  # FUTURE: UpdateGroup()
  # FUTURE: DeleteGroup()

  ### Hotlist methods

  def CreateHotlist(
      self, name, summary, description, editor_ids, issue_ids, is_private,
      default_col_spec):
    # type: (string, string, string, Collection[int], Collection[int], Boolean,
    #     string)
    """Create a hotlist.

    Args:
      name: a valid hotlist name.
      summary: one-line explanation of the hotlist.
      description: one-page explanation of the hotlist.
      editor_ids: a list of user IDs for the hotlist editors.
      issue_ids: a list of issue IDs for the hotlist issues.
      is_private: True if the hotlist can only be viewed by owners and editors.
      default_col_spec: default columns for the hotlist's list view.


    Returns:
      The newly created hotlist.

    Raises:
      HotlistAlreadyExists: A hotlist with the given name already exists.
      InputException: No user is signed in or the proposed name is invalid.
      PermissionException: If the user cannot view all of the issues.
    """
    if not self.mc.auth.user_id:
      raise exceptions.InputException('Anon cannot create hotlists.')

    # GetIssuesDict checks that the user can view all issues.
    self.GetIssuesDict(issue_ids)

    if not framework_bizobj.IsValidHotlistName(name):
      raise exceptions.InputException(
          '%s is not a valid name for a Hotlist' % name)
    if self.services.features.LookupHotlistIDs(
        self.mc.cnxn, [name], [self.mc.auth.user_id]):
      raise features_svc.HotlistAlreadyExists()

    with self.mc.profiler.Phase('creating hotlist %s' % name):
      hotlist = self.services.features.CreateHotlist(
          self.mc.cnxn, name, summary, description, [self.mc.auth.user_id],
          editor_ids, issue_ids=issue_ids, is_private=is_private,
          default_col_spec=default_col_spec, ts=int(time.time()))

    return hotlist

  def UpdateHotlist(
      self, hotlist_id, hotlist_name=None, summary=None, description=None,
      is_private=None, default_col_spec=None, owner_id=None,
      add_editor_ids=None):
    # type: (int, str, str, str, bool, str, int, Collection[int]) -> None
    """Update the given hotlist.

    If a new value is None, the value does not get updated.

    Args:
      hotlist_id: hotlist_id of the hotlist to update.
      hotlist_name: proposed new name for the hotlist.
      summary: new summary for the hotlist.
      description: new description for the hotlist.
      is_private: true if hotlist should be updated to private.
      default_col_spec: new default columns for hotlist list view.
      owner_id: User id of the new owner.
      add_editor_ids: User ids to add as editors.

    Raises:
      InputException: The given hotlist_id is None or proposed new name is not
        a valid hotlist name.
      NoSuchHotlistException: There is no hotlist with the given ID.
      PermissionException: The logged-in user is not allowed to update
        this hotlist's settings.
      NoSuchUserException: Some proposed editors or owner were not found.
      HotlistAlreadyExists: The (proposed new) hotlist owner already owns a
        hotlist with the same (proposed) name.
    """
    hotlist = self.services.features.GetHotlist(
        self.mc.cnxn, hotlist_id, use_cache=False)
    if not permissions.CanAdministerHotlist(
        self.mc.auth.effective_ids, self.mc.perms, hotlist):
      raise permissions.PermissionException(
          'User is not allowed to update hotlist settings.')

    if hotlist.name == hotlist_name:
      hotlist_name = None
    if hotlist.owner_ids[0] == owner_id:
      owner_id = None

    if hotlist_name and not framework_bizobj.IsValidHotlistName(hotlist_name):
      raise exceptions.InputException(
          '"%s" is not a valid hotlist name' % hotlist_name)

    # Check (new) owner does not already own a hotlist with the (new) name.
    if hotlist_name or owner_id:
      owner_ids = [owner_id] if owner_id else None
      if self.services.features.LookupHotlistIDs(
          self.mc.cnxn, [hotlist_name or hotlist.name],
          owner_ids or hotlist.owner_ids):
        raise features_svc.HotlistAlreadyExists(
            'User already owns a hotlist with name %s' %
            hotlist_name or hotlist.name)

    # Filter out existing editors and users that will be added as owner
    # or is the current owner.
    next_owner_id = owner_id or hotlist.owner_ids[0]
    if add_editor_ids:
      new_editor_ids_set = {user_id for user_id in add_editor_ids if
                            user_id not in hotlist.editor_ids and
                            user_id != next_owner_id}
      add_editor_ids = list(new_editor_ids_set)

    # Validate user change requests.
    user_ids = []
    if add_editor_ids:
      user_ids.extend(add_editor_ids)
    else:
      add_editor_ids = None
    if owner_id:
      user_ids.append(owner_id)
    if user_ids:
      self.services.user.LookupUserEmails(self.mc.cnxn, user_ids)

    # Check for other no-op changes.
    if summary == hotlist.summary:
      summary = None
    if description == hotlist.description:
      description = None
    if is_private == hotlist.is_private:
      is_private = None
    if default_col_spec == hotlist.default_col_spec:
      default_col_spec = None

    if ([hotlist_name, summary, description, is_private, default_col_spec,
         owner_id, add_editor_ids] ==
        [None, None, None, None, None, None, None]):
      logging.info('No updates given')
      return

    if (summary is not None) and (not summary):
      raise exceptions.InputException('Hotlist cannot have an empty summary.')
    if (description is not None) and (not description):
      raise exceptions.InputException(
          'Hotlist cannot have an empty description.')
    if default_col_spec is not None and not framework_bizobj.IsValidColumnSpec(
        default_col_spec):
      raise exceptions.InputException(
          '"%s" is not a valid column spec' % default_col_spec)

    self.services.features.UpdateHotlist(
        self.mc.cnxn, hotlist_id, name=hotlist_name, summary=summary,
        description=description, is_private=is_private,
        default_col_spec=default_col_spec, owner_id=owner_id,
        add_editor_ids=add_editor_ids)

  # TODO(crbug/monorail/7104): delete UpdateHotlistRoles.

  def GetHotlist(self, hotlist_id, use_cache=True):
    # int, Optional[Boolean] -> Hotlist
    """Return the specified hotlist.

    Args:
      hotlist_id: int hotlist_id of the hotlist to retrieve.
      use_cache: set to false when doing read-modify-write.

    Returns:
      The specified hotlist.

    Raises:
      NoSuchHotlistException: There is no hotlist with that ID.
      PermissionException: The user is not allowed to view the hotlist.
    """
    if hotlist_id is None:
      raise exceptions.InputException('No hotlist specified')

    with self.mc.profiler.Phase('getting hotlist %r' % hotlist_id):
      hotlist = self.services.features.GetHotlist(
          self.mc.cnxn, hotlist_id, use_cache=use_cache)
    self._AssertUserCanViewHotlist(hotlist)
    return hotlist

  # TODO(crbug/monorail/7104): Remove group_by_spec argument and pre-pend
  # values to sort_spec.
  def ListHotlistItems(self, hotlist_id, max_items, start, can, sort_spec,
                       group_by_spec, use_cache=True):
    # type: (int, int, int, int, str, str, bool) -> ListResult
    """Return a list of HotlistItems for the given hotlist that
       are visible by the user.

    Args:
      hotlist_id: int hotlist_id of the hotlist.
      max_items: int the maximum number of HotlistItems we want to return.
      start: int start position in the total sorted items.
      can: int "canned_query" number to scope the visible issues.
      sort_spec: string that lists the sort order.
      group_by_spec: string that lists the grouping order.
      use_cache: set to false when doing read-modify-write.

    Returns:
      A work_env.ListResult namedtuple.

    Raises:
      NoSuchHotlistException: There is no hotlist with that ID.
      InputException: `max_items` or `start` are negative values.
      PermissionException: The user is not allowed to view the hotlist.
    """
    hotlist = self.GetHotlist(hotlist_id, use_cache=use_cache)
    if start < 0:
      raise exceptions.InputException('Invalid `start`: %d' % start)
    if max_items < 0:
      raise exceptions.InputException('Invalid `max_items`: %d' % max_items)

    hotlist_issues = self.services.issue.GetIssues(
        self.mc.cnxn, [item.issue_id for item in hotlist.items])
    project_ids = hotlist_helpers.GetAllProjectsOfIssues(hotlist_issues)
    config_list = hotlist_helpers.GetAllConfigsOfProjects(
        self.mc.cnxn, project_ids, self.services)
    harmonized_config = tracker_bizobj.HarmonizeConfigs(config_list)

    (sorted_issues, _hotlist_items_context,
     _users_by_id) = hotlist_helpers.GetSortedHotlistIssues(
        self.mc.cnxn, hotlist.items, hotlist_issues, self.mc.auth, can,
        sort_spec, group_by_spec, harmonized_config, self.services,
        self.mc.profiler)


    end = start + max_items
    visible_issues = sorted_issues[start:end]
    hotlist_items_dict = {item.issue_id: item for item in hotlist.items}
    visible_hotlist_items = [hotlist_items_dict.get(issue.issue_id) for
                            issue in visible_issues]

    next_start = None
    if end < len(sorted_issues):
      next_start = end
    return ListResult(visible_hotlist_items, next_start)

  def TransferHotlistOwnership(self, hotlist_id, new_owner_id, remain_editor,
                               use_cache=True, commit=True):
    """Transfer ownership of hotlist from current owner to new_owner.

    Args:
      hotlist_id: int hotlist_id of the hotlist we want to transfer
      new_owner_id: user_id of the new owner
      remain_editor: True if the old owner should remain on the hotlist as
        editor.
      use_cache: set to false when doing read-modify-write.
      commit: True, if changes should be committed.

    Raises:
      NoSuchHotlistException: There is not hotlist with the given ID.
      PermissionException: The logged-in user is not allowed to change ownership
        of the hotlist.
      InputException: The proposed new owner already owns a hotlist with the
        same name.
    """
    hotlist = self.services.features.GetHotlist(
        self.mc.cnxn, hotlist_id, use_cache=use_cache)
    edit_permitted = permissions.CanAdministerHotlist(
        self.mc.auth.effective_ids, self.mc.perms, hotlist)
    if not edit_permitted:
      raise permissions.PermissionException(
          'User is not allowed to update hotlist members.')

    if self.services.features.LookupHotlistIDs(
        self.mc.cnxn, [hotlist.name], [new_owner_id]):
      raise exceptions.InputException(
          'Proposed new owner already owns a hotlist with this name.')

    self.services.features.TransferHotlistOwnership(
        self.mc.cnxn, hotlist, new_owner_id, remain_editor, commit=commit)

  def RemoveHotlistEditors(self, hotlist_id, remove_editor_ids, use_cache=True):
    """Removes editors in a hotlist.

    Args:
      hotlist_id: the id of the hotlist we want to update
      remove_editor_ids: list of user_ids to remove from hotlist editors

    Raises:
      NoSuchHotlistException: There is not hotlist with the given ID.
      PermissionException: The logged-in user is not allowed to administer the
        hotlist.
      InputException: The users being removed are not editors in the hotlist.
    """
    hotlist = self.services.features.GetHotlist(
        self.mc.cnxn, hotlist_id, use_cache=use_cache)
    edit_permitted = permissions.CanAdministerHotlist(
        self.mc.auth.effective_ids, self.mc.perms, hotlist)

    # check if user is only removing themselves from the hotlist.
    # removing linked accounts is allowed but users cannot remove groups
    # they are part of from hotlists.
    user_or_linked_ids = (
        self.mc.auth.user_pb.linked_child_ids + [self.mc.auth.user_id])
    if self.mc.auth.user_pb.linked_parent_id:
      user_or_linked_ids.append(self.mc.auth.user_pb.linked_parent_id)
    removing_self_only = set(remove_editor_ids).issubset(
        set(user_or_linked_ids))

    if not removing_self_only and not edit_permitted:
      raise permissions.PermissionException(
          'User is not allowed to remove editors')

    if not set(remove_editor_ids).issubset(set(hotlist.editor_ids)):
      raise exceptions.InputException(
          'Cannot remove users who are not hotlist editors.')

    self.services.features.RemoveHotlistEditors(
        self.mc.cnxn, hotlist_id, remove_editor_ids)

  def DeleteHotlist(self, hotlist_id):
    """Delete the given hotlist from the DB.

    Args:
      hotlist_id (int): The id of the hotlist to delete.

    Raises:
      NoSuchHotlistException: There is not hotlist with the given ID.
      PermissionException: The logged-in user is not allowed to
        delete the hotlist.
    """
    hotlist = self.services.features.GetHotlist(
        self.mc.cnxn, hotlist_id, use_cache=False)
    edit_permitted = permissions.CanAdministerHotlist(
        self.mc.auth.effective_ids, self.mc.perms, hotlist)
    if not edit_permitted:
      raise permissions.PermissionException(
          'User is not allowed to delete hotlist')

    self.services.features.ExpungeHotlists(
        self.mc.cnxn, [hotlist.hotlist_id], self.services.hotlist_star,
        self.services.user,  self.services.chart)

  def ListHotlistsByUser(self, user_id):
    """Return the hotlists for the given user.

    Args:
      user_id (int): The id of the user to query.

    Returns:
      The hotlists for the given user.
    """
    if user_id is None:
      raise exceptions.InputException('No user specified')

    with self.mc.profiler.Phase('querying hotlists for user %r' % user_id):
      hotlists = self.services.features.GetHotlistsByUserID(
          self.mc.cnxn, user_id)

    # Filter the hotlists that the currently authenticated user cannot see.
    result = [
        hotlist
        for hotlist in hotlists
        if permissions.CanViewHotlist(
            self.mc.auth.effective_ids, self.mc.perms, hotlist)]
    return result

  def ListHotlistsByIssue(self, issue_id):
    """Return the hotlists the given issue is part of.

    Args:
      issue_id (int): The id of the issue to query.

    Returns:
      The hotlists the given issue is part of.
    """
    # Check that the issue exists and the user has permission to see it.
    self.GetIssue(issue_id)

    with self.mc.profiler.Phase('querying hotlists for issue %r' % issue_id):
      hotlists = self.services.features.GetHotlistsByIssueID(
          self.mc.cnxn, issue_id)

    # Filter the hotlists that the currently authenticated user cannot see.
    result = [
        hotlist
        for hotlist in hotlists
        if permissions.CanViewHotlist(
            self.mc.auth.effective_ids, self.mc.perms, hotlist)]
    return result

  def ListRecentlyVisitedHotlists(self):
    """Return the recently visited hotlists for the logged in user.

    Returns:
      The recently visited hotlists for the given user, or an empty list if no
      user is logged in.
    """
    if not self.mc.auth.user_id:
      return []

    with self.mc.profiler.Phase(
        'get recently visited hotlists for user %r' % self.mc.auth.user_id):
      hotlist_ids = self.services.user.GetRecentlyVisitedHotlists(
          self.mc.cnxn, self.mc.auth.user_id)
      hotlists_by_id = self.services.features.GetHotlists(
          self.mc.cnxn, hotlist_ids)
      hotlists = [hotlists_by_id[hotlist_id] for hotlist_id in hotlist_ids]

    # Filter the hotlists that the currently authenticated user cannot see.
    # It might be that some of the hotlists have become private since the user
    # last visited them, or the user has lost access for other reasons.
    result = [
        hotlist
        for hotlist in hotlists
        if permissions.CanViewHotlist(
            self.mc.auth.effective_ids, self.mc.perms, hotlist)]
    return result

  def ListStarredHotlists(self):
    """Return the starred hotlists for the logged in user.

    Returns:
      The starred hotlists for the logged in user.
    """
    if not self.mc.auth.user_id:
      return []

    with self.mc.profiler.Phase(
        'get starred hotlists for user %r' % self.mc.auth.user_id):
      hotlist_ids = self.services.hotlist_star.LookupStarredItemIDs(
          self.mc.cnxn, self.mc.auth.user_id)
      hotlists_by_id, _ = self.services.features.GetHotlistsByID(
          self.mc.cnxn, hotlist_ids)
      hotlists = [hotlists_by_id[hotlist_id] for hotlist_id in hotlist_ids]

    # Filter the hotlists that the currently authenticated user cannot see.
    # It might be that some of the hotlists have become private since the user
    # starred them, or the user has lost access for other reasons.
    result = [
        hotlist
        for hotlist in hotlists
        if permissions.CanViewHotlist(
            self.mc.auth.effective_ids, self.mc.perms, hotlist)]
    return result

  def StarHotlist(self, hotlist_id, starred):
    """Star or unstar the specified hotlist.

    Args:
      hotlist_id: int ID of the hotlist to star/unstar.
      starred: true to add a star, false to remove it.

    Returns:
      Nothing.

    Raises:
      NoSuchHotlistException: There is no hotlist with that ID.
    """
    if hotlist_id is None:
      raise exceptions.InputException('No hotlist specified')

    if not self.mc.auth.user_id:
      raise exceptions.InputException('No current user specified')

    with self.mc.profiler.Phase('(un)starring hotlist %r' % hotlist_id):
      # Make sure the hotlist exists and user has permission to see it.
      self.GetHotlist(hotlist_id)
      self.services.hotlist_star.SetStar(
          self.mc.cnxn, hotlist_id, self.mc.auth.user_id, starred)

  def IsHotlistStarred(self, hotlist_id):
    """Return True if the current hotlist has starred the given hotlist.

    Args:
      hotlist_id: int ID of the hotlist to check.

    Returns:
      True if starred.

    Raises:
      NoSuchHotlistException: There is no hotlist with that ID.
    """
    if hotlist_id is None:
      raise exceptions.InputException('No hotlist specified')

    if not self.mc.auth.user_id:
      return False

    with self.mc.profiler.Phase('checking hotlist star %r' % hotlist_id):
      # Make sure the hotlist exists and user has permission to see it.
      self.GetHotlist(hotlist_id)
      return self.services.hotlist_star.IsItemStarredBy(
        self.mc.cnxn, hotlist_id, self.mc.auth.user_id)

  def GetHotlistStarCount(self, hotlist_id):
    """Return the number of times the hotlist has been starred.

    Args:
      hotlist_id: int ID of the hotlist to check.

    Returns:
      The number of times the hotlist has been starred.

    Raises:
      NoSuchHotlistException: There is no hotlist with that ID.
    """
    if hotlist_id is None:
      raise exceptions.InputException('No hotlist specified')

    with self.mc.profiler.Phase('counting stars for hotlist %r' % hotlist_id):
      # Make sure the hotlist exists and user has permission to see it.
      self.GetHotlist(hotlist_id)
      return self.services.hotlist_star.CountItemStars(self.mc.cnxn, hotlist_id)

  def CheckHotlistName(self, name):
    """Check that a hotlist name is valid and not already in use.

    Args:
      name: str the hotlist name to check.

    Returns:
      None if the user can create a hotlist with that name, or a string with the
      reason the name can't be used.

    Raises:
      InputException: The user is not signed in.
    """
    if not self.mc.auth.user_id:
      raise exceptions.InputException('No current user specified')

    with self.mc.profiler.Phase('checking hotlist name: %r' % name):
      if not framework_bizobj.IsValidHotlistName(name):
        return '"%s" is not a valid hotlist name.' % name
      if self.services.features.LookupHotlistIDs(
          self.mc.cnxn, [name], [self.mc.auth.user_id]):
        return 'There is already a hotlist with that name.'

    return None

  def RemoveIssuesFromHotlists(self, hotlist_ids, issue_ids):
    """Remove the issues given in issue_ids from the given hotlists.

    Args:
      hotlist_ids: a list of hotlist ids to remove the issues from.
      issue_ids: a list of issue_ids to be removed.

    Raises:
      PermissionException: The user has no permission to edit the hotlist.
      NoSuchHotlistException: One of the hotlist ids was not found.
    """
    for hotlist_id in hotlist_ids:
      self._AssertUserCanEditHotlist(self.GetHotlist(hotlist_id))

    with self.mc.profiler.Phase(
        'Removing issues %r from hotlists %r' % (issue_ids, hotlist_ids)):
      self.services.features.RemoveIssuesFromHotlists(
          self.mc.cnxn, hotlist_ids, issue_ids, self.services.issue,
          self.services.chart)

  def AddIssuesToHotlists(self, hotlist_ids, issue_ids, note):
    """Add the issues given in issue_ids to the given hotlists.

    Args:
      hotlist_ids: a list of hotlist ids to add the issues to.
      issue_ids: a list of issue_ids to be added.
      note: a string with a message to record along with the issues.

    Raises:
      PermissionException: The user has no permission to edit the hotlist.
      NoSuchHotlistException: One of the hotlist ids was not found.
    """
    for hotlist_id in hotlist_ids:
      self._AssertUserCanEditHotlist(self.GetHotlist(hotlist_id))

    # GetIssuesDict checks that the user can view all issues
    self.GetIssuesDict(issue_ids)

    added_tuples = [
        (issue_id, self.mc.auth.user_id, int(time.time()), note)
        for issue_id in issue_ids]

    with self.mc.profiler.Phase(
        'Removing issues %r from hotlists %r' % (issue_ids, hotlist_ids)):
      self.services.features.AddIssuesToHotlists(
          self.mc.cnxn, hotlist_ids, added_tuples, self.services.issue,
          self.services.chart)

  # TODO(crbug/monorai/7104): RemoveHotlistItems and RerankHotlistItems should
  # replace RemoveIssuesFromHotlist, AddIssuesToHotlists,
  # RemoveIssuesFromHotlists.
  # The latter 3 methods are still used in v0 API paths and should be removed
  # once those v0 API methods are removed.
  def RemoveHotlistItems(self, hotlist_id, remove_issue_ids):
    # type: (int, Collection[int]) -> None
    """Remove given issues from a hotlist.

    Args:
      hotlist_id: A hotlist ID of the hotlist to remove issues from.
      remove_issue_ids: A list of issue IDs that belong to HotlistItems
        we want to remove from the hotlist.

    Raises:
      NoSuchHotlistException: If the hotlist is not found.
      NoSuchIssueException: if an Issue is not found for a given
        remove_issue_id.
      PermissionException: If the user lacks permissions to edit the hotlist or
        view all the given issues.
      InputException: If there are ids in `remove_issue_ids` that do not exist
        in the hotlist.
    """
    hotlist = self.GetHotlist(hotlist_id)
    self._AssertUserCanEditHotlist(hotlist)
    if not remove_issue_ids:
      raise exceptions.InputException('`remove_issue_ids` empty.')

    item_issue_ids = {item.issue_id for item in hotlist.items}
    if not (set(remove_issue_ids).issubset(item_issue_ids)):
      raise exceptions.InputException('item(s) not found in hotlist.')

    # Raise exception for un-viewable or not found item_issue_ids.
    self.GetIssuesDict(item_issue_ids)

    self.services.features.UpdateHotlistIssues(
        self.mc.cnxn, hotlist_id, [], remove_issue_ids, self.services.issue,
        self.services.chart)

  def AddHotlistItems(self, hotlist_id, new_issue_ids, target_position):
    # type: (int, Sequence[int], int) -> None
    """Add given issues to a hotlist.

    Args:
      hotlist_id: A hotlist ID of the hotlist to add issues to.
      new_issue_ids: A list of issue IDs that should belong to new
        HotlistItems added to the hotlist. HotlistItems will be added
        in the same order the IDs are given in. If some HotlistItems already
        exist in the Hotlist, they will not be moved.
      target_position: The index, starting at 0, of the new position the
        first issue in new_issue_ids should have. This value cannot be greater
        than (# of current hotlist.items).

    Raises:
      PermissionException: If the user lacks permissions to edit the hotlist or
        view all the given issues.
      NoSuchHotlistException: If the hotlist is not found.
      NoSuchIssueException: If an Issue is not found for a given new_issue_id.
      InputException: If the target_position or new_issue_ids are not valid.
    """
    hotlist = self.GetHotlist(hotlist_id)
    self._AssertUserCanEditHotlist(hotlist)
    if not new_issue_ids:
      raise exceptions.InputException('no new issues given to add.')

    item_issue_ids = {item.issue_id for item in hotlist.items}
    confirmed_new_issue_ids = set(new_issue_ids).difference(item_issue_ids)

    # Raise exception for un-viewable or not found item_issue_ids.
    self.GetIssuesDict(item_issue_ids)

    if confirmed_new_issue_ids:
      changed_items = self._GetChangedHotlistItems(
          hotlist, list(confirmed_new_issue_ids), target_position)
      self.services.features.UpdateHotlistIssues(
          self.mc.cnxn, hotlist_id, changed_items, [], self.services.issue,
          self.services.chart)

  def RerankHotlistItems(self, hotlist_id, moved_issue_ids, target_position):
    # type: (int, list(int), int) -> Hotlist
    """Rerank HotlistItems of a Hotlist.

      This method reranks existing hotlist items to the given target_position.
        e.g. For a hotlist with items (a, b, c, d, e), if moved_issue_ids were
        [e.issue_id, c.issue_id] and target_position were 0,
        the hotlist items would be reranked as (e, c, a, b, d).

    Args:
      hotlist_id: A hotlist ID of the hotlist to rerank.
      moved_issue_ids: A list of issue IDs in the hotlist, to be moved
        together, in the order they should have after the reranking.
      target_position: The index, starting at 0, of the new position the
        first issue in moved_issue_ids should have. This value cannot be greater
        than (# of current hotlist.items not being reranked).

    Returns:
      The updated hotlist.

    Raises:
      PermissionException: If the user lacks permissions to rerank the hotlist
        or view all the given issues.
      NoSuchHotlistException: If the hotlist is not found.
      NoSuchIssueException: If an Issue is not found for a given moved_issue_id.
      InputException: If the target_position or moved_issue_ids are not valid.
    """
    hotlist = self.GetHotlist(hotlist_id)
    self._AssertUserCanEditHotlist(hotlist)
    if not moved_issue_ids:
      raise exceptions.InputException('`moved_issue_ids` empty.')

    item_issue_ids = {item.issue_id for item in hotlist.items}
    if not (set(moved_issue_ids).issubset(item_issue_ids)):
      raise exceptions.InputException('item(s) not found in hotlist.')

    # Raise exception for un-viewable or not found item_issue_ids.
    self.GetIssuesDict(item_issue_ids)
    changed_items = self._GetChangedHotlistItems(
        hotlist, moved_issue_ids, target_position)

    if changed_items:
      self.services.features.UpdateHotlistIssues(
          self.mc.cnxn, hotlist_id, changed_items, [], self.services.issue,
          self.services.chart)

    return self.GetHotlist(hotlist.hotlist_id)

  def _GetChangedHotlistItems(self, hotlist, moved_issue_ids, target_position):
    # type: (Hotlist, Sequence(int), int) -> Hotlist
    """Returns HotlistItems that are changed after moving existing/new issues.

      This returns the list of new HotlistItems and existing HotlistItems
      with updated ranks as a result of moving the given issues to the given
      target_position. This list may include HotlistItems whose ranks' must be
      changed as a result of the `moved_issue_ids`.

    Args:
      hotlist: The hotlist that owns the HotlistItems.
      moved_issue_ids: A sequence of issue IDs for new or existing items of the
        Hotlist, to be moved together, in the order they should have after
        the change.
      target_position: The index, starting at 0, of the new position the
        first issue in moved_issue_ids should have. This value cannot be greater
        than (# of current hotlist.items not being reranked).

    Returns:
      The updated hotlist.

    Raises:
      PermissionException: If the user lacks permissions to rerank the hotlist.
      NoSuchHotlistException: If the hotlist is not found.
      InputException: If the target_position or moved_issue_ids are not valid.
    """
    # List[Tuple[issue_id, new_rank]]
    changed_item_ranks = rerank_helpers.GetHotlistRerankChanges(
        hotlist.items, moved_issue_ids, target_position)

    items_by_id = {item.issue_id: item for item in hotlist.items}
    changed_items = []
    current_time = int(time.time())
    for issue_id, rank in changed_item_ranks:
      # Get existing item to update or create new item.
      item = items_by_id.get(
          issue_id,
          features_pb2.Hotlist.HotlistItem(
              issue_id=issue_id,
              adder_id=self.mc.auth.user_id,
              date_added=current_time))
      item.rank = rank
      changed_items.append(item)

    return changed_items

  # TODO(crbug/monorail/7031): Remove this method
  # and corresponding v0 prpc method.
  def RerankHotlistIssues(self, hotlist_id, moved_ids, target_id, split_above):
    """Rerank the moved issues for the hotlist.

    Args:
      hotlist_id: an int with the id of the hotlist.
      moved_ids: The id of the issues to move.
      target_id: the id of the issue to move the issues to.
      split_above: True if moved issues should be moved before the target issue.
    """
    hotlist = self.GetHotlist(hotlist_id)
    self._AssertUserCanEditHotlist(hotlist)
    hotlist_issue_ids = [item.issue_id for item in hotlist.items]
    if not set(moved_ids).issubset(set(hotlist_issue_ids)):
      raise exceptions.InputException('The issue to move is not in the hotlist')
    if target_id not in hotlist_issue_ids:
      raise exceptions.InputException('The target issue is not in the hotlist.')

    phase_name = 'Moving issues %r %s issue %d.' % (
        moved_ids, 'above' if split_above else 'below', target_id)
    with self.mc.profiler.Phase(phase_name):
      lower, higher = features_bizobj.SplitHotlistIssueRanks(
          target_id, split_above,
          [(item.issue_id, item.rank) for item in hotlist.items if
           item.issue_id not in moved_ids])
      rank_changes = rerank_helpers.GetInsertRankings(lower, higher, moved_ids)
      if rank_changes:
        relations_to_change = {
            issue_id: rank for issue_id, rank in rank_changes}
        self.services.features.UpdateHotlistItemsFields(
            self.mc.cnxn, hotlist_id, new_ranks=relations_to_change)

  def UpdateHotlistIssueNote(self, hotlist_id, issue_id, note):
    """Update the given issue of the given hotlist with the given note.

    Args:
      hotlist_id: an int with the id of the hotlist.
      issue_id: an int with the id of the issue.
      note: a string with a message to record for the given issue.
    Raises:
      PermissionException: The user has no permission to edit the hotlist.
      NoSuchHotlistException: The hotlist id was not found.
      InputException: The issue is not part of the hotlist.
    """
    # Make sure the hotlist exists and we have permission to see and edit it.
    hotlist = self.GetHotlist(hotlist_id)
    self._AssertUserCanEditHotlist(hotlist)

    # Make sure the issue exists and we have permission to see it.
    self.GetIssue(issue_id)

    # Make sure the issue belongs to the hotlist.
    if not any(item.issue_id == issue_id for item in hotlist.items):
      raise exceptions.InputException('The issue is not part of the hotlist.')

    with self.mc.profiler.Phase(
        'Editing note for issue %s in hotlist %s' % (issue_id, hotlist_id)):
      new_notes = {issue_id: note}
      self.services.features.UpdateHotlistItemsFields(
          self.mc.cnxn, hotlist_id, new_notes=new_notes)

  def expungeUsersFromStars(self, user_ids):
    """Wipes any starred user or user's stars from all star services.

    This method will not commit the operation. This method will not
    make changes to in-memory data.
    """

    self.services.project_star.ExpungeStarsByUsers(self.mc.cnxn, user_ids)
    self.services.issue_star.ExpungeStarsByUsers(self.mc.cnxn, user_ids)
    self.services.hotlist_star.ExpungeStarsByUsers(self.mc.cnxn, user_ids)
    self.services.user_star.ExpungeStarsByUsers(self.mc.cnxn, user_ids)
    for user_id in user_ids:
      self.services.user_star.ExpungeStars(self.mc.cnxn, user_id, commit=False)

  # Permissions

  # ListFooPermission methods will return the list of permissions in addition to
  # the permission to "VIEW",
  # that the logged in user has for a given resource_id's resource Foo.
  # If the user cannot view Foo, PermissionException will be raised.
  # Not all resources will have predefined lists of permissions
  # (e.g permissions.HOTLIST_OWNER_PERMISSIONS)
  # For most cases, the list of permissions will be created within the
  # ListFooPermissions method.

  def ListHotlistPermissions(self, hotlist_id):
    # type: (int) -> List(str)
    """Return the list of permissions the current user has for the hotlist."""
    # Permission to view checked in GetHotlist()
    hotlist = self.GetHotlist(hotlist_id)
    if permissions.CanAdministerHotlist(self.mc.auth.effective_ids,
                                        self.mc.perms, hotlist):
      return permissions.HOTLIST_OWNER_PERMISSIONS
    if permissions.CanEditHotlist(self.mc.auth.effective_ids, self.mc.perms,
                                  hotlist):
      return permissions.HOTLIST_EDITOR_PERMISSIONS
    return []

  def ListFieldDefPermissions(self, field_id, project_id):
    # type:(int, int) -> List[str]
    """Return the list of permissions the current user has for the fieldDef."""
    project = self.GetProject(project_id)
    # TODO(crbug/monorail/7614): The line below was added temporarily while this
    # bug is fixed.
    self.mc.LookupLoggedInUserPerms(project)
    field = self.GetFieldDef(field_id, project)
    if permissions.CanEditFieldDef(self.mc.auth.effective_ids, self.mc.perms,
                                   project, field):
      return [permissions.EDIT_FIELD_DEF, permissions.EDIT_FIELD_DEF_VALUE]
    if permissions.CanEditValueForFieldDef(self.mc.auth.effective_ids,
                                           self.mc.perms, project, field):
      return [permissions.EDIT_FIELD_DEF_VALUE]
    return []
