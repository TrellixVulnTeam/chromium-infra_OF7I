#!/usr/bin/env python
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import json
import os
import sys
import urllib

from pkg_resources import parse_version


def do_latest():
  versions = []
  for release in json.load(urllib.urlopen('https://golang.org/dl/?mode=json')):
    ver = parse_version(release['version'].replace('go', ''))
    if ver < parse_version('1.16'):
      versions.append(ver)
  versions.sort()
  print versions[-1]


def get_download_url(version, platform, kind):
  if kind == 'prebuilt':
    platform = platform.replace('mac', 'darwin')
    ext = '.zip' if platform.startswith('windows') else '.tar.gz'
    download_url = (('https://storage.googleapis.com/golang/go%(version)s.' +
                     '%(platform)s%(ext)s') % {
                         'version': version,
                         'platform': platform,
                         'ext': ext
                     })
  else:
    ext = '.tar.gz'
    download_url = ('https://storage.googleapis.com/golang/go%s.src.tar.gz' %
                    (version,))

  partial_manifest = {
      'url': [download_url],
      'ext': ext,
  }
  print(json.dumps(partial_manifest))


def main():
  ap = argparse.ArgumentParser()
  ap.add_argument('kind', choices=('prebuilt', 'source'))

  sub = ap.add_subparsers()

  latest = sub.add_parser('latest')
  latest.set_defaults(func=lambda _opts: do_latest())

  download = sub.add_parser('get_url')
  download.set_defaults(func=lambda opts: get_download_url(
      os.environ['_3PP_VERSION'], os.environ['_3PP_PLATFORM'], opts.kind))

  opts = ap.parse_args()
  return opts.func(opts)


if __name__ == '__main__':
  sys.exit(main())
