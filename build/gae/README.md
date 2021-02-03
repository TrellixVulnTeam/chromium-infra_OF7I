GAE tarballs definitions
------------------------

YAMLs here define how to "assemble" tarballs with files needed to deploy
various GAE apps (one app per tarball).

These YAMLs are consumed by [infra-gae-tarballs-continuous] builder that
continuously builds tarballs and uploads them to GCS. Whenever there's a new
tarball (per its SHA256 hash), this builder prepares a [roll CL] to initiate
the deployment process carried out by [gae-deploy] builder.

[infra-gae-tarballs-continuous]: https://ci.chromium.org/p/infra-internal/builders/prod/infra-gae-tarballs-continuous
[roll CL]: https://chrome-internal.googlesource.com/infradata/gae/+/57abfbcc91409ed04dc3aa0f9d4bbcfc7ae1d2dd
[gae-deploy]: https://ci.chromium.org/p/infradata-gae/builders/ci/gae-deploy


Building tarballs locally
-------------------------

To build a tarball locally without uploading it anywhere (e.g. to test newly
added YAML):

```
# Make the directory with this README.md as cwd.
cd build/gae

# Activate the infra go environment to get cloudbuildhelper in PATH.
eval `../../go/env.py`

# Build the tarball locally (from logdog.yaml definition in this case).
cloudbuildhelper stage luci-go/logdog.yaml -output-tarball tb.tar.gz

# Extract the tarball to manually examine it.
mkdir output && cd output
tar -xvf ../tb.tar.gz

# Cleanup.
cd ..
rm -rf output
rm tb.tar.gz
```

Make sure all YAMLs with GAE configs are in the tarball. Notice their paths,
they'll be needed when preparing [infradata/gae] configs.

[infradata/gae]: https://chrome-internal.googlesource.com/infradata/gae
