#!/usr/bin/env python
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import json
import os
import re
import sys
import urllib


# A regex for a name of the release asset to package, available at
# https://github.com/activescott/lessmsi
WINDOWS_ASSET_RE = re.compile(r'^lessmsi-v.*\.zip$')


def do_latest():
  print json.load(
      urllib.urlopen(
          'https://api.github.com/repos/activescott/lessmsi/releases/latest')
  )['tag_name'].lstrip('v')


# TODO(akashmukherjee): Remove.
def do_checkout(version, checkout_path):
  download_url = None
  asset_name = None

  target_tag = 'v%s' % (version,)
  for release in json.load(
      urllib.urlopen(
          'https://api.github.com/repos/activescott/lessmsi/releases')):
    if str(release['tag_name']) == target_tag:
      for asset in release['assets']:
        asset_name = str(asset['name'])
        if WINDOWS_ASSET_RE.match(asset_name):
          download_url = asset['browser_download_url']
          break
      break
  if not download_url:
    raise Exception('could not find download_url')

  print >>sys.stderr, "fetching", download_url
  urllib.urlretrieve(download_url, os.path.join(checkout_path, asset_name))


def get_download_url(version):
  download_url = None

  target_tag = 'v%s' % (version,)
  for release in json.load(
      urllib.urlopen(
          'https://api.github.com/repos/activescott/lessmsi/releases')):
    if str(release['tag_name']) == target_tag:
      for asset in release['assets']:
        asset_name = str(asset['name'])
        if WINDOWS_ASSET_RE.match(asset_name):
          download_url = asset['browser_download_url']
          break
      break
  if not download_url:
    raise Exception('could not find download_url')

  partial_manifest = {
    'url': download_url,
    'ext': '.zip',
  }
  print(json.dumps(partial_manifest))


def main():
  ap = argparse.ArgumentParser()
  sub = ap.add_subparsers()

  latest = sub.add_parser("latest")
  latest.set_defaults(func=lambda _opts: do_latest())

  # TODO(akashmukherjee): Remove.
  checkout = sub.add_parser("checkout")
  checkout.add_argument("checkout_path")
  checkout.set_defaults(
    func=lambda opts: do_checkout(
      os.environ['_3PP_VERSION'], opts.checkout_path))

  download = sub.add_parser("get_url")
  download.set_defaults(
    func=lambda opts: get_download_url(os.environ['_3PP_VERSION']))

  opts = ap.parse_args()
  return opts.func(opts)


if __name__ == '__main__':
  sys.exit(main())
