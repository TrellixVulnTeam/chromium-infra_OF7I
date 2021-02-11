# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""PubSub notifications about builds."""

import json

from google.appengine.api import app_identity
from google.appengine.ext import ndb
import webapp2

from components import decorators
from components import pubsub

from legacy import api_common
import model
import tq


def enqueue_notifications_async(build):
  assert ndb.in_transaction()
  assert build

  def mktask(mode):
    return dict(
        url='/internal/task/buildbucket/notify/%d' % build.key.id(),
        payload=dict(id=build.key.id(), mode=mode),
        retry_options=dict(task_age_limit=model.BUILD_TIMEOUT.total_seconds()),
    )

  tasks = [mktask('global')]
  if build.pubsub_callback and build.pubsub_callback.topic:  # pragma: no branch
    tasks.append(mktask('callback'))
  return tq.enqueue_async('backend-default', tasks)


class TaskPublishNotification(webapp2.RequestHandler):
  """Publishes a PubSub message."""

  @decorators.require_taskqueue('backend-default')
  def post(self, build_id):  # pylint: disable=unused-argument
    body = json.loads(self.request.body)

    assert body.get('mode') in ('global', 'callback')
    bundle = model.BuildBundle.get(
        body['id'],
        infra=True,
        input_properties=True,
        output_properties=True,
    )
    if not bundle:  # pragma: no cover
      return
    build = bundle.build

    message = {
        'build': api_common.build_to_dict(bundle),
        'hostname': app_identity.get_default_version_hostname(),
    }
    attrs = {'build_id': str(build.key.id())}
    if body['mode'] == 'callback':
      # NOTE this is a workaround for crbug.com/1121657 ; Some user code set
      # pubsub_callback topics with surrounding quotes accidentally. We strip
      # them here to allow our notification taskqueue to drain without further
      # errors.
      topic = build.pubsub_callback.topic.strip('"')
      if not topic:  # pragma: no cover
        # This is a workaround for some malformed tasks introduced as part
        # of the Go UpdateBuild migration. If there's no topic, don't bother
        # doing any work for it.
        return
      message['user_data'] = build.pubsub_callback.user_data
      attrs['auth_token'] = build.pubsub_callback.auth_token
    else:
      topic = 'projects/%s/topics/builds' % app_identity.get_application_id()

    pubsub.publish(topic, json.dumps(message, sort_keys=True), attrs)
