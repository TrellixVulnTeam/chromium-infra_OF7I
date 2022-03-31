# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of luci-go.git CI resources."""

load("//lib/infra.star", "infra")

REPO_URL = "https://chromium.googlesource.com/infra/luci/luci-go"

infra.console_view(
    name = "luci-go",
    title = "luci-go repository console",
    repo = REPO_URL,
)
infra.cq_group(name = "luci-go", repo = REPO_URL)

def ci_builder(name, os, tree_closing = False, properties = None, use_python3 = True):
    infra.builder(
        name = name,
        bucket = "ci",
        executable = infra.recipe("luci_go", use_python3 = use_python3),
        os = os,
        properties = properties,
        triggered_by = [
            luci.gitiles_poller(
                name = "luci-go-gitiles-trigger",
                bucket = "ci",
                repo = REPO_URL,
                refs = ["refs/heads/main"],
            ),
        ],
        notifies = infra.tree_closing_notifiers() if tree_closing else None,
    )
    luci.console_view_entry(
        builder = name,
        console_view = "luci-go",
        category = infra.category_from_os(os),
    )

def try_builder(
        name,
        os,
        recipe = None,
        properties = None,
        in_cq = True,
        experiment_percentage = None,
        owner_whitelist = None,
        mode_allowlist = None,
        use_python3 = True):
    infra.builder(
        name = name,
        bucket = "try",
        executable = infra.recipe(recipe or "luci_go", use_python3 = use_python3),
        os = os,
        properties = properties,
    )
    if in_cq:
        luci.cq_tryjob_verifier(
            builder = name,
            cq_group = "luci-go",
            experiment_percentage = experiment_percentage,
            owner_whitelist = owner_whitelist,
            mode_allowlist = mode_allowlist,
        )

# Linux as the main platform to test with the most recent Go version (aka
# "bleeding_edge"). It was picked arbitrarily.
#
# All OSX builders are testing specifically with the older Go version
# (aka "legacy") to reflect the fact that OSX amd64 binaries we build need to
# run on relatively ancient OSX versions that don't support the bleeding edge
# Go.

ci_builder(name = "luci-go-continuous-bionic-64", os = "Ubuntu-18.04", tree_closing = True, properties = {
    "go_version_variant": "bleeding_edge",
    "run_integration_tests": True,
})
ci_builder(name = "luci-go-continuous-mac-10.13-64", os = "Mac-10.13", tree_closing = True, properties = {
    "go_version_variant": "legacy",
})
ci_builder(name = "luci-go-continuous-mac-10.14-64", os = "Mac-10.14", tree_closing = True, properties = {
    "go_version_variant": "legacy",
})
ci_builder(name = "luci-go-continuous-mac-10.15-64", os = "Mac-10.15", tree_closing = True, properties = {
    "go_version_variant": "legacy",
})
ci_builder(name = "luci-go-continuous-win10-64", os = "Windows-10", tree_closing = True)

try_builder(name = "luci-go-try-bionic-64", os = "Ubuntu-18.04", properties = {
    "run_integration_tests": True,
    "go_version_variant": "bleeding_edge",
})
try_builder(name = "luci-go-try-mac", os = "Mac-10.15", properties = {
    "go_version_variant": "legacy",
})
try_builder(name = "luci-go-try-win", os = "Windows-10")

try_builder(
    name = "luci-go-try-presubmit",
    os = "Ubuntu",
    properties = {"presubmit": True},
)

# Experimental trybot for building docker images out of luci-go.git CLs.
try_builder(
    name = "luci-go-try-images",
    os = "Ubuntu",
    recipe = "images_builder",
    experiment_percentage = 100,
    properties = {
        "mode": "MODE_CL",
        "project": "PROJECT_LUCI_GO",
        "infra": "try",
        "manifests": ["infra/build/images/deterministic/luci"],
    },
)

try_builder(
    name = "luci-go-analysis",
    os = "Ubuntu-18.04",
    recipe = "tricium_infra",
    properties = {
        "gclient_config_name": "luci_go",
        "patch_root": "infra/go/src/go.chromium.org/luci",
        "analyzers": ["Gosec", "Spellchecker"],
    },
    owner_whitelist = ["project-infra-tryjob-access"],
    mode_allowlist = [cq.MODE_ANALYZER_RUN],
)

try_builder(
    name = "luci-go-lint",
    os = "Ubuntu-18.04",
    properties = {
        "run_lint": True,
    },
    owner_whitelist = ["project-infra-tryjob-access"],
    mode_allowlist = [cq.MODE_ANALYZER_RUN],
)
