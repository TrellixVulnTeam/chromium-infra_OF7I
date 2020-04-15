# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from framework import paginate
from proto import secrets_pb2


class Paginator(object):
  """Class to manage API pagination.

  Paginator handles the pagination tasks and info of a single List or
  Search API method implementation, given the contents of the request.
  """

  def __init__(self, parent=None, page_size=None, order_by=None, query=None):
    # type: (Optional[str], Optional[int], Optional[str], Optional[str]) -> None
    self.request_contents = secrets_pb2.ListRequestContents(
        parent=parent, page_size=page_size, order_by=order_by, query=query)

  def GetStart(self, page_token):
    # type: (str) -> int
    """Validates a request.page_token and returns the start index for it."""
    if page_token:
      return paginate.ValidateAndParsePageToken(
          page_token, self.request_contents)
    return 0

  def GenerateNextPageToken(self, next_start):
    # type: (int) -> str
    """Generates the `next_page_token` for the API response."""
    return paginate.GeneratePageToken(self.request_contents, next_start)
