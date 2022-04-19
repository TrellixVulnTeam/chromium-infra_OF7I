# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of resources for Code Search system."""

load("//lib/build.star", "build")
load("//lib/infra.star", "infra")
load("//lib/led.star", "led")

luci.bucket(
    name = "codesearch",
    acls = [
        acl.entry(
            roles = acl.BUILDBUCKET_TRIGGERER,
            users = "luci-scheduler@appspot.gserviceaccount.com",
        ),
        acl.entry(
            roles = acl.BUILDBUCKET_TRIGGERER,
            groups = "mdb/chrome-ops-source",
        ),
    ],
)

luci.realm(name = "pools/codesearch")

led.users(
    groups = [
        "mdb/chrome-troopers",
        "google/luci-task-force@google.com",
    ],
    task_realm = "codesearch",
    pool_realm = "pools/codesearch",
)

luci.console_view(
    name = "codesearch",
    repo = "https://chromium.googlesource.com/chromium/src",
    include_experimental_builds = True,
    refs = ["refs/heads/main"],
)

def builder(
        name,
        executable,

        # Builder props.
        os = None,
        cpu_cores = None,
        properties = None,
        builder_group_property_name = "mastername",
        caches = None,
        execution_timeout = None,

        # Console presentation.
        category = None,
        short_name = None,

        # Scheduler parameters.
        triggered_by = None,
        schedule = None):
    """A generic code search builder.

    Args:
      name: name of the builder.
      executable: a recipe to run.
      os: the target OS dimension.
      cpu_cores: the CPU cores count dimension (as string).
      properties: a dict with properties to pass to the recipe.
      builder_group_property_name: the name of the property to set with the
        builder group.
      caches: a list of swarming.cache(...).
      execution_timeout: how long it is allowed to run.
      category: the console category to put the builder under.
      short_name: a short name for the console.
      triggered_by: a list of builders that trigger this one.
      schedule: if given, run the builder periodically under this schedule.
    """

    # Add mastername property so that the gen recipes can find the right
    # config in mb_config.pyl.
    properties = properties or {}
    properties[builder_group_property_name] = "chromium.infra.codesearch"

    properties["$build/goma"] = {
        "server_host": "goma.chromium.org",
        "rpc_extra_params": "?prod",
        "enable_ats": True,  # True for Linux/Win only. Must set to false on Mac.
    }

    luci.builder(
        name = name,
        bucket = "codesearch",
        executable = executable,
        properties = properties,
        dimensions = {
            "os": os or "Ubuntu-18.04",
            "cpu": "x86-64",
            "cores": cpu_cores or "8",
            "pool": "luci.infra.codesearch",
        },
        caches = caches,
        service_account = "infra-codesearch@chops-service-accounts.iam.gserviceaccount.com",
        execution_timeout = execution_timeout,
        build_numbers = True,
        triggered_by = [triggered_by] if triggered_by else None,
        schedule = schedule,
        experiments = {"luci.recipes.use_python3": 100},
    )

    luci.console_view_entry(
        builder = name,
        console_view = "codesearch",
        category = category,
        short_name = short_name,
    )

def chromium_genfiles(short_name, name, os = None, cpu_cores = None):
    builder(
        name = name,
        executable = build.recipe("chromium_codesearch"),
        builder_group_property_name = "builder_group",
        os = os,
        cpu_cores = cpu_cores,
        caches = [swarming.cache(
            path = "generated",
            name = "codesearch_git_genfiles_repo",
        )],
        execution_timeout = 9 * time.hour,
        category = "gen",
        short_name = short_name,
        # Gen builders are triggered by the initiator's recipe.
        triggered_by = "codesearch-gen-chromium-initiator",
    )

def chromiumos_genfiles(name):
    builder(
        name = name,
        executable = build.recipe("chromiumos_codesearch"),
        builder_group_property_name = "builder_group",
        execution_timeout = 9 * time.hour,
        category = "gen",
        # Gen builders are triggered by the initiator's recipe.
        triggered_by = "codesearch-gen-chromiumos-initiator",
    )

# buildifier: disable=function-docstring
def update_submodules_mirror(
        name,
        short_name,
        source_repo,
        target_repo,
        extra_submodules = None,
        triggered_by = None,
        refs = None,
        execution_timeout = time.hour):
    properties = {
        "source_repo": source_repo,
        "target_repo": target_repo,
    }
    if extra_submodules:
        properties["extra_submodules"] = extra_submodules
    if refs:
        properties["refs"] = refs
    builder(
        name = name,
        execution_timeout = execution_timeout,
        executable = infra.recipe("update_submodules_mirror"),
        properties = properties,
        caches = [swarming.cache("codesearch_update_submodules_mirror")],
        category = "update-submodules-mirror",
        short_name = short_name,
        triggered_by = triggered_by,
    )

# Runs every four hours (at predictable times).
builder(
    name = "codesearch-gen-chromium-initiator",
    executable = build.recipe("chromium_codesearch_initiator"),
    builder_group_property_name = "builder_group",
    execution_timeout = 5 * time.hour,
    category = "gen|init",
    schedule = "0 */4 * * *",
)

chromium_genfiles("and", "codesearch-gen-chromium-android")
chromium_genfiles("cro", "codesearch-gen-chromium-chromiumos")
chromium_genfiles("fch", "codesearch-gen-chromium-fuchsia")
chromium_genfiles("lcr", "codesearch-gen-chromium-lacros")
chromium_genfiles("lnx", "codesearch-gen-chromium-linux")
chromium_genfiles(
    "win",
    "codesearch-gen-chromium-win",
    os = "Windows-10",
    cpu_cores = "32",
)

update_submodules_mirror(
    name = "codesearch-update-submodules-mirror-src",
    short_name = "src",
    source_repo = "https://chromium.googlesource.com/chromium/src",
    target_repo = "https://chromium.googlesource.com/codesearch/chromium/src",
    extra_submodules = ["src/out=https://chromium.googlesource.com/chromium/src/out"],
    refs = [
        "refs/heads/main",
        "refs/branch-heads/4044",  # M81
        "refs/branch-heads/4103",  # M83
    ],
    triggered_by = luci.gitiles_poller(
        name = "codesearch-src-trigger",
        bucket = "codesearch",
        repo = "https://chromium.googlesource.com/chromium/src",
    ),
    execution_timeout = 2 * time.hour,
)
update_submodules_mirror(
    name = "codesearch-update-submodules-mirror-infra",
    short_name = "infra",
    source_repo = "https://chromium.googlesource.com/infra/infra",
    target_repo = "https://chromium.googlesource.com/codesearch/infra/infra",
    triggered_by = infra.poller(),
)
update_submodules_mirror(
    name = "codesearch-update-submodules-mirror-build",
    short_name = "build",
    source_repo = "https://chromium.googlesource.com/chromium/tools/build",
    target_repo = "https://chromium.googlesource.com/codesearch/chromium/tools/build",
    triggered_by = build.poller(),
)

# Runs every four hours (at predictable times).
builder(
    name = "codesearch-gen-chromiumos-initiator",
    executable = build.recipe("chromiumos_codesearch_initiator"),
    builder_group_property_name = "builder_group",
    execution_timeout = 5 * time.hour,
    category = "gen|init",
    schedule = "0 */4 * * *",
)

# TODO(crbug.com/1284439): Add more boards.
chromiumos_genfiles("codesearch-gen-chromiumos-amd64-generic")
