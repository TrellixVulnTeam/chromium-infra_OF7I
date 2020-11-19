#!/usr/bin/env python
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import json
import ssl
import sys
import urllib

import certifi

# Make sure up-to-date root certificates are used.
urllib._urlopener = urllib.FancyURLopener(
    context=ssl.create_default_context(cafile=certifi.where()))


def do_latest():
  print json.load(urllib.urlopen(
      'https://registry.npmjs.org/firebase-tools'))['dist-tags']['latest']


def main():
  ap = argparse.ArgumentParser()
  sub = ap.add_subparsers()

  latest = sub.add_parser("latest")
  latest.set_defaults(func=lambda _opts: do_latest())

  checkout = sub.add_parser("checkout")
  checkout.add_argument("checkout_path")
  # we're going to use npm to actually do the fetch in install.sh
  checkout.set_defaults(func=lambda opts: None)

  opts = ap.parse_args()
  return opts.func(opts)


if __name__ == '__main__':
  sys.exit(main())
