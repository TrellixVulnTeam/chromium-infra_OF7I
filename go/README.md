# Chromium Infra Go Area

[TOC]


## Get the code

The steps for getting the code are:

 1. [Install depot_tools](https://www.chromium.org/developers/how-tos/install-depot-tools)
 1. Run `fetch infra`
 1. Run

    ```bash
    eval `infra/go/env.py`
    ```

    On Windows

    ```cmd.exe
    call infra\go\env.cmd
    ```


### Quick Setup

If you are on Linux you can run the [quicksetup script](quicksetup.sh) (instead of the above) like so:

```shell
cd /where/you/want/source/code
wget -O- "https://chromium.googlesource.com/infra/infra/+/master/go/quicksetup.sh?format=TEXT" | base64 -d | bash
```

This will create a self-contained `cr-infra-go-area` directory and populate it
will all necessary tools and source for using or contributing to Chromium's Go
Infrastructure. Once run, look in `cr-infra-go-area/infra/go/src` for the
editable source code.


## Bootstrap

`infra/go` knows how to bootstrap itself from scratch (i.e. from a fresh
checkout) by downloading pinned version of Go toolset, and installing pinned
versions of third party packages and adding a bunch of third party tools (like
`goconvey` and `protoc-gen-go`) to `$PATH`.

The bootstrap (and self-update) procedure is invoked whenever `go/bootstrap.py`
or `go/env.py` run. There's **no** DEPS hook for this. We only want the Go
toolset to be present on systems that need it, since it's somewhat big and
platform-specific.

`go/env.py` can be used in two ways. If invoked without arguments, it verifies
that everything is up-to-date and then just emits a small shell script that
tweaks the environment. This script can be executed in the current shell
process to modify its environment. Once it's done, Go tools can be invoked
directly. This is the recommended way of "entering" `infra/go` build
environment.

For example:

```shell
cd infra/go
eval `./env.py`
cd src/infra
go install ./cmd/bqupload
../../bin/bqupload --help  # infra/go/bin is where executables are installed
bqupload --help            # infra/go/bin is also in $PATH
```

Alternatively `go/env.py` can be used as a wrapping command that sets up an
environment and invokes some other process. It is particularly useful on
Windows.

If the `INFRA_PROMPT_TAG` environment variable is exported while running
`go/env.py`, the new environment will include a modified `PS1` prompt containing
the `INFRA_PROMPT_TAG` value to indicate that the modified environment is being
used. By default, this value is "[cr go] ", but it can be changed by exporting
a different value or disabled by exporting an empty value.


## Dependency management

Infra Go code uses a mix of [gclient DEPS] and [Go Modules] to manage
dependencies.

There are two kinds of dependencies: dependencies between first-party code
(like `infra.git` code depending on `luci-go.git` code) and dependencies between
first-party and third-party code (like `infra.git` code depending on Google
Cloud libraries).

Dependencies between first-party code are managed in the [DEPS] file, which is
usually updated automatically by autoroller bots (e.g. [luci-go => infra]
autoroller). This ensures changes to first-party code propagate quickly
through the dependency tree.

Dependencies on third-party code are managed in `go.mod` and `go.sum` files.
Each Chrome infra repository has them (for `infra.git` they are
[go/src/infra/go.mod] and [go/src/infra/go.sum]), which turns these repositories
into Go modules. Thus an infra gclient checkout is a checkout of a bunch of
Go modules on disk side-by-side (according to the [DEPS] file). `go build` is
made aware of this structure via `replace` directives in the `go.mod` file,
which tells it to use locally checked out modules (at revisions precisely
controlled by the [DEPS]), instead of trying to figure out their intended
versions using some version selection algorithm. This structure also allows to
make changes to one locally checked out module (like `go.chromium.org/luci` from
`luci-go.git`), and verifying they don't break another module (like `infra` from
`infra.git`, which depends on `go.chromium.org/luci`).

Note that when building a specific Go target using Infra builders, only `go.mod`
of its enclosing module is considered when fetching dependencies. For example,
if you setup a CIPD package builder (or a Docker image builder, or GAE tarball
builder) that builds `go.chromium.org/luci/some/package/...`, the builder will use
dependencies specified in the `go.mod` in `luci-go.git` repository (using the
luci-go's revision currently checked out on disk according to the [DEPS] in
`infra.git`). If the same builder then tries to build e.g. `infra/something`, it
will use the `go.mod` from the `infra.git` repository. And it even may end up
using a different version of some third party dependency.

[gclient DEPS]: https://www.chromium.org/developers/how-tos/depottools#TOC-DEPS-file
[Go Modules]: https://golang.org/ref/mod
[DEPS]: ../DEPS
[luci-go => infra]: https://autoroll.skia.org/r/luci-go-infra-autoroll
[go/src/infra/go.mod]: ./src/infra/go.mod
[go/src/infra/go.sum]: ./src/infra/go.sum


### Adding or updating a dependency

Navigate to the directory with `go.mod` file (`go/src/infra`) and use regular
Go modules commands to update `go.mod` and `go.sum` files. See
[Managing dependencies](https://golang.org/doc/modules/managing-dependencies).


### Making a DEPS roll that picks up go.mod changes

If a DEPS roll picks up a change to a `go.mod` file in a first party dependency,
infra's `go.sum` (and sometimes `go.mod`) need to be modified too to pick up
updated version pins and module checksums.

Ensure your local checkout matches `DEPS` and run `go mod tidy` in the infra
module directory:

```shell
cd go/src/infra
go mod tidy
```

Add the resulting `go.mod` and `go.sum` changes to the CL.
