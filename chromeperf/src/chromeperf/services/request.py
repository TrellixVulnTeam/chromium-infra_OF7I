# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
import httplib2
import json
import socket
import urllib

# We depend directly on the google-auth library, as we only intend to use
# the service account credentials in a GCP deployment.
import google.auth

_CACHE_DURATION = 60 * 60 * 24 * 7  # 1 week.
_VULNERABILITY_PREFIX = ")]}'\n"
_EMAIL_SCOPE = 'https://www.googleapis.com/auth/userinfo.email'


class RequestError(httplib2.HttpLib2Error):

  def __init__(self, msg, content):
    super(RequestError, self).__init__(msg)
    self.content = content


class NotFoundError(RequestError):
  """Raised when a request gives a HTTP 404 error."""


def request_json(*args, **kwargs):
  """Fetch a URL and JSON-decode the response.

  See the documentation for Request() for details
  about the arguments and exceptions.
  """
  content = request(*args, **kwargs)
  if content.startswith(_VULNERABILITY_PREFIX):
    content = content[len(_VULNERABILITY_PREFIX):]
  return json.loads(content)


def request(url,
            method='GET',
            body=None,
            use_auth=True,
            scopes=[_EMAIL_SCOPE],
            **parameters):
  """Fetch a URL while authenticated as the service account.

  Args:
    method: The HTTP request method. E.g. 'GET', 'POST', 'PUT'.
    body: The request body as a Python object. It will be JSON-encoded.
    parameters: Parameters to be encoded in the URL query string.

  Returns:
    The response body.

  Raises:
    NotFoundError: The HTTP status code is 404.
    httplib.HTTPException: The request or response is malformed, or there is a
        network or server error, or the HTTP status code is not 2xx.
  """
  if parameters:
    # URL-encode the parameters.
    for key, value in list(parameters.items()):
      if value is None:
        del parameters[key]
      if isinstance(value, bool):
        parameters[key] = str(value).lower()
    url += '?' + urllib.parse.urlencode(sorted(parameters.items()), doseq=True)

  kwargs = {'method': method}
  if body:
    # JSON-encode the body.
    kwargs['body'] = json.dumps(body)
    kwargs['headers'] = {'Content-Type': 'application/json'}

  try:
    content = _request_and_process_http_errors(url, use_auth, scopes, **kwargs)
  except NotFoundError:
    raise
  except (httplib2.HttpLib2Error, socket.error):
    # Retry once.
    content = _request_and_process_http_errors(url, use_auth, scopes, **kwargs)

  return content


def _authorize_client(scopes=[_EMAIL_SCOPE], timeout=None):
  # Use the application default credentials here.
  credentials, _ = google.auth.default(scopes=scopes)
  client = httplib2.Http(timeout=timeout)
  credentials.authorize(client)
  return client


def _request_and_process_http_errors(url, use_auth, scopes, **kwargs):
  """Requests a URL, converting HTTP errors to Python exceptions."""

  if use_auth:
    http = _authorize_client(scopes=scopes, timeout=60)
  else:
    http = httplib2.Http(timeout=60)

  response, content = http.request(url, **kwargs)

  if response['status'] == '404':
    raise NotFoundError(
        'HTTP status code %s: %s' % (response['status'], repr(content[0:100])),
        content)
  if not response['status'].startswith('2'):
    raise RequestError(
        'Failure in request for `%s`; HTTP status code %s: %s' %
        (url, response['status'], repr(content[0:100])), content)

  return content
