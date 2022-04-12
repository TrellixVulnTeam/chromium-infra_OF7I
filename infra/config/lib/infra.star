# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Functions and constants related to infra.git used by all modules."""

def poller():
    """Defines a gitiles poller polling infra.git repo."""
    return luci.gitiles_poller(
        name = "infra-gitiles-trigger",
        bucket = "ci",
        repo = infra.REPO_URL,
        refs = ["refs/heads/main"],
    )

def recipe(name, use_python3 = False):
    """Defines a recipe hosted in the infra.git recipe bundle.

    Args:
      name: name of the recipe.
      use_python3: Run the recipe via py3.

    Returns:
      A luci.recipe(...) object.
    """
    return luci.recipe(
        name = name,
        recipe = name,
        cipd_package = "infra/recipe_bundles/chromium.googlesource.com/infra/infra",
        use_python3 = use_python3,
    )

def console_view(name, title, repo = None):
    """Defines a console view with infra header."""
    luci.console_view(
        name = name,
        title = title,
        repo = repo or infra.REPO_URL,
        header = "//data/infra_console_header.textpb",
        refs = ["refs/heads/main"],
    )

def cq_group(name, repo = None, tree_status_host = None):
    """Defines a CQ group watching refs/heads/main.

    Args:
      name: The human- and machine-readable name of the CQ group.
      repo: https URL of the git repo for this CQ group to monitor.
      tree_status_host: Hostname of the tree_status_host for this CQ group.
    """
    luci.cq_group(
        name = name,
        watch = cq.refset(
            repo = repo or infra.REPO_URL,
            refs = ["refs/heads/main"],
        ),
        tree_status_host = tree_status_host,
        retry_config = cq.RETRY_NONE,
    )

def builder(
        *,

        # Basic required stuff.
        name,
        bucket,
        executable,

        # Dimensions.
        os,
        cpu = None,
        pool = None,

        # Swarming environ.
        service_account = None,

        # Misc tweaks.
        properties = None,
        schedule = None,
        extra_dimensions = None,

        # Triggering relations.
        triggered_by = None,

        # LUCI-Notify config.
        notifies = None):
    """Defines a basic infra builder (CI or Try).

    It is a builder that needs an infra.git checkout to do stuff.

    Depending on value of `bucket`, will chose a default pool (ci or flex try),
    the service account and build_numbers settings.

    Args:
      name: name of the builder.
      bucket: name of the bucket to put it in.
      executable: a recipe or other luci.executable(...) to run.
      os: a target OS dimension.
      cpu: a target CPU dimension.
      pool: target Swarming pool.
      service_account: a service account to run the build as.
      properties: a dict with properties to pass to the builder.
      schedule: a string with builder schedule for cron-like builders.
      extra_dimensions: a dict with additional Swarming dimensions.
      triggered_by: builders that trigger this one.
      notifies: what luci.notifier(...) to notify when its done.
    """
    if bucket == "ci":
        pool = pool or "luci.flex.ci"
        service_account = service_account or infra.SERVICE_ACCOUNT_CI
        build_numbers = True
    elif bucket == "try":
        pool = pool or "luci.flex.try"
        service_account = service_account or infra.SERVICE_ACCOUNT_TRY
        build_numbers = None  # leave it unset in the generated file
    else:
        fail("unknown bucket")

    caches = []
    if os.startswith("Mac"):
        caches.append(infra.cache_osx_sdk)

    dimensions = {"os": os, "cpu": cpu or "x86-64", "pool": pool}
    if extra_dimensions:
        if ("cpu" in extra_dimensions or "os" in extra_dimensions or
            "pool" in extra_dimensions):
            fail("specify 'cpu', 'os', or 'pool' directly")
        dimensions.update(extra_dimensions)

    luci.builder(
        name = name,
        bucket = bucket,
        executable = executable,
        dimensions = dimensions,
        service_account = service_account,
        properties = properties,
        caches = caches,
        build_numbers = build_numbers,
        schedule = schedule,
        task_template_canary_percentage = 30,
        triggered_by = triggered_by,
        notifies = notifies,
        resultdb_settings = resultdb.settings(
            enable = True,
            history_options = resultdb.history_options(
                by_timestamp = True,
            ),
        ),
        experiments = {"luci.buildbucket.agent.cipd_installation": 50},
    )

def _tree_closing_notifiers():
    return [
        luci.tree_closer(
            name = "infra tree closer",
            tree_status_host = "infra-status.appspot.com",
            template = "status",
        ),
    ]

_OS_TO_CATEGORY = {
    "Ubuntu": "linux",
    "Mac": "mac",
    "Windows": "win",
}

def category_from_os(os, short = False):
    """Given e.g. 'Ubuntu-20.10' returns e.g. 'linux|20.10'.

    Args:
      os: OS dimension name.
      short: if True, strip the version.

    Returns:
      A console category name.
    """
    os, _, ver = os.partition("-")
    category = _OS_TO_CATEGORY.get(os, os.lower())
    if not short:
        category += "|" + ver
    return category

infra = struct(
    REPO_URL = "https://chromium.googlesource.com/infra/infra",

    # Note: try account is also used by all presubmit builders in this project.
    SERVICE_ACCOUNT_TRY = "infra-try-builder@chops-service-accounts.iam.gserviceaccount.com",
    SERVICE_ACCOUNT_CI = "infra-ci-builder@chops-service-accounts.iam.gserviceaccount.com",
    cache_osx_sdk = swarming.cache("osx_sdk"),
    poller = poller,
    recipe = recipe,
    console_view = console_view,
    cq_group = cq_group,
    builder = builder,
    tree_closing_notifiers = _tree_closing_notifiers,
    category_from_os = category_from_os,
)
