# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pylint: skip-file

import traceback

from lib import pages


def process_page(request):
  """Fetch, process and upload Rietveld pages.

  Defines a cloud function to fetch pages from an existing Rietveld instance,
  process it to remove dynamic content, and upload the resulting page to Google
  Storage.

  Args:
    request: A dict containing entries for 'path', 'type' and 'private'.
      path: The page to fetch, e.g. '/1234/patchset/5'
      type: One of 'Issue', 'PatchSet' or 'Patch'.
      private: Whether this page is from a private Rietveld issue.
  """
  try:
    params = request.get_json(force=True, silent=True)
    path = params['Path']
    page_type = params['EntityKind']
    private = params['Private']

    pages.process_page(path, page_type, private)

    return ''

  except pages.FatalError as e:
    # Log exception using structured logs, so problems are easy to find.
    # https://cloud.google.com/functions/docs/monitoring/logging#writing_structured_logs
    print(json.dumps({
      'severity': 'ERROR',
      'exception': 'FatalError',
      'message': str(e),
      'path': path,
      'page_type': page_type,
      'private': private,
    }))
    # Ignore exception, so that cloud tasks doesn't retry it.
    return ''
