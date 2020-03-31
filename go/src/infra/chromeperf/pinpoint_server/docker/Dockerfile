# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

FROM gcr.io/distroless/static:latest

COPY bin/pinpoint_server ./pinpoint_server

USER nobody

ENTRYPOINT ["./pinpoint_server"]
