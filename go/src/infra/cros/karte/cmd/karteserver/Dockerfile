# Copyright 2021 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# TODO(gregorynisbet): switch base container to gcr.io/distroless/static:latest
# more info here https://github.com/GoogleContainerTools/distroless
FROM ubuntu:focal
COPY ./karteserver /opt/srv/karteserver
RUN chmod a+x /opt/srv/karteserver
USER nobody
ENTRYPOINT ["/opt/srv/karteserver"]
