# Adding a new wheel for vpython

For this example, we'll be adding 'scandir' at version 1.7.

1. Go to pypi and find the wheel at the appropriate version in question.
   1. Project: https://pypi.org/project/scandir
   1. Versions: https://pypi.org/project/scandir/#history
   1. Version 1.7: https://pypi.org/project/scandir/1.7/
   1. Files for 1.7: https://pypi.org/project/scandir/1.7/#files

1. Determine what type of wheel it is (in order of preference):
   1. Universal
      1. Pure-python libraries already packaged as wheels.
      1. These will have a `*-py2.py3-none-any.whl` file (may be just py2)
      1. Example: https://pypi.org/project/requests/#files
   1. UniversalSource
      1. Pure-python libraries distributed as a tarball.
      1. These will have a `*.tar.gz` file. You'll have to fetch this tarball
         and look to see if it contains any .c or .cc files. If it does, then
         this is either `Prebuilt` or `SourceOrPrebuilt`.
      1. Example: https://pypi.org/project/httplib2/#files
   1. Prebuilt
      1. Python libs with c extensions, pre-built for platforms we care about.
      1. These will have many .whl files for various platforms. Look at the list
         to see if it covers all the platforms your users care about. If not
         then you may have to use `SourceOrPrebuilt.`
      1. Example: https://pypi.org/project/pillow/#files
   1. SourceOrPrebuilt
      1. Python libs with c extensions, pre-built for some platforms we care
         about. These don't require extra C libraries though, just typical
         system/python C libraries.
      1. They will include `*.tar.gz` with the library source, but may also
         contain `.whl` files for some platforms.
      1. Example: https://pypi.org/project/scandir/#files
      1. Example (no .whl): https://pypi.org/project/wrapt/#files
   1. "Special" wheels
      1. These deviate from the wheels above in some way, usually by requiring
         additional C libraries.
      1. We always prepare our wheels and their C extensions to be as static as
         possible. Generally this means building the additional C libraries as
         static ('.a') files, and adjusting the python setup.py to find this.
      1. See the various implementations referenced by wheels.py to get a feel
         for these.
      1. These are (fortunately) pretty rare (but they do come up occasionally).
   1. The "infra_libs" wheel
      1. This one is REALLY special, but essentially packages the
         [packages/infra_libs](/packages/infra_libs) wheel. Check
         wheel_infra.py.


Once you've identified the wheel type, open [wheels.py](./wheels.py) and find
the relevant section. Each section is ordered by wheel name and then by symver.
If you put the wheel definition in the wrong place, dockerbuild will tell you :)

So for `scandir`, we see that there are prebuilts for windows, but for
everything else we have to build it ourself.

The wheels are built for linux platforms using Docker (hence "dockerbuild").
Unfortunately this tool ONLY supports building for linux this way. For building
mac and windows, this can use the ambient toolchain (i.e. have XCode or MSVS
installed on your system).

*** note
I actually haven't ever run this on windows. Usually python wheels with
C extensions that chromium may actually need have pre-built windows wheels.

That said, this is essentially just doing `setup.py bdist_wheel` to generate the
wheel contents, so if that process works with MSVS, it SHOULD work.
***

The upshot of this is that if you need to build for e.g. mac or windows, you
need to run this from one of those platforms with an appropriate SDK installed.

Back to our example, we'll be adding a new entry to the SourceOrPrebuilt
section:

    SourceOrPrebuilt('scandir', '1.9.0',
        packaged=[
          'windows-x86',
          'windows-x64',
        ],
    ),

This says the wheel `scandir-1.9.0` is either built from source (.tar.gz) or is
prebuilt (for the following `packaged` platforms).

*** note
When adding a new version of an existing wheel, please only ADD it
(don't replace an existing version). This is because existing .vpython specs
will likely still reference the old version, and it's good to keep wheels.md
as a full registry of available versions.
***

And update the wheel.md documentation:

    path/to/infra.git/run.py         \
       infra.tools.dockerbuild       \
       wheel-dump

Now, test that your wheel builds successfully using the following:

    path/to/infra.git/run.py         \
       infra.tools.dockerbuild       \
       --logs-debug                  \
       wheel-build                   \
       --wheel 'scandir-1.9.0'       \

Notable options (check `--help` for details):
  * `--wheel_re` - Use in place of `--wheel` to run for multiple wheels or
    versions.
  * `--platform` - Specify a specific platform to build for.

Then you upload your CL and commit as usual.

Once your CL is committed, the wheels will be automatically built and uploaded
by the following builders:

* [Linux](https://ci.chromium.org/p/infra-internal/builders/prod/Linux%20wheel%20builder)
* [Mac](https://ci.chromium.org/p/infra-internal/builders/prod/Mac%20wheel%20builder)
* [Windows](https://ci.chromium.org/p/infra-internal/builders/prod/Windows%20wheel%20builder)

## Custom patches

While we strongly prefer to not patch anything, sometimes we need a backport
or local fix for our system.

Here's the quick overview:

* Patches are only supported with `UniversalSource` since we need to unpack
  the source & patch it directly before building the wheel.
* All patches live under `patches/`.
* All patches must be in the `-p1` format.
* The filenames must start with the respective package name & version and end
  in `.patch`.  e.g. `UniversalSource('scandir', '1.9.0')` will have a prefix
  of `scandir-1.9.0-` and a suffix of `.patch`.
* Add the shortnames into the `patches=(...)` tuple to `UniversalSource`.
* All patches should be well documented in the file header itself.

A short example:

```python
  UniversalSource('scandir', '1.9.0', patches=(
      'some-fix',
      'another-change',
  )),
```

This will apply the two patches:
* `patches/scandir-1.9.0-some-fix.patch`
* `patches/scandir-1.9.0-another-change.patch`
