# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from framework import exceptions
from framework import paginate
from proto import secrets_pb2


def CoercePageSize(page_size, max_size, default_size=None):
  # type: (int, int, Optional[int]) -> int
  """Validates page_size and coerces it to max_size if needed.

  Args:
    page_size: The page_size requested by the user.
    max_size: the maximum page size allowed. Must be > 0.
        Also used as default if default_size not provided
    default_size: default size to use if page_size not provided. Must be > 0.

  Returns:
    The appropriate page size to use for the request, based on the parameters.
    Specifically this means
      - page_size if not greater than max_size
      - max_size if page_size > max_size
      - max_size if page_size is not provided and default_size is not provided
      - default_size if page_size is not provided

  Raises:
    InputException: if page_size is negative.
  """
  # These are programming errors. They are not user input.
  assert max_size > 0
  assert default_size is None or default_size > 0

  # Check for invalid user provided page_size.
  if page_size and page_size < 0:
    raise exceptions.InputException('`page_size` cannot be negative.')

  if not page_size:
    return default_size or max_size
  if page_size > max_size:
    return max_size
  return page_size


class Paginator(object):
  """Class to manage API pagination.

  Paginator handles the pagination tasks and info of a single List or
  Search API method implementation, given the contents of the request.
  """

  def __init__(self, parent=None, page_size=None, order_by=None, query=None,
               projects=None):
    # type: (Optional[str], Optional[int], Optional[str], Optional[str],
    #   Optional[Collection[str]]]) -> None
    # TOD(crbug/monorail/7663): Add `projects` for SearchIssues.
    self.request_contents = secrets_pb2.ListRequestContents(
        parent=parent, page_size=page_size, order_by=order_by, query=query,
        projects=projects)

  def GetStart(self, page_token):
    # type: (Optional[str]) -> int
    """Validates a request.page_token and returns the start index for it."""
    if page_token:
      return paginate.ValidateAndParsePageToken(
          page_token, self.request_contents)
    return 0

  def GenerateNextPageToken(self, next_start):
    # type: (Optional[int]) -> str
    """Generates the `next_page_token` for the API response.

    Args:
      next_start: The start index of the next page, or None if no more results.

    Returns:
      A string clients can use to request the next page. Returns None if
      next_start was None
    """
    if next_start is None:
      return None
    return paginate.GeneratePageToken(self.request_contents, next_start)
