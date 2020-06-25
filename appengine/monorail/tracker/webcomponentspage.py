# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Classes that implement a web components page.

Summary of classes:
 WebComponentsPage: Show one web components page.
"""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import


import logging

import settings
from framework import servlet
from framework import framework_helpers
from framework import permissions
from framework import urls


class WebComponentsPage(servlet.Servlet):

  _PAGE_TEMPLATE = 'tracker/web-components-page.ezt'

  def AssertBasePermission(self, mr):
    # type: (MonorailRequest) -> None
    """Check that the user has permission to visit this page."""
    super(WebComponentsPage, self).AssertBasePermission(mr)

  def GatherPageData(self, mr):
    # type: (MonorailRequest) -> Mapping[str, Any]
    """Build up a dictionary of data values to use when rendering the page.

    Args:
      mr: commonly used info parsed from the request.

    Returns:
      Dict of values used by EZT for rendering the page.
    """
    # Create link to view in old UI for the list view pages.
    old_ui_url = None
    url = mr.request.url
    if '/hotlists/' in url:
      hotlist = self.services.features.GetHotlist(mr.cnxn, mr.hotlist_id)
      if '/people' in url:
        old_ui_url = '/u/%s/hotlists/%s/people' % (
            hotlist.owner_ids[0], hotlist.name)
      elif '/settings' in url:
        old_ui_url = '/u/%s/hotlists/%s/details' % (
            hotlist.owner_ids[0], hotlist.name)
      else:
        old_ui_url = '/u/%s/hotlists/%s' % (hotlist.owner_ids[0], hotlist.name)

    return {
       'local_id': mr.local_id,
       'old_ui_url': old_ui_url,
      }


class ProjectListPage(WebComponentsPage):

  def GatherPageData(self, mr):
    # type: (MonorailRequest) -> Mapping[str, Any]
    """Build up a dictionary of data values to use when rendering the page.

    May redirect the user to a default project if one is configured for
    the current domain.

    Args:
      mr: commonly used info parsed from the request.

    Returns:
      Dict of values used by EZT for rendering the page.
    """
    redirect_msg = self._MaybeRedirectToDomainDefaultProject(mr)
    logging.info(redirect_msg)
    return super(ProjectListPage, self).GatherPageData(mr)

  def _MaybeRedirectToDomainDefaultProject(self, mr):
    # type: (MonorailRequest) -> str
    """If there is a relevant default project, redirect to it.

      This function is copied from: sitewide/hostinghome.py

      Args:
        mr: commonly used info parsed from the request.

      Returns:
        String with a message about what happened for logging purposes.
    """
    project_name = settings.domain_to_default_project.get(mr.request.host)
    if not project_name:
      return 'No configured default project redirect for this domain.'

    project = None
    try:
      project = self.services.project.GetProjectByName(mr.cnxn, project_name)
    except exceptions.NoSuchProjectException:
      pass

    if not project:
      return 'Domain default project %s not found' % project_name

    if not permissions.UserCanViewProject(mr.auth.user_pb,
                                          mr.auth.effective_ids, project):
      return 'User cannot view default project: %r' % project

    project_url = '/p/%s' % project_name
    self.redirect(project_url, abort=True)
    return 'Redirected to %r' % project_url
