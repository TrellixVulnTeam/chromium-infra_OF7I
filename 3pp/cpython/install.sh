#!/bin/bash
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"
DEPS_PREFIX="$2"

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

# Determine our Python interpreter version. It will use PEP440's "local
# version identifier" to specify a local Python version based on our
# $PATCH_VERSION.
PY_VERSION="$_3PP_VERSION+${_3PP_PATCH_VERSION}"

# Don't add $DEPS_PREFIX to CPPFLAGS and LDFLAGS until after we've built
# the bootstrap python! LDFLAGS, in particular, is already in the environment
# for dockcross containers, so these changes are picked up by the configure
# script. The 3pp-built libraries in $DEPS_PREFIX cannot be used to build the
# bootstrap python because we build it with shared modules, but the
# $DEPS_PREFIX libraries are not compiled with -fPIC. Fortunately, the system
# versions of these libraries are sufficient for running the bootstrap
# interpreter.

export CONFIG_ARGS="--host $CROSS_TRIPLE"

# This module is broken, and seems to reference a non-existent symbol
# at compile time.
SETUP_LOCAL_SKIP=(_testcapi)
SETUP_LOCAL_ATTACH=(
  "$DEPS_PREFIX/lib/libbz2.a"
  "$DEPS_PREFIX/lib/libreadline.a"
  "$DEPS_PREFIX/lib/libpanel.a"
  "$DEPS_PREFIX/lib/libncurses.a"
  "$DEPS_PREFIX/lib/libsqlite3.a"
  "$DEPS_PREFIX/lib/libz.a"
  "$DEPS_PREFIX/lib/libssl.a"
  "$DEPS_PREFIX/lib/libcrypto.a"
)

if [[ $_3PP_PLATFORM == mac* ]]; then
    CONFIGURE_FFI="--with-system-ffi"
else
    CONFIGURE_FFI="--without-system-ffi"
fi

# First see if we have to build a runnable python; we'll need it later.
if [[ $_3PP_PLATFORM == "$_3PP_TOOL_PLATFORM" ]]; then  # not cross compiling
  # we need to make all the rest of the stuff like _struct and _functools in
  # order to use this interpreter. If we're cross compiling then we're using
  # the interpreter from path, which is already complete.

  autoconf

  ./configure --prefix="$(pwd)/host_interp" \
    --disable-shared ${CONFIGURE_FFI} --enable-ipv6

  # We need a couple of modules explicitly, so edit Modules/Setup

  # Insert "*static*" at the top of the file, make zlib and binascii
  # builtin modules.
  echo -en "*static*\n\n" > Modules/Setup.new
  sed -e "s/^#zlib/zlib/g" -e "s/^#binascii/binascii/g" Modules/Setup \
      >> Modules/Setup.new
  mv -f Modules/Setup.new Modules/Setup

  # Now regenerate the Makefile/config.c to pick up changes to Modules/Setup.
  rm Modules/config.c
  make Modules/config.c

  # Because of... something... setup.py tries to load in all modules which it
  # builds. This gets seriously horked when loading readline and running the 3pp
  # processes in make to touch stdin, so we redirect it to /dev/null.
  # recipe when attached to a terminal. We don't actually ever want any
  make install < /dev/null

  INTERP=$(pwd)/host_interp/bin/python
elif [[ $_3PP_PLATFORM == mac* ]]; then
  # When cross-compiling on Mac, explicitly force the system python2, so
  # we don't pick up vpython-native's python2 from $PATH. Running vpython
  # with "-s -S" below causes problems where built-in modules such as argparse
  # can't be loaded.
  INTERP=/usr/bin/python2
else
  # When cross-compiling on Linux, use the host python2 in the docker
  # container.
  INTERP=python2
fi

CPPFLAGS="-I$DEPS_PREFIX/include"
LDFLAGS="-L$DEPS_PREFIX/lib"

# OpenSSL 1.1.1 depends on pthread, so it needs to come LAST. Python's
# Makefile has an LDLAST to allow some ldflags to be the very last thing on
# the link line.
LDLAST="-lpthread"

# TODO(iannucci): Remove this once the fleet is using GLIBC 2.25 and
# macOS 10.12 or higher.
#
# See comment in 3pp/openssl/install.sh for more detail.
export ac_cv_func_getentropy=0

# Now, really configure and build.
if [[ $_3PP_PLATFORM == mac* ]]; then
  # Mac Python installations use 2-byte Unicode.
  UNICODE_TYPE=ucs2
  # Flags gathered from stock Python installation.
  EXTRA_CONFIGURE_ARGS="--with-threads --enable-toolbox-glue"

  # Instruct Mac to prefer ".a" files in earlier library search paths
  # rather than search all of the paths for a ".dylib" and then, failing
  # that, do a second sweep for ".a".
  LDFLAGS="$LDFLAGS -Wl,-search_paths_first"

  # Statically linking the _ctypes module requires linking against libffi.
  SETUP_LOCAL_ATTACH+=('_ctypes::-lffi')

  # Our builder system is missing X11 headers, so this module does not build.
  SETUP_LOCAL_SKIP+=(_tkinter)
  # QuickTime.framework is no longer present on macOS 10.15
  SETUP_LOCAL_SKIP+=(_Qt)

  # For use with cross-compiling.
  if [[ $_3PP_TOOL_PLATFORM == mac-arm64 ]]; then
      host_cpu="aarch64"
  else
      host_cpu="x86_64"
  fi
  EXTRA_CONFIGURE_ARGS="$EXTRA_CONFIGURE_ARGS --build=${host_cpu}-apple-darwin"
else
  # Linux Python (Ubuntu) installations use 4-byte Unicode.
  UNICODE_TYPE=ucs4
  EXTRA_CONFIGURE_ARGS="--with-fpectl --with-dbmliborder=bdb:gdbm"
  # NOTE: This can break building on Mac builder, causing it to freeze
  # during execution.
  #
  # Maybe look into this if we have time later.
  EXTRA_CONFIGURE_ARGS="$EXTRA_CONFIGURE_ARGS --enable-optimizations"

  # TODO(iannucci) This assumes we're building for linux under docker (which is
  # currently true).
  EXTRA_CONFIGURE_ARGS="$EXTRA_CONFIGURE_ARGS --build=x86_64-linux-gnu"

  # On Linux, we need to manually configure the embedded 'libffi' package
  # so the '_ctypes' module can link against it.
  #
  # This mirrors the non-Darwin 'libffi' path in the '_ctypes' code in
  # '//setup.py'.
  mkdir tpp_libffi
  (cd tpp_libffi && ../Modules/_ctypes/libffi/configure --host="$CROSS_TRIPLE")
  CPPFLAGS="$CPPFLAGS -Itpp_libffi -Itpp_libffi/include"

  # On Linux, we need to ensure that most symbols from our static-embedded
  # libraries (notably OpenSSL) don't get exported. If they do, they can
  # conflict with the same libraries from wheels or other dynamically
  # linked sources.
  #
  # This set of symbols was determined by trial, see:
  # - crbug.com/763792
  #
  # We'd like to use LDFLAGS_NODIST instead of LDFLAGS here, so that distutils
  # doesn't use this for building extensions. It would break the build, as
  # gnu_version_script.txt isn't available when we build wheels, and it's not
  # necessary there anyway.
  #
  # But LDFLAGS_NODIST isn't available on Python 2.7, so instead we have to
  # manually trim this from the output later on. Note: don't use a colon
  # (':') in LDFLAGS_TO_STRIP as that's used as a delimeter later.
  LDFLAGS_TO_STRIP="-Wl,--version-script=$SCRIPT_DIR/gnu_version_script.txt"
  LDFLAGS+=" $LDFLAGS_TO_STRIP"

  # The "crypt" module needs to link against glibc's "crypt" function. We link
  # it statically because our docker environment uses libcrypt.so.2, which isn't
  # available where we run the resulting binary.
  SETUP_LOCAL_ATTACH+=('crypt::-l:libcrypt.a')
  SETUP_LOCAL_ATTACH+=("$DEPS_PREFIX/lib/libnsl.a")
fi

# Assert blindly that the target distro will have /dev/ptmx and not /dev/ptc.
# This is likely to be true, since all linuxes that we know of have this
# configuration.
export ac_cv_file__dev_ptmx=y
export ac_cv_file__dev_ptc=n

# Generate our configure script.
autoconf

export LDFLAGS
export CPPFLAGS
export LDLAST
# Configure our production Python build with our static configuration
# environment and generate our basic platform.
#
# We're going to use our system python interpreter to generate our static module
# list.
./configure --prefix "$PREFIX" --host="$CROSS_TRIPLE" \
  --disable-shared ${CONFIGURE_FFI} --enable-ipv6 \
  --enable-py-version-override="$PY_VERSION" \
  --enable-unicode=$UNICODE_TYPE \
  --without-ensurepip \
  $EXTRA_CONFIGURE_ARGS

# Generate our "pybuilddir.txt" file. This also generates
# "_sysconfigdata.py" from our current Python, which we need to
# generate our module list, since it includes our "configure_env"'s
# CPPFLAGS, LDFLAGS, etc.
make platform

# Generate our static module list, "Modules/Setup.local". Python
# reads this during build and projects it into its Makefile.
#
# The "python_mod_gen.py" script extracts a list of modules by
# strategically invoking "setup.py", pretending that it's trying to
# build the modules, and capturing their output. It generates a
# "Setup.local" file.
#
# We need to run it with a Python interpreter that is compatible with
# this checkout. Enter the bootstrap interpreter! However, that is
# tailored to the bootstrap interpreter's environment ("bootstrap_dir"),
# not the production one ("checkout_dir"). We use the
# "python_build_bootstrap.py" script to strip that out and reorient
# it to point to our production directory prior to invoking
# "python_mod_gen.py".
#
# This is all a very elaborate (but adaptable) way to not hardcode
# "Setup.local" for each set of platforms that we support.
SETUP_LOCAL_FLAGS=()
for x in "${SETUP_LOCAL_SKIP[@]}"; do
  SETUP_LOCAL_FLAGS+=(--skip "$x")
done
for x in "${SETUP_LOCAL_ATTACH[@]}"; do
  SETUP_LOCAL_FLAGS+=(--attach "$x")
done

$INTERP -s -S "$SCRIPT_DIR/python_mod_gen.py" \
  --pybuilddir $(cat pybuilddir.txt) \
  --output ./Modules/Setup.local \
  "${SETUP_LOCAL_FLAGS[@]}"

# Build production Python.
make install

# Augment the Python installation.

# Read / augment / write the "ssl.py" module to implement custom SSL
# certificate loading logic.
#
# We do this here instead of "usercustomize.py" because the latter
# isn't propagated when a VirtualEnv is cut.
cat < "$SCRIPT_DIR/ssl_suffix.py" >> "$PREFIX/lib/python2.7/ssl.py"

# TODO: maybe strip python executable?

$INTERP `which pip_bootstrap.py` "$PREFIX"

# As mentioned above, there are some LDFLAGS that we use during build, but we
# don't want to use for building native extensions. Here we strip them from the
# config file which distutils will read.
if [[ $LDFLAGS_TO_STRIP ]]; then
  sed -i 's: $LDFLAGS_TO_STRIP::g' $PREFIX/lib/python*/_sysconfigdata.py
fi

# Clear out any pre-compiled versions of this file that might persist the bad
# LDFLAGS, just to be safe.
rm -f $PREFIX/lib/python*/_sysconfigdata.pyc
rm -f $PREFIX/lib/python*/_sysconfigdata.pyo

# Cleanup!
rm -rf $PREFIX/lib/*.a
rm -rf $PREFIX/lib/python*/test
rm -rf $PREFIX/lib/python*/*/*test*
rm -rf $PREFIX/lib/python*/config
rm $PREFIX/lib/python*/lib-dynload/*.{so,dylib} || true
