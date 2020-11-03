# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses
import flask
import json
import typing

from google.cloud import datastore
from google.protobuf import json_format as proto_json_format
from google.protobuf import message as proto_message

import chromeperf.pinpoint.models.job as job_module
import chromeperf.pinpoint.models.task as task_module
from chromeperf.pinpoint import find_isolate_task_payload_pb2
from chromeperf.pinpoint import result_reader_payload_pb2
from chromeperf.pinpoint import test_runner_payload_pb2
from chromeperf.pinpoint.evaluators import isolate_finder
from chromeperf.pinpoint.evaluators import result_reader
from chromeperf.pinpoint.evaluators import test_runner
from chromeperf.pinpoint.models import repository

_PAYLOAD_TYPE_MAP = {
    'find_isolate': find_isolate_task_payload_pb2.FindIsolateTaskPayload,
    'read_value': result_reader_payload_pb2.ResultReaderPayload,
    'run_test': test_runner_payload_pb2.TestRunnerPayload,
}

_TASK_MODULE_MAP = {
    'find_isolate': isolate_finder,
    'read_value': result_reader,
    'run_test': test_runner,
}


class FromDictError(TypeError):
    pass


def _from_dict(client, kind, d):
    try:
        if dataclasses.is_dataclass(kind):
            # Repository disabled construction directly
            if kind is repository.Repository:
                if d.get('name'):
                    return repository.Repository.FromName(client, d['name'])
                elif d.get('url'):
                    return repository.Repository.FromUrl(client, d['url'])

            fieldtypes = {f.name: f.type for f in dataclasses.fields(kind)}
            return kind(
                **{f: _from_dict(client, fieldtypes[f], d[f])
                   for f in d})

        # Type 'list' doesn't tell us how to construct it's member. Also
        # type hint is not a real class we can use.
        if type(kind) is typing._GenericAlias:
            if kind.__origin__ is list:
                return [_from_dict(client, kind.__args__[0], x) for x in d]
            raise NotImplementedError('Only support typing.List')

        if issubclass(kind, proto_message.Message):
            return proto_json_format.Parse(json.dumps(d))
        return d
    except FromDictError:
        raise
    except Exception:
        raise FromDictError(f'Parse {kind} from {repr(d)}')


def _task_entity_to_dict(task_entity: datastore.entity.Entity):
    task_dict = dataclasses.asdict(task_module.Task.FromEntity(task_entity))
    task_dict['key'] = task_dict['key'].id_or_name
    task_dict['created'] = task_dict['created'].isoformat()
    task_dict['dependencies'] = [
        d.id_or_name for d in task_dict['dependencies']
    ]
    try:
        if task_dict['task_type'] in _PAYLOAD_TYPE_MAP:
            payload = task_dict['payload']
            proto_type = _PAYLOAD_TYPE_MAP[task_dict['task_type']]
            result = proto_type()
            if not payload.Unpack(result):
                raise TypeError(
                    f'Mismatched payload type: '
                    f'(expecting: {proto_type.__name__}, '
                    f'got: {payload.type_url})', )
            task_dict['payload'] = proto_json_format.MessageToDict(
                result, including_default_value_fields=True)
    except Exception as e:
        task_dict['payload'] = proto_json_format.MessageToDict(
            task_dict['payload'])
        task_dict['payload_decode_error'] = str(e)
    return task_dict


def create_app(client: datastore.Client = datastore.Client()):
    app = flask.Flask(__name__)

    @app.route("/debug/jobs/<job_id>", methods=['GET', 'POST'])
    def job(job_id):  # pylint: disable=unused-variable
        job_key = client.key('Job', job_id)

        if flask.request.method == 'GET':
            with client.transaction():
                entities = client.query(kind='Task', ancestor=job_key).fetch()
                tasks = [_task_entity_to_dict(t) for t in entities]
            return flask.jsonify(tasks)

        if flask.request.method == 'POST':
            req = flask.request.get_json()
            mod = _TASK_MODULE_MAP[req['type']]
            options = _from_dict(client, mod.TaskOptions, req['options'])
            graph = mod.create_graph(options)
            job = job_module.Job(
                key=job_key,
                user='test-user@example.com',
                url='https://pinpoint.service/job',
            )

            task_module.populate_task_graph(client, job, graph)
            return flask.jsonify({})

    @app.route("/debug/jobs/<job_id>/tasks/<task_id>")
    def task(job_id, task_id):  # pylint: disable=unused-variable
        job_key = client.key('Job', job_id)
        entity = client.get(client.key('Task', task_id, parent=job_key))
        return flask.jsonify(_task_entity_to_dict(entity))

    return app
