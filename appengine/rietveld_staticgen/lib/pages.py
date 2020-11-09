# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.


def process_page(path, page_type, private):
  """Fetch, process and upload Rietveld pages.

  Fetch pages from an existing Rietveld instance, process it to remove dynamic
  content, and upload the resulting page to Google Storage.

  Args:
    path: The page to fetch, e.g. '/1234/patchset/5'
    page_type: One of 'Issue', 'PatchSet' or 'Patch'.
    private: Whether this page is from a private Rietveld issue.
  """
  del path
  del page_type
  del private
  raise Exception('Not implemented')
