# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
"""Unittest for the exceptions module."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from framework import exceptions
from framework import permissions


class ErrorsManagerTest(unittest.TestCase):

  def testRaiseIfErrors_Errors(self):
    """We raise the given exception if there are errors."""
    err_aggregator = exceptions.ErrorAggregator(exceptions.InputException)

    err_aggregator.AddErrorMessage('The chickens are missing.')
    err_aggregator.AddErrorMessage('The foxes are free.')
    with self.assertRaisesRegexp(
        exceptions.InputException,
        'The chickens are missing.\nThe foxes are free.'):
      err_aggregator.RaiseIfErrors()

  def testErrorsManager_NoErrors(self):
    """ We don't raise exceptions if there are not errors. """
    err_aggregator = exceptions.ErrorAggregator(exceptions.InputException)
    err_aggregator.RaiseIfErrors()

  def testWithinContext_ExceptionPassedIn(self):
    """We do not suppress exceptions raised within wrapped code."""

    with self.assertRaisesRegexp(exceptions.InputException,
                                 'We should raise this'):
      with exceptions.ErrorAggregator(exceptions.InputException) as errors:
        errors.AddErrorMessage('We should ignore this error.')
        raise exceptions.InputException('We should raise this')

  def testWithinContext_NoExceptionPassedIn(self):
    """We raise an exception for any errors if no exceptions are passed in."""
    with self.assertRaisesRegexp(exceptions.InputException,
                                 'We can raise this now.'):
      with exceptions.ErrorAggregator(exceptions.InputException) as errors:
        errors.AddErrorMessage('We can raise this now.')
        return True
