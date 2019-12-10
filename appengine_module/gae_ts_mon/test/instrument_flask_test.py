# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import print_function

import flask
import mock
import werkzeug

from google.appengine.api.runtime import runtime

from .test_support import test_case

from infra_libs.ts_mon import config
from infra_libs.ts_mon import instrument_flask
from infra_libs.ts_mon import shared
from infra_libs.ts_mon.common import http_metrics
from infra_libs.ts_mon.common import interface
from infra_libs.ts_mon.common import targets
from parameterized import parameterized


class InstrumentFlaskTest(test_case.TestCase):

  def setUp(self):
    super(InstrumentFlaskTest, self).setUp()
    config.reset_for_unittest()
    self.next_time = 42.0
    self.time_increment = 3.0

    self.app = flask.Flask('test_app')
    self.app.config['TESTING'] = True
    instrument_flask.instrument(self.app, self.fake_time)

  def fake_time(self):
    ret = self.next_time
    self.next_time += self.time_increment
    return ret

  def validate_http_metrics(self,
                            name,
                            status_code,
                            response_data,
                            is_robot=False):
    """Validates the standard HTTP metrics with the HTTP transcation."""
    fields = {'name': name, 'status': status_code, 'is_robot': is_robot}
    self.assertEqual(1, http_metrics.server_response_status.get(fields))
    self.assertLessEqual(3000, http_metrics.server_durations.get(fields).sum)
    self.assertEqual(
        len(response_data),
        http_metrics.server_response_bytes.get(fields).sum)

  def test_instrument_flask_with_middleware_loop(self):
    app1, app2 = flask.Flask('app1'), flask.Flask('app2')
    app1.wsgi_app = app2
    app2.wsgi_app = app1

    with self.assertRaises(Exception) as e:
      instrument_flask.instrument(app1, self.fake_time)
    self.assertTrue('max-depth' in str(e.exception))

  def test_instrument_flask_invoked_multiple_times(self):
    instrumentor = self.app.wsgi_app
    self.assertNotEqual(instrumentor, self.app)
    self.assertTrue(instrument_flask._is_instrumented(self.app))
    self.assertIsInstance(instrumentor, instrument_flask.FlaskInstrumentor)

    # self.app.wsgi_app should point to the same object.
    instrument_flask.instrument(self.app, self.fake_time)
    self.assertTrue(instrumentor, self.app.wsgi_app)

    # trigger a test page handler and check if the value of the HTTP metric
    # didn't increase twice.
    @self.app.route('/')
    def page():  # pylint: disable=unused-variable
      return 'response'

    resp = self.app.test_client().get('/')
    self.validate_http_metrics('/', 200, resp.get_data())

  @parameterized.expand([
      ('/', '/'),
      ('/user/<name>', '/user/foo'),
      ('/<name>/<int:uid>', '/bar/123'),
  ])
  def test_success(self, url_rule, request):

    @self.app.route(url_rule)
    def page(*args, **kwargs):  # pylint: disable=unused-variable,unused-argument
      return 'response %s' % request

    resp = self.app.test_client().get(request)
    self.validate_http_metrics(url_rule, 200, resp.get_data())

  def test_abort(self):

    @self.app.route('/')
    def page():  # pylint: disable=unused-variable
      flask.abort(417)

    resp = self.app.test_client().get('/')
    self.validate_http_metrics('/', 417, resp.get_data())

  def test_exception(self):

    @self.app.route('/uncaught_exception')
    def page():  # pylint: disable=unused-variable
      raise Exception('an exception')

    with self.assertRaises(Exception) as e:
      self.app.test_client().get('/uncaught_exception')
    self.assertTrue('an exception' in e.exception)
    self.validate_http_metrics('/uncaught_exception', 500, '')

  @parameterized.expand([
      (werkzeug.exceptions.BadRequest,),
      (werkzeug.exceptions.Unauthorized,),
      (werkzeug.exceptions.Forbidden,),
  ])
  def test_http_exception(self, ex):

    @self.app.route('/http_exception')
    def page():  # pylint: disable=unused-variable
      raise ex()

    resp = self.app.test_client().get('/http_exception')
    self.validate_http_metrics('/http_exception', ex.code, resp.get_data())

  def test_with_non_existing_page(self):
    resp = self.app.test_client().get('/not_found')

    # If there is no matching url_rule for the HTTP request, then the response
    # should be reported with an empty string in name. Otherwise, it can cause
    # an metric explosion with a massive number of requests with random paths.
    self.validate_http_metrics('', werkzeug.exceptions.NotFound.code,
                               resp.get_data())

  def test_with_existing_page_raising_404(self):

    @self.app.route('/found')
    def page():  # pylint: disable=unused-variable
      return werkzeug.exceptions.NotFound('found')

    resp = self.app.test_client().get('/found')

    # This is the case where there is a url_rule matching with the requested
    # path, but the handler returned a 404 response. In this case, the HTTP
    # traffic should be reported with the requested path.
    self.validate_http_metrics('/found', werkzeug.exceptions.NotFound.code,
                               resp.get_data())

  @parameterized.expand([
      ('200', 200),
      ('200 OK', 200),
      ('226 IM Used ', 226),
      ('418 I\'m a teapot', 418),
  ])
  def test_with_valid_status_code(self, status, expected_code):

    @self.app.route('/foo')
    def page():  # pylint: disable=unused-variable
      return flask.wrappers.Response('bar', status=status)

    resp = self.app.test_client().get('/foo')
    self.validate_http_metrics('/foo', expected_code, resp.get_data())

  @parameterized.expand([
      ('C8 OK'),
      ('0xC8 OK'),
  ])
  def test_with_invalid_status_code(self, status):

    @self.app.route('/foo')
    def page():  # pylint: disable=unused-variable
      return flask.wrappers.Response('bar', status=status)

    resp = self.app.test_client().get('/foo')
    # flask/werkzeug inserts 0 response code into the status line,
    # if the first word in the status line is not a valid 10-based number.
    self.validate_http_metrics('/foo', 0, resp.get_data())

  def test_with_valid_content_length(self):
    length = 128

    @self.app.route('/foo')
    def page():  # pylint: disable=unused-variable
      response = flask.wrappers.Response('bar', status='200 OK')
      response.automatically_set_content_length = False
      response.headers['Content-Length'] = str(length)
      return response

    self.app.test_client().get('/foo')
    self.validate_http_metrics('/foo', 200, '1' * length)

  def test_with_empty_content_length(self):

    @self.app.route('/foo')
    def page():  # pylint: disable=unused-variable
      response = flask.wrappers.Response('bar', status='200 OK')
      response.automatically_set_content_length = False
      response.headers['Content-Length'] = ''
      return response

    self.app.test_client().get('/foo')
    self.validate_http_metrics('/foo', 200, '')

  def test_without_content_length(self):

    @self.app.route('/foo')
    def page():  # pylint: disable=unused-variable
      response = flask.wrappers.Response('bar', status='200 OK')
      response.automatically_set_content_length = False
      return response

    resp = self.app.test_client().get('/foo')

    # Even if a handler returns a Response() with False in
    # automatically_set_content_length, werkzug auto sets the value with the
    # length of the response body, if the response has no 'Content-Length'
    # in the headers.
    self.validate_http_metrics('/foo', 200, resp.get_data())

  def test_with_invalid_content_length(self):

    @self.app.route('/foo')
    def page():  # pylint: disable=unused-variable
      response = flask.wrappers.Response('bar', status='200 OK')
      response.automatically_set_content_length = False
      response.headers['Content-Length'] = 'abc123xyz'
      return response

    self.app.test_client().get('/foo')
    self.validate_http_metrics('/foo', 200, '')

  @parameterized.expand([
      ('', False),
      ('MyAgent', False),
      ('GoogleBot', True),
  ])
  def test_with_user_agent(self, agent, is_robot):

    @self.app.route('/foo')
    def page():  # pylint: disable=unused-variable
      return 'bar'

    resp = self.app.test_client().get(
        '/foo', environ_base={'HTTP_USER_AGENT': agent})
    self.validate_http_metrics('/foo', 200, resp.get_data(), is_robot)

  @mock.patch(
      'gae_ts_mon.instrument_flask.FlaskHTTPStat._get_status_code',
      autospec=True,
      return_value=False)
  def test_HTTP_stat_parse_failure(self, mock_get_status_code):
    mock_get_status_code.side_effect = Exception('tsmon bug')

    @self.app.route('/foo')
    def page():  # pylint: disable=unused-variable
      return 'bar'

    self.app.test_client().get('/foo')
    # an exception raised inside FlaskHTTPStat.parse() should not be re-raised
    # outside, but results in the request reported with 0.
    self.validate_http_metrics('/foo', 0, '', False)


class AssignTaskNumTest(test_case.TestCase):

  def setUp(self):
    super(AssignTaskNumTest, self).setUp()
    config.reset_for_unittest()
    target = targets.TaskTarget('test_service', 'test_job', 'test_region',
                                'test_host')
    self.mock_state = interface.State(target=target)
    self.app = flask.Flask('test_app')
    self.app.config['TESTING'] = True
    instrument_flask.instrument(self.app)

  def tearDown(self):
    mock.patch.stopall()
    super(AssignTaskNumTest, self).tearDown()

  def test_success(self):
    with mock.patch.object(runtime, 'memory_usage') as m:
      resp = self.app.test_client().get(
          shared.CRON_REQUEST_PATH_TASKNUM_ASSIGNER,
          headers={'X-Appengine-Cron': 'true'})
      self.assertEqual(resp.status_code, 200)
      m.assert_called()

  def test_unauthorized(self):
    with mock.patch.object(runtime, 'memory_usage') as m:
      resp = self.app.test_client().get(
          shared.CRON_REQUEST_PATH_TASKNUM_ASSIGNER)
      self.assertEqual(resp.status_code, 403)
      m.assert_called()
