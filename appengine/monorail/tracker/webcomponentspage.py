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

from framework import servlet
from framework import framework_helpers
from framework import urls


class WebComponentsPage(servlet.Servlet):

  _PAGE_TEMPLATE = 'tracker/web-components-page.ezt'

  def AssertBasePermission(self, mr):
    """Check that the user has permission to visit this page."""
    super(WebComponentsPage, self).AssertBasePermission(mr)

  def GatherPageData(self, mr):
    """Build up a dictionary of data values to use when rendering the page.

    Args:
      mr: commonly used info parsed from the request.

    Returns:
      Dict of values used by EZT for rendering the page.
    """
    # Create link to view in old UI for the list view pages.
    old_ui_url = None
    url = self.request.url
    if 'issues/list' in url:
      old_ui_url = url.replace('issues/list', 'issues/list_old')
    elif '/hotlists/' in url:
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


class IssueDetailRedirect(servlet.Servlet):
  def GatherPageData(self, mr):
    logging.info(
        'Redirecting from approval page to the new issue detail page.')
    url = framework_helpers.FormatAbsoluteURL(
        mr, urls.ISSUE_DETAIL, id=mr.local_id)
    return self.redirect(url, abort=True, permanent=True)


class IssueListRedirect(servlet.Servlet):

  def GatherPageData(self, mr):
    logging.info('Redirecting from list_new to list.')

    url = self.request.url.replace(urls.ISSUE_NEW_GRID, urls.ISSUE_LIST)
    return self.redirect(url, abort=True, permanent=True)
