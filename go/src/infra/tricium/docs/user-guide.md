## Tricium User Guide

This doc is intended for people who either want to set up a project to use
Tricium, or for those who want to configure where and how Tricium is triggered.

## Note about future migration to luci-change-verifier

The Tricium service will be merged in the future with CQ in a new service called
[Change
Verifier](https://source.chromium.org/chromium/infra/infra/+/master:go/src/go.chromium.org/luci/cv/README.md).
Tricium analysis jobs will be regular tryjobs that happen to output comments;
Tricium project configs will no longer be used, but most of the configuration
and behavior of Tricium analyzers will be determined by the recipe that runs the
analyzer.

## Setting up a project to use Tricium

Tricium supports projects that use Gerrit for code review and LUCI. Tricium
previously supported only public projects because analysis for most jobs was run
on public swarming instances; but now, only recipe-based analyzers are
supported, and so internal projects should theoretically be supported.

### Adding a recipe and builder

When enabling Tricium for a new project, you will need to:

*   Add a recipe; see example: https://crrev.com/c/2477246. This recipe gets a
    checkout, commit message (optional), and affected files and calls
    `api.tricium.run_legacy`.
*   Add a builder; see example https://crrev.com/c/2521848.

### Adding or editing a project config

Project-specific configs are kept in individual project repositories, alongside
other LUCI configs for the project.

For example, they may be in the "infra/config" branch of your project's
repository. To make changes on this branch, you can make a new branch with
origin/infra/config as the upstream, for example by running:

```
git checkout -B branch-name --track origin/infra/config
```

A project config file for the production instance of Tricium must be named
`tricium-prod.cfg`, and generally looks something like this:

```
# Analyzer definition.
functions {
  type: ANALYZER
  name: "Wrapper"
  needs: GIT_FILE_DETAILS
  provides: RESULTS
  owner: "someone@chromium.org"
  monorail_component: "Infra>Platform>Tricium>Analyzer"
  impls {
    runtime_platform: LINUX
    provides_for_platform: LINUX
    # The recipe determines the actual behavior, including what is run.
    recipe {
      project: "my-project"
      bucket: "try"
      builder: "tricium-analysis"
    }
    deadline: 900
  }
}
selections {
  function: "Wrapper"
  platform: LINUX  # Must match platform in definition.
}
repos {
  gerrit_project {
    host: "chromium-review.googlesource.com"
    project: "my/project"
    git_url: "https://chromium.googlesource.com/my-project"
  }
}
service_account: "tricium-prod@appspot.gserviceaccount.com"
```

## Analyzer development

See [contribute.md](./contribute.md) for more details about adding your own
analyzers.

If you are adding a new Analyzer to be used across multiple projects, you should
add it to the
[tricium recipe module](https://source.chromium.org/chromium/infra/infra/+/master:recipes-py/recipe_modules/tricium/api.py).

## Disabling Tricium for a particular CL

To make Tricium skip a particular change, you can add "Tricium: disable" to the
CL description.
