# Copyright 2021 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# Use an empty base image
# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static:latest

# TODO(gregorynisbet): perform the build as part of the Dockerfile.
# The binary is built on the host. Copy it into the image.
COPY ./karteserver ./karteserver

USER nobody
ENTRYPOINT ["./karteserver"]
