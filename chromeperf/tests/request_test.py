# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest

import httplib2
import socket

from chromeperf.services import request


@pytest.fixture
def authorize_mock(mocker):
  mocked_client = mocker.MagicMock()
  mocked_client.request = mocker.MagicMock()
  mocked_client.return_value = mocked_client
  mocker.patch('chromeperf.services.request._authorize_client', mocked_client)
  return mocked_client


def test_Request(authorize_mock):
  authorize_mock.request.return_value = ({'status': '200'}, 'response')
  response = request.request('https://example.com')
  authorize_mock.assert_called_once_with(
      scopes=[request._EMAIL_SCOPE], timeout=60)
  assert response == 'response'


def test_RequestJson(authorize_mock):
  authorize_mock.request.return_value = ({'status': '200'}, '"response"')
  response = request.request_json('https://example.com')
  authorize_mock.request.assert_called_once_with(
      'https://example.com', method='GET')
  authorize_mock.assert_called_once_with(
      scopes=[request._EMAIL_SCOPE], timeout=60)
  assert response == 'response'


def test_RequestJsonWithPrefix(authorize_mock):
  authorize_mock.request.return_value = ({'status': '200'}, ')]}\'\n"response"')
  response = request.request_json('https://example.com')
  authorize_mock.request.assert_called_once_with(
      'https://example.com', method='GET')
  authorize_mock.assert_called_once_with(
      scopes=[request._EMAIL_SCOPE], timeout=60)
  assert response == 'response'


def test_RequestWithBodyAndParameters(authorize_mock):
  authorize_mock.request.return_value = ({'status': '200'}, 'response')
  response = request.request(
      'https://example.com',
      'POST',
      body='a string',
      url_param_1='value_1',
      url_param_2='value_2')
  authorize_mock.request.assert_called_once_with(
      'https://example.com?url_param_1=value_1&url_param_2=value_2',
      method='POST',
      body='"a string"',
      headers={'Content-Type': 'application/json'})
  assert response == 'response'

def _test_retry(authorize_mock):
  response = request.request('https://example.com')
  authorize_mock.request.assert_called_with('https://example.com', method='GET')
  assert authorize_mock.request.call_count == 2
  assert response == 'response'

def test_Request_HttpErrorCode(authorize_mock):
  authorize_mock.request.return_value = ({'status': '500'}, '')
  with pytest.raises(httplib2.HttpLib2Error):
    request.request('https://example.com')
  authorize_mock.request.assert_called_with('https://example.com', method='GET')
  assert authorize_mock.request.call_count == 2

def test_Request_HttpException(authorize_mock):
  authorize_mock.request.side_effect = httplib2.HttpLib2Error
  with pytest.raises(httplib2.HttpLib2Error):
    request.request('https://example.com')
  authorize_mock.request.assert_called_with('https://example.com', method='GET')
  assert authorize_mock.request.call_count == 2


def test_Request_SocketError(authorize_mock):
  authorize_mock.request.side_effect = socket.error
  with pytest.raises(socket.error):
    request.request('https://example.com')
  authorize_mock.request.assert_called_with(
      'https://example.com', method='GET')
  assert authorize_mock.request.call_count == 2


def test_Request_NotFound(authorize_mock):
  authorize_mock.request.return_value = ({'status': '404'}, '')
  with pytest.raises(request.NotFoundError):
    request.request('https://example.com')
  authorize_mock.request.assert_called_with('https://example.com', method='GET')
  assert authorize_mock.request.call_count == 1

def test_Request_HttpNotAuthorized(authorize_mock):
  authorize_mock.request.return_value = ({'status': '403'}, b'\x00\xe2')
  with pytest.raises(request.RequestError):
    request.request('https://example.com')


def test_Request_HttpErrorCodeSuccessOnRetry(authorize_mock):
  failure_return_value = ({'status': '500'}, '')
  success_return_value = ({'status': '200'}, 'response')
  authorize_mock.request.side_effect = (failure_return_value,
                                        success_return_value)
  _test_retry(authorize_mock)


def test_Request_HttpExceptionSuccessOnRetry(authorize_mock):
  return_value = ({'status': '200'}, 'response')
  authorize_mock.request.side_effect = (httplib2.HttpLib2Error, return_value)
  _test_retry(authorize_mock)

def test_Request_SocketErrorSuccessOnRetry(authorize_mock):
  return_value = ({'status': '200'}, 'response')
  authorize_mock.request.side_effect = (socket.error, return_value)
  _test_retry(authorize_mock)

def test_Request_NoAuth(authorize_mock, mocker):
  http = mocker.MagicMock()
  http.request.return_value = ({'status': '200'}, 'response')
  patcher = mocker.patch('httplib2.Http')
  patcher.return_value = http

  response = request.request('https://example.com', use_auth=False)

  patcher.assert_called_once_with(timeout=60)
  http.request.assert_called_once_with('https://example.com', method='GET')
  assert authorize_mock.call_count == 0
  assert response == 'response'
