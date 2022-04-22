# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Validates V2 proto messages.

Internally, this module is a bit magical. It keeps a stack of fields currently
being validated per thread. It is used to construct a path to an invalid field
value.
"""

import contextlib
import logging
import re
import threading

from components import cipd

from go.chromium.org.luci.buildbucket.proto import common_pb2

import buildtags
import config
import errors
import experiments
import model


class Error(Exception):
  """Raised on validation errors."""


PUBSUB_USER_DATA_MAX_LENGTH = 4096

# Maximum size of Build.summary_markdown field. Defined in build.proto.
MAX_BUILD_SUMMARY_MARKDOWN_SIZE = 4000  # 4 KB


################################################################################
# Validation of common.proto messages.
# The order of functions must match the order of messages in common.proto.


def validate_gerrit_change(change):
  """Validates common_pb2.GerritChange."""
  # project is not required.
  _check_truth(change, 'host', 'change', 'patchset')


def validate_gitiles_commit(commit):
  """Validates common_pb2.GitilesCommit."""
  _check_truth(commit, 'host', 'project')

  _check_truth(commit, 'ref')
  if not commit.ref.startswith('refs/'):
    _enter_err('ref', 'must start with "refs/"')

  if commit.id:  # pragma: no cover
    with _enter('id'):
      _validate_hex_sha1(commit.id)


def validate_tags(string_pairs, mode):
  """Validates a list of common.StringPair tags.

  For mode, see buildtags.validate_tags docstring.
  """
  for p in string_pairs:
    if ':' in p.key:
      _err('tag key "%s" cannot have a colon', p.key)

  with _handle_invalid_input_error():
    tags = ['%s:%s' % (p.key, p.value) for p in string_pairs]
    buildtags.validate_tags(tags, mode)


################################################################################
# Validation of build.proto messages.
# The order of functions must match the order of messages in common.proto.


def validate_builder_id(builder_id):
  """Validates builder_common_pb2.BuilderID."""
  _check_truth(builder_id, 'project')
  _check_truth(builder_id, 'bucket')

  with _enter('project'), _handle_invalid_input_error():
    config.validate_project_id(builder_id.project)

  with _enter('bucket'), _handle_invalid_input_error():
    config.validate_bucket_name(builder_id.bucket)
    parts = builder_id.bucket.split('.')
    if len(parts) >= 3 and parts[0] == 'luci':
      _err(
          'invalid usage of v1 bucket format in v2 API; use %r instead',
          parts[2]
      )

  with _enter('builder'), _handle_invalid_input_error():
    if builder_id.builder:  # pragma: no cover
      errors.validate_builder_name(builder_id.builder)


################################################################################
# Validation of rpc.proto messages.
# The order of functions must match the order of messages in rpc.proto.


def validate_requested_dimension(dim):
  """Validates common_pb2.RequestedDimension."""
  _check_truth(dim, 'key', 'value')

  with _enter('key'):
    if dim.key == 'caches':
      _err('"caches" is invalid; define caches instead')
    if dim.key == 'pool':
      _err('"pool" is not allowed')

  with _enter('expiration'):
    with _enter('seconds'):
      if dim.expiration.seconds < 0:
        _err('must not be negative')
      if dim.expiration.seconds % 60 != 0:
        _err('must be a multiple of 60')

    if dim.expiration.nanos:
      _enter_err('nanos', 'must be 0')


def validate_schedule_build_request(req, well_known_experiments):
  if '/' in req.request_id:  # pragma: no cover
    _enter_err('request_id', 'must not contain /')

  if not req.HasField('builder') and not req.template_build_id:
    _err('builder or template_build_id is required')

  if req.HasField('builder'):
    with _enter('builder'):
      validate_builder_id(req.builder)

  with _enter('exe'):
    _check_falsehood(req.exe, 'cipd_package')
    if req.exe.cipd_version:
      with _enter('cipd_version'):
        _validate_cipd_version(req.exe.cipd_version)

  with _enter('properties'):
    validate_struct(req.properties)

  if req.HasField('gitiles_commit'):
    with _enter('gitiles_commit'):
      validate_gitiles_commit(req.gitiles_commit)

  _check_repeated(
      req,
      'gerrit_changes',
      validate_gerrit_change,
  )

  with _enter('tags'):
    validate_tags(req.tags, 'new')

  _check_repeated(req, 'dimensions', validate_requested_dimension)

  if req.priority < 0 or req.priority > 255:
    _enter_err('priority', 'must be in [0, 255]')

  if req.HasField('notify'):  # pragma: no branch
    with _enter('notify'):
      validate_notification_config(req.notify)

  for exp_name in req.experiments:
    with _enter('experiment "%s"' % (exp_name,)):
      _maybe_err(
          experiments.check_invalid_name(exp_name, well_known_experiments)
      )


def validate_struct(struct):
  for name, value in struct.fields.iteritems():
    if not value.WhichOneof('kind'):
      _enter_err(name, 'value is not set; for null, initialize null_value')


def validate_notification_config(notify):
  _check_truth(notify, 'pubsub_topic')
  if len(notify.user_data) > PUBSUB_USER_DATA_MAX_LENGTH:
    _enter_err('user_data', 'must be <= %d bytes', PUBSUB_USER_DATA_MAX_LENGTH)


################################################################################
# Internals.


def _validate_cipd_version(version):
  if not cipd.is_valid_version(version):
    _err('invalid version "%s"', version)


def _validate_hex_sha1(sha1):
  pattern = r'[a-z0-9]{40}'
  if not re.match(pattern, sha1):
    _err('does not match r"%s"', pattern)


def _check_truth(msg, *field_names):
  """Validates that the field values are truish."""
  assert field_names, 'at least 1 field is required'
  for f in field_names:
    if not getattr(msg, f):
      _enter_err(f, 'required')


def _check_falsehood(msg, *field_names):
  """Validates that the field values are falsish."""
  for f in field_names:
    if getattr(msg, f):
      _enter_err(f, 'disallowed')


def _check_repeated(msg, field_name, validator):
  """Validates each element of a repeated field."""
  for i, c in enumerate(getattr(msg, field_name)):
    with _enter('%s[%d]' % (field_name, i)):
      validator(c)


@contextlib.contextmanager
def _enter(*names):
  _field_stack().extend(names)
  try:
    yield
  finally:
    _field_stack()[-len(names):] = []


def _maybe_err(err):
  if not err:
    return
  _err('%s', err)


def _err(fmt, *args):
  field_path = '.'.join(_field_stack())
  raise Error('%s: %s' % (field_path, fmt % args))


@contextlib.contextmanager
def _handle_invalid_input_error():
  try:
    yield
  except errors.InvalidInputError as ex:
    _err('%s', ex.message)


def _enter_err(name, fmt, *args):
  with _enter(name):
    _err(fmt, *args)


def _field_stack():
  if not hasattr(_CONTEXT, 'field_stack'):  # pragma: no cover
    _CONTEXT.field_stack = []
  return _CONTEXT.field_stack


# Validation context of the current thread.
_CONTEXT = threading.local()
