# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Trybot definitions for 3PP infrastructure."""

luci.cq_group(
    name = "wheels_3pp",
    watch = cq.refset(
        repo = "https://chromium.googlesource.com/infra/infra",
        refs = ["refs/heads/main"],
    ),
    retry_config = cq.RETRY_TRANSIENT_FAILURES,
)

def wheel_tryjob(builder):
    luci.cq_tryjob_verifier(
        builder = builder,
        cq_group = "wheels_3pp",
        location_regexp = [
            ".+/[+]/infra/tools/dockerbuild/.+",
            ".+/[+]/recipes/recipes/build_wheels.py",
        ],
    )

wheel_tryjob("infra-internal:try/Linux wheel builder")
wheel_tryjob("infra-internal:try/Mac wheel builder")
wheel_tryjob("infra-internal:try/Mac ARM64 wheel builder")
wheel_tryjob("infra-internal:try/Windows-x64 wheel builder")
wheel_tryjob("infra-internal:try/Windows-x86 wheel builder")
