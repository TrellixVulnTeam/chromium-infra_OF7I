# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import functools
import logging
import pprint
import pytest
import pytest_bdd
import random
import uuid

from chromeperf.engine import evaluator as evaluator_module
from chromeperf.engine import event as event_module
from chromeperf.pinpoint import find_isolate_task_payload_pb2
from chromeperf.pinpoint.evaluators import isolate_finder
from chromeperf.pinpoint.models import change as change_module
from chromeperf.pinpoint.models import commit as commit_module
from chromeperf.pinpoint.models import isolate as isolate_module
from chromeperf.pinpoint.models import repository as repository_module
from chromeperf.pinpoint.models import task as task_module

from .. import test_utils

pytest_bdd.scenarios('evaluators/isolate_finder.feature')


@pytest.fixture
def get_build_status(mocker):
    return mocker.patch('chromeperf.services.buildbucket_service.get')


@pytest.fixture
def put_job(mocker):
    return mocker.patch('chromeperf.services.buildbucket_service.put')


@pytest.fixture
def eval_context():
    return dict()


@pytest_bdd.given(
    pytest_bdd.parsers.cfparse('a repository named {repo:w} at {url}'),
    target_fixture='seed_repository',
)
def seed_repository(repo, url, datastore_client):
    repository_module.add_repository(datastore_client, repo, url)


@pytest_bdd.given(
    pytest_bdd.parsers.cfparse(
        'a commit {repo:w}@{commit:w} and change {change:d}'),
    target_fixture='seed_commit_and_change',
)
def seed_commit_and_change(repo, commit, change, mocker):
    change_map = {
        change: {
            '_number': 567890,
            'id': f'chromium~master~{commit}',
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
                            'url': test_utils.CHROMIUM_URL,
                            'ref': 'refs/changes/90/567890/5',
                        },
                    },
                    'commit_with_footers':
                    'Subject\n\nCommit message.\n'
                    'Change-Id: I0123456789abcdef',
                },
            },
        },
        f'{repo}~master~{commit}': {
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
                            'url': test_utils.CHROMIUM_URL,
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

    def _commit_info_stub(repository_url, git_hash, override=False):
        del repository_url
        return {
            'author': {
                'email': 'author@chromium.org',
            },
            'commit':
            commit,
            'committer': {
                'time': 'Fri Jan 01 00:01:00 2018 +1000'
            },
            'message':
            ('Subject.\n\nCommit message.\n'
             f'Reviewed-on: https://foo.bar/+/{change}\n'
             'Change-Id: 9d7cfd0abead41b83499dd0759b05939\n'
             f'Cr-Commit-Position: refs/heads/master@{{#{change}}}'),
            'parents': [],
        }

    def _get_change(server_url, change_id, fields=None):
        del server_url
        del fields
        return change_map.get(change_id)

    commit_info_mock = mocker.patch(
        'chromeperf.services.gitiles_service.commit_info')
    commit_info_mock.side_effect = _commit_info_stub
    mock_changes_handler = mocker.patch(
        'chromeperf.services.gerrit_service.get_change')
    mock_changes_handler.side_effect = _get_change
    return commit_info_mock, mock_changes_handler


@pytest_bdd.given('a Pinpoint job', target_fixture='pinpoint_job')
def pinpoint_job(datastore_client):
    return test_utils.MockJob(datastore_client.key('Job', str(uuid.uuid4())))


@pytest_bdd.given(
    pytest_bdd.parsers.cfparse('a cached isolate for commit {repo}@{commit}'),
    target_fixture='seeded_isolate_for_commit')
def seeded_isolate_for_commit(repo, commit, datastore_client):
    change = change_module.Change(commits=[
        commit_module.Commit(
            repository_module.Repository.FromName(datastore_client, repo),
            commit),
    ], )
    isolate_module.put(
        ((
            'Test Builder',
            change,
            'performance_test',
            'https://isolate.server',
            ''.join(random.choice('123456789abcdef') for _ in range(16)),
        ), ),
        datastore_client=datastore_client,
    )


@pytest_bdd.given(
    pytest_bdd.parsers.cfparse('a isolate-finding task graph for commit '
                               '{repo:w}@{commit:w}'),
    target_fixture='seed_task_graph',
)
def seed_task_graph(repo, commit, pinpoint_job, datastore_client):
    logging.debug('Seeding job = %s', pinpoint_job.key)
    graph = isolate_finder.create_graph(
        isolate_finder.TaskOptions(
            builder='Test Builder',
            target='performance_test',
            bucket='some.bucket',
            change=change_module.Change(commits=[
                commit_module.Commit(repository=repository_module.Repository.
                                     FromName(datastore_client, repo),
                                     git_hash=commit)
            ])))
    task_module.populate_task_graph(datastore_client, pinpoint_job, graph)
    logging.debug('graph = %s', pprint.pformat(graph))
    return graph


@pytest_bdd.given(
    'an isolate-finding evaluator',
    target_fixture='seeded_evaluator',
)
def seeded_evaluator(pinpoint_job, datastore_client):
    assert pinpoint_job != None
    return isolate_finder.Evaluator(pinpoint_job, datastore_client)


@pytest_bdd.given('attempts to schedule builds succeed')
def seed_mock_put_job(put_job):
    put_job.return_value = {
        'build': {
            'id': ''.join(random.choice('123456789') for _ in range(24))
        }
    }


@pytest_bdd.when('we evaluate the task graph')
def evaluate_task_graph(
    pinpoint_job,
    datastore_client,
    put_job,
    eval_context,
):
    event = event_module.build_event(
        type='initiate',
        payload=find_isolate_task_payload_pb2.BuildUpdate(
            id=str(uuid.uuid4())),
    )
    logging.debug('pinpoint_job = %s', pinpoint_job)
    logging.debug('put_job = %s', put_job)
    logging.debug('put_job.return_value = %s', put_job.return_value)
    context = evaluator_module.evaluate_graph(
        event,
        isolate_finder.Evaluator(
            job=pinpoint_job,
            datastore_client=datastore_client,
        ),
        task_module.task_graph_loader(datastore_client, pinpoint_job),
    )
    assert context != {}
    eval_context.clear()
    eval_context.update(context)


@pytest_bdd.then(
    pytest_bdd.parsers.cfparse('we must have scheduled {num:d} build'))
@pytest_bdd.then(
    pytest_bdd.parsers.cfparse('we must have scheduled {num:d} builds'))
def check_scheduled_build(num, eval_context, put_job):
    assert put_job.call_count == num


@pytest_bdd.then(
    pytest_bdd.parsers.cfparse(
        'the task payload has {article} buildbucket build'))
def check_payload_has_buildbucket_build(article, eval_context):
    assert eval_context != {}
    assert len(eval_context) == 1
    context = [v for v in eval_context.values()][0]
    payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    assert context.payload.Unpack(payload)
    if article == 'a':
        assert payload.buildbucket_build.id != ''
    elif article == 'no':
        assert payload.buildbucket_build.id == ''
    else:
        assert False, f'invalid article: "{article}"; please fix the test!'


@pytest_bdd.then(pytest_bdd.parsers.cfparse('the task is {state}'))
def check_task_is_ongoing(state, eval_context):
    assert eval_context != {}
    assert len(eval_context) == 1
    context = [v for v in eval_context.values()][0]
    assert context.state == state


@pytest_bdd.then(pytest_bdd.parsers.cfparse('the task has isolate details'))
def check_task_has_isolate_details(eval_context):
    assert eval_context != {}
    assert len(eval_context) == 1
    context = [v for v in eval_context.values()][0]
    payload = find_isolate_task_payload_pb2.FindIsolateTaskPayload()
    assert context.payload.Unpack(payload)
    assert payload.isolate_server != ''
    assert payload.isolate_hash != ''