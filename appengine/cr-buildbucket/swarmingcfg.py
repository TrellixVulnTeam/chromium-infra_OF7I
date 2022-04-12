# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pylint: disable=line-too-long

import collections
import copy
import json
import re

from components.config import validation

from go.chromium.org.luci.buildbucket.proto import project_config_pb2
import errors
import experiments

_DIMENSION_KEY_RGX = re.compile(r'^[a-zA-Z\_\-]+$')
# Copied from
# https://github.com/luci/luci-py/blob/75de6021b50a73e140eacfb80760f8c25aa183ff/appengine/swarming/server/task_request.py#L101
# Keep it synchronized.
_CACHE_NAME_RE = re.compile(ur'^[a-z0-9_]{1,4096}$')
# See https://chromium.googlesource.com/infra/luci/luci-py/+/master/appengine/swarming/server/service_accounts.py
_SERVICE_ACCOUNT_RE = re.compile(r'^[0-9a-zA-Z_\-\.\+\%]+@[0-9a-zA-Z_\-\.]+$')


def _validate_hostname(hostname, ctx):
  if not hostname:
    ctx.error('unspecified')
  if '://' in hostname:
    ctx.error('must not contain "://"')


def _validate_service_account(service_account, ctx):
  if (service_account != 'bot' and
      not _SERVICE_ACCOUNT_RE.match(service_account)):
    ctx.error(
        'value "%s" does not match %s', service_account,
        _SERVICE_ACCOUNT_RE.pattern
    )


def _parse_dimension(string):  # pragma: no cover
  """Parses a dimension string to a tuple (key, value, expiration_secs)."""
  key, value = string.split(':', 1)
  expiration_secs = 0
  try:
    expiration_secs = int(key)
  except ValueError:
    pass
  else:
    key, value = value.split(':', 1)
  return key, value, expiration_secs


def _parse_dimensions(strings):  # pragma: no cover
  """Parses dimension strings to a dict {key: {(value, expiration_secs)}}."""
  out = collections.defaultdict(set)
  for s in strings:
    key, value, expiration_secs = _parse_dimension(s)
    out[key].add((value, expiration_secs))
  return out


def _format_dimension(key, value, expiration_secs):  # pragma: no cover
  """Formats a dimension to a string. Opposite of parse_dimension."""
  if expiration_secs:
    return '%d:%s:%s' % (expiration_secs, key, value)
  return '%s:%s' % (key, value)


# The below is covered by swarming_test.py and swarmbucket_api_test.py
def read_dimensions(builder_cfg):  # pragma: no cover
  """Read the dimensions for a builder config.

  Returns:
    dimensions is returned as dict {key: {(value, expiration_secs)}}.
  """
  dimensions = _parse_dimensions(builder_cfg.dimensions)
  if (builder_cfg.auto_builder_dimension == project_config_pb2.YES and
      u'builder' not in dimensions):
    dimensions[u'builder'] = {(builder_cfg.name, 0)}
  return dimensions


def _validate_tag(tag, ctx):
  # a valid swarming tag is a string that contains ":"
  if ':' not in tag:
    ctx.error('does not have ":": %s', tag)
  name = tag.split(':', 1)[0]
  if name.lower() == 'builder':
    ctx.error(
        'do not specify builder tag; '
        'it is added by swarmbucket automatically'
    )


def _validate_resultdb(resultdb, ctx):
  with ctx.prefix('history_options: '):
    if resultdb.history_options.HasField('commit'):
      ctx.error('commit must be unset')


def _validate_dimensions(field_name, dimensions, ctx):
  parsed = collections.defaultdict(set)  # {key: {(value, expiration_secs)}}
  expirations = set()

  for dim in dimensions:
    with ctx.prefix('%s "%s": ', field_name, dim):
      parts = dim.split(':', 1)
      if len(parts) != 2:
        ctx.error('does not have ":"')
        continue
      key, value = parts
      expiration_secs = 0
      try:
        expiration_secs = int(key)
      except ValueError:
        pass
      else:
        parts = value.split(':', 1)
        if len(parts) != 2 or not parts[1]:
          ctx.error('has expiration_secs but missing value')
          continue
        key, value = parts

      valid_key = False
      if not key:
        ctx.error('no key')
      elif not _DIMENSION_KEY_RGX.match(key):
        ctx.error(
            'key "%s" does not match pattern "%s"', key,
            _DIMENSION_KEY_RGX.pattern
        )
      elif key == 'caches':
        ctx.error(
            'dimension key must not be "caches"; '
            'caches must be declared via caches field'
        )
      else:
        valid_key = True

      valid_expiration_secs = False
      if expiration_secs < 0 or expiration_secs > 21 * 24 * 60 * 60:
        ctx.error('expiration_secs is outside valid range; up to 21 days')
      elif expiration_secs % 60:
        ctx.error('expiration_secs must be a multiple of 60 seconds')
      else:
        expirations.add(expiration_secs)
        valid_expiration_secs = True

      if valid_key and valid_expiration_secs:
        parsed[key].add((value, expiration_secs))

  if len(expirations) >= 6:
    ctx.error('at most 6 different expiration_secs values can be used')

  # Ensure that tombstones are not mixed with non-tomstones for the same key.
  TOMBSTONE = ('', 0)
  for key, entries in parsed.iteritems():
    if TOMBSTONE not in entries or len(entries) == 1:  # pragma: no cover
      continue
    for value, expiration_secs in entries:
      if (value, expiration_secs) == TOMBSTONE:
        continue
      dim = _format_dimension(key, value, expiration_secs)
      with ctx.prefix('%s "%s": ', field_name, dim):
        ctx.error('mutually exclusive with "%s:"', key)


def _validate_relative_path(path, ctx):
  if not path:
    ctx.error('required')
  if '\\' in path:
    ctx.error(
        'cannot contain \\. On Windows forward-slashes will be '
        'replaced with back-slashes.'
    )
  if '..' in path.split('/'):
    ctx.error('cannot contain ".."')
  if path.startswith('/'):
    ctx.error('cannot start with "/"')


def _validate_exe_cfg(exe, ctx, final=True):
  """Validates an Executable message.

  If final is False, does not validate for completeness.
  """
  if final:
    if not exe.cipd_package:
      ctx.error('cipd_package: unspecified')


def _validate_recipe_cfg(recipe, ctx, final=True):
  """Validates a Recipe message.

  If final is False, does not validate for completeness.
  """
  if final:
    if not recipe.name:
      ctx.error('name: unspecified')
    if not recipe.cipd_package:
      ctx.error('cipd_package: unspecified')
  validate_recipe_properties(recipe.properties, recipe.properties_j, ctx)


def validate_recipe_property(key, value, ctx):
  if not key:
    ctx.error('key not specified')
  elif key == 'buildbucket':
    ctx.error('reserved property')
  elif key == '$recipe_engine/runtime':
    if not isinstance(value, dict):
      ctx.error('not a JSON object')
    else:
      for k in ('is_luci', 'is_experimental'):
        if k in value:
          ctx.error('key %r: reserved key', k)


def validate_recipe_properties(properties, properties_j, ctx):
  keys = set()

  def validate(props, is_json):
    for p in props:
      with ctx.prefix('%r: ', p):
        if ':' not in p:
          ctx.error('does not have a colon')
          continue

        key, value = p.split(':', 1)
        if is_json:
          try:
            value = json.loads(value)
          except ValueError as ex:
            ctx.error('%s', ex)
            continue

        validate_recipe_property(key, value, ctx)
        if key in keys:
          ctx.error('duplicate property')
        else:
          keys.add(key)

  with ctx.prefix('properties '):
    validate(properties, False)
  with ctx.prefix('properties_j '):
    validate(properties_j, True)


def _validate_properties(properties, ctx):
  try:
    properties_d = json.loads(properties)
  except ValueError as ex:
    ctx.error('%s', ex)
    return
  if not isinstance(properties_d, dict):
    ctx.error('properties is not a dict')


def validate_builder_cfg(builder, well_known_experiments, final, ctx):
  """Validates a Builder message.

  If final is False, does not validate for completeness.
  """
  if final or builder.name:
    try:
      errors.validate_builder_name(builder.name)
    except errors.InvalidInputError as ex:
      ctx.error('name: %s', ex.message)

  if final or builder.swarming_host:
    with ctx.prefix('swarming_host: '):
      _validate_hostname(builder.swarming_host, ctx)

  for i, t in enumerate(builder.swarming_tags):
    with ctx.prefix('tag #%d: ', i + 1):
      _validate_tag(t, ctx)

  _validate_dimensions('dimension', builder.dimensions, ctx)
  with ctx.prefix('resultdb:'):
    _validate_resultdb(builder.resultdb, ctx)

  cache_paths = set()
  cache_names = set()
  fallback_secs = set()
  for i, c in enumerate(builder.caches):
    with ctx.prefix('cache #%d: ', i + 1):
      _validate_cache_entry(c, ctx)
      if c.name:
        if c.name in cache_names:
          ctx.error('duplicate name')
        else:
          cache_names.add(c.name)
      if c.path:
        if c.path in cache_paths:
          ctx.error('duplicate path')
        else:
          cache_paths.add(c.path)
        if c.wait_for_warm_cache_secs:
          with ctx.prefix('wait_for_warm_cache_secs: '):
            if c.wait_for_warm_cache_secs < 60:
              ctx.error('must be at least 60 seconds')
            elif c.wait_for_warm_cache_secs % 60:
              ctx.error('must be rounded on 60 seconds')
          fallback_secs.add(c.wait_for_warm_cache_secs)
  if len(fallback_secs) > 7:
    # There can only be 8 task_slices.
    ctx.error(
        'too many different (%d) wait_for_warm_cache_secs values; max 7' %
        len(fallback_secs)
    )

  has_exe = builder.HasField('exe')
  has_recipe = builder.HasField('recipe')
  if final and ((not has_exe and not has_recipe) or (has_exe and has_recipe)):
    ctx.error('exactly one of exe or recipe must be specified')
  if has_exe:
    with ctx.prefix('exe: '):
      _validate_exe_cfg(builder.exe, ctx, final=final)
  elif has_recipe:
    if builder.properties:
      ctx.error('recipe and properties cannot be set together')
    with ctx.prefix('recipe: '):
      _validate_recipe_cfg(builder.recipe, ctx, final=final)

  if builder.priority and (builder.priority < 20 or builder.priority > 255):
    ctx.error('priority: must be in [20, 255] range; got %d', builder.priority)

  if builder.properties:
    _validate_properties(builder.properties, ctx)

  if builder.service_account:
    with ctx.prefix('service_account: '):
      _validate_service_account(builder.service_account, ctx)

  with ctx.prefix('experiments: '):
    for exp_name, percent in sorted(builder.experiments.items()):
      with ctx.prefix('"%s": ' % (exp_name)):
        err = experiments.check_invalid_name(exp_name, well_known_experiments)
        if err:
          ctx.error(err)

        if percent < 0 or percent > 100:
          ctx.error('value must be in [0, 100]')


def _validate_cache_entry(entry, ctx):
  if not entry.name:
    ctx.error('name: required')
  elif not _CACHE_NAME_RE.match(entry.name):
    ctx.error(
        'name: "%s" does not match %s', entry.name, _CACHE_NAME_RE.pattern
    )

  with ctx.prefix('path: '):
    _validate_relative_path(entry.path, ctx)


def validate_project_cfg(swarming, well_known_experiments, ctx):
  """Validates a project_config_pb2.Swarming message.

  Args:
    swarming (project_config_pb2.Swarming): the config to validate.
    well_known_experiments (Set[string]) - The set of well-known experiments.
  """

  def make_subctx():
    return validation.Context(
        on_message=lambda msg: ctx.msg(msg.severity, '%s', msg.text)
    )

  if swarming.task_template_canary_percentage.value > 100:
    ctx.error('task_template_canary_percentage.value must must be in [0, 100]')

  seen = set()
  for i, b in enumerate(swarming.builders):
    with ctx.prefix('builder %s: ' % (b.name or '#%s' % (i + 1))):
      # Validate b before merging, otherwise merging will fail.
      subctx = make_subctx()
      validate_builder_cfg(b, well_known_experiments, False, subctx)
      if subctx.result().has_errors:
        # Do not validate invalid configs.
        continue

      if b.name in seen:
        ctx.error('name: duplicate')
      else:
        seen.add(b.name)
      validate_builder_cfg(b, well_known_experiments, True, ctx)


def _validate_package(package, ctx, allow_predicate=True):
  if not package.package_name:
    ctx.error('package_name is required')
  if not package.version:
    ctx.error('version is required')

  if allow_predicate:
    _validate_builder_predicate(package.builders, ctx)
  elif package.HasField('builders'):  # pragma: no cover
    ctx.error('builders is not allowed')


def _validate_builder_predicate(predicate, ctx):
  for regex in predicate.regex:
    with ctx.prefix('regex %r: ', regex):
      _validate_regex(regex, ctx)

  for regex in predicate.regex_exclude:
    with ctx.prefix('regex_exclude %r: ', regex):
      _validate_regex(regex, ctx)


def _validate_regex(regex, ctx):
  try:
    re.compile(regex)
  except re.error as ex:
    ctx.error('invalid: %s', ex)


def validate_service_cfg(swarming, ctx):
  with ctx.prefix('milo_hostname: '):
    _validate_hostname(swarming.milo_hostname, ctx)

  # Validate packages.
  for i, p in enumerate(swarming.user_packages):
    with ctx.prefix('user_package[%d]: ' % i):
      _validate_package(p, ctx)
  with ctx.prefix('bbagent_package: '):
    _validate_package(swarming.bbagent_package, ctx)
  with ctx.prefix('kitchen_package: '):
    _validate_package(swarming.kitchen_package, ctx, allow_predicate=False)
