# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses
import typing

from google.protobuf import message
from google.protobuf import any_pb2
from google.cloud import datastore

from chromeperf.pinpoint.models import change as change_module
from chromeperf.engine import evaluator
from chromeperf.engine import event as event_module
from chromeperf.pinpoint.models import task as task_module
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint import find_isolate_task_payload_pb2


@dataclasses.dataclass
class UpdateWrapper:
    datastore_client: datastore.Client
    job: typing.Any
    task: task_module.Task
    new_state: str
    payload: message.Message

    def __call__(self, _):
        encoded_payload = any_pb2.Any()
        encoded_payload.Pack(self.payload)
        return updates.update_task(
            self.datastore_client,
            self.job,
            self.task.id,
            new_state=self.new_state,
            payload=encoded_payload,
        )


def FakeNotFoundIsolate(datastore_client, job, task, *_):
    if task.state == 'completed':
        return None

    return [
        UpdateWrapper(datastore_client, job, task, 'completed', task.payload)
    ]


@dataclasses.dataclass
class FakeFoundIsolate(task_module.PayloadUnpackingMixin):
    datastore_client: datastore.Client
    job: typing.Any

    def __call__(self, task, *_):
        if task.task_type != 'find_isolate':
            return None

        if task.state == 'completed':
            return None

        task_payload = self.unpack(
            find_isolate_task_payload_pb2.FindIsolateTaskPayload, task.payload)
        task_payload.buildbucket_build.id = '345982437987234'
        task_payload.buildbucket_build.url = 'https://builbucket/url'
        task_payload.buildbucket_build.status = 'COMPLETED'
        task_payload.buildbucket_build.result = 'SUCCESS'
        task_payload.buildbucket_build.result_details_json = '{}'
        task_payload.isolate_server = 'some-isolate-server'
        task_payload.isolate_hash = '14aaaaaaaaaaa514'
        return [
            UpdateWrapper(
                self.datastore_client,
                self.job,
                task,
                'completed',
                task_payload,
            )
        ]


@dataclasses.dataclass
class FakeFindIsolateFailed(task_module.PayloadUnpackingMixin):
    datastore_client: datastore.Client
    job: typing.Any

    def __call__(self, task, *_):
        if task.task_type != 'find_isolate':
            return None

        if task.state == 'failed':
            return None

        task_payload = self.unpack(
            find_isolate_task_payload_pb2.FindIsolateTaskPayload, task.payload)
        task_payload.buildbucket_build.status = 'COMPLETED'
        task_payload.buildbucket_build.result = 'FAILURE'
        task_payload.buildbucket_build.result_details_json = '{}'
        task_payload.tries = 1
        task.payload.Pack(task_payload)
        return [
            UpdateWrapper(
                self.datastore_client,
                self.job,
                task,
                'failed',
                task.payload,
            )
        ]
