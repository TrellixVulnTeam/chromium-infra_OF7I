# Copyright 2022 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

create {
  source {
    git {
      repo: "https://github.com/evanw/esbuild.git"
      tag_pattern: "v%s"
    }
    patch_version: "chromium.1"
  }
  build {
    tool: "tools/go"
  }
}

upload { pkg_prefix: "tools" }
