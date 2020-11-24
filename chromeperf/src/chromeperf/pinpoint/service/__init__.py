# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses
import flask
import json
import typing

from google.cloud import datastore
from google.protobuf import empty_pb2
from google.protobuf import json_format as proto_json_format
from google.protobuf import message as proto_message

from chromeperf.engine import combinators
from chromeperf.engine import predicates
from chromeperf.pinpoint import find_culprit_task_payload_pb2
from chromeperf.pinpoint import find_isolate_task_payload_pb2
from chromeperf.pinpoint import result_reader_payload_pb2
from chromeperf.pinpoint import test_runner_payload_pb2
from chromeperf.pinpoint.evaluators import culprit_finder
from chromeperf.pinpoint.evaluators import isolate_finder
from chromeperf.pinpoint.evaluators import result_reader
from chromeperf.pinpoint.evaluators import test_runner
from chromeperf.pinpoint.models import repository
import chromeperf.engine.evaluator as evaluator_module
import chromeperf.engine.event as event_module
import chromeperf.pinpoint.models.job as job_module
import chromeperf.pinpoint.models.task as task_module

_TASK_MODULE_MAP = {
    'culprit_finder': culprit_finder,
    'find_isolate': isolate_finder,
    'read_value': result_reader,
    'run_test': test_runner,
}

_TASK_PAYLOAD_TYPE_MAP = {
    'find_culprit': find_culprit_task_payload_pb2.FindCulpritTaskPayload,
    'find_isolate': find_isolate_task_payload_pb2.FindIsolateTaskPayload,
    'read_value': result_reader_payload_pb2.ResultReaderPayload,
    'run_test': test_runner_payload_pb2.TestRunnerPayload,
}

_EVENT_PAYLOAD_TYPE_MAP = {
    'build': find_isolate_task_payload_pb2.BuildUpdate,
    'none': empty_pb2.Empty,
}


class FromDictError(TypeError):
    pass


def _maybe_unwrap_optional(kind):
    """If kind is a typing.Optional[SomeT], return SomeT (else return kind)."""
    # TODO: use typing.get_origin / typing.get_args
    if (kind.__origin__ is typing.Union
        and len(kind.__args__) == 2
        and kind.__args__[1] is type(None)):
        return kind.__args__[0]
    return kind


def _from_dict(client, kind, d):
    if d is None: return None
    try:
        if dataclasses.is_dataclass(kind):
            # Repository disabled construction directly
            if kind is repository.Repository:
                if type(d) is not dict:
                    raise FromDictError(
                        f'Expected Repository dictionary, got {d!r}')
                if d.get('name'):
                    return repository.Repository.FromName(client, d['name'])
                elif d.get('url'):
                    return repository.Repository.FromUrl(client, d['url'])

            fields = dataclasses.fields(kind)
            def has_default(f):
                return f.default is not None or f.default_factory is not None
            missing_fields = set(
                f.name for f in fields if not has_default(f)) - set(d.keys())
            if missing_fields:
                raise FromDictError(f'Missing fields for {kind.__name__}: '
                                    f'{", ".join(missing_fields)}')
            fieldtypes = {f.name: f.type for f in dataclasses.fields(kind)}
            try:
                return kind(
                    **{f: _from_dict(client, fieldtypes[f], d[f])
                       for f in d})
            except KeyError as err:
                raise FromDictError(
                    f'Unexpected field {err} for kind {kind.__name__}')

        # Type 'list' doesn't tell us how to construct it's member. Also
        # type hint is not a real class we can use.
        if type(kind) is typing._GenericAlias:
            kind = _maybe_unwrap_optional(kind)
            # TODO: use typing.get_origin / typing.get_args
            if kind.__origin__ is list:
                return [_from_dict(client, kind.__args__[0], x) for x in d]
            raise NotImplementedError('Only support typing.List')

        if issubclass(kind, proto_message.Message):
            return proto_json_format.ParseDict(d, kind())
        return d
    except FromDictError:
        raise
    except Exception as e:
        raise FromDictError(f'Parse {kind} from {repr(d)}: {e}')


def _task_entity_to_dict(task_entity: datastore.entity.Entity):
    task_dict = dataclasses.asdict(task_module.Task.FromEntity(task_entity))
    task_dict['key'] = task_dict['key'].id_or_name
    task_dict['created'] = task_dict['created'].isoformat()
    task_dict['dependencies'] = [
        d.id_or_name for d in task_dict['dependencies']
    ]
    try:
        if task_dict['task_type'] in _TASK_PAYLOAD_TYPE_MAP:
            payload = task_dict['payload']
            proto_type = _TASK_PAYLOAD_TYPE_MAP[task_dict['task_type']]
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
            task_dict['payload'], including_default_value_fields=True)
        task_dict['payload_decode_error'] = str(e)
    return task_dict


def _task_context_to_dict(context):
    d = dataclasses.asdict(context)
    d['payload'] = proto_json_format.MessageToDict(
        d['payload'], including_default_value_fields=True)
    return d


def create_app(client: datastore.Client = datastore.Client()):
    try:
        import googleclouddebugger
        googleclouddebugger.enable(breakpoint_enable_canary=True)
    except ImportError:
        pass

    app = flask.Flask(__name__)

    # TODO(fancl): Send a real request to check status?
    @app.route("/healthcheck", methods=['GET'])
    def healthcheck():  # pylint: disable=unused-variable
        return "ok"

    @app.route("/debug/jobs/<job_id>", methods=['GET', 'POST', 'PATCH'])
    def job(job_id):  # pylint: disable=unused-variable
        job_key = client.key('Job', job_id)

        if flask.request.method == 'GET':
            with client.transaction():
                entities = client.query(kind='Task', ancestor=job_key).fetch()
                tasks = [_task_entity_to_dict(t) for t in entities]
            return flask.jsonify(tasks)

        @dataclasses.dataclass
        class JobPostRequest:
            type: str
            options: dict

        if flask.request.method == 'POST':
            req = _from_dict(
                client,
                JobPostRequest,
                flask.request.get_json(),
            )
            mod = _TASK_MODULE_MAP[req.type]
            options = _from_dict(client, mod.TaskOptions, req.options)
            graph = mod.create_graph(options)
            job = job_module.Job(
                key=job_key,
                user='test-user@example.com',
                url='https://pinpoint.service/job',
            )

            task_module.populate_task_graph(client, job, graph)
            return flask.jsonify({})

        @dataclasses.dataclass
        class JobUpdateEvent:
            type: str
            payload_type: str
            payload: dict
            target_task: typing.Optional[str] = None

        @dataclasses.dataclass
        class JobUpdateRequest:
            evaluator: str
            event: JobUpdateEvent

        if flask.request.method == 'PATCH':
            job = job_module.Job(
                key=job_key,
                user='test-user@example.com',
                url='https://pinpoint.service/job',
            )
            req = _from_dict(
                client,
                JobUpdateRequest,
                flask.request.get_json(),
            )
            evaluator = _TASK_MODULE_MAP[req.evaluator].Evaluator
            proto_type = _EVENT_PAYLOAD_TYPE_MAP[req.event.payload_type]
            context = evaluator_module.evaluate_graph(
                event_module.build_event(
                    type=req.event.type,
                    target_task=req.event.target_task,
                    payload=proto_json_format.ParseDict(
                        req.event.payload,
                        proto_type(),
                    ),
                ),
                evaluator(job, client),
                task_module.task_graph_loader(client, job),
            )
            return flask.jsonify(
                {k: _task_context_to_dict(v)
                 for k, v in context.items()})

    @app.route("/debug/jobs/<job_id>/tasks/<task_id>", methods=['GET'])
    def task(job_id, task_id):  # pylint: disable=unused-variable
        job_key = client.key('Job', job_id)
        entity = client.get(client.key('Task', task_id, parent=job_key))
        return flask.jsonify(_task_entity_to_dict(entity))

    return app
