# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses
import pytest
import uuid

import collections
from google.cloud import datastore
from deepdiff import DeepDiff

from chromeperf.engine import evaluator as evaluator_module
from chromeperf.engine import event as event_module
from chromeperf.engine import event_pb2
from chromeperf.pinpoint import change_pb2
from chromeperf.pinpoint import find_isolate_task_payload_pb2
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.evaluators import isolate_finder
from chromeperf.pinpoint.models import change as change_module
from chromeperf.pinpoint.models import commit as commit_module
from chromeperf.pinpoint.models import isolate as isolate_module
from chromeperf.pinpoint.models import repository as repository_module
from chromeperf.pinpoint.models import task as task_module

CHROMIUM_URL = 'https://chromium.googlesource.com/chromium/src'


@dataclasses.dataclass
class MockJob:
    key: datastore.Key
    user: str = dataclasses.field(default='test-user@example.com')
    url: str = dataclasses.field(default='https://pinpoint.service/job')

    @property
    def job_id(self):
        return str(self.key.id)


@pytest.fixture(autouse=True)
def pinpoint_seeded_data(datastore_client):
    # Add some test repositories.
    repository_module.add_repository(
        datastore_client,
        'catapult',
        'https://chromium.googlesource.com/catapult',
    )
    repository_module.add_repository(
        datastore_client,
        'chromium',
        CHROMIUM_URL,
    )


@pytest.fixture
def populate_job_tasks(request, pinpoint_seeded_data, datastore_client):
    job = MockJob(datastore_client.key('Job', str(uuid.uuid4())))
    git_hash = request.node.get_closest_marker('git_hash')
    if not git_hash:
        git_hash = '7c7e90be'
    else:
        git_hash = git_hash.args[0]

    task_module.populate_task_graph(
        datastore_client, job,
        isolate_finder.create_graph(
            isolate_finder.TaskOptions(
                builder='Mac Builder',
                target='telemetry_perf_tests',
                bucket='luci.bucket',
                change=change_module.Change(commits=[
                    commit_module.Commit(repository='chromium',
                                         git_hash=git_hash)
                ]))))
    return job


@pytest.fixture
def get_build_status(mocker):
    return mocker.patch('chromeperf.services.buildbucket_service.get')


@pytest.fixture
def put_job(mocker):
    return mocker.patch('chromeperf.services.buildbucket_service.put')


@pytest.fixture(autouse=True)
def seed_commit_info(mocker):
    def _commit_info_stub(repository_url, git_hash, override=False):
        del repository_url
        if git_hash == 'HEAD':
            git_hash = 'git hash at HEAD'
        components = git_hash.split('_')
        parents = []
        if not override and len(components) > 1:
            if components[0] == 'commit':
                parent_num = int(components[1]) - 1
                parents.append('commit_' + str(parent_num))

        return {
            'author': {
                'email': 'author@chromium.org',
            },
            'commit':
            git_hash,
            'committer': {
                'time': 'Fri Jan 01 00:01:00 2018 +1000'
            },
            'message':
            'Subject.\n\nCommit message.\n'
            'Reviewed-on: https://foo.bar/+/123456\n'
            'Change-Id: If32lalatdfg325simon8943washere98j589\n'
            'Cr-Commit-Position: refs/heads/master@{#123456}',
            'parents':
            parents,
        }

    commit_info_mock = mocker.patch(
        'chromeperf.services.gitiles_service.commit_info')
    commit_info_mock.side_effect = _commit_info_stub
    return commit_info_mock


@pytest.fixture(autouse=True)
def seed_changes(mocker, request):
    git_hash = request.node.get_closest_marker('git_hash')
    if not git_hash:
        git_hash = '7c7e90be'
    else:
        git_hash = git_hash.args[0]
    change_map = {
        123456: {
            '_number': 567890,
            'id': f'chromium~master~{git_hash}',
            'current_revision': 'abc123',
            'project': 'project/name',
            'subject': 'Patch subject.',
            'revisions': {
                'abc123': {
                    '_number':
                    5,
                    'created':
                    '2018-02-01 23:46:56.000000000',
                    'uploader': {
                        'email': 'author@codereview.com'
                    },
                    'fetch': {
                        'http': {
                            'url': CHROMIUM_URL,
                            'ref': 'refs/changes/90/567890/5',
                        },
                    },
                    'commit_with_footers':
                    'Subject\n\nCommit message.\n'
                    'Change-Id: I0123456789abcdef',
                },
            },
        },
        f'chromium~master~{git_hash}': {
            'current_revision': 'abc123',
            'project': 'project/name',
            'subject': 'Patch subject.',
            'revisions': {
                'abc123': {
                    '_number':
                    5,
                    'created':
                    '2018-02-01 23:46:56.000000000',
                    'uploader': {
                        'email': 'author@codereview.com'
                    },
                    'fetch': {
                        'http': {
                            'url': CHROMIUM_URL,
                            'ref': 'refs/changes/90/567890/5',
                        },
                    },
                    'commit_with_footers':
                    'Subject\n\nCommit message.\n'
                    'Change-Id: I0123456789abcdef',
                },
            }
        }
    }

    def _get_change(server_url, change_id, fields=None):
        del server_url
        del fields
        return change_map.get(change_id)

    mock_changes_handler = mocker.patch(
        'chromeperf.services.gerrit_service.get_change')
    mock_changes_handler.side_effect = _get_change
    return mock_changes_handler


def test_IsolateFinder_Initiate_FoundIsolate(mocker, datastore_client,
                                             populate_job_tasks):
    # Seed the isolate for this change.
    change = change_module.Change(
        commits=[commit_module.Commit('chromium', '7c7e90be')])
    isolate_module.put((('Mac Builder', change, 'telemetry_perf_tests',
                         'https://isolate.server', '7c7e90be'), ),
                       datastore_client=datastore_client)

    # Ensure that we can find the seeded isolate for the specified revision.
    event = event_module.build_event(
        type='initiate',
        target_task='find_isolate_chromium@7c7e90be',
        payload=find_isolate_task_payload_pb2.BuildUpdate(
            id=str(uuid.uuid4())),
    )
    context = evaluator_module.evaluate_graph(
        event,
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    assert context['find_isolate_chromium@7c7e90be'].payload.Unpack(payload)
    # FIXME: Increase test coverage here.
    assert payload.bucket == 'luci.bucket'


@pytest.mark.git_hash('600dfeed')
def test_IsolateFinder_Initiate_ScheduleBuild(mocker, put_job,
                                              populate_job_tasks,
                                              datastore_client):
    # We then need to make sure that the buildbucket put was called.
    put_job.return_value = {'build': {'id': '345982437987234'}}

    # This time we don't seed the isolate for the change to force the build.
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task='find_isolate_chromium@600dfeed',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    task_context = context['find_isolate_chromium@600dfeed']
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    assert task_context.payload.Unpack(task_payload)
    assert task_payload.buildbucket_build.id == '345982437987234'
    assert task_payload.tries == 1
    assert task_context.state == 'ongoing'
    assert put_job.call_count == 1


def test_IsolateFinder_Update_BuildSuccessful(mocker, put_job,
                                              get_build_status,
                                              populate_job_tasks,
                                              datastore_client):
    put_job.return_value = {
        'build': {
            'id': '345982437987234',
            'url': 'https://some.buildbucket/url'
        }
    }
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task='find_isolate_chromium@7c7e90be',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    task_context = context['find_isolate_chromium@7c7e90be']
    assert task_context.state == 'ongoing'
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    assert task_context.payload.Unpack(task_payload)
    assert task_payload.buildbucket_build.id == '345982437987234'
    assert (
        task_payload.buildbucket_build.url == 'https://some.buildbucket/url')
    assert put_job.call_count == 1

    # Now we send an update event which should cause us to poll the status of
    # the build on demand.
    json = """
    {
      "properties": {
          "got_revision_cp": "refs/heads/master@7c7e90be",
          "isolate_server": "https://isolate.server",
          "swarm_hashes_refs/heads/master(at)7c7e90be_without_patch":
              {"telemetry_perf_tests": "192923affe212adf"}
      }
    }"""
    get_build_status.return_value = {
        'build': {
            'id': '345982437987234',
            'url': 'https://some.buildbucket/url',
            'status': 'COMPLETED',
            'result': 'SUCCESS',
            'result_details_json': json,
        }
    }
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='update',
            target_task='find_isolate_chromium@7c7e90be',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    task_context = context['find_isolate_chromium@7c7e90be']
    assert task_context.state == 'completed'
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    task_context.payload.Unpack(task_payload)
    assert task_payload.isolate_hash == '192923affe212adf'
    assert task_payload.isolate_server == 'https://isolate.server'
    assert get_build_status.call_count == 1
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='unimportant',
            target_task=None,
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Serializer(),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    assert context == {
        'find_isolate_chromium@7c7e90be': {
            'completed':
            True,
            'exception':
            None,
            'details': [{
                'key': 'builder',
                'value': 'Mac Builder',
                'url': 'https://some.buildbucket/url',
            }, {
                'key': 'build',
                'value': '345982437987234',
                'url': mocker.ANY,
            }, {
                'key':
                'isolate',
                'value':
                '192923affe212adf',
                'url':
                'https://isolate.server/browse?digest=192923affe212adf',
            }]
        }
    }


@pytest.fixture
def seed_build_data(mocker, populate_job_tasks, datastore_client, put_job,
                    seed_changes, seed_commit_info, request):
    # Here we set up the pre-requisite for polling, where we've already had a
    # successful build scheduled.
    put_job.return_value = {'build': {'id': '345982437987234'}}
    git_hash = request.node.get_closest_marker('git_hash')
    if not git_hash:
        git_hash = '7c7e90be'
    else:
        git_hash = git_hash.args[0]
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=f'find_isolate_chromium@{git_hash}',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    assert put_job.call_count == 1
    task_context = context[f'find_isolate_chromium@{git_hash}']
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    assert task_context.payload.Unpack(task_payload)
    assert task_context.state == 'ongoing'
    expected_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload(
        change=change_pb2.Change(commits=[
            change_pb2.Commit(repository='chromium', git_hash=git_hash)
        ]),
        buildbucket_build=find_isolate_task_payload_pb2.BuildBucketBuild(
            id='345982437987234'),
        builder='Mac Builder',
        bucket='luci.bucket',
        target='telemetry_perf_tests',
        tries=1,
    )
    assert task_payload == expected_payload
    return put_job


def test_IsolateFinder_Update_BuildFailed_HardFailure(mocker, seed_build_data,
                                                      get_build_status,
                                                      datastore_client,
                                                      populate_job_tasks):
    get_build_status.return_value = {
        'build': {
            'status': 'COMPLETED',
            'result': 'FAILURE',
            'result_details_json': '{}',
        }
    }
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='update',
            target_task='find_isolate_chromium@7c7e90be',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4()),
                state=find_isolate_task_payload_pb2.BuildUpdate.BuildState.
                COMPLETED),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    assert get_build_status.call_count == 1
    task_context = context['find_isolate_chromium@7c7e90be']
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    task_context.payload.Unpack(task_payload)
    assert len(task_payload.errors) == 1
    assert task_payload.errors[0].reason == 'BuildFailed'
    assert '345982437987234' in task_payload.errors[0].message
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='unimportant',
            target_task=None,
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Serializer(),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    assert context == {
        'find_isolate_chromium@7c7e90be': {
            'completed':
            True,
            'exception':
            mocker.ANY,
            'details': [{
                'key': 'builder',
                'value': 'Mac Builder',
                'url': None,
            }, {
                'key': 'build',
                'value': '345982437987234',
                'url': mocker.ANY,
            }]
        }
    }


def test_IsolateFinder_Update_BuildFailed_Cancelled(mocker, seed_build_data,
                                                    get_build_status,
                                                    datastore_client,
                                                    populate_job_tasks):
    get_build_status.return_value = {
        'build': {
            'status': 'COMPLETED',
            'result': 'CANCELLED',
            'result_details_json': '{}',
        }
    }
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='update',
            target_task='find_isolate_chromium@7c7e90be',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    task_context = context['find_isolate_chromium@7c7e90be']
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    task_context.payload.Unpack(task_payload)
    assert task_context.state == 'failed'
    assert task_payload.buildbucket_build.result == 'CANCELLED'
    assert get_build_status.call_count == 1


def test_IsolateFinder_Update_MissingIsolates_Server(mocker, seed_build_data,
                                                     get_build_status,
                                                     datastore_client,
                                                     populate_job_tasks):
    json = """
    {
        "properties": {
            "got_revision_cp": "refs/heads/master@7c7e90be",
            "swarm_hashes_refs/heads/master(at)7c7e90be_without_patch":
                {"telemetry_perf_tests": "192923affe212adf"}
        }
    }"""
    get_build_status.return_value = {
        'build': {
            'status': 'COMPLETED',
            'result': 'SUCCESS',
            'result_details_json': json,
        }
    }
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='update',
            target_task='find_isolate_chromium@7c7e90be',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    task_context = context['find_isolate_chromium@7c7e90be']
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    task_context.payload.Unpack(task_payload)
    assert task_context.state == 'failed'
    assert task_payload.errors[0].reason == 'BuildIsolateNotFound'
    assert get_build_status.call_count == 1


def test_IsolateFinder_Update_MissingIsolates_Revision(mocker, seed_build_data,
                                                       get_build_status,
                                                       datastore_client,
                                                       populate_job_tasks):
    json = """
    {
        "properties": {
            "isolate_server": "https://isolate.server",
            "swarm_hashes_refs/heads/master(at)7c7e90be_without_patch":
                {"telemetry_perf_tests": "192923affe212adf"}
        }
    }"""
    get_build_status.return_value = {
        'build': {
            'status': 'COMPLETED',
            'result': 'SUCCESS',
            'result_details_json': json,
        }
    }
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='update',
            target_task='find_isolate_chromium@7c7e90be',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    task_context = context['find_isolate_chromium@7c7e90be']
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    task_context.payload.Unpack(task_payload)
    assert task_context.state == 'failed'
    assert task_payload.errors[0].reason == 'BuildIsolateNotFound'
    assert get_build_status.call_count == 1


def test_IsolateFinder_Update_MissingIsolates_Hashes(mocker, seed_build_data,
                                                     get_build_status,
                                                     datastore_client,
                                                     populate_job_tasks):
    json = """
    {
      "properties": {
          "got_revision_cp": "refs/heads/master@7c7e90be",
          "isolate_server": "https://isolate.server"
      }
    }"""
    get_build_status.return_value = {
        'build': {
            'status': 'COMPLETED',
            'result': 'SUCCESS',
            'result_details_json': json,
        }
    }
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='update',
            target_task='find_isolate_chromium@7c7e90be',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4())),
        ),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    task_context = context['find_isolate_chromium@7c7e90be']
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    assert task_context.payload.Unpack(task_payload)
    assert task_context.state == 'failed'
    assert task_payload.errors[0].reason == 'BuildIsolateNotFound'
    assert get_build_status.call_count == 1


@pytest.mark.git_hash('600df00d')
def test_IsolateFinder_Update_MissingIsolates_InvalidJson(
        mocker, seed_build_data, get_build_status, datastore_client,
        populate_job_tasks):
    json = '{ invalid }'
    get_build_status.return_value = {
        'build': {
            'status': 'COMPLETED',
            'result': 'SUCCESS',
            'result_details_json': json,
        }
    }
    context = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='update',
            target_task='find_isolate_chromium@600df00d',
            payload=find_isolate_task_payload_pb2.BuildUpdate(
                id=str(uuid.uuid4()))),
        isolate_finder.Evaluator(populate_job_tasks, datastore_client),
        task_module.task_graph_loader(datastore_client, populate_job_tasks),
    )
    task_context = context['find_isolate_chromium@600df00d']
    task_payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    assert task_context.payload.Unpack(task_payload)
    assert task_context.state == 'failed'
    assert task_payload.errors[0].reason == 'BuildIsolateNotFound'
    assert get_build_status.call_count == 1


@pytest.mark.skip(reason='Not implemented yet.')
def test_IsolateFinder_Update_BuildFailed_ScheduleRetry():
    pass
