#!/usr/bin/env python
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import json
import os
import sys
import urllib


def do_latest():
  print urllib.urlopen(
      'https://raw.githubusercontent.com/'
      'GoogleCloudPlatform/gsutil/master/VERSION').read().strip()


# TODO(akashmukherjee): Remove
def do_checkout(version, checkout_path):
  download_url = (
    'https://storage.googleapis.com/pub/gsutil_%s.tar.gz' % version)
  urllib.urlretrieve(download_url, os.path.join(checkout_path,
                                                'archive.tar.gz'))


def get_download_url(version):
  download_url = (
    'https://storage.googleapis.com/pub/gsutil_%s.tar.gz' % version)
  partial_manifest = {
    'url': download_url,
    'ext': '.tar.gz',
  }
  print(json.dumps(partial_manifest))


def main():
  ap = argparse.ArgumentParser()
  sub = ap.add_subparsers()

  latest = sub.add_parser("latest")
  latest.set_defaults(func=lambda _opts: do_latest())

  # TODO(akashmukherjee): Remove
  checkout = sub.add_parser("checkout")
  checkout.add_argument("checkout_path")
  checkout.set_defaults(
    func=lambda opts: do_checkout(os.environ['_3PP_VERSION'],
                                  opts.checkout_path))

  download = sub.add_parser("get_url")
  download.set_defaults(
    func=lambda opts: get_download_url(os.environ['_3PP_VERSION']))

  opts = ap.parse_args()
  return opts.func(opts)


if __name__ == '__main__':
  sys.exit(main())
