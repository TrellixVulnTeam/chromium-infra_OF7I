# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pylint: skip-file

import io
import json
import os
import unittest
import unittest.mock as mock

import requests

from requests.exceptions import HTTPError
from google.cloud import storage

mock.patch('requests.Session').start()
mock.patch('google.cloud.storage.Client').start()

import main

from lib import pages


class ProcessPageTest(unittest.TestCase):
  def setUp(self):
    self.env = {
      'RIETVELD_HOST': 'https://www.example.com',
      'BUCKET_NAME': 'BUCKET_NAME',
    }
    self.content = '\n'.join([
      '<html>',
      ' <body>',
      '  <p>',
      '   content',
      '  </p>',
      ' </body>',
      '</html>',
    ])
    self.stdout = io.StringIO()
    mock.patch('lib.pages.client').start()
    mock.patch('lib.pages.session').start()
    mock.patch('lib.pages._get_auth_headers').start()
    mock.patch('os.getenv', self.env.get).start()
    mock.patch('sys.stdout', self.stdout).start()
    self.addCleanup(mock.patch.stopall)

    self.request = mock.Mock()
    self.request.get_json.return_value = {
        'Path': '/123',
        'EntityKind': pages.ISSUE,
        'Private': False,
    }

    self.get_fn = pages.session.get
    self.response = self.get_fn.return_value
    self.response.status_code = 200
    self.response.text = self.content
    self.response.headers = {'content-type': 'text/html; charset=UTF-8'}

    self.blob_fn = pages.client.get_bucket.return_value.blob
    self.blob = self.blob_fn.return_value

  def testUnknownPageType(self):
    self.request.get_json.return_value = {
        'Path': '/path',
        'EntityKind': 'UNKNOWN_PAGE_TYPE',
        'Private': False,
    }

    _, status_code = main.process_page(self.request)

    self.assertEqual(500, status_code)

    log = json.loads(self.stdout.getvalue())
    self.assertEqual('ERROR', log['severity'])
    self.assertEqual(
        'AssertionError: Expected page type to be one of '
        '(\'Issue\', \'PatchSet\', \'Patch\'), got UNKNOWN_PAGE_TYPE',
        log['message'].splitlines()[-1])
    self.assertEqual(self.request.get_json.return_value, log['params'])

  def test5XXResponse(self):
    self.response.raise_for_status.side_effect = HTTPError('503')
    self.response.status_code = 503

    _, status_code = main.process_page(self.request)

    self.assertEqual(500, status_code)

    log = json.loads(self.stdout.getvalue())
    self.assertEqual('ERROR', log['severity'])
    self.assertEqual(
        'requests.exceptions.HTTPError: 503',
        log['message'].splitlines()[-1])
    self.assertEqual(self.request.get_json.return_value, log['params'])

    self.get_fn.assert_called_once_with(
        'https://www.example.com/123', headers=mock.ANY)
    self.blob_fn.assert_called_once_with('/123/index.html')
    self.blob.upload_from_string.assert_called_once_with(self.content)
    self.assertEqual(
        {'Rietveld-Private': False, 'Status-Code': 503},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()

  def test4XXResponse(self):
    self.response.raise_for_status.side_effect = HTTPError('429')
    self.response.status_code = 429

    _, status_code = main.process_page(self.request)

    self.assertEqual(500, status_code)

    log = json.loads(self.stdout.getvalue())
    self.assertEqual('ERROR', log['severity'])
    self.assertEqual(
        'requests.exceptions.HTTPError: 429',
        log['message'].splitlines()[-1])
    self.assertEqual(self.request.get_json.return_value, log['params'])

    self.get_fn.assert_called_once_with(
        'https://www.example.com/123', headers=mock.ANY)
    self.blob_fn.assert_called_once_with('/123/index.html')
    self.blob.upload_from_string.assert_called_once_with(self.content)
    self.assertEqual(
        {'Rietveld-Private': False, 'Status-Code': 429},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()

  def testNonTransientStatusCode(self):
    self.response.status_code = 404

    _, status_code = main.process_page(self.request)

    self.assertEqual('', self.stdout.getvalue())
    self.get_fn.assert_called_once_with(
        'https://www.example.com/123', headers=mock.ANY)
    self.blob_fn.assert_called_once_with('/123/index.html')
    self.blob.upload_from_string.assert_called_once_with(self.content)
    self.assertEqual(
        {'Rietveld-Private': False, 'Status-Code': 404},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()

  def testUploadsToGoogleStorage_NoLeadingSlash(self):
    self.request.get_json.return_value['Path'] = '123'

    _, status_code = main.process_page(self.request)

    self.get_fn.assert_called_once_with(
        'https://www.example.com/123', headers=mock.ANY)
    self.blob_fn.assert_called_once_with('/123/index.html')
    self.blob.upload_from_string.assert_called_once_with(self.content.encode())
    self.assertEqual(
        {'Rietveld-Private': False, 'Status-Code': 200},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()

  def testUploadsToGoogleStorage_Issue(self):
    _, status_code = main.process_page(self.request)

    self.get_fn.assert_called_once_with(
        'https://www.example.com/123', headers=mock.ANY)
    self.blob_fn.assert_called_once_with('/123/index.html')
    self.blob.upload_from_string.assert_called_once_with(self.content.encode())
    self.assertEqual(
        {'Rietveld-Private': False, 'Status-Code': 200},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()

  def testUploadsToGoogleStorage_PatchSet(self):
    self.request.get_json.return_value['Path'] = '/123/patch/1'
    self.request.get_json.return_value['EntityKind'] = pages.PATCH_SET

    _, status_code = main.process_page(self.request)

    self.get_fn.assert_called_once_with(
        'https://www.example.com/123/patch/1', headers=mock.ANY)
    self.blob_fn.assert_called_once_with('/123/patch/1')
    self.blob.upload_from_string.assert_called_once_with(self.content.encode())
    self.assertEqual(
        {'Rietveld-Private': False, 'Status-Code': 200},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()

  def testUploadsToGoogleStorage_Patch(self):
    self.request.get_json.return_value['Path'] = '/123/patch/1/3'
    self.request.get_json.return_value['EntityKind'] = pages.PATCH

    _, status_code = main.process_page(self.request)

    self.get_fn.assert_called_once_with(
        'https://www.example.com/123/patch/1/3', headers=mock.ANY)
    self.blob_fn.assert_called_once_with('/123/patch/1/3')
    self.blob.upload_from_string.assert_called_once_with(self.content.encode())
    self.assertEqual(
        {'Rietveld-Private': False, 'Status-Code': 200},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()

  def testUploadsToGoogleStorage_Private(self):
    self.request.get_json.return_value['Private'] = True

    _, status_code = main.process_page(self.request)

    self.get_fn.assert_called_once_with(
        'https://www.example.com/123', headers=mock.ANY)
    self.blob_fn.assert_called_once_with('/123/index.html')
    self.blob.upload_from_string.assert_called_once_with(self.content.encode())
    self.assertEqual(
        {'Rietveld-Private': True, 'Status-Code': 200},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()


if __name__ == '__main__':
  unittest.main()
