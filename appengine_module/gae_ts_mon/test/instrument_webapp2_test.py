# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import mock
import webapp2

from .test_support import test_case

from google.appengine.api.runtime import runtime

from infra_libs.ts_mon import config
from infra_libs.ts_mon import instrument_webapp2
from infra_libs.ts_mon.common import http_metrics
from infra_libs.ts_mon.common import interface
from infra_libs.ts_mon.common import targets


class InstrumentWebapp2Test(test_case.TestCase):

  def setUp(self):
    super(InstrumentWebapp2Test, self).setUp()
    config.reset_for_unittest()

    self.next_time = 42.0
    self.time_increment = 3.0

  def fake_time(self):
    ret = self.next_time
    self.next_time += self.time_increment
    return ret

  def test_instrument_webapp2_invoked_multiple_times(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        self.response.write('success!')

    app = webapp2.WSGIApplication([('/', Handler)])
    self.assertFalse(instrument_webapp2._is_instrumented(app))

    instrument_webapp2.instrument(app, time_fn=self.fake_time)
    self.assertTrue(instrument_webapp2._is_instrumented(app))
    instrument_webapp2.instrument(app, time_fn=self.fake_time)
    self.assertTrue(instrument_webapp2._is_instrumented(app))

    # trigger a test page handler and check if the value of the HTTP metric
    # didn't increase twice.
    app.get_response('/')
    fields = {'name': '^/$', 'status': 200, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))

  def test_success(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        self.response.write('success!')

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app, time_fn=self.fake_time)

    app.get_response('/')

    fields = {'name': '^/$', 'status': 200, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))
    self.assertLessEqual(3000, http_metrics.server_durations.get(fields).sum)
    self.assertEqual(
        len('success!'),
        http_metrics.server_response_bytes.get(fields).sum)

  def test_abort(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        self.abort(417)

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app)

    app.get_response('/')

    fields = {'name': '^/$', 'status': 417, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))

  def test_set_status(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        self.response.set_status(418)

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app)

    app.get_response('/')

    fields = {'name': '^/$', 'status': 418, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))

  def test_exception(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        raise ValueError

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app)

    app.get_response('/')

    fields = {'name': '^/$', 'status': 500, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))

  def test_http_exception(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        raise webapp2.exc.HTTPExpectationFailed()

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app)

    app.get_response('/')

    fields = {'name': '^/$', 'status': 417, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))

  def test_return_response(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        ret = webapp2.Response()
        ret.set_status(418)
        return ret

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app)

    app.get_response('/')

    fields = {'name': '^/$', 'status': 418, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))

  def test_robot(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        ret = webapp2.Response()
        ret.set_status(200)
        return ret

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app)

    app.get_response('/', user_agent='GoogleBot')

    fields = {'name': '^/$', 'status': 200, 'is_robot': True}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))

  def test_missing_response_content_length(self):

    class Handler(webapp2.RequestHandler):

      def get(self):
        del self.response.headers['content-length']

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app)

    app.get_response('/')

    fields = {'name': '^/$', 'status': 200, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))
    self.assertIsNone(http_metrics.server_response_bytes.get(fields))

  def test_not_found(self):
    app = webapp2.WSGIApplication([])
    instrument_webapp2.instrument(app)

    app.get_response('/notfound')

    fields = {'name': '', 'status': 404, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))

  def test_post(self):

    class Handler(webapp2.RequestHandler):

      def post(self):
        pass

    app = webapp2.WSGIApplication([('/', Handler)])
    instrument_webapp2.instrument(app)

    app.get_response('/', POST='foo')

    fields = {'name': '^/$', 'status': 200, 'is_robot': False}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))
    self.assertEqual(
        len('foo'),
        http_metrics.server_request_bytes.get(fields).sum)


class TaskNumAssignerHandlerTest(test_case.TestCase):

  def setUp(self):
    super(TaskNumAssignerHandlerTest, self).setUp()
    config.reset_for_unittest()
    target = targets.TaskTarget('test_service', 'test_job', 'test_region',
                                'test_host')
    self.mock_state = interface.State(target=target)

    # Workaround the fact that 'system' module is not mocked.
    class _memory_usage(object):

      def current(self):
        return 10.0

    env = os.environ.copy()
    env['SERVER_SOFTWARE'] = 'PRODUCTION'
    self.mock(runtime, 'memory_usage', _memory_usage)
    self.mock(os, 'environ', env)

    self.app = webapp2.WSGIApplication()
    instrument_webapp2.instrument(self.app)

  def tearDown(self):
    mock.patch.stopall()
    super(TaskNumAssignerHandlerTest, self).tearDown()

  def test_success(self):
    response = self.app.get_response(
        '/internal/cron/ts_mon/send', headers=[('X-Appengine-Cron', 'true')])
    self.assertEqual(response.status_int, 200)

  def test_unauthorized(self):
    response = self.app.get_response('/internal/cron/ts_mon/send')
    self.assertEqual(response.status_int, 403)
