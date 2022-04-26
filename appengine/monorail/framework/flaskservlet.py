# Copyright 2022 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd
"""Base classes for Monorail Flask servlets.

This is derived from  servlet.py
This base class provides handler methods that conveniently drive
the process of parsing the request, checking base permissions,
gathering common page information, gathering page-specific information,
and adding on-page debugging information (when appropriate).
Subclasses can simply implement the page-specific logic.

Summary of page classes:
  FlaskServlet: abstract base class for all Monorail flask servlets.
"""

import gc
import httplib
import logging
import time

import flask

import settings
from framework import exceptions
from framework import framework_constants
from framework import monorailrequest
from framework import permissions
from framework import ratelimiter
from framework import template_helpers
from framework import xsrf

if not settings.unit_test_mode:
  import MySQLdb


class FlaskServlet(object):
  """Base class for all Monorail flask servlets.

  Defines a framework of methods that build up parts of the EZT page data.

  Subclasses should override GatherPageData and/or ProcessFormData to
  handle requests.
  """

  # This value should not typically be overridden.
  _TEMPLATE_PATH = framework_constants.TEMPLATE_PATH

  _PAGE_TEMPLATE = None  # Normally overridden in subclasses.
  _ELIMINATE_BLANK_LINES = False

  _MISSING_PERMISSIONS_TEMPLATE = 'sitewide/403-page.ezt'

  def __init__(self, services=None, content_type='text/html; charset=UTF-8'):
    """Load and parse the template, saving it for later use."""
    if self._PAGE_TEMPLATE:  # specified in subclasses
      template_path = self._TEMPLATE_PATH + self._PAGE_TEMPLATE
      self.template = template_helpers.GetTemplate(
          template_path, eliminate_blank_lines=self._ELIMINATE_BLANK_LINES)
    else:
      self.template = None

    self._missing_permissions_template = template_helpers.MonorailTemplate(
        self._TEMPLATE_PATH + self._MISSING_PERMISSIONS_TEMPLATE)
    # TODO(https://crbug.com/monorail/6511): set service in app config
    self.services = services
    self.content_type = content_type
    self.mr = None
    self.request = flask.request
    self.response = None
    self.ratelimiter = ratelimiter.RateLimiter()

  def preProcess(self):

    self.mr = monorailrequest.MonorailRequest(self.services)
    # TODO: add the header to response
    # self.response.headers.add('Strict-Transport-Security',
    #     'max-age=31536000; includeSubDomains')

    if 'X-Cloud-Trace-Context' in self.request.headers:
      self.mr.profiler.trace_context = (
          self.request.headers.get('X-Cloud-Trace-Context'))

    if self.services.cache_manager:
      try:
        with self.mr.profiler.Phase('distributed invalidation'):
          self.services.cache_manager.DoDistributedInvalidation(self.mr.cnxn)

      except MySQLdb.OperationalError as e:
        logging.exception(e)
        page_data = {
            'http_response_code': httplib.SERVICE_UNAVAILABLE,
            'requested_url': self.request.url,
        }
        self.template = template_helpers.GetTemplate(
            'templates/framework/database-maintenance.ezt',
            eliminate_blank_lines=self._ELIMINATE_BLANK_LINES)
        # TODO: create the function for create falsk response
        # self.template.WriteResponse(
        #   self.response, page_data, content_type='text/html')
        return str(page_data)

  def handler(self):
    """Do common stuff then dispatch the request to get() or put() methods."""
    self.response = flask.make_response()
    handler_start_time = time.time()
    logging.info('\n\n\nRequest handler: %r', self)

    self.preProcess()

    try:
      self.ratelimiter.CheckStart(self.request)

      with self.mr.profiler.Phase('parsing request and doing lookups'):
        self.mr.ParseFlaskRequest(self.request, self.services)

      self.response.headers['X-Frame-Options'] = 'SAMEORIGIN'

      if self.request.method == 'POST':
        self.post()
      elif self.request.method == 'GET':
        self.get()
    except exceptions.NoSuchUserException as e:
      logging.info('Trapped NoSuchUserException %s', e)
      flask.abort(404, 'user not found')

    except exceptions.NoSuchGroupException as e:
      logging.warning('Trapped NoSuchGroupException %s', e)
      flask.abort(404, 'user group not found')

    except exceptions.InputException as e:
      logging.info('Rejecting invalid input: %r', e)
      self.response.status_code = httplib.BAD_REQUEST

    except exceptions.NoSuchProjectException as e:
      logging.info('Rejecting invalid request: %r', e)
      self.response.status_code = httplib.NOT_FOUND

    except xsrf.TokenIncorrect as e:
      logging.info('Bad XSRF token: %r', e.message)
      self.response.status_code = httplib.BAD_REQUEST

    except permissions.BannedUserException as e:
      logging.warning('The user has been banned')
      #TODO: reidrect to userbanned page

    except ratelimiter.RateLimitExceeded as e:
      logging.info('RateLimitExceeded Exception %s', e)
      self.response.status_code = httplib.BAD_REQUEST
      self.response.response = 'Slow your roll.'

    finally:
      self.mr.CleanUp()
      self.ratelimiter.CheckEnd(self.request, time.time(), handler_start_time)

    total_processing_time = time.time() - handler_start_time
    logging.info(
        'Processed request in %d ms', int(total_processing_time * 1000))

    end_count0, end_count1, end_count2 = gc.get_count()
    logging.info('gc counts: %d %d %d', end_count0, end_count1, end_count2)
    # TODO: get the GC event back
    # if (end_count0 < count0) or (end_count1 < count1) or(end_count2 < count2):
    #   GC_EVENT_REQUEST.increment()

    if settings.enable_profiler_logging:
      self.mr.profiler.LogStats()

    return self.response

  def get(self):
    #TODO: implement basic data processing
    logging.info('process get request')

  def post(self):
    #TODO: implement basic data processing
    logging.info('process post request')
