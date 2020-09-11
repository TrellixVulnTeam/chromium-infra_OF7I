# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Event model for Task evaluation."""
import dataclasses
import typing
import uuid

from google.protobuf import any_pb2
from google.protobuf import empty_pb2
from google.protobuf import message

from chromeperf.engine import event_pb2


@dataclasses.dataclass
class Event:
    id: str
    type: str
    target_task: str
    payload: any_pb2.Any


def build_event(payload: message.Message,
                type: str,
                target_task: typing.Union[str, None] = None) -> Event:
    encoded_payload = any_pb2.Any()
    encoded_payload.Pack(payload)
    return Event(id=str(uuid.uuid4()),
                 type=type,
                 target_task=target_task or '',
                 payload=encoded_payload)


def select_event(target_task=None):
    encoded_payload = any_pb2.Any()
    encoded_payload.Pack(empty_pb2.Empty())
    return Event(id=str(uuid.uuid4()),
                 type='select',
                 target_task=target_task,
                 payload={})
