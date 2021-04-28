# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

create {
  source {
    git {
      repo: "https://github.com/golangci/golangci-lint.git"
      tag_pattern: "v%s"
    }
  }
  build {
    tool: "tools/go"
  }
}

upload { pkg_prefix: "tools" }
