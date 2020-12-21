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

The list of "benign file extensions" and corresponding filepaths is configured
on a per-repo basis. Here is a sample `repo_config` that allows all `.xtb` filepaths
in `chromium/src`, and a few subpaths for`.txt` files.

    repo_configs {
      key: "chromium/src"
      value: {
        benign_file_pattern {
          file_extension_map {
            key: ".xtb"
            value: {
              paths: "**"
            }
          }
          file_extension_map {
            key: ".txt"
            value: {
              paths: "a/b.txt",
              paths: "a/*/c.txt",
              paths: "d/"
            }
          }
          file_extension_map {
            key: "*"
            value: {
              paths: "z/"
            }
          }
        }
      }
    }

Using `**` in `paths` allows all paths in that repo. Using `*` in the key of
`file_extension_map` allows all the file extensions.

`file_extension_map` is a map contains the information about which files are
considered benign under which directories. The `key`s are file extensions,
while the `value`s are paths of files or directories. For paths to specific
files, these files can be considered benign files; for paths to directories,
files under these directories with corresponding extensions can be considered
as benign files. For files with no extensions, their key should be an empty
string "".

We use the Match function from the `path` package to judge whether a file belongs to
a path. Therefore,`paths` here should follow the same syntax as the `pattern`
variable in the [Match](https://golang.org/pkg/path/#Match) function.

If you need to add a file whose suffix already exists in the pattern, you can
simply add another `paths` under the existing key. For example, adding another
"a/c.txt" in the above config would be like:

    benign_file_pattern {
      file_extension_map {
        key: ".txt"
        value: {
          paths: "a/b.txt",
          paths: "a/*/c.txt",
          paths: "d/",
          paths: "a/c.txt"
        }
      }
    }

If you need to add a file whose suffix does not exist yet, you need to add a
new `file_extension_map`, like:

    benign_file_pattern {
      file_extension_map {
        key: ".txt"
        value: {
          paths: "a/b.txt",
          paths: "a/*/c.txt",
          paths: "d/"
        }
      }
      file_extension_map {
        key: ""
        value: {
          paths: "a/DEPS"
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
