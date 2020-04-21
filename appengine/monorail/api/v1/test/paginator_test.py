# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
"""Tests for the Paginator class."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from google.appengine.ext import testbed

from api.v1 import paginator
from api.v1.api_proto import hotlists_pb2
from framework import exceptions
from framework import paginate

class PaginatorTest(unittest.TestCase):

  def setUp(self):
    self.testbed = testbed.Testbed()
    self.testbed.activate()
    self.testbed.init_memcache_stub()
    self.testbed.init_datastore_v3_stub()

    self.paginator = paginator.Paginator(
        parent='animal/goose/sound/honks', query='chaos')

  def testGetStart(self):
    """We can get the start index from a page_token."""
    start = 5
    page_token = paginate.GeneratePageToken(
        self.paginator.request_contents, start)
    self.assertEqual(self.paginator.GetStart(page_token), start)

  def testGetStart_EmptyPageToken(self):
    """We return the default start for an empty page_token."""
    request = hotlists_pb2.ListHotlistItemsRequest()
    self.assertEqual(0, self.paginator.GetStart(request.page_token))

  def testGenerateNextPageToken(self):
    """We return the next page token."""
    next_start = 10
    expected_page_token = paginate.GeneratePageToken(
        self.paginator.request_contents, next_start)
    self.assertEqual(
        self.paginator.GenerateNextPageToken(next_start), expected_page_token)

  def testGenerateNextPageToken_NoStart(self):
    """We return None if start is not provided."""
    next_start = None
    self.assertEqual(self.paginator.GenerateNextPageToken(next_start), None)

  def testCoercePageSize(self):
    """A valid page_size is used when provided."""
    self.assertEqual(1, paginator.CoercePageSize(1, 5))

  def testCoercePageSize_Negative(self):
    """An exception is raised for a negative page_size."""
    with self.assertRaises(exceptions.InputException):
      paginator.CoercePageSize(-1, 5)

  def testCoercePageSize_TooBig(self):
    """A page_size above the max is coerced to the max."""
    self.assertEqual(5, paginator.CoercePageSize(6, 5, 2))

  def testCoercePageSize_Default(self):
    """A default page_size different from max_size is used when provided."""
    self.assertEqual(2, paginator.CoercePageSize(None, 5, 2))

  def testCoercePageSize_NotProvided(self):
    """max_size is used if no page_size or default_size provided."""
    self.assertEqual(5, paginator.CoercePageSize(None, 5))

  def testCoercePageSize_Zero(self):
    """Handles zero equivalently to None."""
    self.assertEqual(5, paginator.CoercePageSize(0, 5))