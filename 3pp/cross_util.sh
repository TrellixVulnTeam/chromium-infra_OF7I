#!/bin/bash
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Utility functions to handle cross-compiling from 3pp install scripts.

# Switches to building for the host platform, if applicable.
# Typically this should be invoked in a subshell, so the environment
# is reset afterwards to compile for the target platform.
3pp_toggle_host() {
  if [[ "$_3PP_PLATFORM" != "$_3PP_TOOL_PLATFORM" ]]; then
    if [[ $_3PP_PLATFORM == mac* ]]; then
      unset CROSS_TRIPLE
      unset CCC_OVERRIDE_OPTIONS
    else
      # Assume dockcross.
      . /install-util.sh
      toggle_host

      # TODO(iannucci): fix toggle_host to correctly export the 'host' compiler.
      # This is because the docker images currently set an alternative for `cc`
      # and `gcc` in /usr/bin to be the xcompile gcc. None of the other tools in
      # /usr/bin are switched though...
      if command -v gcc-9 > /dev/null; then
        export CC=gcc-9
      elif command -v gcc-8 > /dev/null; then
        export CC=gcc-8
      elif command -v gcc-6 > /dev/null; then
        export CC=gcc-6
      elif command -v gcc-4.9 > /dev/null; then
        export CC=gcc-4.9
      fi
    fi
  fi
}
