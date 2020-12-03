# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pylint: skip-file

import os
import unittest
import unittest.mock as mock

import requests

from requests.exceptions import HTTPError
from google.cloud import storage

mock.patch('requests.Session').start()
mock.patch('google.cloud.storage.Client').start()

import pages


class ProcessPageTest(unittest.TestCase):
  def setUp(self):
    self.env = {
      'HOST': 'https://www.example.com',
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
    mock.patch('os.getenv', self.env.get).start()
    mock.patch('pages.client').start()
    mock.patch('pages.session').start()
    self.addCleanup(mock.patch.stopall)

    self.response = pages.session.get()
    self.response.status_code = 200
    self.response.text = self.content
    self.response.headers = {'content-type': 'text/html; charset=UTF-8'}

    self.blob_fn = pages.client.get_bucket.return_value.blob
    self.blob = self.blob_fn.return_value

  def testUnknownPageType(self):
    with self.assertRaises(pages.FatalError):
      pages.process_page('/path', 'UNKNOWN_PAGE_TYPE', False)

  def testTransientErrorIsForwarded(self):
    self.response.raise_for_status.side_effect = HTTPError('503')

    self.response.status_code = 503
    with self.assertRaises(HTTPError):
      pages.process_page('/path', pages.PATCH, False)

    self.response.status_code = 429
    with self.assertRaises(requests.exceptions.HTTPError):
      pages.process_page('/path', pages.PATCH, False)

  def testUploadsToGoogleStorage_Issue(self):
    pages.process_page('/123', pages.ISSUE, False)

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
    pages.process_page('/123/patch/1', pages.PATCH_SET, False)

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
    pages.process_page('/123/patch/1/3', pages.PATCH, False)

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
    pages.process_page('/123/patch/1/3', pages.PATCH, True)

    self.blob_fn.assert_called_once_with('/123/patch/1/3')
    self.blob.upload_from_string.assert_called_once_with(self.content.encode())
    self.assertEqual(
        {'Rietveld-Private': True, 'Status-Code': 200},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()

  def testUploadsToGoogleStorage_StatusCode(self):
    self.response.status_code = 404

    pages.process_page('/123/patch/1/3', pages.PATCH, False)

    self.blob_fn.assert_called_once_with('/123/patch/1/3')
    self.blob.upload_from_string.assert_called_once_with(self.content)
    self.assertEqual(
        {'Rietveld-Private': False, 'Status-Code': 404},
        self.blob.metadata)
    self.assertEqual(
        'text/html; charset=UTF-8',
        self.blob.content_type)
    self.blob.patch.assert_called_once_with()


if __name__ == '__main__':
  unittest.main()
