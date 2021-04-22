# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of CQ for the infra/gerrit-plugins repos."""

load("//lib/build.star", "build")

BASE_REPO_URL = "https://chromium.googlesource.com/infra/gerrit-plugins/"

luci.cq_group(
    name = "gerrit-plugins",
    watch = cq.refset(
        # TODO(gavinmak): Include other plugins.
        repo = BASE_REPO_URL + "tricium",
        refs = ["refs/heads/main", "refs/heads/master"],
    ),
)

build.presubmit(
    name = "Gerrit Plugins Presubmit",
    cq_group = "gerrit-plugins",
    repo_name = "gerrit_plugins",
)
