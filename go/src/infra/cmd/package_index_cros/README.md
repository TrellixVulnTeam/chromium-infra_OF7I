# Package index pregenerator

A script to generate build artifacts for Chrome OS packages.

WARNING: this script is only manually tested for very limited purpuses and
         by very limited engineer. The engineer tried to list possible
         pitfalls here but certainly missed something. Use at your own
         risk.

## Overview

The utltimate goal for the script is to generate input for
[package_index](https://source.chromium.org/chromium/infra/infra/+/main:go/src/infra/cmd/package_index)
which makes references available for Chrome on Codesearch page. The script
should help to reuse `package_index` for ChromeOS purpuses.

Currently, it is able to generate, fix and merge:

- compile_commands.json
- gn_targets.json
- build dir (generated source files, ninja artifacts etc)

## Usage

1. Clone `package_index_cros` somewhere in Chrome OS checkout. E.g
  `chromeos/package_index_cros`, `chromeos/tmp/package_index_cros`. The
  script needs `chromite` in one of parent directories.
1. Make `package_index_cros/main.py` executable.
1. Run `./package_index_cros/main.py --help` for details.

TODO: Instead of the last step add actual steps on how to get kzip.

## Local usage

Side effect of the `compile_commands.json` generator is that it can be used
locally with clangd.

1. Install `package_index_cros` with steps above.
1. Install [clangd](go/clangd).
1. Generate `compile_commands.json`:

```bash
./package_index_cros/main.py \
  --with-build
  --compile-commands \
  /path/to/clangd/compile/commands/dir/compile_commands.json \
  package1 package2 package3
```

Where:

* `--with-build` will build packages with `FEATURES=noclean` to keep build
  artifacts. If not much changed, build artifacts are still there, and you
  want to regenerate compile commands, you may skip the flag.

  NOTE: The scripts build workon (`*.ebuild-9999`) version of the packages.
        It calls `cros_workon start` and `cros_workon stop` for not yet
        workon packages. If something goes wrong inbetween calls, you might
        end up with `cros_workon list` being different than you expected.

* `--compile-commands /some/path/compile_commands.json` will store
  tells the script to generate compile commands and store it in given file.
* packages: which packages you want to include. Compile commands will
  include these packages plus their dependencies of unlimited depth.
  You can skip packages, then compile commands will be generated for all
  available and supported packages (may take a while because of build).
* Optional `--build-dir ${some_dir}` will create a dir and merge all
  packages' build dirs and generated files there. Can be useful when
  working with proto files but fully optional.

  NOTE: The existing `${some_dir}` is removed each script run. Be careful
        with that axe.

1. Bosh. Done. Now you can use clangd and click references in your
   favourite IDE.

  NOTE: The script does not clean up after itself. You might want to use
        `cros_sdk clean`.
