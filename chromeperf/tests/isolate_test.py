# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest

import base64
import unittest
import zlib

from chromeperf.services import isolate

_FILE_HASH = 'c6911f39564106542b28081c81bde61c43121bda'
_ISOLATED_HASH = 'fc5e63011ae25b057b3097eba4413fc357c05cff'


def test_IsolateService_RetrieveUrl(request_json, service_request):
  request_json.return_value = {
      'url':
          'https://isolateserver.storage.googleapis.com/default-gzip/' +
          _FILE_HASH
  }
  service_request.return_value = zlib.compress(b'file contents')

  file_contents = isolate.retrieve('https://isolate.com', _FILE_HASH)
  assert file_contents == b'file contents'

  url = 'https://isolate.com/_ah/api/isolateservice/v1/retrieve'
  body = {'namespace': {'namespace': 'default-gzip'}, 'digest': _FILE_HASH}
  request_json.assert_called_once_with(url, 'POST', body)

  service_request.assert_called_once_with(
      'https://isolateserver.storage.googleapis.com/default-gzip/' + _FILE_HASH,
      'GET')


def test_IsolateService_RetrieveContent(request_json, service_request):
  request_json.return_value = {
      'content': base64.b64encode(zlib.compress(b'file contents'))
  }

  isolate_contents = isolate.retrieve('https://isolate.com', _ISOLATED_HASH)
  assert isolate_contents == b'file contents'

  url = 'https://isolate.com/_ah/api/isolateservice/v1/retrieve'
  body = {
      'namespace': {
          'namespace': 'default-gzip'
      },
      'digest': _ISOLATED_HASH,
  }
  request_json.assert_called_once_with(url, 'POST', body)


def test_IsolateService_RetrieveUnknownFormat(request_json, service_request):
  request_json.return_value = {}

  with pytest.raises(NotImplementedError):
    isolate.retrieve('https://isolate.com', 'digest')
