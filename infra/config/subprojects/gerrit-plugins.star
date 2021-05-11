# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of CQ for the infra/gerrit-plugins repos."""

load("//lib/infra.star", "infra")

BASE_REPO_URL = "https://chromium.googlesource.com/infra/gerrit-plugins/"
BUILDER_NAME = "Gerrit Plugins Tester"

luci.cq_group(
    name = "gerrit-plugins",
    watch = cq.refset(
        # TODO(gavinmak): Include other plugins.
        repo = BASE_REPO_URL + "tricium",
        refs = ["refs/heads/main"],
    ),
)

luci.builder(
    name = BUILDER_NAME,
    bucket = "try",
    executable = infra.recipe("gerrit_plugins"),
    dimensions = {
        # TODO(tandrii): switch entirely to 18.04 once pool has enough of them.
        "os": "Ubuntu-16.04|Ubuntu-18.04",
        "cpu": "x86-64",
        "pool": "luci.flex.try",
    },
    service_account = infra.SERVICE_ACCOUNT_TRY,
)

luci.cq_tryjob_verifier(
    builder = BUILDER_NAME,
    cq_group = "gerrit-plugins",
)
