# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of luci-py.git CI resources."""

load("//lib/build.star", "build")
load("//lib/infra.star", "infra")

cq_group = "luci-py"

infra.cq_group(
    name = cq_group,
    repo = "https://chromium.googlesource.com/infra/luci/luci-py",
)

def try_builder(
        name,
        os,
        recipe = None,
        experiment_percentage = None,
        properties = None,
        in_cq = True,
        use_python3 = True):
    infra.builder(
        name = name,
        bucket = "try",
        executable = infra.recipe(recipe or "luci_py", use_python3 = use_python3),
        os = os,
        properties = properties,
    )
    if in_cq:
        luci.cq_tryjob_verifier(
            builder = name,
            cq_group = cq_group,
            experiment_percentage = experiment_percentage,
        )

build.presubmit(
    name = "luci-py-try-presubmit",
    cq_group = cq_group,
    repo_name = "luci_py",
    os = "Ubuntu-18.04",
    # The default 8-minute timeout is a problem for luci-py.
    # See https://crbug.com/917479 for context.
    timeout_s = 900,
)

try_builder(
    name = "luci-py-analysis",
    os = "Ubuntu-18.04",
    recipe = "tricium_infra",
    properties = {
        "gclient_config_name": "luci_py",
        "patch_root": "infra/luci",
        "analyzers": ["Spellchecker"],
    },
    in_cq = False,
)

try_builder(
    name = "luci-py-try-bionic-64",
    os = "Ubuntu-18.04",
)

try_builder(
    name = "luci-py-try-mac10.15-64",
    os = "Mac-10.15",
)

try_builder(
    name = "luci-py-try-win10-64",
    os = "Windows-10",
)
