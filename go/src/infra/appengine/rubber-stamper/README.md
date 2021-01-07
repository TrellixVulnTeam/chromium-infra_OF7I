# Rubber-Stamper

[go/rubber-stamper-user-guide](go/rubber-stamper-user-guide)

Rubber-Stamper provides a security-sensitive bot which can provide timely
second-eyes code review by verifying the safety of CLs that it is asked to
review.

See go/chops-rubber-stamp-bot-design for design plans.

## Rubber Stamper User Guide

Currently, our expected CL patterns can be generally divided into two types:
- Changes to benign files (translation, whitespace, test expectation files,
directories that contain no code)
- Clean reverts/relands/cherry-picks (will be implemented in Q1 2021;
[crbug/1092608](crbug/1092608))

If you need to add/change pattern configs, go to the first section "Configure
your patterns"; if a pattern is already configured, and you need to use
Rubber-Stamper as a reviewer for CLs, go to the second section "Have
Rubber-Stamper review your CLs".

### Configure your patterns
1. Add a config for your patterns in [config.cfg](https://chrome-internal.googlesource.com/infradata/config/+/refs/heads/master/configs/rubber-stamper/config.cfg).
Config definitions can be found in [config.proto](https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/appengine/rubber-stamper/config/config.proto).
2. Send a review request to xinyuoffline@ and yulanlin@. Once the config change
is submitted, the config will be pushed to Rubber-Stamper via luci-config.

#### Examples of benign file patterns

When you need Rubber-Stamper to treat your files as benign files, you need to
add the path to the files/directories in the config.

The list of filepaths that follow `.gitignore` style ignore patterns, is configured
on a per-repo basis. Here is a sample `repo_config` that allows all `.xtb` filepaths
in `chromium/src`, and a few subpaths for`.txt` files.

    repo_configs {
      key: "chromium/src"
      value: {
        benign_file_pattern {
          paths: "**.xtb"
          paths: "a/b.txt"
          paths: "a/*/c.txt"
          paths: "z/*"
        }
      }
    }

We use the same rule as in `.gitignore` files to parse these `paths`. Here, each
element of `paths` is treated as a line in a `.gitignore` file. Syntax of
`.gitignore` can be found in [gitignore](https://git-scm.com/docs/gitignore). To
be noticed, in a `.gitignore` file, the last matching pattern decides the
outcome. We also follow this rule here. For example, with the following
`repo_config`, file `test/a/1.txt` would be valid, while file
`test/a/b/2.txt` would not be allowed.

    repo_configs {
      key: "chromium/src"
      value: {
        benign_file_pattern {
          paths: "test/a/**",
          paths: "!test/a/b/*",
        }
      }
    }

### Have Rubber-Stamper review your CLs
For every CL that requires Rubber-Stamper's review, add `Rubber Stamper (rubber-stamper@appspot.gserviceaccount.com)`
as a reviewer. Rubber-Stamper will respond in ~1 min. It will either add a
`Bot-Commit +1` label or leave a comment providing the reason why this CL does
not meet the configured rules and also remove itself as reviewer. The `Bot-Commit +1`
label is not sticky, which means additional patchsets will need to be reviewed.

If a CL does not pass, you can upload a new patchset that addresses the
comments and re-add Rubber-Stamper as a reviewer.

## Found a bug?

Please file a bug in [crbug.com](http://crbug.com) under the Infra>Security
component for support.
