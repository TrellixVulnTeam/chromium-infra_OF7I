# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import logging
import pytest
import re
import uuid

from google.protobuf import any_pb2
from google.protobuf import empty_pb2

from chromeperf.engine import actions
from chromeperf.engine import combinators
from chromeperf.engine import evaluator as evaluator_module
from chromeperf.engine import event as event_module
from chromeperf.engine import task_pb2
from chromeperf.pinpoint import change_pb2
from chromeperf.pinpoint import find_culprit_task_payload_pb2
from chromeperf.pinpoint import result_reader_payload_pb2
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.evaluators import culprit_finder
from chromeperf.pinpoint.models import change as change_module
from chromeperf.pinpoint.models import task as task_module

from . import test_utils  # pylint: disable=relative-beyond-top-level
from . import bisection_test_util  # pylint: disable=relative-beyond-top-level



@pytest.fixture
def gitiles_commit_info(mocker):
    mock = mocker.patch('chromeperf.services.gitiles_service.commit_info')
    def _commit_info_stub(repository_url, git_hash, override=False):
        del repository_url
        return {
            'author': {
                'email': 'author@chromium.org',
            },
            'commit': git_hash,
            'committer': {
                'time': 'Fri Jan 01 00:01:00 2018 +1000'
            },
            'message':
            ('Subject.\n\nCommit message.\n'
             f'Reviewed-on: https://foo.bar/+/baz\n'
             'Change-Id: badc0ffeedeadbeef\n'
             f'Cr-Commit-Position: refs/heads/master@{{#{git_hash}}}'),
            'parents': [],
        }

    mock.side_effect = _commit_info_stub
    return mock


@pytest.fixture
def gitiles_commit_range(mocker):
    mock = mocker.patch('chromeperf.services.gitiles_service.commit_range')
    def _commit_range_stub(repository_url, first_git_hash, last_git_hash):
        assert first_git_hash.startswith('commit_')
        assert last_git_hash.startswith('commit_')
        first_idx = int(first_git_hash[len('commit_'):])
        last_idx = int(last_git_hash[len('commit_'):])
        return [
            {
                'author': {
                    'email': 'author@chromium.org',
                },
                'commit': f'commit_{idx}',
                'committer': {
                    'time': 'Fri Jan 01 00:01:00 2018 +1000'
                },
                'message':
                ('Subject.\n\nCommit message.\n'
                 f'Reviewed-on: https://foo.bar/+/baz\n'
                 'Change-Id: badc0ffeedeadbeef\n'
                 f'Cr-Commit-Position: refs/heads/master@{{#commit_{idx}}}'),
                'parents': [f'commit_{idx-1}'],
            }
            for idx in reversed(range(first_idx + 1, last_idx + 1))]

    mock.side_effect = _commit_range_stub
    return mock



def test_convert_params_smoketest(datastore_client, gitiles_commit_info):
    options = culprit_finder.convert_params(
        {
            'start_git_hash': 'commit_0',
            'end_git_hash': 'commit_5',
            'repository': 'chromium',
            'target': 'some target',
            'benchmark': 'some benchmark',
        },
        datastore_client)
    assert options.start_change.base_commit.git_hash == 'commit_0'
    assert options.start_change.base_commit.repository.name == 'chromium'


def test_create_graph_smoketest(datastore_client, gitiles_commit_info):
    job = test_utils.MockJob(datastore_client.key('Job', str(uuid.uuid4())))
    options = culprit_finder.convert_params(
        {
            'start_git_hash': 'commit_0',
            'end_git_hash': 'commit_5',
            'repository': 'chromium',
            'target': 'some target',
            'benchmark': 'some benchmark',
        },
        datastore_client)
    options.analysis_options.min_attempts = 2
    options.test_options.attempts = 2

    graph = culprit_finder.create_graph(options)
    assert set(
            [e.to for e in graph.edges if e.from_ == 'performance_bisection']
        ) == set(['read_value_chromium@commit_0_0',
                  'read_value_chromium@commit_0_1',
                  'read_value_chromium@commit_5_0',
                  'read_value_chromium@commit_5_1']), (
            "There are edges from performance_bisection to read_value for "
            "start and end commits")
    assert set([v.id for v in graph.vertices]) == set(
            ['performance_bisection',
             'find_isolate_chromium@commit_0',
             'find_isolate_chromium@commit_5',
             'run_test_chromium@commit_0_0',
             'run_test_chromium@commit_0_1',
             'read_value_chromium@commit_0_0',
             'read_value_chromium@commit_0_1',
             'run_test_chromium@commit_5_0',
             'run_test_chromium@commit_5_1',
             'read_value_chromium@commit_5_0',
             'read_value_chromium@commit_5_1',
             ]), "Includes subgraphs of find_isolate, run_test, and read_value."

    task_module.populate_task_graph(datastore_client, job, graph)
    # Doesn't explode.


@pytest.fixture
def simple_job(datastore_client, gitiles_commit_info):
    job = test_utils.MockJob(datastore_client.key('Job', str(uuid.uuid4())))
    options = culprit_finder.convert_params(
        {
            'start_git_hash': 'commit_0',
            'end_git_hash': 'commit_5',
            'repository': 'chromium',
            'target': 'some target',
            'benchmark': 'some benchmark',
            'builder': 'Some builder',
            'target': 'telemetry_perf_tests',
            'bucket': 'luci.bucket',
        },
        datastore_client)

    task_module.populate_task_graph(
        datastore_client, job,
        culprit_finder.create_graph(options)
        )
    return job


def select_find_culprit_task(graph_loader):
    find_culprit_tasks = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select', target_task=None, payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='find_culprit'),
        graph_loader,
    )
    assert len(find_culprit_tasks) == 1
    return list(find_culprit_tasks.values())[0]


def test_evaluate_graph_initially_expands_commit_range(
        simple_job, datastore_client, gitiles_commit_range):
    loader = task_module.task_graph_loader(datastore_client, simple_job)
    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate', target_task=None, payload=empty_pb2.Empty()),
        culprit_finder.Evaluator(datastore_client, simple_job),
        loader,
    ) == {}

    # The first phase runs PrepareCommitsAction (call gitiles to expand the
    # start and end commit into a complete range), so check that the full
    # range of changes to explore has been populated.
    find_culprit = select_find_culprit_task(loader)
    assert find_culprit.state == 'ongoing'
    task_payload = find_culprit_task_payload_pb2.FindCulpritTaskPayload()
    find_culprit.payload.Unpack(task_payload)
    assert len(task_payload.state.changes) == 6
    assert [ch.commits[0].git_hash for ch in task_payload.state.changes] == [
            'commit_0', 'commit_1', 'commit_2', 'commit_3', 'commit_4', 'commit_5']


class FakeReadValueEvaluator:
    def __init__(self, datastore_client, job, values_fn, errors_fn):
        self.datastore_client = datastore_client
        self.job = job
        self.values_fn = values_fn
        self.errors_fn = errors_fn

    def __call__(self, task, event, context):
        del event
        if task.state in {'completed', 'failed'}: return []
        match = re.match(
            'read_value_chromium@commit_([0-9]+)_([0-9]+)$', task.id)
        assert match is not None, f'unexpected task id: {task.id}'
        commit_num = int(match.group(1), 10)
        attempt = int(match.group(2), 10)
        fake_payload = result_reader_payload_pb2.ResultReaderPayload()
        fake_payload.input.change.commits.add(
                repository='chromium', git_hash=f'commit_{commit_num}')
        fake_payload.output.result_values.extend(
                self.values_fn(commit_num, attempt))
        new_state = 'completed'
        if self.errors_fn is not None:
            errors = self.errors_fn(commit_num, attempt)
            fake_payload.errors.extend(errors)
            if errors: new_state = 'failed'
        return [updates.UpdateTaskAction(self.datastore_client, self.job,
                                         task.id, new_state=new_state,
                                         payload=test_utils.as_any(fake_payload))]


@pytest.fixture
def fake_evalulator_factory(datastore_client):
    class _Factory:
        @staticmethod
        def read_value_fake(job, values_fn, errors_fn=None):
            """Make a fake read value task.

            values_fn: (commit num, attempt num) -> result values list
            errors_fn: (commit num, attempt num) -> ErrorMessage list.
            """
            return FakeReadValueEvaluator(datastore_client, job, values_fn,
                                          errors_fn)
    return _Factory()


def wrap_in_payload_lifter(evaluator):
    return combinators.SequenceEvaluator(
            [evaluator, combinators.TaskPayloadLiftingEvaluator()])


@pytest.fixture
def simple_bisection_job(request, datastore_client):
    """Make a simple bisection graph for a range of 10 commits.

    Only the find_culprit task and its direct dependencies (read_value tasks)
    are populated.

    Set the 'simple_bisection_job_overrides' marker to override default values
    for 'commit_count' and 'analysis_min_attempts'.
    """
    marker = request.node.get_closest_marker('simple_bisection_job_overrides')
    kwargs = marker.kwargs if marker is not None else {}
    commit_count = kwargs.get('commit_count', 10)
    analysis_min_attempts = kwargs.get('analysis_min_attempts', 10)

    payload = find_culprit_task_payload_pb2.FindCulpritTaskPayload()
    payload.input.start_change.commits.add(
            repository='chromium', git_hash='commit_0')
    payload.input.end_change.commits.add(
            repository='chromium', git_hash=f'commit_{commit_count-1}')
    payload.input.analysis_options.comparison_magnitude = 1.0
    payload.input.analysis_options.min_attempts = analysis_min_attempts
    payload.input.analysis_options.max_attempts = 60
    for i in range(commit_count):
        payload.state.changes.add(commits=[change_pb2.Commit(
                repository='chromium', git_hash=f'commit_{i}')])
    find_culprit_task = evaluator_module.TaskVertex(
         id='performance_bisection',
         vertex_type='find_culprit',
         payload=test_utils.as_any(payload),
    )
    read_values_tasks = [
        evaluator_module.TaskVertex(
            id=f'read_value_chromium@commit_{commit_num}_{attempt}',
            vertex_type='read_value',
            payload=test_utils.as_any(
                result_reader_payload_pb2.ResultReaderPayload()))
        for commit_num in range(commit_count)
        for attempt in range(analysis_min_attempts)
    ]

    graph = evaluator_module.TaskGraph(
            vertices=[find_culprit_task] + read_values_tasks,
            edges=[evaluator_module.Dependency(
                    from_='performance_bisection', to=rv_task.id)
                   for rv_task in read_values_tasks])
    job = test_utils.MockJob(datastore_client.key('Job', str(uuid.uuid4())))
    task_module.populate_task_graph(datastore_client, job, graph)
    updates.update_task(datastore_client, job, 'performance_bisection',
                        new_state='ongoing')
    return job


def fake_read_values_subgraph_context_evaluator():
    def fake_evalulator(task, event, context):
        del event
        del task
        if 'fake_read_values_subgraph_context_evaluator' in context: return
        for commit_num in range(10):
            for attempt_num in range(100):
                change_id = f'commit_{commit_num}@chromium'
                context[f'read_value_{change_id}_{attempt_num}'] = 'dummy value'
                #context[f'read_value_{change_id}_{attempt_num}'] = 'dummy value'
        context['fake_read_values_subgraph_context_evaluator'] = None
    return fake_evalulator


def test_evaluate_graph_success_no_repro(datastore_client, simple_bisection_job,
                                         fake_evalulator_factory, mocker):
    job = simple_bisection_job
    loader = task_module.task_graph_loader(datastore_client, job)
    evaluator = wrap_in_payload_lifter(
        combinators.DispatchByTaskType(
            {'find_culprit': culprit_finder.Evaluator(datastore_client, job),
             'read_value': fake_evalulator_factory.read_value_fake(
                     job, lambda ignore_commit, ignore_attempt: [1.0])}))
    evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate', target_task=None, payload=empty_pb2.Empty()),
        evaluator,
        loader,
    )
    find_culprit = select_find_culprit_task(loader)
    assert find_culprit.state == 'completed'
    task_payload = find_culprit_task_payload_pb2.FindCulpritTaskPayload()
    find_culprit.payload.Unpack(task_payload)
    assert len(task_payload.output.culprits) == 0


@pytest.mark.simple_bisection_job_overrides(commit_count=6)
def test_evaluate_graph_success_speculate_bisection(datastore_client,
                                                    simple_bisection_job,
                                                    fake_evalulator_factory):
    job = simple_bisection_job
    loader = task_module.task_graph_loader(datastore_client, job)
    # There's significant change between commit_1 and commit_2.
    values_for_commits = {
        0: [1.0] * 10,
        1: [1.0] * 10,
        2: [2.0] * 10,
        3: [2.0] * 10,
        4: [2.0] * 10,
        5: [2.0] * 10,
    }
    evaluator = wrap_in_payload_lifter(
        combinators.DispatchByTaskType(
            {'find_culprit': culprit_finder.Evaluator(datastore_client, job),
             'read_value': fake_evalulator_factory.read_value_fake(
                     job, lambda commit_no, _: values_for_commits[commit_no])}))
    evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate', target_task=None, payload=empty_pb2.Empty()),
        evaluator,
        loader,
    )
    find_culprit = select_find_culprit_task(loader)
    assert find_culprit.state == 'completed'
    task_payload = find_culprit_task_payload_pb2.FindCulpritTaskPayload()
    find_culprit.payload.Unpack(task_payload)
    # We found the change between commit_1 and commit_2.
    assert len(task_payload.output.culprits) == 1
    culprit = task_payload.output.culprits[0]
    assert change_module.Change.FromProto(
            datastore_client, culprit.from_).id_string == 'chromium@commit_1'
    assert change_module.Change.FromProto(
            datastore_client, culprit.to).id_string == 'chromium@commit_2'


@pytest.mark.simple_bisection_job_overrides(commit_count=6)
def test_evaluate_graph_success_need_to_refine(datastore_client,
                                               simple_bisection_job,
                                               fake_evalulator_factory):
    job = simple_bisection_job
    loader = task_module.task_graph_loader(datastore_client, job)
    values_for_commits = {
        0: list(range(0, 10)),
        1: list(range(1, 11)),
        2: list(range(2, 12)),
        3: list(range(3, 13)),
        4: list(range(3, 13)),
        5: list(range(3, 13)),
    }
    evaluator = wrap_in_payload_lifter(
        combinators.DispatchByTaskType(
            {'find_culprit': culprit_finder.Evaluator(datastore_client, job),
             'read_value': fake_evalulator_factory.read_value_fake(
                     job, lambda commit_no, _: values_for_commits[commit_no])}))
    evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate', target_task=None, payload=empty_pb2.Empty()),
        evaluator,
        loader,
    )

    # Here we test that we have more than the minimum attempts for the change
    # between commit_1 and commit_2.
    read_value_tasks = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select', target_task=None, payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        loader,
    ).values()
    assert len(read_value_tasks) != 0, read_value_tasks
    attempt_counts = collections.defaultdict(lambda: 0)
    for rv_task in read_value_tasks:
        rv_payload = result_reader_payload_pb2.ResultReaderPayload()
        rv_task.payload.Unpack(rv_payload)
        change = change_module.Change.FromProto(
                datastore_client, rv_payload.input.change).id_string
        attempt_counts[change] += 1
    logging.debug(f'{attempt_counts!r}')
    assert 10 < attempt_counts['chromium@commit_2'] < 100, attempt_counts

    # We know that we will refine the graph until we see the progression from
    # commit_0 -> commit_1 -> commit_2 -> commit_3 and stabilize.
    find_culprit = select_find_culprit_task(loader)
    assert find_culprit.state == 'completed'
    task_payload = find_culprit_task_payload_pb2.FindCulpritTaskPayload()
    find_culprit.payload.Unpack(task_payload)
    assert len(task_payload.output.culprits) == 3


def test_evaluate_graph_failure_dependencies_failed(
        datastore_client, simple_bisection_job, fake_evalulator_factory):
    job = simple_bisection_job
    loader = task_module.task_graph_loader(datastore_client, job)
    def read_value_fail(task, event, context):
        if task.state == 'failed': return []
        payload = result_reader_payload_pb2.ResultReaderPayload(
            errors=[task_pb2.ErrorMessage(reason='SomeReason',
                                          message='message')])
        encoded_payload = test_utils.as_any(payload)
        return [updates.UpdateTaskAction(datastore_client, job,
                                         task.id, new_state='failed',
                                         payload=encoded_payload)]
    evaluator = wrap_in_payload_lifter(
        combinators.DispatchByTaskType(
            {'find_culprit': culprit_finder.Evaluator(datastore_client, job),
             'read_value': read_value_fail}))
    evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate', target_task=None, payload=empty_pb2.Empty()),
        evaluator,
        loader,
    )

    find_culprit = select_find_culprit_task(loader)
    assert find_culprit.state == 'failed'
    task_payload = find_culprit_task_payload_pb2.FindCulpritTaskPayload()
    find_culprit.payload.Unpack(task_payload)
    assert len(task_payload.errors) != 0


def test_evaluate_graph_failure_dependencies_no_results(
        datastore_client, simple_bisection_job, fake_evalulator_factory):
    job = simple_bisection_job
    loader = task_module.task_graph_loader(datastore_client, job)
    evaluator = wrap_in_payload_lifter(
        combinators.DispatchByTaskType(
            {'find_culprit': culprit_finder.Evaluator(datastore_client, job),
             'read_value': fake_evalulator_factory.read_value_fake(
                 job,
                 values_fn=lambda *_: [],
                 errors_fn=lambda *_: [task_pb2.ErrorMessage(
                         reason='SomeReason', message='message')])
             }))
    evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate', target_task=None, payload=empty_pb2.Empty()),
        evaluator,
        loader,
    )
    find_culprit = select_find_culprit_task(loader)
    assert find_culprit.state == 'failed'
    task_payload = find_culprit_task_payload_pb2.FindCulpritTaskPayload()
    find_culprit.payload.Unpack(task_payload)
    assert len(task_payload.errors) != 0

@pytest.mark.skip(reason=
    'Implement the case where intermediary builds/tests failed but we can find '
    'some non-failing intermediary CLs')
def test_evaluate_ambiguous_intermediate_partial_failure():
    pass

@pytest.mark.skip(reason=
    'Implement the case where the likely culprit is an auto-roll commit, in '
    'which case we want to embellish the commit range with commits from the '
    'remote repositories')
def test_evaluate_ambiguous_intermediate_culprit_is_autoroll():
    pass

@pytest.mark.skip(reason=
    'Implement the case where we have already found a culprit and we still '
    'have ongoing builds/tests running but have the chance to cancel those.')
def test_evaluate_ambiguous_intermediate_culprit_found_cancel_ogoing(self):
    pass

@pytest.mark.skip(reason=
    'Implement the case where either the start or end commits are broken.')
def test_evaluate_failure_extent_cls_failed(self):
    pass
