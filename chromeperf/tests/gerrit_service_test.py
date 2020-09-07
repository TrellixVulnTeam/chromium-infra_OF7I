# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest

from chromeperf.services import gerrit_service


def test_GerritService_GetChange(request_json):
  request_json.return_value = {'content': {}}
  server = 'https://chromium-review.googlesource.com'
  response = gerrit_service.get_change(server, 672011)
  assert response == {'content': {}}

  request_json.assert_called_once_with(
      server + '/changes/672011',
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE,
      o=None)


def test_GerritService_GetChangeWithFields(request_json):
  request_json.return_value = {'content': {}}
  server = 'https://chromium-review.googlesource.com'
  response = gerrit_service.get_change(server, 672011, fields=('FIELD_NAME',))
  assert response == {'content': {}}
  request_json.assert_called_once_with(
      server + '/changes/672011',
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE,
      o=('FIELD_NAME',))


def test_GerritService_PostChangeComment(request_json, service_request):
  request_json.return_value = {'content': {}}
  server = 'https://chromium-review.googlesource.com'
  gerrit_service.post_change_comment(server, 12334, 'hello!')
  service_request.assert_called_once_with(
      'https://chromium-review.googlesource.com/a/changes/12334'
      '/revisions/current/review',
      body='hello!',
      scope=gerrit_service.GERRIT_SCOPE,
      use_cache=False,
      method='POST',
      use_auth=True)
