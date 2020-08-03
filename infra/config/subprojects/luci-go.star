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

def ci_builder(name, os, tree_closing = False):
    infra.builder(
        name = name,
        bucket = "ci",
        executable = infra.recipe("luci_go"),
        os = os,
        triggered_by = [
            luci.gitiles_poller(
                name = "luci-go-gitiles-trigger",
                bucket = "ci",
                repo = REPO_URL,
            ),
        ],
        gatekeeper_group = "chromium.infra",
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
        experiment_percentage = None,
        properties = None):
    infra.builder(
        name = name,
        bucket = "try",
        executable = infra.recipe(recipe or "luci_go"),
        os = os,
        properties = properties,
    )
    luci.cq_tryjob_verifier(
        builder = name,
        cq_group = "luci-go",
        experiment_percentage = experiment_percentage,
        disable_reuse = (properties or {}).get("presubmit"),
    )

ci_builder(name = "luci-go-continuous-trusty-64", os = "Ubuntu-14.04", tree_closing = True)
ci_builder(name = "luci-go-continuous-xenial-64", os = "Ubuntu-16.04", tree_closing = True)
ci_builder(name = "luci-go-continuous-bionic-64", os = "Ubuntu-18.04", tree_closing = True)
ci_builder(name = "luci-go-continuous-mac-10.13-64", os = "Mac-10.13", tree_closing = True)
ci_builder(name = "luci-go-continuous-mac-10.14-64", os = "Mac-10.14", tree_closing = True)
ci_builder(name = "luci-go-continuous-mac-10.15-64", os = "Mac-10.15", tree_closing = True)
ci_builder(name = "luci-go-continuous-win10-64", os = "Windows-10", tree_closing = True)

try_builder(name = "luci-go-try-trusty-64", os = "Ubuntu-14.04")
try_builder(name = "luci-go-try-xenial-64", os = "Ubuntu-16.04", properties = {
    "run_integration_tests": True,
})

# TODO(tandrii): bump to 10.15 once available.
try_builder(name = "luci-go-try-mac", os = "Mac-10.13")
try_builder(name = "luci-go-try-win", os = "Windows-10")

try_builder(
    name = "luci-go-try-presubmit",
    os = "Ubuntu-16.04",
    properties = {"presubmit": True},
)

# Experimental trybot for building docker images out of luci-go.git CLs.
try_builder(
    name = "luci-go-try-images",
    os = "Ubuntu-16.04",
    recipe = "images_builder",
    experiment_percentage = 100,
    properties = {
        "mode": "MODE_CL",
        "project": "PROJECT_LUCI_GO",
        "infra": "try",
        "manifests": ["infra/build/images/deterministic/luci"],
    },
)
