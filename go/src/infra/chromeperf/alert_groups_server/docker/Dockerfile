# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

FROM gcr.io/distroless/static:latest

COPY bin/alert_groups_server ./alert_groups_server

USER nobody

ENTRYPOINT ["./alert_groups_server"]
