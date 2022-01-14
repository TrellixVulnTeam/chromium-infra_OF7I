# nsjail-wrapper

self-link: [go/nsjail-wrapper]

## Overview

This is a minimal wrapper around [nsjail] for the purposes of task isolation as
part of the [verified builds] project. It is intended to be deployed in the same
directory as `nsjail` with `setuid` root. This binary is narrowly defined to
exclusively work within the context of `bbagent` running in a swarming task.

This will be implemented such that the launched process will always have less
permissions than the calling process.

The wrapper will fulfill a few functions

-   Store the config (and potentially options) to pass to `nsjail`
-   Fulfill the [luciexe] contract

### Store the config & optionally options for isolation

This will include things like:

-   namespacing
-   seccomp-bpf filter

### Fulfilling the `luciexe` contract

This includes things like:

-   ensure that `stdin` is undisturbed
-   forwarding `SIGTERM`
-   ensuring that the file pointed to by `$LUCI_CONTEXT` is available
-   ensuring that the logdog domain socket file/envvar is available

[go/nsjail-wrapper]: go/nsjail-wrapper
[luciexe]: go/luciexe
[nsjail]: https://github.com/google/nsjail
[verified builds]: go/chops-verified-builds
