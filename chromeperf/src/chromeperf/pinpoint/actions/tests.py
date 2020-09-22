# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses
import typing
import logging

from google.cloud import datastore
from google.protobuf import json_format
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.models import task as task_module
from chromeperf.pinpoint import test_runner_payload_pb2
from chromeperf.services import swarming


@dataclasses.dataclass
class MarkTestFailedAction(
        task_module.PayloadUnpackingMixin,
        updates.ErrorAppendingMixin,
):
    job: typing.Any
    task: task_module.Task
    datastore_client: datastore.Client
    reason: str
    message: str

    def __str__(self):
        return f'MarkTestFailedAction(task = {self.task.id})'

    @updates.log_transition_failures
    def __call__(self, _):
        task_payload = self.unpack(
            test_runner_payload_pb2.TestRunnerPayload,
            self.task.payload,
        )
        return self.update_task_with_error(
            datastore_client=self.datastore_client,
            job=self.job,
            task=self.task,
            payload=task_payload,
            reason=self.reason,
            message=self.message,
        )


@dataclasses.dataclass
class ScheduleTestAction(task_module.PayloadUnpackingMixin,
                         updates.ErrorAppendingMixin):
    job: typing.Any
    task: task_module.Task
    datastore_client: datastore.Client

    def __str__(self):
        return 'ScheduleTestAction(job = %s, task = %s)' % (self.job.job_id,
                                                            self.task.id)

    @updates.log_transition_failures
    def __call__(self, _):
        logging.debug('Scheduling a Swarming task to run a test.')
        task_payload = self.unpack(test_runner_payload_pb2.TestRunnerPayload,
                                   self.task.payload)
        # TODO: Figure out whether we can just turn the proto to json directly.
        swarming_request_body = json_format.MessageToDict(
            task_payload.output.swarming_request)

        # Ensure that this thread/process/handler is the first to mark this task
        # 'ongoing'. Only proceed in scheduling a Swarming request if we're the
        # first one to do so.
        updates.update_task(self.datastore_client,
                            self.job,
                            self.task.id,
                            new_state='ongoing',
                            payload=self.task.payload)

        # At this point we know we were successful in transitioning to
        # 'ongoing'.
        try:
            response = swarming.Swarming(
                task_payload.input.swarming_server).Tasks().New(
                    swarming_request_body)
            task_payload.output.swarming_response.task_id = response.get(
                'task_id')
            task_payload.output.tries += 1
        except request.RequestError as e:
            return self.update_task_with_error(
                datastore_client=self.datastore_client,
                job=self.job,
                task=self.task,
                payload=task_payload,
                reason=type(e).__name__,
                message=f'Encountered failure in swarming request: {e}',
            )

        # Update the payload with the task id from the Swarming request. Note
        # that this could also fail to commit.
        self.task.payload.Pack(task_payload)
        return updates.update_task(
            self.datastore_client,
            self.job,
            self.task.id,
            payload=self.task.payload,
        )


@dataclasses.dataclass
class PollSwarmingTaskAction(task_module.PayloadUnpackingMixin,
                             updates.ErrorAppendingMixin):
    job: typing.Any
    task: task_module.Task
    datastore_client: datastore.Client

    def __str__(self):
        return 'PollSwarmingTaskAction(job = %s, task = %s)' % (
            self.job.job_id, self.task.id)

    @updates.log_transition_failures
    def __call__(self, _):
        task_payload = self.unpack(
            test_runner_payload_pb2.TestRunnerPayload,
            self.task.payload,
        )
        swarming_task = swarming.Swarming(
            task_payload.input.swarming_server).Task(
                task_payload.output.swarming_response.task_id)
        result = swarming_task.Result()
        required_keys = {'bot_id', 'state', 'failure'}
        missing_required_keys = required_keys - set(result.keys())
        if missing_required_keys:
            return self.update_task_with_error(
                datastore_client=self.datastore_client,
                job=self.job,
                task=self.task,
                payload=task_payload,
                reason='MissingRequiredKeys',
                message=(f'Missing required keys from Swarming response = '
                         f'{missing_required_keys}.'),
            )

        task_payload.output.swarming_response.bot_id = result.get('bot_id')
        task_payload.output.swarming_response.state = result.get('state')
        task_payload.output.swarming_response.failure = result.get(
            'failure', False)
        swarming_task_state = result.get('state')
        if swarming_task_state in {'PENDING', 'RUNNING'}:
            # Commit the task payload still.
            self.task.payload.Pack(task_payload)
            return task_module.update_task(
                datastore_client=self.datastore_client,
                job=self.job,
                task_id=self.task.id,
                payload=self.task.payload)

        if swarming_task_state == 'EXPIRED':
            # TODO(dberris): Do a retry, reset the payload and run an
            # "initiate"?
            return self.update_task_with_error(
                datastore_client=self.datastore_client,
                job=self.job,
                task=self.task,
                payload=task_payload,
                reason='SwarmingExpired',
                message=
                f'Task {task_payload.output.swarming_response.task_id} expired'
            )

        if swarming_task_state != 'COMPLETED':
            return self.update_task_with_error(
                datastore_client=self.datastore_client,
                job=self.job,
                task=self.task,
                payload=task_payload,
                reason='SwarmingFailed',
                message=f'Task state is not "COMPLETED"; response={result}',
            )

        task_payload.output.task_output.isolate_server = result.get(
            'outputs_ref', {}).get('isolatedserver')
        task_payload.output.task_output.isolate_hash = result.get(
            'outputs_ref', {}).get('isolated')

        if result.get('failure', False):
            return self.update_task_with_error(
                datastore_client=self.datastore_client,
                job=self.job,
                task=self.task,
                payload=task_payload,
                reason='RunTestFailed',
                message=
                (f'Running the test failed, see isolate output: '
                 f'https://{task_payload.output.task_output.isolate_server}/'
                 f'browse?digest={task_payload.output.task_output.isolate_hash}'
                 ),
            )

        self.task.payload.Pack(task_payload)
        return updates.update_task(
            datastore_client=self.datastore_client,
            job=self.job,
            task_id=self.task.id,
            new_state='completed',
            payload=self.task.payload,
        )
