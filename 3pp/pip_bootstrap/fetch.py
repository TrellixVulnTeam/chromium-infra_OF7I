#!/usr/bin/env python
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import os
import re
import sys

import pkg_resources
import requests


def _get_wheel_url(pkgname, version):
  r = requests.get('https://pypi.org/pypi/%s/json' % pkgname)
  r.raise_for_status()
  for filedata in r.json()['releases'][version]:
    if filedata['packagetype'] == 'bdist_wheel':
      return filedata['url'], filedata['filename']
  raise AssertionError('could not find wheel for %s @ %s' % (pkgname, version))


def _get_version(pkgname, bad_versions=()):
  r = requests.get('https://pypi.org/pypi/%s/json' % pkgname)
  r.raise_for_status()
  # Find the latest python2-compatible version.
  releases = r.json()['releases']
  versions = [pkg_resources.parse_version(v) for v in releases.keys()]
  for version in sorted(versions, reverse=True):
    version_str = str(version)
    for filedata in releases[version_str]:
      if (version_str not in bad_versions and
          filedata['packagetype'] == 'bdist_wheel' and
          not filedata['yanked'] and filedata['python_version'] != 'py3'):
        return version_str
  raise AssertionError('could not find a compatible version for %s' % pkgname)


def do_latest():
  setuptools_bad_versions = frozenset([
      '45.0.0',  # Advertises python_version='py2.py3', but also requires >= 3.5
  ])

  print 'pip%s.setuptools%s.wheel%s' % (_get_version(
      'pip'), _get_version('setuptools', bad_versions=setuptools_bad_versions),
                                        _get_version('wheel'))


def do_checkout(version, checkout_path):
  # split version pip<vers>.setuptools<vers>.wheel<vers>
  m = re.match(
    r'^pip(.*)\.setuptools(.*)\.wheel(.*)$',
    version)
  versions = {
    'pip': m.group(1),
    'setuptools': m.group(2),
    'wheel': m.group(3),
  }
  for pkgname, vers in versions.iteritems():
    url, name = _get_wheel_url(pkgname, vers)
    print >>sys.stderr, "fetching", url
    r = requests.get(url, stream=True)
    r.raise_for_status()
    with open(os.path.join(checkout_path, name), 'wb') as f:
      for chunk in r.iter_content(1024**2):
        f.write(chunk)


def main():
  ap = argparse.ArgumentParser()
  sub = ap.add_subparsers()

  latest = sub.add_parser("latest")
  latest.set_defaults(func=lambda _opts: do_latest())

  checkout = sub.add_parser("checkout")
  checkout.add_argument("checkout_path")
  checkout.set_defaults(
    func=lambda opts: do_checkout(
      os.environ['_3PP_VERSION'], opts.checkout_path))

  opts = ap.parse_args()
  return opts.func(opts)


if __name__ == '__main__':
  sys.exit(main())
