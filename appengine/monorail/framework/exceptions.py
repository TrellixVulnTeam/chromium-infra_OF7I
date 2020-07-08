# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Exception classes used throughout monorail.
"""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import


class ErrorAggregator():
  """Class for holding errors and raising an exception for many."""

  def __init__(self, exc_type):
    # type: (type) -> None
    self.exc_type = exc_type
    self.error_messages = []

  def __enter__(self):
    return self

  def __exit__(self, exc_type, exc_value, exc_traceback):
    # If no exceptions were raised within the context, we check
    # if any error messages were accumulated that we should raise
    # an exception for.
    if exc_type == None:
      self.RaiseIfErrors()
    # If there were exceptions raised within the context, we do
    # nothing to suppress them.

  def AddErrorMessage(self, message):
    # type: (str) -> None
    """Add a new error message."""
    self.error_messages.append(message)

  def RaiseIfErrors(self):
    # type: () -> None
    """If there are errors, raise one exception."""
    if self.error_messages:
      raise self.exc_type("\n".join(self.error_messages))


class Error(Exception):
  """Base class for errors from this module."""
  pass


class ActionNotSupported(Error):
  """The user is trying to do something we do not support yet."""
  pass


class InputException(Error):
  """Error in user input processing."""
  pass


class ProjectAlreadyExists(Error):
  """Tried to create a project that already exists."""


class FieldDefAlreadyExists(Error):
  """Tried to create a custom field that already exists."""


class NoSuchProjectException(Error):
  """No project with the specified name exists."""
  pass


class NoSuchTemplateException(Error):
  """No template with the specified name exists."""
  pass


class NoSuchUserException(Error):
  """No user with the specified name exists."""
  pass


class NoSuchIssueException(Error):
  """The requested issue was not found."""
  pass


class NoSuchAttachmentException(Error):
  """The requested attachment was not found."""
  pass


class NoSuchCommentException(Error):
  """The requested comment was not found."""
  pass


class NoSuchAmendmentException(Error):
  """The requested amendment was not found."""
  pass


class NoSuchComponentException(Error):
  """No component with the specified name exists."""
  pass


class InvalidComponentNameException(Error):
  """The component name is invalid."""
  pass


class InvalidHotlistException(Error):
  """The specified hotlist is invalid."""
  pass


class NoSuchFieldDefException(Error):
  """No field def for specified project exists."""
  pass


class InvalidFieldTypeException(Error):
  """Expected field type and actual field type do not match."""
  pass


class NoSuchIssueApprovalException(Error):
  """The requested approval for the issue was not found."""
  pass


class CircularGroupException(Error):
  """Circular nested group exception."""
  pass


class GroupExistsException(Error):
  """Group already exists exception."""
  pass


class NoSuchGroupException(Error):
  """Requested group was not found exception."""
  pass


class InvalidExternalIssueReference(Error):
  """Improperly formatted external issue reference.

  External issue references must be of the form:

      $tracker_shortname/$tracker_specific_id

  For example, issuetracker.google.com issues:

      b/123456789
  """
  pass


class PageTokenException(Error):
  """Incorrect page tokens."""
  pass


class FilterRuleException(Error):
  """Violates a filter rule that should show error."""
  pass
