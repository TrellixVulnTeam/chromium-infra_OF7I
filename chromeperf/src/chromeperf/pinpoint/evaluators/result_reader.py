# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import dataclasses
import itertools
import json
import logging
import math
import ntpath
import posixpath
import re

from google.cloud import datastore
from google.protobuf import any_pb2
from google.protobuf import empty_pb2

from chromeperf.engine import combinators
from chromeperf.engine import evaluator
from chromeperf.engine import predicates
from chromeperf.pinpoint import errors
from chromeperf.pinpoint import change_pb2
from chromeperf.pinpoint import result_reader_payload_pb2
from chromeperf.pinpoint import test_runner_payload_pb2
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.actions import results
from chromeperf.pinpoint.evaluators import isolate_finder
from chromeperf.pinpoint.evaluators import test_runner
from chromeperf.pinpoint.models import job as job_module
from chromeperf.pinpoint.models import task as task_module
from chromeperf.services import isolate


@dataclasses.dataclass
class HistogramOptions:
    grouping_label: str
    story: str
    statistic: str
    histogram_name: str


@dataclasses.dataclass
class GraphJsonOptions:
    chart: str
    trace: str


@dataclasses.dataclass
class TaskOptions:
    test_options: test_runner.TaskOptions
    benchmark: str
    histogram_options: HistogramOptions
    graph_json_options: GraphJsonOptions
    mode: str
    results_filename: str


@dataclasses.dataclass
class ResultReaderEvaluator(task_module.PayloadUnpackingMixin,
                            updates.ErrorAppendingMixin):
    datastore_client: datastore.Client
    job: job_module.Job

    def complete_with_error(self, task, task_payload, reason, message):
        return self.update_task_with_error(
            datastore_client=self.datastore_client,
            job=self.job,
            task=task,
            payload=task_payload,
            reason=reason,
            message=message,
        )

    def __call__(self, task, _, accumulator):
        # TODO(dberris): Validate!
        # Outline:
        #   - Retrieve the data given the options.
        #   - Parse the data from the result file.
        #   - Update the state and payload with an action.

        if task.state in {'completed', 'failed'}:
            return None
        task_payload = self.unpack(
            result_reader_payload_pb2.ResultReaderPayload, task.payload)
        task_payload.tries += 1

        dep = accumulator.get(
            task.dependencies[0],
            combinators.TaskContext(state='unknown',
                                    payload=empty_pb2.Empty()))
        test_runner_payload = self.unpack(
            test_runner_payload_pb2.TestRunnerPayload, dep.payload)
        isolate_server = test_runner_payload.output.task_output.isolate_server
        isolate_hash = test_runner_payload.output.task_output.isolate_hash

        if dep.state == 'failed':
            return self.complete_with_error(
                task, task_payload, 'DependencyFailed',
                'Task dependency "%s" ended in failed state.' %
                (task.dependencies[0], ))

        if dep.state in {'pending', 'ongoing'}:
            return None

        try:
            data = retrieve_output_json(isolate_server, isolate_hash,
                                        task_payload.input.results_filename)
            if task_payload.input.mode == 'histogram_sets':
                return self.handle_histogram_sets(task, task_payload, data)
            elif task_payload.input.mode == 'graph_json':
                return self.handle_graph_json(task, task_payload, data)
            else:
                return self.complete_with_error(
                    task, task_payload, 'UnsupportedMode',
                    ('Pinpoint only currently supports reading '
                     'HistogramSets and GraphJSON formatted files.'))
        except (errors.FatalError, errors.InformationalError,
                errors.RecoverableError) as e:
            return self.complete_with_error(task, task_payload,
                                            type(e).__name__, repr(e))

    def handle_histogram_sets(self, task, task_payload, histogram_dicts):
        histogram_options = task_payload.input.histogram_options
        grouping_label = histogram_options.grouping_label
        histogram_name = histogram_options.histogram_name
        story = histogram_options.story
        statistic = histogram_options.statistic

        histograms = HistogramSet()
        histograms.import_dicts(histogram_dicts)
        histograms_by_path = create_histogram_by_test_path(histograms)
        histograms_by_path_optional_grouping_label = (
            create_histogram_by_test_path(histograms,
                                          ignore_grouping_label=True))
        trace_urls = _find_trace_urls(histograms)
        test_paths_to_match = set([
            _compute_test_path_from_components(histogram_name,
                                               grouping_label=grouping_label,
                                               story_name=story),
            _compute_test_path_from_components(histogram_name,
                                               grouping_label=grouping_label,
                                               story_name=story,
                                               needs_escape=False)
        ])
        logging.debug('Test paths to match: %s', test_paths_to_match)
        try:
            result_values = _extract_values_from_histograms(
                test_paths_to_match, histograms_by_path, histogram_name,
                grouping_label, story, statistic)
        except errors.ReadValueNotFound:
            result_values = _extract_values_from_histograms(
                test_paths_to_match,
                histograms_by_path_optional_grouping_label, histogram_name,
                None, story, statistic)
        logging.debug('Results: %s', result_values)
        task_payload.output.result_values.extend(result_values)
        task_payload.output.trace_urls.extend(
            result_reader_payload_pb2.ResultReaderPayload.Output.TraceUrl(
                key='trace',
                value=url['name'],
                url=url['url'],
            ) for url in trace_urls or [])

        task.payload.Pack(task_payload)
        return [
            results.CompleteResultReaderAction(
                datastore_client=self.datastore_client,
                job=self.job,
                task=task,
                state='completed',
            )
        ]

    def handle_graph_json(self, task, task_payload, data):
        chart = task_payload.input.graph_json_options.chart
        trace = task_payload.input.graph_json_options.trace
        task_payload.output.ClearField('result_values')
        if chart and trace:
            if chart not in data:
                raise errors.ReadValueChartNotFound(chart)
            if trace not in data[chart]['traces']:
                raise errors.ReadValueTraceNotFound(trace)
            task_payload.output.result_values.append(
                float(data[chart]['traces'][trace][0]))
        task.payload.Pack(task_payload)
        return [
            results.CompleteResultReaderAction(
                datastore_client=self.datastore_client,
                job=self.job,
                task=task,
                state='completed',
            )
        ]


class Evaluator(combinators.FilteringEvaluator):
    def __init__(self, datastore_client, job):
        super(Evaluator, self).__init__(
            predicate=predicates.All(predicates.TaskTypeEq('read_value'),
                                     predicates.TaskStateIn({'pending'})),
            delegate=combinators.SequenceEvaluator(evaluators=(
                combinators.TaskPayloadLiftingEvaluator(),
                ResultReaderEvaluator(datastore_client, job),
            )))


def ResultSerializer(task, _, accumulator):
    results = accumulator.setdefault(task.id, {})
    results.update({
        'completed':
        task.status in {'completed', 'failed', 'cancelled'},
        'exception':
        ','.join(e.get('reason') for e in task.payload.get('errors', []))
        or None,
        'details': []
    })

    trace_urls = task.payload.get('trace_urls')
    if trace_urls:
        results['details'].extend(trace_urls)


class Serializer(combinators.FilteringEvaluator):
    def __init__(self):
        super(Serializer, self).__init__(
            predicate=predicates.All(
                predicates.TaskTypeEq('read_value'),
                predicates.TaskStateIn(
                    {'ongoing', 'failed', 'completed', 'cancelled'}),
            ),
            delegate=ResultSerializer,
        )


def create_graph(options: TaskOptions):
    subgraph = test_runner.create_graph(options.test_options)
    path = None
    if _is_windows({'dimensions': options.test_options.dimensions}):
        path = ntpath.join(options.benchmark, options.results_filename)
    else:
        path = posixpath.join(options.benchmark, options.results_filename)

    # We create a 1:1 mapping between a result_reader task and a run_test task.
    def generate_vertex_and_dep(attempts):
        for attempt in range(attempts):
            change_id = isolate_finder.change_id(
                options.test_options.build_options.change)
            result_reader_id = f'read_value_{change_id}_{attempt}'
            run_test_id = test_runner.task_id(change_id, attempt)
            result_reader_payload = result_reader_payload_pb2.ResultReaderPayload(
                input=result_reader_payload_pb2.ResultReaderPayload.Input(
                    benchmark=options.benchmark,
                    mode=options.mode,
                    results_filename=path,
                    histogram_options=result_reader_payload_pb2.
                    ResultReaderPayload.Input.HistogramOptions(
                        **dataclasses.asdict(options.histogram_options)),
                    graph_json_options=result_reader_payload_pb2.
                    ResultReaderPayload.Input.GraphJsonOptions(
                        **dataclasses.asdict(options.graph_json_options)),
                    change=change_pb2.Change(
                        commits=[
                            change_pb2.Commit(
                                repository=c.repository.name,
                                git_hash=c.git_hash,
                            ) for c in
                            options.test_options.build_options.change.commits
                        ],
                        patch=(change_pb2.GerritPatch(
                            server=options.test_options.build_options.change.
                            patch.server,
                            change=options.test_options.build_options.change.
                            patch.change,
                            revision=options.test_options.build_options.change.
                            patch.revision) if
                               options.test_options.build_options.change.patch
                               else None),
                    ),
                ),
                index=attempt,
            )
            encoded_payload = any_pb2.Any()
            encoded_payload.Pack(result_reader_payload)

            yield evaluator.TaskVertex(
                id=result_reader_id,
                vertex_type='read_value',
                payload=encoded_payload,
            ), evaluator.Dependency(from_=result_reader_id, to=run_test_id)

    for vertex, edge in generate_vertex_and_dep(options.test_options.attempts):
        subgraph.vertices.append(vertex)
        subgraph.edges.append(edge)

    return subgraph


def retrieve_isolate_output(isolate_server, isolate_hash, filename):
    logging.debug(f'Retrieving output'
                  '({isolate_server}, {isolate_hash}, {filename})')

    retrieve_result = isolate.retrieve(isolate_server, isolate_hash)
    response = json.loads(retrieve_result)
    output_files = response.get('files', {})

    if filename not in output_files:
        if 'performance_browser_tests' not in filename:
            raise errors.ReadValueNoFile(filename)

        # TODO(simonhatch): Remove this once crbug.com/947501 is resolved.
        filename = filename.replace('performance_browser_tests',
                                    'browser_tests')
        if filename not in output_files:
            raise errors.ReadValueNoFile(filename)

    output_isolate_hash = output_files[filename]['h']
    logging.debug('Retrieving %s', output_isolate_hash)

    return isolate.retrieve(isolate_server, output_isolate_hash)


def retrieve_output_json(isolate_server, isolate_hash, filename):
    isolate_output = retrieve_isolate_output(
        isolate_server,
        isolate_hash,
        filename,
    )

    # TODO(dberris): Use incremental json parsing through a file interface, to
    # avoid having to load the whole string contents in memory. See
    # https://crbug.com/998517 for more context.
    return json.loads(isolate_output)


def create_histogram_by_test_path(histograms, ignore_grouping_label=False):
    histograms_by_path = collections.defaultdict(list)
    for h in histograms:
        histograms_by_path[_compute_test_path(
            h,
            ignore_grouping_label,
        )].append(h)
    return histograms_by_path


def _is_windows(arguments):
    dimensions = arguments.get('dimensions', ())
    if isinstance(dimensions, str):
        dimensions = json.loads(dimensions)
    for dimension in dimensions:
        if dimension['key'] == 'os' and dimension['value'].startswith('Win'):
            return True
    return False


def _find_trace_urls(histograms):
    # Get and cache any trace URLs.
    unique_trace_urls = set()
    for hist in histograms:
        trace_urls = hist.diagnostics.get(RESERVED_INFO_TRACE_URLS.name)
        if trace_urls:
            unique_trace_urls.update(trace_urls)

    sorted_urls = sorted(unique_trace_urls)

    # TODO(crbug.com/1123554) All these names read 'trace.html'. We should
    # show the name of the story.
    return [{'name': t.split('/')[-1], 'url': t} for t in sorted_urls]


def _extract_values_from_histograms(test_paths_to_match, histograms_by_path,
                                    histogram_name, grouping_label, story,
                                    statistic):
    result_values = []
    matching_histograms = list(
        itertools.chain.from_iterable(
            histograms_by_path.get(histogram)
            for histogram in test_paths_to_match
            if histogram in histograms_by_path))
    logging.debug('Histograms in results: %s', histograms_by_path.keys())
    if matching_histograms:
        logging.debug('Found %s matching histograms: %s',
                      len(matching_histograms),
                      [h.name for h in matching_histograms])
        for h in matching_histograms:
            result_values.extend(_get_values_or_statistic(statistic, h))
    elif histogram_name:
        # Histograms don't exist, which means this is summary
        summary_value = []
        for test_path, histograms_for_test_path in histograms_by_path.items():
            for test_path_to_match in test_paths_to_match:
                if test_path.startswith(test_path_to_match):
                    for h in histograms_for_test_path:
                        summary_value.extend(
                            _get_values_or_statistic(statistic, h))
                        matching_histograms.append(h)

        logging.debug('Found %s matching summary histograms',
                      len(matching_histograms))
        if summary_value:
            result_values.append(sum(summary_value))

        logging.debug('result values: %s', result_values)

    if not result_values and histogram_name:
        if matching_histograms:
            raise errors.ReadValueNoValues()
        else:
            conditions = {'histogram': histogram_name}
            if grouping_label:
                conditions['grouping_label'] = grouping_label
            if story:
                conditions['story'] = story
            reason = ', '.join(list(':'.join(i) for i in conditions.items()))
            raise errors.ReadValueNotFound(reason)
    return result_values


def _get_values_or_statistic(statistic, hist):
    if not statistic:
        return hist.sample_values

    if not hist.sample_values:
        return []

    if statistic == 'avg':
        return [hist.running.mean]
    elif statistic == 'min':
        return [hist.running.min]
    elif statistic == 'max':
        return [hist.running.max]
    elif statistic == 'sum':
        return [hist.running.sum]
    elif statistic == 'std':
        return [hist.running.stddev]
    elif statistic == 'count':
        return [hist.running.count]
    raise errors.ReadValueUnknownStat(statistic)


class _ReservedInfo(object):
    def __init__(self, name, _type=None, entry_type=None):
        self._name = name
        self._type = _type
        if entry_type is not None and self._type != 'GenericSet':
            raise ValueError(
                'entry_type should only be specified if _type is GenericSet')
        self._entry_type = entry_type

    @property
    def name(self):
        return self._name

    @property
    def type(self):
        return self._type

    @property
    def entry_type(self):
        return self._entry_type


RESERVED_INFO_STORIES = _ReservedInfo('stories', 'GenericSet', str)
RESERVED_INFO_STORY_TAGS = _ReservedInfo('storyTags', 'GenericSet', str)
RESERVED_INFO_SUMMARY_KEYS = _ReservedInfo('summaryKeys', 'GenericSet', str)
RESERVED_INFO_TRACE_URLS = _ReservedInfo('traceUrls', 'GenericSet', str)

# This should be equal to sys.float_info.max, but that value might differ
# between platforms, whereas ECMA Script specifies this value for all platforms.
# The specific value should not matter in normal practice.
JS_MAX_VALUE = 1.7976931348623157e+308
UNIT_NAMES = [
    'ms',
    'msBestFitFormat',
    'tsMs',
    'n%',
    'sizeInBytes',
    'bytesPerSecond',
    'J',  # Joule
    'W',  # Watt
    'A',  # Ampere
    'Ah',  # Ampere-hours
    'V',  # Volt
    'Hz',  # Hertz
    'unitless',
    'count',
    'sigma',
]


class Histogram(object):
    def __init__(self, name, unit, bin_boundaries=None):
        assert unit in UNIT_NAMES, f'Unrecognized unit {unit}'
        self._name = name
        self._diagnostics = dict()
        self._running = None
        self._sample_values = []
        self._unit = unit

    @property
    def unit(self):
        return self._unit

    @property
    def running(self):
        return self._running

    @property
    def sample_values(self):
        return self._sample_values

    @property
    def name(self):
        return self._name

    @property
    def diagnostics(self):
        return self._diagnostics

    @staticmethod
    def from_dict(dct, shared_diagnostics):
        hist = Histogram(dct['name'], dct['unit'])
        if 'diagnostics' in dct:
            for name, value in dct['diagnostics'].items():
                if name == 'tagmap':
                    continue
                if isinstance(value, str):
                    hist._diagnostics[name] = shared_diagnostics.get(value)
                else:
                    hist._diagnostics[name] = value.get('values')
        if 'running' in dct:
            hist._running = RunningStatistics.from_dict(dct['running'])
        if 'sampleValues' in dct:
            hist._sample_values = dct['sampleValues']
        return hist


class HistogramSet(object):
    def __init__(self, histograms=()):
        self._histograms = set()
        self._shared_diagnostics_by_guid = {}
        for hist in histograms:
            self.add_histogram(hist)

    def add_histogram(self, hist, diagnostics=None):
        if diagnostics:
            for name, diag in diagnostics.items():
                hist.diagnostics[name] = diag

        self._histograms.add(hist)

    def __len__(self):
        return len(self._histograms)

    def __iter__(self):
        for hist in self._histograms:
            yield hist

    def import_dicts(self, dicts):
        self._shared_diagnostics = {
            d.get('guid'): d.get('values')
            for d in dicts if d.get('type') == 'GenericSet'
        }
        for d in dicts:
            if 'type' not in d:
                hist = Histogram.from_dict(d, self._shared_diagnostics)
                self.add_histogram(hist)


class RunningStatistics(object):
    __slots__ = ('_count', '_mean', '_max', '_min', '_sum', '_variance',
                 '_meanlogs')

    def __init__(self):
        self._count = 0
        self._mean = 0.0
        self._max = -JS_MAX_VALUE
        self._min = JS_MAX_VALUE
        self._sum = 0.0
        self._variance = 0.0
        # Mean of logarithms of samples, or None if any samples were <= 0.
        self._meanlogs = 0.0

    @property
    def count(self):
        return self._count

    @property
    def geometric_mean(self):
        if self._meanlogs is None:
            return None
        return math.exp(self._meanlogs)

    @property
    def mean(self):
        if self._count == 0:
            return None
        return self._mean

    @property
    def max(self):
        return self._max

    @property
    def min(self):
        return self._min

    @property
    def sum(self):
        return self._sum

    # This returns the variance of the samples after Bessel's correction has
    # been applied.
    @property
    def variance(self):
        if self.count == 0 or self._variance is None:
            return None
        if self.count == 1:
            return 0
        return self._variance / (self.count - 1)

    # This returns the standard deviation of the samples after Bessel's
    # correction has been applied.
    @property
    def stddev(self):
        if self.count == 0 or self._variance is None:
            return None
        return math.sqrt(self.variance)

    @staticmethod
    def from_dict(dct):
        result = RunningStatistics()
        if len(dct) != 7:
            return result

        def AsFloatOrNone(x):
            if x is None:
                return x
            return float(x)

        [
            result._count,
            result._max,
            result._meanlogs,
            result._mean,
            result._min,
            result._sum,
            result._variance,
        ] = [int(dct[0])] + [AsFloatOrNone(x) for x in dct[1:]]
        return result


def _escape_name(name):
    """Escapes a trace name so it can be stored in a row.

  Args:
    name: A string representing a name.

  Returns:
    An escaped version of the name.
  """
    return re.sub(r'[\:|=/#&,]', '_', name)


def _compute_test_path(hist, ignore_grouping_label=False):
    # If a Histogram represents a summary across multiple stories, then its
    # 'stories' diagnostic will contain the names of all of the stories.
    # If a Histogram is not a summary, then its 'stories' diagnostic will
    # contain the singular name of its story.
    is_summary = list(
        hist.diagnostics.get(
            RESERVED_INFO_SUMMARY_KEYS.name,
            [],
        ))

    grouping_label = _get_grouping_label_from_histogram(
        hist) if not ignore_grouping_label else None

    story_name = hist.diagnostics.get(RESERVED_INFO_STORIES.name)
    if story_name and len(story_name) == 1:
        story_name = story_name[0]
    else:
        story_name = None
    return _compute_test_path_from_components(
        hist.name,
        grouping_label=grouping_label,
        story_name=story_name,
        is_summary=is_summary,
    )


def _compute_test_path_from_components(
    hist_name,
    grouping_label=None,
    story_name=None,
    is_summary=None,
    needs_escape=True,
):
    path = hist_name or ''

    if grouping_label and (not is_summary
                           or RESERVED_INFO_STORY_TAGS.name in is_summary):
        path += '/' + grouping_label

    if story_name and not is_summary:
        if needs_escape:
            escaped_story_name = _escape_name(story_name)
            path += '/' + escaped_story_name
        else:
            path += '/' + story_name

    return path


def _get_grouping_label_from_histogram(hist):
    tags = hist.diagnostics.get(RESERVED_INFO_STORY_TAGS.name) or []
    tags_to_use = [t.split(':') for t in tags if ':' in t]
    return '_'.join(v for _, v in sorted(tags_to_use))
