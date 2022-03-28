# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import functools
import json

from google.appengine.ext import ndb
from protorpc import messages
from protorpc import remote
import endpoints

from components import auth
from components import utils
import gae_ts_mon

from legacy import api
import config
import errors
import swarming


def swarmbucket_api_method(
    request_message_class, response_message_class, **kwargs
):
  """Defines a swarmbucket API method."""

  endpoints_decorator = auth.endpoints_method(
      request_message_class, response_message_class, **kwargs
  )

  def decorator(fn):
    fn = adapt_exceptions(fn)
    fn = auth.public(fn)
    fn = endpoints_decorator(fn)

    ts_mon_time = lambda: utils.datetime_to_timestamp(utils.utcnow()) / 1e6
    fn = gae_ts_mon.instrument_endpoint(time_fn=ts_mon_time)(fn)
    # ndb.toplevel must be the last one.
    # See also the comment in endpoint decorator in api.py
    return ndb.toplevel(fn)

  return decorator


def adapt_exceptions(fn):

  @functools.wraps(fn)
  def decorated(*args, **kwargs):
    try:
      return fn(*args, **kwargs)
    except errors.InvalidInputError as ex:  # pragma: no cover
      raise endpoints.BadRequestException(ex.message)

  return decorated


class GetTaskDefinitionRequestMessage(messages.Message):
  # A build creation request. Buildbucket will not create the build and won't
  # allocate a build number, but will return a definition of the swarming task
  # that would be created for the build. Build id will be 1 and build number (if
  # configured) will be 0.
  build_request = messages.MessageField(api.PutRequestMessage, 1, required=True)


class GetTaskDefinitionResponseMessage(messages.Message):
  # A definition of the swarming task that would be created for the specified
  # build.
  task_definition = messages.StringField(1)

  # The swarming host that we would send this task request to.
  swarming_host = messages.StringField(2)


@auth.endpoints_api(
    name='swarmbucket', version='v1', title='Buildbucket-Swarming integration'
)
class SwarmbucketApi(remote.Service):
  """API specific to swarmbucket."""

  @swarmbucket_api_method(
      GetTaskDefinitionRequestMessage, GetTaskDefinitionResponseMessage
  )
  def get_task_def(self, request):
    """Returns a swarming task definition for a build request."""
    settings = config.get_settings_async().get_result()
    well_known_experiments = set(
        exp.name for exp in settings.experiment.experiments
    )

    try:
      # Checks access too.
      request.build_request.bucket = api.convert_bucket(
          request.build_request.bucket
      )

      build_request = api.put_request_message_to_build_request(
          request.build_request,
          well_known_experiments,
      )

      builder_id = build_request.schedule_build_request.builder
      builder = config.Builder.make_key(
          builder_id.project, builder_id.bucket, builder_id.builder
      ).get()
      if not builder:
        raise endpoints.NotFoundException(
            'Builder %s/%s/%s not found' %
            (builder_id.project, builder_id.bucket, builder_id.builder)
        )

      settings = config.get_settings_async().get_result()

      # Create a fake build and prepare a task definition.
      identity = auth.get_current_identity()
      build = build_request.create_build_async(
          1, settings, builder.config, identity, utils.utcnow()
      ).get_result()
      if builder.config.resultdb.enable:  # pragma: no branch
        # Create a dummy invocation, so that when `led launch`, the led build
        # will have resultdb enabled.
        build.proto.infra.resultdb.invocation = 'invocations/dummy'
      assert build.proto.HasField('infra')
      build.proto.number = 1
      settings = config.get_settings_async().get_result()
      task_def = swarming.compute_task_def(build, settings, fake_build=True)
      task_def_json = json.dumps(task_def)

      return GetTaskDefinitionResponseMessage(
          task_definition=task_def_json,
          swarming_host=build.proto.infra.swarming.hostname,
      )
    except errors.InvalidInputError as ex:
      raise endpoints.BadRequestException(
          'invalid build request: %s' % ex.message
      )
