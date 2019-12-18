# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import logging
import time
import webapp2

from infra_libs.ts_mon import exporter
from infra_libs.ts_mon import handlers
from infra_libs.ts_mon import shared
from infra_libs.ts_mon.common import http_metrics
from infra_libs.ts_mon.common import interface

from google.appengine.api import runtime as apiruntime


def instrument(app, time_fn=time.time):
  """Instruments an webapp2 WSGI application."""
  if _is_instrumented(app):
    return

  old_dispatcher = app.router.dispatch

  def dispatch(_, request, response):
    return _instrumented_dispatcher(
        old_dispatcher, request, response, time_fn=time_fn)

  app.router.add((r'/internal/cron/ts_mon/send', TaskNumAssignerHandler))
  app.router.set_dispatcher(dispatch)
  app.router.__instrumented_by_ts_mon = True


def _is_instrumented(app):
  """Checks if tsmon dispatcher has been installed into the app."""
  return hasattr(app.router, '__instrumented_by_ts_mon')


def _instrumented_dispatcher(dispatcher, request, response, time_fn=time.time):
  start_time = time_fn()
  response_status = 0
  time_now = time_fn()

  try:
    with exporter.parallel_flush(time_now):
      ret = dispatcher(request, response)
  except webapp2.HTTPException as ex:
    response_status = ex.code
    raise
  except Exception:
    response_status = 500
    raise
  else:
    if isinstance(ret, webapp2.Response):
      response = ret
    response_status = response.status_int
  finally:
    elapsed_ms = int((time_fn() - start_time) * 1000)

    # Use the route template regex, not the request path, to prevent an
    # explosion in possible field values.
    name = request.route.template if request.route is not None else ''

    http_metrics.update_http_server_metrics(
        name,
        response_status,
        elapsed_ms,
        request_size=request.content_length,
        response_size=response.content_length,
        user_agent=request.user_agent)

  return ret


def report_memory(handler):
  """Wraps an app so handlers log when memory usage increased by at least 0.5MB
  after the handler completed.
  """
  if os.environ.get('SERVER_SOFTWARE', '').startswith('Development'):
    # Otherwise this fails with:
    # AssertionError: No api proxy found for service "system"
    return handler  # pragma: no cover

  min_delta = 0.5

  def dispatch_and_report(*args, **kwargs):
    before = apiruntime.runtime.memory_usage().current()
    try:
      return handler(*args, **kwargs)
    finally:
      after = apiruntime.runtime.memory_usage().current()
      if after >= before + min_delta:  # pragma: no cover
        logging.debug('Memory usage: %.1f -> %.1f MB; delta: %.1f MB', before,
                      after, after - before)

  return dispatch_and_report


class TaskNumAssignerHandler(webapp2.RequestHandler):

  @report_memory
  def get(self):
    if self.request.headers.get('X-Appengine-Cron') != 'true':
      self.abort(403)

    with shared.instance_namespace_context():
      handlers._assign_task_num()

    interface.invoke_global_callbacks()
