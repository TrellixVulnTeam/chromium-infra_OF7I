This directory is similar to the `/cmd` directory, but for executables that
implement the luciexe protocol. These executables can run locally, and can also
be run on hosts that support luciexe executables, e.g. Buildbucket.

See related docs:
- [luciexe Go package](https://pkg.go.dev/go.chromium.org/luci/luciexe): Overview of the luciexe
protocol.
- [Running luciexe on Buildbucket](https://pkg.go.dev/go.chromium.org/luci/luciexe#hdr-LUCI_Executables_on_Buildbucket): How to run luciexe on Buildbucket.
**Note that buildbucket expects the executable to be named "luciexe"**.
- [luciexe/build Go package](https://pkg.go.dev/go.chromium.org/luci/luciexe/build): Suggested
framework to implement luciexe binaries.