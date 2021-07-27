# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import logging

from google.appengine.ext import ndb

from components import endpoints_webapp2
from components import ereporter2
from components import utils
import gae_ts_mon
import webapp2

import handlers
import metrics
import swarming


def disable_memcache(app):  # pragma: no cover
  """Overrides the dispatcher to disable memcache."""
  # Because Buildbucket Go service cannot invalidate Python's memcache,
  # it must be disabled once the Go service performs any writes.
  # ndb.Key (unused) -> bool indicating whether to use memcache or not.
  old_dispatcher = app.router.dispatch

  def dispatch(_, req, res):
    ndb.get_context().set_memcache_policy(lambda _: False)
    old_dispatcher(req, res)

  app.router.set_dispatcher(dispatch)


def create_frontend_app():  # pragma: no cover
  """Returns WSGI app for frontend."""
  app = webapp2.WSGIApplication(
      handlers.get_frontend_routes(), debug=utils.is_local_dev_server()
  )
  gae_ts_mon.initialize(app)
  return app


def create_backend_app():  # pragma: no cover
  """Returns WSGI app for backend."""
  routes = handlers.get_backend_routes() + swarming.get_backend_routes()
  app = webapp2.WSGIApplication(routes, debug=utils.is_local_dev_server())
  gae_ts_mon.initialize(app, cron_module='backend')
  gae_ts_mon.register_global_metrics(metrics.GLOBAL_METRICS)
  gae_ts_mon.register_global_metrics_callback(
      'buildbucket_global', metrics.update_global_metrics
  )
  return app


def initialize():  # pragma: no cover
  """Bootstraps the global state and creates WSGI applications."""
  ereporter2.register_formatter()
  fe, be = create_frontend_app(), create_backend_app()
  logging.info('disabling memcache')
  disable_memcache(fe)
  disable_memcache(be)
  return fe, be
