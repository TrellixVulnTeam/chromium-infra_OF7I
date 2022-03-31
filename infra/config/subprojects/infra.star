# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of infra.git CI resources."""

load("//lib/build.star", "build")
load("//lib/infra.star", "infra")
load("//lib/recipes.star", "recipes")

infra.console_view(name = "infra", title = "infra/infra repository console")
infra.cq_group(name = "infra", tree_status_host = "infra-status.appspot.com")

def ci_builder(
        name,
        os,
        cpu = None,
        recipe = None,
        use_python3 = True,
        console_category = None,
        properties = None,
        extra_dimensions = None,
        schedule = None,
        infra_triggered = True,
        tree_closing = False):
    infra.builder(
        name = name,
        bucket = "ci",
        executable = infra.recipe(recipe or "infra_continuous", use_python3 = use_python3),
        os = os,
        cpu = cpu,
        triggered_by = [infra.poller()] if infra_triggered else None,
        schedule = schedule,
        properties = properties,
        extra_dimensions = extra_dimensions,
        notifies = infra.tree_closing_notifiers() if tree_closing else None,
    )
    luci.console_view_entry(
        builder = name,
        console_view = "infra",
        category = console_category or infra.category_from_os(os),
    )

def try_builder(
        name,
        os,
        cpu = None,
        recipe = None,
        experiment_percentage = None,
        properties = None,
        in_cq = True,
        use_python3 = True):
    infra.builder(
        name = name,
        bucket = "try",
        executable = infra.recipe(recipe or "infra_repo_trybot", use_python3 = use_python3),
        os = os,
        cpu = cpu,
        properties = properties,
    )
    if in_cq:
        luci.cq_tryjob_verifier(
            builder = name,
            cq_group = "infra",
            experiment_percentage = experiment_percentage,
        )

# Linux as the main platform to test with the most recent Go version (aka
# "bleeding_edge"). It was picked arbitrarily.
#
# All OSX builders are testing specifically with the older Go version
# (aka "legacy") to reflect the fact that OSX amd64 binaries we build need to
# run on relatively ancient OSX versions that don't support the bleeding edge
# Go.

# CI Linux.
ci_builder(name = "infra-continuous-bionic-arm64", os = "Ubuntu-18.04", cpu = "arm64", console_category = "linux|18.04|ARM", properties = {
    "go_version_variant": "bleeding_edge",
})
ci_builder(name = "infra-continuous-bionic-64", os = "Ubuntu-18.04", tree_closing = True, properties = {
    "go_version_variant": "bleeding_edge",
})

# CI OSX.
ci_builder(name = "infra-continuous-mac-10.13-64", os = "Mac-10.13", tree_closing = True, properties = {
    "go_version_variant": "legacy",
})
ci_builder(name = "infra-continuous-mac-10.14-64", os = "Mac-10.14", tree_closing = True, properties = {
    "go_version_variant": "legacy",
})
ci_builder(name = "infra-continuous-mac-10.15-64", os = "Mac-10.15", tree_closing = True, properties = {
    "go_version_variant": "legacy",
})

# CI Win.
ci_builder(name = "infra-continuous-win10-64", os = "Windows-10", tree_closing = True)

# CI for building docker images.
ci_builder(
    name = "infra-continuous-images",
    os = "Ubuntu",  # note: exact Linux version doesn't really matter
    recipe = "images_builder",
    console_category = "misc",
    properties = {
        "mode": "MODE_CI",
        "project": "PROJECT_INFRA",
        "infra": "ci",
        "manifests": ["infra/build/images/deterministic"],
    },
)

# Builds arm64-flavored docker images for swarm_docker.
ci_builder(
    name = "swarm-docker-arm64-image-builder",
    os = "Ubuntu-18.04",
    cpu = "arm64",
    # Make sure we're not building Docker images inside another container.
    extra_dimensions = {"inside_docker": "0"},
    infra_triggered = False,  # No need to build at every commit.
    schedule = "triggered",
    recipe = "docker_image_builder",
    console_category = "misc",
    properties = {
        "container_name": "swarm_docker_arm64",
        "dir_name": "swarm_docker",
    },
)

# All trybots.
try_builder(name = "infra-try-bionic-64", os = "Ubuntu-18.04", properties = {
    "go_version_variant": "bleeding_edge",
})

try_builder(name = "infra-try-mac", os = "Mac-10.15", properties = {
    "go_version_variant": "legacy",
})
try_builder(name = "infra-try-win", os = "Windows-10")

try_builder(name = "infra-try-frontend", os = "Ubuntu-18.04", recipe = "infra_frontend_tester")

try_builder(
    name = "infra-analysis",
    os = "Ubuntu-18.04",
    recipe = "tricium_infra",
    properties = {
        "gclient_config_name": "infra",
        "patch_root": "infra",
        "analyzers": ["Copyright", "Eslint", "Gosec", "Spellchecker", "InclusiveLanguageCheck"],
    },
    in_cq = False,
)

# Experimental trybot for building docker images out of infra.git CLs.
try_builder(
    name = "infra-try-images",
    os = "Ubuntu",
    recipe = "images_builder",
    experiment_percentage = 100,
    properties = {
        "mode": "MODE_CL",
        "project": "PROJECT_INFRA",
        "infra": "try",
        "manifests": ["infra/build/images/deterministic"],
    },
)

# Presubmit trybot.
build.presubmit(name = "infra-try-presubmit", cq_group = "infra", repo_name = "infra")

# Recipes ecosystem.
recipes.simulation_tester(
    name = "infra-continuous-recipes-tests",
    project_under_test = "infra",
    triggered_by = infra.poller(),
    console_view = "infra",
    console_category = "misc",
)

# Recipe rolls from Infra.
recipes.roll_trybots(
    upstream = "infra",
    downstream = [
        "build",
    ],
    cq_group = "infra",
)

luci.cq_tryjob_verifier(
    builder = "infra-internal:try/build_limited Roll Tester (infra)",
    cq_group = "infra",
)

luci.cq_tryjob_verifier(
    builder = "infra-internal:try/chrome_release Roll Tester (infra)",
    cq_group = "infra",
)

# Tryjobs for 3pp wheel builders.
def wheel_tryjob(builder):
    luci.cq_tryjob_verifier(
        builder = builder,
        cq_group = "infra",
        location_regexp = [
            ".+/[+]/infra/tools/dockerbuild/.+",
            ".+/[+]/recipes/recipes/build_wheels.py",
        ],
    )

wheel_tryjob("infra-internal:try/Universal wheel builder")
wheel_tryjob("infra-internal:try/Linux x64 wheel builder")
wheel_tryjob("infra-internal:try/Linux ARM wheel builder")
wheel_tryjob("infra-internal:try/Mac wheel builder")
wheel_tryjob("infra-internal:try/Mac ARM64 wheel builder")
wheel_tryjob("infra-internal:try/Windows-x64 wheel builder")
wheel_tryjob("infra-internal:try/Windows-x86 wheel builder")

# Tryjobs for 3pp package builders.
def tpp_tryjob(builder):
    luci.cq_tryjob_verifier(
        builder = builder,
        cq_group = "infra",
        location_regexp = [
            ".+/[+]/3pp/.+",
        ],
    )

tpp_tryjob("infra-internal:try/3pp linux-amd64")
tpp_tryjob("infra-internal:try/3pp linux-arm64")
tpp_tryjob("infra-internal:try/3pp linux-armv6l")
tpp_tryjob("infra-internal:try/3pp mac-amd64")
tpp_tryjob("infra-internal:try/3pp mac-arm64")
tpp_tryjob("infra-internal:try/3pp windows-386")
tpp_tryjob("infra-internal:try/3pp windows-amd64")

# Placeholder tryjob for Buildbucket integration testing
infra.builder(
    bucket = "try",
    name = "placeholder",
    os = "Ubuntu",
    executable = luci.recipe(
        name = "engine_placeholder",
        recipe = "placeholder",
        cipd_package = "infra/recipe_bundles/chromium.googlesource.com/infra/luci/recipes-py",
        use_python3 = True,
    ),
    properties = {
        "status": "SUCCESS",
    },
)
