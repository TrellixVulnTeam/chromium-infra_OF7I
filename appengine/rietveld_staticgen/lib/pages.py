# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pylint: skip-file

import os
import posixpath
import requests

from google.cloud import storage

from . import process_content


ISSUE = 'Issue'
PATCH_SET = 'PatchSet'
PATCH = 'Patch'
KNOWN_PAGE_TYPES = (ISSUE, PATCH_SET, PATCH)


session = requests.Session()
client = storage.Client()


class FatalError(Exception):
  """Fatal failure to process a Rietveld page.

  The task to process the rietveld page failed with a fatal error and shouldn't
  be retried.
  """
  pass


def process_page(path, page_type, private):
  """Fetch, process and upload Rietveld pages.

  Fetch pages from an existing Rietveld instance, process it to remove dynamic
  content, and upload the resulting page to Google Storage.

  Args:
    path: The page to fetch, e.g. '/1234/patchset/5'
    page_type: One of 'Issue', 'PatchSet' or 'Patch'.
    private: Whether this page is from a private Rietveld issue.

  Raises FatalError if we failed to process the page due to a non-transient
  issue.
  """
  if page_type not in KNOWN_PAGE_TYPES:
    message = 'Expected page type to be one of {}, got {}'.format(
        KNOWN_PAGE_TYPES, page_type)
    raise FatalError(message)

  response = session.get(
      posixpath.join(os.getenv('RIETVELD_HOST'), path),
      headers=_get_auth_headers())

  # Forward transient errors to the client so tasks can be retried.
  if response.status_code >= 500 or response.status_code == 429:
    response.raise_for_status()

  # Process page content to remove dynamic links and unarchived pages.
  content = response.text
  if response.status_code == 200:
    content = _process_content(page_type, content)

  # Add a '/index.html' for issue pages. This makes it more convenient to browse
  # issues on Google Storage.
  if page_type == ISSUE:
    if not path.endswith('/'):
      path += '/'
    path += 'index.html'

  # Upload processed page to Google Storage
  bucket = client.get_bucket(os.getenv('BUCKET_NAME'))
  blob = bucket.blob(path)
  blob.upload_from_string(content)
  blob.metadata = {
    'Rietveld-Private': private,
    'Status-Code': response.status_code,
  }
  blob.content_type = response.headers['content-type']
  blob.patch()


def _get_auth_headers():
  # Fetch access token from metadata server.
  # https://cloud.google.com/run/docs/securing/service-identity#access_tokens
  TOKEN_URL = ('http://metadata.google.internal/computeMetadata/v1'
               '/instance/service-accounts/default/token')
  TOKEN_HEADERS = {'Metadata-Flavor': 'Google'}

  response = session.get(TOKEN_URL, headers=TOKEN_HEADERS)
  response.raise_for_status()

  # Extract the access token from the response.
  access_token = response.json()['access_token']

  return {'Authorization': 'Bearer {}'.format(access_token)}


def _process_content(page_type, content):
  try:
    if page_type == ISSUE:
      return process_content.process_issue(content)
    if page_type == PATCH_SET:
      return process_content.process_patch_set(content)
    if page_type == PATCH:
      return process_content.process_patch(content)
  except Exception as e:
    raise FatalError(e)
