#!/bin/bash
# Copyright 2022 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script builds httpd and php and all of the dependencies needed.We are
# doing this because it's difficult to build all of the libraries statically.
# We are instead building shared libraries and bundling them with the executable.
# This works because we can use install_name_tool to change the library search
# path on Mac

set -e
set -x

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

apr_version="1.6.5"
apr_util_version="1.6.1"
expat_version="2.4.7"
httpd_version="2.4.38"
libxml2_version="2.9.12"
openssl_version="1.1.1j"
pcre_version="8.41"
php_version="7.3.31"
zlib_version="1.2.12"

out="$1"
build="$PWD/build"
src="$PWD/src"

jobs=8

mkdir "${build}"
mkdir "${src}"

cd "${src}"

echo "Building zlib"
tar xf "../zlib-${zlib_version}.tar.gz"
cd "zlib-${zlib_version}"
# This is necessary for zlib to detect that we're using gcc
cc=$CC ./configure --prefix="${build}"
make -j"${jobs}"
make install
cd ..

echo "Building OpenSSL"
tar xf "../openssl-${openssl_version}.tar.gz"
cd "openssl-${openssl_version}"
./config no-tests no-asm --prefix="${build}"
make -j"${jobs}"
make install_sw
cd ..

echo "Building PCRE"
tar xf "../pcre-${pcre_version}.tar.gz"
cd "pcre-${pcre_version}"
./configure --prefix="${build}"
make -j"${jobs}"
make install
cd ..

echo "Building APR"
tar xf "../apr-${apr_version}.tar.gz"
cd "apr-${apr_version}"
patch -p1 < ${SCRIPT_DIR}/patches/0001-Handle-macOS-11-and-later-properly.patch
CFLAGS="-Wno-format -Wno-implicit-function-declaration" ./configure --prefix="${build}"
make -j"${jobs}"
make install
cd ..

echo "Building expat"
tar xf "../expat-${expat_version}.tar.gz"
cd "expat-${expat_version}"
./configure --host="$CROSS_TRIPLE" --prefix="${build}"
make "-j$(jobs)"
make install
cd ..

echo "Building APR-util"
tar xf "../apr-util-${apr_util_version}.tar.gz"
cd "apr-util-${apr_util_version}"
./configure --prefix="${build}" --with-apr="${build}" --with-expat="${build}"
make -j"${jobs}"
make install
cd ..

echo "Building httpd"
tar xf "../httpd-${httpd_version}.tar.gz"
cd "httpd-${httpd_version}"
# See third_party/blink/tools/apache_config/apache2-httpd-2.4-php7.conf for the
# modules to enable. Build modules as shared libraries to match the LoadModule
# lines (the ServerRoot option will let httpd discover them), but we statically
# link dependencies to avoid runtime linker complications.
./configure --prefix="${build}" \
    --enable-access-compat=shared \
    --enable-actions=shared \
    --enable-alias=shared \
    --enable-asis=shared \
    --enable-authz-core=shared \
    --enable-authz-host=shared \
    --enable-autoindex=shared \
    --enable-cgi=shared \
    --enable-env=shared \
    --enable-headers=shared \
    --enable-imagemap=shared \
    --enable-include=shared \
    --enable-log-config=shared \
    --enable-mime=shared \
    --enable-modules=none \
    --enable-negotiation=shared \
    --enable-rewrite=shared \
    --enable-ssl=shared \
    --enable-unixd=shared \
    --libexecdir="${build}/libexec/apache2" \
    --with-apr-util="${build}" \
    --with-apr="${build}" \
    --with-mpm=prefork \
    --with-pcre="${build}" \
    --with-ssl="${build}"
make -j"${jobs}"
make install
cd ..

echo "Building libxml2"
tar xf "../libxml2-${libxml2_version}.tar.gz"
cd "libxml2-${libxml2_version}"
./configure --with-python=no --prefix="${build}"
make "-j$(jobs)"
make install
cd ..

echo "Building PHP"
tar xf "../php-${php_version}.tar.gz"
cd "php-${php_version}"
patch -p1 < ${SCRIPT_DIR}/patches/libtool.patch.txt
./buildconf --force
./configure --prefix="${build}" \
    --disable-cgi \
    --disable-cli \
    --with-apxs2="${build}/bin/apxs" \
    --with-zlib="${build}" \
    --with-libxml-dir="${build}" \
    --without-iconv
make -j"${jobs}"
make install
cd ..

bin_files="
    bin/httpd
    bin/openssl"
if [[ $OSTYPE == darwin* ]]
then
  lib_files="
      lib/libapr-1.0.dylib
      lib/libaprutil-1.0.dylib
      lib/libcrypto.1.1.dylib
      lib/libexpat.1.dylib
      lib/libpcre.1.dylib
      lib/libpcrecpp.0.dylib
      lib/libpcreposix.0.dylib
      lib/libssl.1.1.dylib
      lib/libxml2.2.dylib
      lib/libz.1.dylib"
else
  lib_files="
      lib/libapr-1.so.0
      lib/libaprutil-1.so.0
      lib/libcrypto.so.1.1
      lib/libexpat.so.1
      lib/libpcre.so.1
      lib/libpcrecpp.so.0
      lib/libpcreposix.so.0
      lib/libssl.so.1.1
      lib/libxml2.so.2
      lib/libz.so.1"
fi
libexec_files="
    libexec/apache2/libphp7.so
    libexec/apache2/mod_access_compat.so
    libexec/apache2/mod_actions.so
    libexec/apache2/mod_alias.so
    libexec/apache2/mod_asis.so
    libexec/apache2/mod_authz_core.so
    libexec/apache2/mod_authz_host.so
    libexec/apache2/mod_autoindex.so
    libexec/apache2/mod_cgi.so
    libexec/apache2/mod_env.so
    libexec/apache2/mod_headers.so
    libexec/apache2/mod_imagemap.so
    libexec/apache2/mod_include.so
    libexec/apache2/mod_log_config.so
    libexec/apache2/mod_mime.so
    libexec/apache2/mod_negotiation.so
    libexec/apache2/mod_rewrite.so
    libexec/apache2/mod_ssl.so
    libexec/apache2/mod_unixd.so"
license_files="
    apr-${apr_version}/LICENSE
    apr-${apr_version}/NOTICE
    apr-util-${apr_util_version}/LICENSE
    apr-util-${apr_util_version}/NOTICE
    expat-${expat_version}/COPYING
    httpd-${httpd_version}/LICENSE
    httpd-${httpd_version}/NOTICE
    libxml2-${libxml2_version}/Copyright
    openssl-${openssl_version}/LICENSE
    pcre-${pcre_version}/LICENCE
    php-${php_version}/LICENSE"

echo "Copying files"
mkdir "${out}/bin"
mkdir "${out}/lib"
mkdir "${out}/libexec"
mkdir "${out}/libexec/apache2"

cat > "${out}/LICENSE" <<EOT
This directory contains binaries for Apache httpd, PHP, and their dependencies.
License and notices for each are listed below:
EOT

for f in ${license_files}; do
  echo >> "${out}/LICENSE"
  echo "=======================" >> "${out}/LICENSE"
  echo >> "${out}/LICENSE"
  echo "${f}:" >> "${out}/LICENSE"
  cat "${src}/${f}" >> "${out}/LICENSE"
done

# zlib does not have a standalone LICENSE file. Extract it from the README
# instead.
echo >> "${out}/LICENSE"
echo "=======================" >> "${out}/LICENSE"
echo >> "${out}/LICENSE"
echo "From zlib-${zlib_version}/README:" >> "${out}/LICENSE"
sed -n -e '/^Copyright notice:/,//p' "${src}/zlib-${zlib_version}/README" >> "${out}/LICENSE"

for f in ${bin_files} ${lib_files} ${libexec_files}; do
  cp "${build}/${f}" "${out}/${f}"
  if [[ $OSTYPE == darwin* ]]
  then
    for lib in ${lib_files}; do
      install_name_tool -change "${build}/${lib}" "@rpath/$(basename "${lib}")" "${out}/${f}"
    done
  fi
done
if [[ $OSTYPE == darwin* ]]
then
  for f in ${bin_files}; do
    install_name_tool -add_rpath "@executable_path/../lib" "${out}/${f}"
  done
  for f in ${lib_files}; do
    install_name_tool -id "@rpath/$(basename "${f}")" "${out}/${f}"
  done
  for f in ${libexec_files}; do
    install_name_tool -id "@rpath/../libexec/$(basename "${f}")" "${out}/${f}"
  done

  # Verify that no absolute build paths have leaked into the output.
  for f in ${bin_files} ${lib_files} ${libexec_files}; do
    if otool -l "${out}/${f}" | grep ${build}; then
      echo "ERROR: Absolute path found in binary ${f}"
      exit 1
    fi
  done
fi

if [[ $OSTYPE == linux* ]]
then
  # The docker environment uses libcrypt.so.2, which isn't available
  # where we run the resulting binary.
  cp /usr/local/lib/libcrypt.so.2 "${out}/lib"
fi
