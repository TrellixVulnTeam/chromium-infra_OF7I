
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
#
# This file is appended to Chromium Python's "ssl.py" module.
#
# Chromium modifies "ssl.py" to set this default SSL certificate path to the
# sort of path a native system Python would include. We determine this by
# probing the local environment on startup and seeing if we can identfy an
# OpenSSL certificate layout.
#
# If we can't, no default certificate authority bundle will be installed.
#
# The contents of this file are appended to "//lib/python3.7/ssl.py".

import sys as _ssl_suffix_sys
import os as _ssl_suffix_os

def _darwin_synthesize_cert_pem():
  import io
  import ctypes
  import ctypes.util
  from ctypes import byref, memmove
  from ctypes import c_void_p, c_char, c_long, c_int32, c_char_p

  CF = ctypes.CDLL(ctypes.util.find_library('CoreFoundation'))
  CF.CFArrayAppendValue.argtypes = [c_void_p, c_void_p]
  CF.CFArrayCreateMutableCopy.argtypes = [c_void_p, c_long, c_void_p]
  CF.CFArrayCreateMutableCopy.restype = c_void_p
  CF.CFArrayGetValueAtIndex.argtypes = [c_void_p, c_int32]
  CF.CFArrayGetValueAtIndex.restype = c_void_p
  CF.CFDataGetBytePtr.argtypes = [c_void_p]
  CF.CFDataGetBytePtr.restype = c_void_p
  CF.CFDataGetLength.argtypes = [c_void_p]
  CF.CFDataGetLength.restype = c_long
  CF.CFDictionaryAddValue.argtypes = [c_void_p, c_void_p, c_void_p]
  CF.CFDictionaryAddValue.restype = None
  CF.CFDictionaryCreateMutable.restype = c_void_p
  CF.CFRelease.argtypes = [c_void_p]

  SEC = ctypes.CDLL(ctypes.util.find_library('Security'))
  SEC.SecItemCopyMatching.argtypes = [c_void_p, c_void_p]
  SEC.SecItemCopyMatching.restype = c_int32
  SEC.SecKeychainCopySearchList.argtypes = [c_void_p]
  SEC.SecKeychainCopySearchList.restype = c_int32
  SEC.SecKeychainOpen.argtypes = [c_char_p, c_void_p]
  SEC.SecKeychainOpen.restype = c_long

  errSecItemNotFound = -25300

  to_release = []
  try:
    lst = c_void_p(0)
    assert SEC.SecKeychainCopySearchList(byref(lst)) == 0
    to_release.append(lst)

    search_list = CF.CFArrayCreateMutableCopy(
        None, CF.CFArrayGetCount(lst)+1, lst)
    to_release.append(search_list)

    # Attempt to add the SystemRootCertificates keychain to the search path
    # as well.
    root_cert_kc = c_void_p(0)
    root_certs = (
      '/System/Library/Keychains/SystemRootCertificates.keychain'
    ).encode('utf-8')
    if SEC.SecKeychainOpen(root_certs, byref(root_cert_kc)) == 0:
      CF.CFArrayAppendValue(search_list, root_cert_kc)
      to_release.append(root_cert_kc)

    def getConst(refname):
      return c_void_p.in_dll(SEC, refname)

    query = CF.CFDictionaryCreateMutable(None, 3, None, None)
    to_release.append(query)

    # We want to find all the "Certificate" items in the keychains we're
    # searching.
    CF.CFDictionaryAddValue(
      query, getConst('kSecClass'), getConst('kSecClassCertificate'))
    # We want ALL the certs loaded into the system (the default would only
    # return the first cert).
    CF.CFDictionaryAddValue(
      query, getConst('kSecMatchLimit'), getConst('kSecMatchLimitAll'))
    # Return raw data (CFDataRef's). Since kSecClass == kSecClassCertificate,
    # these will be DER-encoded certs. Not specifying this will return
    # SecCertificateRef's instead, which requires an extra function call
    # to get them as DER-encoded data.
    CF.CFDictionaryAddValue(
      query, getConst('kSecReturnData'), getConst('kCFBooleanTrue'))
    # Search these keychains (default search list + SystemRootCertificates).
    CF.CFDictionaryAddValue(
      query, getConst('kSecMatchSearchList'), search_list)

    # Items is going to be a CFArrayRef wit CFDataRef's in it, once
    # SecItemCopyMatching fills it in.
    items = c_void_p(0)
    result = SEC.SecItemCopyMatching(query, byref(items))
    if result == errSecItemNotFound:
      _ssl_suffix_sys.stderr.write('found zero certs in System Keychain\n')
      return
    elif result != 0:
      _ssl_suffix_sys.stderr.write(
        'failed to find certs in System Keychain: OSStatus(%r)' % result)
      return
    to_release.append(items)

    # Now we've got all the certs in DER encoding. Since we want to be able to
    # call load_verify_locations with cadata we can either give it ASN.1
    # DER-encoded certs, or PEM certs. We don't have an easy way (that
    # I could find) to generate an ASN.1 DER encoded cert bundle here, but PEM
    # certs are bundled by just cat'ing them together, so we do that.
    cert_pem = io.StringIO()
    for i in range(CF.CFArrayGetCount(items)):
      data = CF.CFArrayGetValueAtIndex(items, i)
      siz = CF.CFDataGetLength(data)
      buf = bytearray(siz)
      char_array = c_char * len(buf)
      memmove(char_array.from_buffer(buf), CF.CFDataGetBytePtr(data), siz)
      # DER_cert_to_PEM_cert is a symbol inside of ssl.py
      # pylint: disable=undefined-variable
      cert_pem.write(DER_cert_to_PEM_cert(buf))

    return cert_pem.getvalue()
  finally:
    for itm in reversed(to_release):
      CF.CFRelease(itm)


def _linux_find_load_verify_locations_kwargs():
  kwargs = {}

  # Borrowed from
  # https://github.com/golang/go/blob/master/src/crypto/x509/root_linux.go#L9
  ca_files = [
    # Debian/Ubuntu/Gentoo etc.
    '/etc/ssl/certs/ca-certificates.crt',
    '/etc/pki/tls/certs/ca-bundle.crt',                  # Fedora/RHEL 6
    '/etc/ssl/ca-bundle.pem',                            # OpenSUSE
    '/etc/pki/tls/cacert.pem',                           # OpenELEC
    '/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem', # CentOS/RHEL 7
    '/etc/ssl/cert.pem',                                 # Alpine Linux
  ]
  for fname in ca_files:
    if _ssl_suffix_os.path.isfile(fname):
      kwargs['cafile'] = fname
      break

  # Borrowed from
  # https://github.com/golang/go/blob/master/src/crypto/x509/root_unix.go#L16
  ca_paths = [
    # SLES10/SLES11, https://golang.org/issue/12139
    '/etc/ssl/certs',
    '/system/etc/security/cacerts', # Android
    '/usr/local/share/certs',       # FreeBSD
    '/etc/pki/tls/certs',           # Fedora/RHEL
    '/etc/openssl/certs',           # NetBSD
    '/var/ssl/certs',               # AIX
  ]
  for dname in ca_paths:
    if _ssl_suffix_os.path.isdir(dname):
      kwargs['capath'] = dname
      break

  return kwargs


def _override_set_default_verify_paths():
  kwargs = {}

  if _ssl_suffix_sys.platform.startswith('darwin'):
    # On OS X, we can use the Security.framework to obtain all the certs
    # installed to the system keychains. We calculate them once and cache them.
    #
    # If you install new certs in the various keychains, you'll need to restart
    # the python process... but that seems like a fair tradeoff to make.
    kwargs = {'cadata': _darwin_synthesize_cert_pem()}

  elif _ssl_suffix_sys.platform.startswith('linux'):
    # On linux we have an easier job; we search well-known locations for
    # cert.pem.
    #
    # As soon as we find one with certs in it, we stop and change
    # set_default_verify_paths to load from that location.
    #
    # We look for a cert.pem as well as a 'certs' folder.
    kwargs = _linux_find_load_verify_locations_kwargs()

  # Now we override set_default_verify_paths.
  if kwargs:
    # DER_cert_to_PEM_cert is a symbol inside of ssl.py
    # pylint: disable=undefined-variable
    SSLContext.set_default_verify_paths = (
        lambda self: self.load_verify_locations(**kwargs))
  else:
    _ssl_suffix_sys.stderr.write(
        "WARNING: no system SSL certs found. SSL verification may fail.\n"
    )

_override_set_default_verify_paths()
del _darwin_synthesize_cert_pem
del _linux_find_load_verify_locations_kwargs
del _override_set_default_verify_paths
del _ssl_suffix_sys
del _ssl_suffix_os
