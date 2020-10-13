# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses

from google.cloud import datastore

from chromeperf.engine import task_pb2
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.models import job as job_module


@dataclasses.dataclass
class CompleteResultReaderAction:
    datastore_client: datastore.Client
    job: job_module.Job
    task: task_pb2.Task
    state: str

    def __str__(self):
        return 'CompleteResultReaderAction(job = %s, task = %s)' % (
            self.job.job_id, self.task.id)

    @updates.log_transition_failures
    def __call__(self, _):
        updates.update_task(
            self.datastore_client,
            self.job,
            self.task.id,
            new_state=self.state,
            payload=self.task.payload,
        )