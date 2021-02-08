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
- Clean reverts/cherry-picks

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
          paths: "*.xtb"
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

Note: when a revert/cherry-pick cannot pass the `clean_revert_pattern`/
`clean_cherry_pick_pattern` check, but can pass the `benign_file_pattern`
check, we will still approve the CL because `benign_file_pattern` is used
as a fallback in our design.

#### Examples of clean revert patterns

A revert will be approved if the Gerrit API marks it as a [pure revert](https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#get-pure-revert),
it is within the configured `time_window` in the config, and none of the
modified files is in the `excluded_paths`.

A `time_window` represents a length of time in `<int><unit>` form. Reverts need
to be within this `time_window` to be valid. Valid units are `s`, `m`, `h`,
`d`, meaning "seconds", "minutes", "hours", "days" respectively. By default,
Rubber-Stamper has a global `default_time_window` of `7d`, which allows reverts
that are reverted no later than 7 days. You can override this value at a
host-level (by modifying `clean_revert_time_window` in `host_configs`) or
repo-level (by modifying `time_window` in `clean_revert_pattern` of
`repo_configs`). It is always the more granular level that is applied. For
example, in the following config, the global `7d`, host-level `1h` will all be
overriden by repo-level `5m`.

    default_time_window: "7d"

    host_configs {
      key: "chromium"
      value: {
        clean_revert_time_window: "1h"
        repo_configs {
          key: "infra/experimental"
          value: {
            clean_revert_pattern {
              time_window: "5m"
            }
          }
        }
      }
    }

`excluded_paths` defines files/directories that always require a human to
review. If a revert modifies any file that is contained in `excluded_paths`,
the revert will not be approved by Rubber-Stamper. The syntax of
`excluded_paths` is the same as that of `.gitignore` files. Gitignore file
syntax is documented in [gitignore](https://git-scm.com/docs/gitignore). In the
following sample config, any revert that changes `a/b.md` or any files ending
in `.go` will not be approved.

    clean_revert_pattern {
      excluded_paths: "a/b.md"
      excluded_paths: "*.go"
    }

#### Examples of clean cherry-pick patterns
A cherry-pick will be approved if:
1. Only one revision uploaded;
2. It is within the configured `time_window` in the config;
3. It is cherry-picked after the original CL has been merged;
4. None of the modified files is in the `excluded_paths`;
5. It is marked as mergeable by the Gerrit API
[GetMergeable](https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#get-mergeable).

The format of `time_window` is the same as that for clean revert patterns,
which you can find in the section above. Similarly, a host-level value
(`clean_cherry_pick_time_window` in `host_configs`) can override the global
value (`default_time_window`), while a repo-level value (`time_window` in
`clean_cherry_pick_pattern` of `repo_configs`) can override the host-level
value.

The format of `excluded_paths` is also the same as that for clean revert
patterns.

Here is an example of a clean cherry-pick pattern. Any cherry-pick that is 
cherry-picked more than 3 hours ago or modifies any files ending in `.md`
will not be approved.

    clean_cherry_pick_pattern {
      time_window: "3h"
      excluded_paths: "*.md"
    }

### Have Rubber-Stamper review your CLs
For every CL that requires Rubber-Stamper's review, add `Rubber Stamper (rubber-stamper@appspot.gserviceaccount.com)`
as a reviewer. Rubber-Stamper will respond in ~1 min. It will either add a
`Bot-Commit +1` label or leave a comment providing the reason why this CL does
not meet the configured rules and also remove itself as reviewer. The `Bot-Commit +1`
label is not sticky, which means additional patchsets will need to be reviewed.

If you set the Auto-Submit label, Rubber Stamper will also set `Commit-Queue +2`
if it approves the CL.

If a CL does not pass, you can upload a new patchset that addresses the
comments and re-add Rubber-Stamper as a reviewer.

## Found a bug?

Please file a bug in [crbug.com](http://crbug.com) under the Infra>Security
component for support.
