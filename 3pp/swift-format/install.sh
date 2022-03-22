#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

set -e
set -x
set -o pipefail

PREFIX="$1"

swift build -c release
cp $(swift build -c release --show-bin-path)/swift-format $PREFIX/swift-format

# swift-format depends on lib_InternalSwiftSyntaxParser.dylib.
# Apple github recommends to compile it or to ship the library with the application.
# see https://github.com/apple/swift-syntax#embedding-swiftsyntax-in-an-application
cp $(xcode-select -p)/Toolchains/XcodeDefault.xctoolchain/usr/lib/swift/macosx/lib_InternalSwiftSyntaxParser.dylib $PREFIX/lib_InternalSwiftSyntaxParser.dylib
cp LICENSE.txt $PREFIX/LICENSE.txt

