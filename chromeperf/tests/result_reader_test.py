# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import itertools
import json
import uuid

import pytest
from google.protobuf import empty_pb2

from chromeperf.engine import combinators
from chromeperf.engine import evaluator as evaluator_module
from chromeperf.engine import event as event_module
from chromeperf.engine import predicates
from chromeperf.pinpoint import result_reader_payload_pb2
from chromeperf.pinpoint.evaluators import isolate_finder
from chromeperf.pinpoint.evaluators import result_reader
from chromeperf.pinpoint.evaluators import test_runner
from chromeperf.pinpoint.models import task as task_module
from chromeperf.pinpoint.models import change as change_module
from chromeperf.pinpoint.models import commit as commit_module
from chromeperf.pinpoint.models import repository as repository_module

from . import bisection_test_util  # pylint: disable=relative-beyond-top-level
from . import test_utils  # pylint: disable=relative-beyond-top-level


def evaluator(datastore_client, job):
    return combinators.SequenceEvaluator(evaluators=(
        combinators.FilteringEvaluator(
            predicate=predicates.TaskTypeEq('find_isolate'),
            delegate=combinators.SequenceEvaluator(evaluators=(
                bisection_test_util.FakeFoundIsolate(datastore_client, job),
                combinators.TaskPayloadLiftingEvaluator()))),
        combinators.FilteringEvaluator(
            predicate=predicates.TaskTypeEq('run_test'),
            delegate=combinators.SequenceEvaluator(
                evaluators=(bisection_test_util.FakeSuccessfulRunTest(
                    datastore_client, job),
                            combinators.TaskPayloadLiftingEvaluator()))),
        result_reader.Evaluator(datastore_client, job),
    ))


@pytest.fixture
def isolate_retrieve(mocker):
    mocked = mocker.patch('chromeperf.services.isolate.retrieve',
                          mocker.MagicMock())
    return mocked


@pytest.fixture
def populate_task_graph(datastore_client):
    job = test_utils.MockJob(datastore_client.key('Job', str(uuid.uuid4())))

    def populate(benchmark=None,
                 chart=None,
                 grouping_label=None,
                 story=None,
                 statistic=None,
                 trace='some_trace',
                 mode='histogram_sets'):
        task_option = result_reader.TaskOptions(
            test_options=test_runner.TaskOptions(
                build_options=isolate_finder.TaskOptions(
                    builder='Some Builder',
                    target='telemetry_perf_tests',
                    bucket='luci.bucket',
                    change=change_module.Change(commits=[
                        commit_module.Commit(
                            repository=repository_module.Repository.FromName(
                                datastore_client,
                                'chromium',
                            ),
                            git_hash='7c7e90be',
                        )
                    ])),
                swarming_server='some_server',
                dimensions=[],
                extra_args=[],
                attempts=10),
            benchmark=benchmark,
            histogram_options=result_reader.HistogramOptions(
                grouping_label=grouping_label,
                story=story,
                statistic=statistic,
                histogram_name=chart,
            ),
            graph_json_options=result_reader.GraphJsonOptions(chart=chart,
                                                              trace=trace),
            mode=mode,
            results_filename='perf_results.json',
        )
        task_module.populate_task_graph(
            datastore_client, job, result_reader.create_graph(task_option))

        return job

    return populate


def test_Evaluate_Success_WithData(datastore_client, populate_task_graph,
                                   isolate_retrieve):
    # Seed the response to the call to the isolate service.
    histograms = json.dumps([{
        'values': ['story'],
        'guid': '2f076467-3ae3-4618-9f50-aee0ab0c7359',
        'type': 'GenericSet'
    }, {
        'values': ['group:label'],
        'guid': '5a349379-3dd9-4b80-851c-40822e531183',
        'type': 'GenericSet'
    }, {
        'sampleValues': [0, 1, 2],
        'name': 'some_chart',
        'running': [3, 2, None, 1, 0, 3, 2],
        'binBoundaries': [1, [1, 1000.0, 20]],
        'diagnostics': {
            'storyTags': '5a349379-3dd9-4b80-851c-40822e531183',
            'stories': '2f076467-3ae3-4618-9f50-aee0ab0c7359'
        },
        'allBins': {
            0: [1],
            1: [1],
            3: [1]
        },
        'unit': 'count'
    }])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([
        ('{"files": {"some_benchmark/perf_results.json": '
         '{"h": "394890891823812873798734a"}}}'),
        histograms,
    ], 10))

    # Set it up so that we are building a graph that's looking for no statistic.
    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
        grouping_label='label',
        story='story',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    # Ensure we find the find a value, and the histogram (?) associated with the
    # data we're looking for.
    # task_payload = test_runner_payload_pb2.TestRunnerPayload()
    # assert task_context.payload.Unpack(task_payload)
    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'completed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == 'label'
        assert histogram_options.story == 'story'
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == [0, 1, 2]

        assert task_payload.index == attempt
        assert task_payload.tries == 1


def test_Evaluate_Success_HistogramStat(datastore_client, populate_task_graph,
                                        isolate_retrieve):
    histograms = json.dumps([{
        'values': ['story'],
        'guid': '6352ba96-d169-439d-a1aa-b0db0ffedd5c',
        'type': 'GenericSet'
    }, {
        'values': ['group:label'],
        'guid': 'd8432a1d-1920-41b2-8591-d5f8652bbe28',
        'type': 'GenericSet'
    }, {
        'sampleValues': [0, 1, 2],
        'name': 'some_chart',
        'running': [3, 2, None, 1, 0, 3, 2],
        'binBoundaries': [1, [1, 1000.0, 20]],
        'diagnostics': {
            'storyTags': 'd8432a1d-1920-41b2-8591-d5f8652bbe28',
            'stories': '6352ba96-d169-439d-a1aa-b0db0ffedd5c'
        },
        'allBins': {
            0: [1],
            1: [1],
            3: [1]
        },
        'unit': 'count'
    }])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))
    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
        grouping_label='label',
        story='story',
        statistic='avg',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'completed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == 'label'
        assert histogram_options.story == 'story'
        assert histogram_options.statistic == 'avg'
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == [1]

        assert task_payload.index == attempt
        assert task_payload.tries == 1


def test_Evaluate_Success_HistogramStoryNeedsEscape(datastore_client,
                                                    populate_task_graph,
                                                    isolate_retrieve):
    histograms = json.dumps([{
        'values': ['https://story'],
        'guid': '1d4458ea-cda5-472d-aafb-38e330d036c3',
        'type': 'GenericSet'
    }, {
        'values': ['group:label'],
        'guid': '350e4d8a-91b6-4e1f-85a2-9f863954769e',
        'type': 'GenericSet'
    }, {
        'sampleValues': [0, 1, 2],
        'name': 'some_chart',
        'running': [3, 2, None, 1, 0, 3, 2],
        'binBoundaries': [1, [1, 1000.0, 20]],
        'diagnostics': {
            'storyTags': '350e4d8a-91b6-4e1f-85a2-9f863954769e',
            'stories': '1d4458ea-cda5-472d-aafb-38e330d036c3'
        },
        'allBins': {
            0: [1],
            1: [1],
            3: [1]
        },
        'unit': 'count'
    }])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
        grouping_label='label',
        story='https://story',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'completed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == 'label'
        assert histogram_options.story == 'https://story'
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == [0, 1, 2]

        assert task_payload.index == attempt
        assert task_payload.tries == 1


def test_Evaluate_Success_MultipleHistograms(datastore_client,
                                             populate_task_graph,
                                             isolate_retrieve):
    histograms = json.dumps([{
        'values': ['group:label'],
        'guid': '8b31cd6d-ad4b-4976-a79f-36c35515a7d4',
        'type': 'GenericSet'
    }, {
        'values': ['story'],
        'guid': '05d08804-4d8f-4a48-9cd2-803da045121c',
        'type': 'GenericSet'
    }] + [{
        'sampleValues': [0, 1, 2],
        'name': name,
        'running': [3, 2, None, 1, 0, 3, 2],
        'binBoundaries': [1, [1, 1000.0, 20]],
        'diagnostics': {
            'storyTags': '8b31cd6d-ad4b-4976-a79f-36c35515a7d4',
            'stories': '05d08804-4d8f-4a48-9cd2-803da045121c'
        },
        'allBins': {
            0: [1],
            1: [1],
            3: [1]
        },
        'unit': 'count'
    } for name in ('some_chart', 'some_chart', 'some_other_chart')])

    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
        grouping_label='label',
        story='story',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'completed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == 'label'
        assert histogram_options.story == 'story'
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == [0, 1, 2, 0, 1, 2]

        assert task_payload.index == attempt
        assert task_payload.tries == 1


def test_Evaluate_Success_HistogramsTraceUrls(datastore_client,
                                              populate_task_graph,
                                              isolate_retrieve):
    histograms = json.dumps([{
        'sampleValues': [0],
        'name': 'some_chart',
        'running': [1, 0, None, 0, 0, 0, 0],
        'binBoundaries': [1, [1, 1000.0, 20]],
        'diagnostics': {
            'traceUrls': {
                'values': ['trace_url1', 'trace_url2'],
                'type': 'GenericSet'
            }
        },
        'allBins': {
            0: [1]
        },
        'unit': 'count'
    }, {
        'binBoundaries': [1, [1, 1000.0, 20]],
        'name': 'hist3',
        'unit': 'count',
        'diagnostics': {
            'traceUrls': {
                'values': ['trace_url2'],
                'type': 'GenericSet'
            }
        }
    }, {
        'binBoundaries': [1, [1, 1000.0, 20]],
        'name': 'hist2',
        'unit': 'count',
        'diagnostics': {
            'traceUrls': {
                'values': ['trace_url3'],
                'type': 'GenericSet'
            }
        }
    }])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'completed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == ''
        assert histogram_options.story == ''
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == [0]
        for i, trace_url in zip(range(3), task_payload.output.trace_urls):
            assert trace_url.key == 'trace'
            assert trace_url.value == f'trace_url{i + 1}'
            assert trace_url.url == f'trace_url{i + 1}'

        assert task_payload.index == attempt
        assert task_payload.tries == 1


def test_Evaluate_Success_HistogramSkipRefTraceUrls(datastore_client,
                                                    populate_task_graph,
                                                    isolate_retrieve):
    histograms = json.dumps([{
        'sampleValues': [0],
        'name': 'some_chart',
        'running': [1, 0, None, 0, 0, 0, 0],
        'binBoundaries': [1, [1, 1000.0, 20]],
        'diagnostics': {
            'traceUrls': {
                'values': ['trace_url1', 'trace_url2'],
                'type': 'GenericSet'
            }
        },
        'allBins': {
            0: [1]
        },
        'unit': 'count'
    }, {
        'binBoundaries': [1, [1, 1000.0, 20]],
        'name': 'hist2',
        'unit': 'count',
        'diagnostics': {
            'traceUrls': 'foo'
        }
    }])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'completed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == ''
        assert histogram_options.story == ''
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == [0]
        for i, trace_url in zip(range(2), task_payload.output.trace_urls):
            assert trace_url.key == 'trace'
            assert trace_url.value == f'trace_url{i + 1}'
            assert trace_url.url == f'trace_url{i + 1}'

        assert task_payload.index == attempt
        assert task_payload.tries == 1


def test_Evaluate_Success_HistogramSummary(datastore_client,
                                           populate_task_graph,
                                           isolate_retrieve):
    histograms = json.dumps([{
        'values': ['group:label'],
        'guid': '3c2ef810-c8e2-4abb-8f65-f83e56cd0ad1',
        'type': 'GenericSet'
    }] + [{
        'sampleValues': [0, 1, 2],
        'name': 'some_chart',
        'running': [3, 2, None, 1, 0, 3, 2],
        'binBoundaries': [1, [1, 1000.0, 20]],
        'diagnostics': {
            'storyTags': '3c2ef810-c8e2-4abb-8f65-f83e56cd0ad1',
            'stories': {
                'values': [f'{i}_story_{j}'],
                'type': 'GenericSet'
            }
        },
        'allBins': {
            0: [1],
            1: [1],
            3: [1]
        },
        'unit': 'count'
    } for j in range(10) for i in range(2)])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'completed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == ''
        assert histogram_options.story == ''
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == [60]  # sum([0, 1, 2]) * 20

        assert task_payload.index == attempt
        assert task_payload.tries == 1


def test_Evaluate_Failure_HistogramNoSamples(datastore_client,
                                             populate_task_graph,
                                             isolate_retrieve):
    histograms = json.dumps([{
        'values': ['https://story'],
        'guid': 'd10f9682-7665-41ce-a7a1-ea730c24e8f3',
        'type': 'GenericSet'
    }, {
        'values': ['group:label'],
        'guid': '404cab67-9f15-4e0e-a2a4-2f7aed75a85b',
        'type': 'GenericSet'
    }, {
        'binBoundaries': [1, [1, 1000.0, 20]],
        'name': 'some_chart',
        'unit': 'count',
        'diagnostics': {
            'storyTags': '404cab67-9f15-4e0e-a2a4-2f7aed75a85b',
            'stories': 'd10f9682-7665-41ce-a7a1-ea730c24e8f3'
        }
    }])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
        grouping_label='label',
        story='https://story',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'failed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == 'label'
        assert histogram_options.story == 'https://story'
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == []

        assert task_payload.index == attempt
        assert task_payload.tries == 1

        assert task_payload.errors[0].reason == 'ReadValueNoValues'


def test_Evaluate_Failure_EmptyHistogramSet(datastore_client,
                                            populate_task_graph,
                                            isolate_retrieve):
    histograms = json.dumps([])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
        grouping_label='label',
        story='https://story',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'failed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == 'label'
        assert histogram_options.story == 'https://story'
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == []

        assert task_payload.index == attempt
        assert task_payload.tries == 1

        assert task_payload.errors[0].reason == 'ReadValueNotFound'


def test_Evaluate_Failure_HistogramNoValues(datastore_client,
                                            populate_task_graph,
                                            isolate_retrieve):
    histograms = json.dumps([{
        'binBoundaries': [1, [1, 1000.0, 20]],
        'name': 'some_benchmark',
        'unit': 'count'
    }])
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), histograms], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='some_chart',
        grouping_label='label',
        story='https://story',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'failed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'histogram_sets'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == 'label'
        assert histogram_options.story == 'https://story'
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'some_chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'some_chart'
        assert graph_json_options.trace == 'some_trace'

        assert task_payload.output.result_values == []

        assert task_payload.index == attempt
        assert task_payload.tries == 1

        assert task_payload.errors[0].reason == 'ReadValueNotFound'


def test_Evaluate_Success_GraphJson(datastore_client, populate_task_graph,
                                    isolate_retrieve):
    graph_json = json.dumps({
        'chart': {
            'traces': {
                'trace': ['126444.869721', '0.0']
            }
        },
    })
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), graph_json], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='chart',
        trace='trace',
        mode='graph_json',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'completed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'graph_json'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == ''
        assert histogram_options.story == ''
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'chart'
        assert graph_json_options.trace == 'trace'

        assert task_payload.output.result_values == [126444.869721]

        assert task_payload.index == attempt
        assert task_payload.tries == 1


def testEvaluateFailure_GraphJsonMissingFile(datastore_client,
                                             populate_task_graph,
                                             isolate_retrieve):
    isolate_retrieve.return_value = json.dumps({"files": {}})

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='chart',
        trace='trace',
        mode='graph_json',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'failed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'graph_json'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == ''
        assert histogram_options.story == ''
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'chart'
        assert graph_json_options.trace == 'trace'

        assert task_payload.output.result_values == []

        assert task_payload.index == attempt
        assert task_payload.tries == 1

        assert task_payload.errors[0].reason == 'ReadValueNoFile'


def test_Evaluate_Fail_GraphJsonMissingChart(datastore_client,
                                             populate_task_graph,
                                             isolate_retrieve):
    graph_json = json.dumps({})
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), graph_json], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='chart',
        trace='trace',
        mode='graph_json',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'failed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'graph_json'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == ''
        assert histogram_options.story == ''
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'chart'
        assert graph_json_options.trace == 'trace'

        assert task_payload.output.result_values == []

        assert task_payload.index == attempt
        assert task_payload.tries == 1

        assert task_payload.errors[0].reason == 'ReadValueChartNotFound'


def test_Evaluate_Fail_GraphJsonMissingTrace(datastore_client,
                                             populate_task_graph,
                                             isolate_retrieve):
    graph_json = json.dumps({
        'chart': {
            'traces': {
                'trace': ['126444.869721', '0.0']
            }
        },
    })
    isolate_retrieve.side_effect = itertools.chain(*itertools.repeat([(
        '{"files": {"some_benchmark/perf_results.json": '
        '{"h": "394890891823812873798734a"}}}'), graph_json], 10))

    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='chart',
        trace='must_not_be_found',
        mode='graph_json',
    )

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator(datastore_client, job),
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'failed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'graph_json'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == ''
        assert histogram_options.story == ''
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'chart'
        assert graph_json_options.trace == 'must_not_be_found'

        assert task_payload.output.result_values == []

        assert task_payload.index == attempt
        assert task_payload.tries == 1

        assert task_payload.errors[0].reason == 'ReadValueTraceNotFound'


def test_Evaluate_FailedDependency(datastore_client, populate_task_graph):
    job = populate_task_graph(
        benchmark='some_benchmark',
        chart='chart',
        trace='must_not_be_found',
        mode='graph_json',
    )

    evaluator = combinators.SequenceEvaluator(evaluators=(
        combinators.FilteringEvaluator(
            predicate=predicates.TaskTypeEq('find_isolate'),
            delegate=combinators.SequenceEvaluator(evaluators=(
                bisection_test_util.FakeFoundIsolate(datastore_client, job),
                combinators.TaskPayloadLiftingEvaluator()))),
        combinators.FilteringEvaluator(
            predicate=predicates.TaskTypeEq('run_test'),
            delegate=combinators.SequenceEvaluator(evaluators=(
                bisection_test_util.FakeFailedRunTest(datastore_client, job),
                combinators.TaskPayloadLiftingEvaluator()))),
        result_reader.Evaluator(datastore_client, job),
    ))

    assert evaluator_module.evaluate_graph(
        event_module.build_event(
            type='initiate',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        evaluator,
        task_module.task_graph_loader(datastore_client, job),
    ) != {}

    read_values = evaluator_module.evaluate_graph(
        event_module.build_event(
            type='select',
            target_task=None,
            payload=empty_pb2.Empty(),
        ),
        combinators.Selector(task_type='read_value'),
        task_module.task_graph_loader(datastore_client, job),
    )
    for attempt, (k, v) in zip(range(10), sorted(read_values.items())):
        assert 'read_value_chromium@7c7e90be_%s' % (attempt, ) == k
        assert v.state == 'failed'

        task_payload = result_reader_payload_pb2.ResultReaderPayload()
        assert v.payload.Unpack(task_payload)

        assert task_payload.input.benchmark == 'some_benchmark'
        assert task_payload.input.mode == 'graph_json'
        assert task_payload.input.results_filename == 'some_benchmark/perf_results.json'

        histogram_options = task_payload.input.histogram_options
        assert histogram_options.grouping_label == ''
        assert histogram_options.story == ''
        assert histogram_options.statistic == ''
        assert histogram_options.histogram_name == 'chart'

        graph_json_options = task_payload.input.graph_json_options
        assert graph_json_options.chart == 'chart'
        assert graph_json_options.trace == 'must_not_be_found'

        assert task_payload.output.result_values == []

        assert task_payload.index == attempt
        assert task_payload.tries == 1

        assert task_payload.errors[0].reason == 'DependencyFailed'
