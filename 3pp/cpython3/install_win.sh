#!/bin/bash
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"

# Remove all .msi bits we don't want.
rm -vrf -- *_d.msi *_pdb.msi test.msi tcl*.msi doc.msi launcher.msi path.msi \
  pip.msi tools.msi

# Extract the rest of the msi's to the current directory.
for x in *.msi; do
  lessmsi x "$x" "$(cygpath -w "$(pwd)")\\"
done

# Move the meat where we actually want it.
mkdir "$PREFIX/bin"
mv SourceDir/* "$PREFIX/bin"
ls "$PREFIX/bin"

# Install pip_bootstrap things.
"$PREFIX/bin/python.exe" "$(where pip_bootstrap.py)" "$PREFIX/bin"

# This is full of .exe shims which don't work correctly unless you put
# python.exe on %PATH% (via a hack in pip_bootstrap.py). Currently (2018/11/12)
# we don't put python.exe on %PATH% for devs, and we don't use these shims on
# bots.
#
# Rather than have a folder full of maybe-broken exes, we remove them here.
#
# However, when https://bitbucket.org/vinay.sajip/simple_launcher/issues/4 is
# fixed, we can stop doing this (but will maybe have to tweak pip_bootstrap
# somehow to take advantage of the new syntax).
rm -vrf "$PREFIX/bin/Scripts"

mv "$PREFIX/bin/python.exe" "$PREFIX/bin/python3.exe"

# Don't distribute __pycache__. Because the file modification times are not
# preserved in the CIPD package, Python will try to regenerate the compiled
# code, but will not overwrite an existing read-only file, effectively
# disabling the compiled code cache.
find "$PREFIX" -name __pycache__ -exec rm -rf {} +
