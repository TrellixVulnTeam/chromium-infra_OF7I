# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""The generic Pinpoint bisection workflow."""

import collections
import dataclasses
import itertools
import logging
import math
from typing import Iterable, List, Optional, Tuple

import scipy.stats
from google.cloud import datastore
from google.protobuf import any_pb2

from chromeperf.engine import combinators
from chromeperf.engine import evaluator
from chromeperf.engine import predicates
from chromeperf.engine import task_pb2
from chromeperf.pinpoint import find_culprit_task_payload_pb2
from chromeperf.pinpoint import result_reader_payload_pb2
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.evaluators import isolate_finder  # builder, target, bucket, change
from chromeperf.pinpoint.evaluators import result_reader  # test_options, benchmark, histogram_options, graph_json_options, mode
from chromeperf.pinpoint.evaluators import test_runner  # build_options, swarming_server, dimensions, extra_args, attempts
from chromeperf.pinpoint.models import exploration
from chromeperf.pinpoint.models import job as job_module
from chromeperf.pinpoint.models import task as task_module
from chromeperf.pinpoint.models import change as change_module
from chromeperf.pinpoint.models import commit as commit_module
from chromeperf.pinpoint.models import repository as repository_module
from chromeperf.pinpoint.models.compare import compare
from chromeperf.services import gitiles_service


_ERROR_BUG_ID = 'Bug ID must be an integer.'
_ERROR_PRIORITY = 'Priority must be an integer.'

_DEFAULT_SPECULATION_LEVELS = 2

# Alias a very long type name.
_PayloadOutput = find_culprit_task_payload_pb2.FindCulpritTaskPayload.Output


@dataclasses.dataclass
class AnalysisOptions:
    comparison_magnitude: float
    min_attempts: int
    max_attempts: int

    def to_proto(self) -> find_culprit_task_payload_pb2.AnalysisOptions:
        return find_culprit_task_payload_pb2.AnalysisOptions(
            comparison_magnitude=self.comparison_magnitude,
            min_attempts=self.min_attempts,
            max_attempts=self.max_attempts)

    @classmethod
    def from_proto(cls, proto: find_culprit_task_payload_pb2.AnalysisOptions):
        return AnalysisOptions(
            comparison_magnitude=proto.comparison_magnitude,
            min_attempts=proto.min_attempts,
            max_attempts=proto.max_attempts)


@dataclasses.dataclass
class TaskOptions:
    build_options: isolate_finder.TaskOptions
    test_options: test_runner.TaskOptions
    read_options: result_reader.TaskOptions
    analysis_options: AnalysisOptions
    start_change: change_module.Change
    end_change: change_module.Change
    # TODO: remove pinned_patch?
    # This is slightly odd to have in the options, as it is already baked into
    # the start_change/end_change.
    pinned_patch: Optional[change_module.GerritPatch] = None

    @classmethod
    def from_proto(
            cls,
            datastore_client,
            proto: find_culprit_task_payload_pb2.FindCulpritTaskPayload.Input):
        return TaskOptions(
            build_options=isolate_finder.TaskOptions.from_proto(
                    datastore_client, proto.build_option_template),
            test_options=test_runner.TaskOptions.from_proto(
                    proto.test_option_template),
            read_options=result_reader.TaskOptions.from_proto(
                    proto.read_option_template),
            analysis_options=AnalysisOptions.from_proto(proto.analysis_options),
            start_change=change_module.Change.FromProto(
                    datastore_client, proto.start_change),
            end_change=change_module.Change.FromProto(
                    datastore_client, proto.end_change),
            pinned_patch=None,   # TODO (maybe delete this options field?)
        )

    def make_read_options_for_change(self, change):
        build_options = dataclasses.replace(self.build_options, change=change)
        test_options = dataclasses.replace(
            self.test_options, build_options=build_options,
            attempts=self.analysis_options.min_attempts)
        read_options = dataclasses.replace(
            self.read_options, test_options=test_options)
        return read_options



def create_graph(options: TaskOptions): # -> evaluator.TaskGraph
    ##  start_change = options.start_change
    ##  end_change = options.end_change
    ##  if options.pinned_change:
    ##    start_change.Update(options.pinned_change)
    ##    end_change.Update(options.pinned_change)
    ##

    # Given the start_change and end_change, we create two subgraphs that we
    # depend on from the 'find_culprit' task. This means we'll need to create
    # independent test options and build options from the template provided by
    # the caller.
    start_subgraph = result_reader.create_graph(
            options.make_read_options_for_change(options.start_change))
    end_subgraph = result_reader.create_graph(
            options.make_read_options_for_change(options.end_change))

    # Then we add a dependency from the 'FindCulprit' task with the payload
    # describing the options set for the performance bisection.
    payload = find_culprit_task_payload_pb2.FindCulpritTaskPayload()
    payload.input.start_change.CopyFrom(options.start_change.to_proto())
    payload.input.end_change.CopyFrom(options.end_change.to_proto())
    payload.input.analysis_options.CopyFrom(options.analysis_options.to_proto())
    payload.input.build_option_template.CopyFrom(options.build_options.to_proto())
    payload.input.test_option_template.CopyFrom(options.test_options.to_proto())
    payload.input.read_option_template.CopyFrom(
            options.read_options.to_input_proto())

    ## payload.find_isolate_payload.SetInParent()
    ## isolate_payload = payload.find_isolate_payload
    ## if options.pinned_patch:
    ##     isolate_payload.patch.CopyFrom(options.pinned_patch.to_proto())
    ## isolate_payload.builder = options.build_options.builder
    ## isolate_payload.target = options.build_options.target
    ## isolate_payload.bucket = options.build_options.bucket
    encoded_payload = any_pb2.Any()
    encoded_payload.Pack(payload)

    find_culprit_task = evaluator.TaskVertex(
         id='performance_bisection',
         vertex_type='find_culprit',
         payload=encoded_payload)

    subgraph_vertices = (
            list(start_subgraph.vertices) + list(end_subgraph.vertices))
    return evaluator.TaskGraph(
            vertices=subgraph_vertices + [find_culprit_task],
            edges=list(start_subgraph.edges) + list(end_subgraph.edges) + [
                    evaluator.Dependency(from_=find_culprit_task.id, to=v.id)
                    for v in subgraph_vertices
                    if v.vertex_type == 'read_value'])


class InputValidationError(Exception):
    pass


def convert_params(params: dict, datastore_client: datastore.Client
                   ) -> TaskOptions:
    """Convert a params dict (a JSON-ish struct) to TaskOptions.

    Raises InputValidationError.
    """
    _validate_required_params(params)

    # Apply per-configuation defaults.
    if params.get('configuration'):
        # Was called _ArgumentsWithConfiguration in past
        params = _update_params_with_configuration_defaults(params, client)
        logging.debug('Updated Params: %s', params)

    # Process params that require some validation or transformation prior to use
    # in TaskOptions.
    # All other params (like 'target') are simply used as-is.
    priority = _extract_priority(params)
    bug_id, bug_project = _extract_bug_id(params)
    comparison_magnitude = _extract_comparison_magnitude(params)
    pinned_patch = _extract_patch(params)
    repository = _extract_repository(params, datastore_client)
    start_change, end_change = _extract_changes(
            params, datastore_client, repository, pinned_patch)
    bucket = params.get('bucket', 'master.tryserver.chromium.perf')

    min_attempts = 10

    task_options = TaskOptions(
            build_options=isolate_finder.TaskOptions(
                    builder=params.get('builder'),
                    target=params['target'],
                    bucket=bucket,
                    change=None,
            ),
            test_options=test_runner.TaskOptions(
                    swarming_server=params.get('swarming_server'),
                    dimensions=params.get('dimensions', {}),
                    extra_args=params.get('extra_test_args'),
                    attempts=min_attempts,
                    build_options=None,
            ),
            read_options=result_reader.TaskOptions(
                    benchmark=params['benchmark'],
                    histogram_options=result_reader.HistogramOptions(
                        grouping_label=params.get('grouping_label'),
                        story=params.get('story'),
                        statistic=params.get('statistic'),
                        histogram_name=params.get('chart'),
                    ),
                    graph_json_options=result_reader.GraphJsonOptions(
                            chart=params.get('chart'),
                            trace=params.get('trace')),
                    mode=('histogram_sets'
                          if params['target'] in EXPERIMENTAL_TARGET_SUPPORT
                          else 'graph_json'),
                    results_filename='perf_results.json',
                    test_options=None,
            ),
            analysis_options=AnalysisOptions(
                    comparison_magnitude=comparison_magnitude,
                    min_attempts=min_attempts,
                    max_attempts=60,
            ),
            start_change=start_change,
            end_change=end_change,
            pinned_patch=pinned_patch,
    )
    return task_options



# TODO: update to not rely on ndb objects.
def _update_params_with_configuration_defaults(datastore_client, original_arguments):
#def _ArgumentsWithConfiguration(datastore_client, original_arguments):
  # "configuration" is a special argument that maps to a list of preset
  # arguments. Pull any arguments from the specified "configuration", if any.
  new_arguments = original_arguments.copy()

  configuration = original_arguments.get('configuration')
  if configuration:
    try:
      # TODO: bot_configurations needs to come from somewhere.
      default_arguments = bot_configurations.Get(configuration)
    except ValueError:
      # Reraise with a clearer message.
      raise ValueError("Bot Config: %s doesn't exist." % configuration)
    logging.info('Bot Config: %s', default_arguments)

    if default_arguments:
      for k, v in list(default_arguments.items()):
        # We special-case the extra_test_args argument to be additive, so that
        # we can respect the value set in bot_configurations in addition to
        # those provided from the UI.
        if k == 'extra_test_args':
          # First, parse whatever is already there. We'll canonicalise the
          # inputs as a JSON list of strings.
          provided_args = new_arguments.get('extra_test_args', '')
          extra_test_args = []
          if provided_args:
            try:
              extra_test_args = json.loads(provided_args)
            except ValueError:
              extra_test_args = shlex.split(provided_args)

          try:
            configured_args = json.loads(v)
          except ValueError:
            configured_args = shlex.split(v)

          new_arguments['extra_test_args'] = json.dumps(extra_test_args +
                                                        configured_args)
        else:
          new_arguments.setdefault(k, v)

  return new_arguments


# Functions to extract (and validate) parameters

def _extract_bug_id(params) -> Tuple[Optional[int], str]:
    bug_id = params.get('bug_id')
    project = params.get('project', 'chromium')
    if not bug_id:
        return None, project

    try:
        # TODO(dberris): Figure out a way to check the issue tracker if the project
        # is valid at creation time. That might involve a user credential check, so
        # we might need to update the scopes we're asking for. For now trust that
        # the inputs are valid.
        return int(bug_id), project
    except ValueError:
        raise InputValidationError(_ERROR_BUG_ID)



def _extract_priority(params) -> Optional[int]:
    priority = params.get('priority')
    if not priority:
        return None

    try:
        return int(priority)
    except ValueError:
        raise InputValidationError(_ERROR_PRIORITY)


def _extract_repository(params, datastore_client) -> repository_module.Repository:
    """Returns short name of repository extracted from 'repository' param.

    The 'repository' param may be a short name or a repository URL.
    """
    repository = params['repository']

    if repository.startswith('https://'):
        return repository_module.Repository.FromUrl(datastore_client,
                                                    repository)

    try:
        return repository_module.Repository.FromName(datastore_client,
                                                     repository)
    except KeyError as e:
        raise InputValidationError(str(e))


def _extract_changes(params, datastore_client,
                     repository: repository_module.Repository,
                     patch: Optional[change_module.GerritPatch]
                     ) -> Tuple[change_module.Change, change_module.Change]:

    commit_1 = commit_module.Commit.MakeValidated(
            datastore_client, repository, params['start_git_hash'])
    commit_2 = commit_module.Commit.MakeValidated(
            datastore_client, repository, params['end_git_hash'])

    # If we find a patch in the request, this means we want to apply it even to
    # the start commit.
    change_1 = change_module.Change(commits=(commit_1,), patch=patch)
    change_2 = change_module.Change(commits=(commit_2,), patch=patch)

    return change_1, change_2


def _extract_patch(params) -> Optional[change_module.GerritPatch]:
    patch_data = params.get('patch')
    if patch_data:
        return change_module.GerritPatch.FromData(patch_data)
    return None


def _extract_comparison_magnitude(params) -> float:
    comparison_magnitude = params.get('comparison_magnitude')
    if not comparison_magnitude:
        return 1.0
    return float(comparison_magnitude)



_REQUIRED_NON_EMPTY_PARAMS = {'target', 'benchmark', 'repository',
                              'start_git_hash', 'end_git_hash'}


def _validate_required_params(params) -> None:
    missing = _REQUIRED_NON_EMPTY_PARAMS - set(params.keys())
    if missing:
        raise InputValidationError(
                f'Missing required parameters: {list(missing)}')
    # Check that they're not empty.
    empty_keys = [key for key in _REQUIRED_NON_EMPTY_PARAMS if not params[key]]
    if empty_keys:
        raise InputValidationError(
                f'Parameters must not be empty: {empty_keys}')


# TODO(crbug.com/1203798): Add fallback logic like in crrev.com/c/2951291 once
# work on the new execution engine resumes.
EXPERIMENTAL_TELEMETRY_BENCHMARKS = {
    'performance_webview_test_suite',
    'telemetry_perf_webview_tests',
}
SUFFIXED_EXPERIMENTAL_TELEMETRY_BENCHMARKS = {
    'performance_test_suite',
    'telemetry_perf_tests',
}
SUFFIXES = {
    '',
    '_android_chrome',
    '_android_monochrome',
    '_android_monochrome_bundle',
    '_android_weblayer',
    '_android_webview',
    '_android_clank_chrome',
    '_android_clank_monochrome',
    '_android_clank_monochrome_64_32_bundle',
    '_android_clank_monochrome_bundle',
    '_android_clank_trichrome_bundle',
    '_android_clank_trichrome_webview',
    '_android_clank_trichrome_webview_bundle',
    '_android_clank_webview',
    '_android_clank_webview_bundle',
}
for test in SUFFIXED_EXPERIMENTAL_TELEMETRY_BENCHMARKS:
  for suffix in SUFFIXES:
    EXPERIMENTAL_TELEMETRY_BENCHMARKS.add(test + suffix)

EXPERIMENTAL_VR_BENCHMARKS = {'vr_perf_tests'}
EXPERIMENTAL_TARGET_SUPPORT = (
    EXPERIMENTAL_TELEMETRY_BENCHMARKS | EXPERIMENTAL_VR_BENCHMARKS)


@dataclasses.dataclass
class PrepareCommitsAction(task_module.PayloadUnpackingMixin,
                           updates.ErrorAppendingMixin):
    """Populates payload's state.changes by querying gitiles.

    This takes the start_change/end_change from the payload, and uses gitiles to
    expand that out into individual commits.
    """
    datastore_client: datastore.Client
    job: job_module.Job
    task: task_pb2.Task

    @updates.log_transition_failures
    def __call__(self, context):
        del context
        task_payload = self.unpack(
                find_culprit_task_payload_pb2.FindCulpritTaskPayload,
                self.task.payload)

        try:
            # We're storing this once, so that we don't need to always get this
            # when working with the individual commits. This reduces our
            # reliance on datastore operations throughout the course of handling
            # the culprit finding process.
            #
            # TODO(dberris): Expand the commits into the full table of
            # dependencies?  Because every commit in the chromium repository is
            # likely to be building against different versions of the
            # dependencies (v8, skia, etc.) we'd need to expand the concept of a
            # changelist (CL, or Change in the Pinpoint codebase) so that we
            # know which versions of the dependencies to use in specific CLs.
            # Once we have this, we might be able to operate cleanly on just
            # Change instances instead of just raw commits.
            #
            # TODO(dberris): Model the "merge-commit" like nature of auto-roll
            # CLs by allowing the preparation action to model the non-linearity
            # of the history. This means we'll need a concept of levels, where
            # changes in a single repository history (the main one) operates at
            # a higher level linearly, and if we're descending into rolls that
            # we're exploring a lower level in the linear history. This is
            # similar to the following diagram:
            #
            #     main -> m0 -> m1 -> m2 -> roll0 -> m3 -> ...
            #                                                            |
            #     dependency ..............    +-> d0 -> d1
            #
            # Ideally we'll already have this expanded before we go ahead and
            # perform a bisection, to amortise the cost of making requests to
            # back-end services for this kind of information in tight loops.
            start_change = change_module.Change.FromProto(
                    self.datastore_client, task_payload.input.start_change)
            end_change = change_module.Change.FromProto(
                    self.datastore_client, task_payload.input.end_change)
            gitiles_commits = commit_module.commit_range(
                    start_change.base_commit, end_change.base_commit)
            task_payload.state.changes.append(task_payload.input.start_change)
            # change (w/ pinned commit), not commit here:
            for commit in reversed(gitiles_commits):
                task_payload.state.changes.extend(
                    [dataclasses.replace(
                        start_change,
                        commits=[commit_module.Commit(
                            repository=start_change.base_commit.repository,
                            git_hash=commit['commit'])],
                    ).to_proto()])
        except gitiles_service.NotFoundError as e:
            self.update_task_with_error(
                datastore_client=self.datastore_client,
                job=self.job,
                task=self.task,
                payload=task_payload,
                reason='GitilesFetchError',
                message=e.message)

        encoded_payload = any_pb2.Any()
        encoded_payload.Pack(task_payload)
        updates.update_task(
            self.datastore_client,
            self.job,
            self.task.id,
            new_state='ongoing',
            payload=encoded_payload,
        )


@dataclasses.dataclass
class RefineExplorationAction(task_module.PayloadUnpackingMixin,
                              updates.ErrorAppendingMixin):

    datastore_client: datastore.Client
    job: job_module.Job
    task: task_pb2.Task
    change: change_module.Change
    new_size: int

    @updates.log_transition_failures
    def __call__(self, context):
        task_payload = self.unpack(
                find_culprit_task_payload_pb2.FindCulpritTaskPayload,
                self.task.payload)
        task_options = TaskOptions.from_proto(
                self.datastore_client, task_payload.input)
        # Outline:
        #     - Given the job and task, extend the TaskGraph to add new tasks and
        #         dependencies, being careful to filter the IDs from what we
        #         already see in the accumulator to avoid graph amendment
        #         errors.
        #     - If we do encounter graph amendment errors, we should log those
        #         and not block progress because that can only happen if there's
        #         concurrent updates being performed with the same actions.

        analysis_options = task_options.analysis_options
        if self.new_size:
            max_attempts = (analysis_options.max_attempts
                            if analysis_options.max_attempts else 100)
            analysis_options.min_attempts = min(self.new_size, max_attempts)

        logging.debug(f'making subgraph for change {self.change.id_string} '
                      f'x {analysis_options.min_attempts} attempts')
        new_subgraph = result_reader.create_graph(
                task_options.make_read_options_for_change(self.change))
        try:
            # Add all of the new vertices we do not have in the graph yet.
            additional_vertices = [
                    v for v in new_subgraph.vertices if v.id not in context
            ]

            # All all of the new edges that aren't in the graph yet, and the
            # dependencies from the find_culprit task to the new read_value tasks if
            # there are any.
            additional_dependencies = [
                    new_edge for new_edge in new_subgraph.edges
                    if new_edge.from_ not in context
            ] + [
                    evaluator.Dependency(from_=self.task.id, to=v.id)
                    for v in new_subgraph.vertices
                    if v.id not in context and v.vertex_type == 'read_value'
            ]

            logging.debug(
                    'Extending the graph with %s new vertices and %s new edges.',
                    len(additional_vertices), len(additional_dependencies))
            updates.extend_task_graph(
                    self.datastore_client,
                    self.job,
                    vertices=additional_vertices,
                    dependencies=additional_dependencies)
        except updates.InvalidAmendment as e:
            logging.error('Failed to amend graph: %s', e)


@dataclasses.dataclass
class CompleteExplorationAction:
    """Sets task's state to 'complete'."""
    datastore_client: datastore.Client
    job: job_module.Job
    task: task_pb2.Task
    payload: any_pb2.Any

    def __call__(self, context):
        del context
        updates.update_task(
            self.datastore_client,
            self.job,
            self.task.id,
            new_state='completed',
            payload=self.payload
        )


@dataclasses.dataclass
class FindCulprit(task_module.PayloadUnpackingMixin,
                  updates.ErrorAppendingMixin):
    """Finds a culprit by bisection.

    Expects to be called with a context that contains:

    - a ResultReaderPayload for each direct dependency of the task, and
    - entries for each result reader task subgraph.
    """
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

    def __call__(self, task, _, context):
        # Outline:
        #  - If the task is still pending, this means this is the first time we're
        #  encountering the task in an evaluation. Set up the payload data to
        #  include the full range of commits, so that we load it once and have it
        #  ready, and emit an action to mark the task ongoing.
        #
        #  - If the task is ongoing, gather all the dependency data (both results
        #  and state) and see whether we have enough data to determine the next
        #  action. We have three main cases:
        #
        #    1. We cannot detect a significant difference between the results from
        #       two different CLs. We call this the NoReproduction case.
        #
        #    2. We do not have enough confidence that there's a difference. We call
        #       this the Indeterminate case.
        #
        #    3. We have enough confidence that there's a difference between any two
        #       ordered changes. We call this the SignificantChange case.
        #
        # - Delegate the implementation to handle the independent cases for each
        #   change point we find in the CL continuum.
        logging.debug(f'FindCulprit.__call__, task.state={task.state}')
        if task.state == 'pending':
            return [PrepareCommitsAction(self.datastore_client, self.job, task)]

        task_payload = self.unpack(
                find_culprit_task_payload_pb2.FindCulpritTaskPayload,
                task.payload)

        actions = []
        all_changes = [
                change_module.Change.FromProto(self.datastore_client, change)
                for change in task_payload.state.changes]

        if task.state == 'ongoing':
            # TODO(dberris): Validate and fail gracefully instead of asserting?
            if len(all_changes) == 0:
                return self.complete_with_error(
                        task, task_payload, 'AssertionError',
                        'Programming error, need commits to proceed!')

            analysis_options = task_payload.input.analysis_options

            # Collect all the dependency task data and analyse the results
            # (remember the dependencies of the find_culprit task are read_value
            # tasks, which have had their state lifted into the accumulator).
            # Group them by change.
            # Order them by appearance in the CL range.
            # Also count the state per CL (failed, ongoing, etc.)
            results_by_change = collections.defaultdict(list)
            state_by_change = collections.defaultdict(dict)
            changes_with_data = set()
            changes_by_state = collections.defaultdict(set)

            associated_results = [
                (change_module.Change.FromProto(self.datastore_client,
                                                rv_payload.input.change),
                 rv_state, rv_payload.output.result_values)
                for (rv_state, rv_payload) in self._read_values_payloads(task, context)]

            for change, state, result_values in associated_results:
                if result_values:
                    filtered_results = [r for r in result_values if r is not None]
                    if filtered_results:
                        results_by_change[change].append(filtered_results)
                state_by_change[change].update({
                        state: state_by_change[change].get(state, 0) + 1,
                })
                changes_by_state[state].add(change)
                changes_with_data.add(change)

            # If the dependencies have converged into a single state, we can make
            # decisions on the terminal state of the bisection.
            if len(changes_by_state) == 1 and changes_with_data:
                # Check whether all dependencies are completed and if we do
                # not have data in any of the dependencies.
                if changes_by_state.get('completed') == changes_with_data:
                    changes_with_empty_results = [
                            change for change in changes_with_data
                            if not results_by_change.get(change)
                    ]
                    if changes_with_empty_results:
                        return self.complete_with_error(
                                task, task_payload, 'BisectionFailed',
                                'We did not find any results from successful '
                                'test runs.')

                # Check whether all the dependencies had the tests fail consistently.
                elif changes_by_state.get('failed') == changes_with_data:
                    return self.complete_with_error(
                                task, task_payload, 'BisectionFailed',
                                'All attempts in all dependencies failed.')
                # If they're all pending or ongoing, then we don't do anything yet.
                else:
                    return actions

            # We want to reduce the list of ordered changes to only the ones that have
            # data available.
            change_index = {change: index for index, change in enumerate(all_changes)}
            ordered_changes = [c for c in all_changes if c in changes_with_data]

            # From here we can then do the analysis on a pairwise basis, as we're
            # going through the list of Change instances we have data for.
            # NOTE: A lot of this algorithm is already in pinpoint/models/job_state.py
            # which we're adapting.
            def Compare(a, b):
                # This is the comparison function which determines whether the samples
                # we have from the two changes (a and b) are statistically significant.
                if a is None or b is None:
                    return None

                if 'pending' in state_by_change[a] or 'pending' in state_by_change[b]:
                    return compare.ComparisonResult(compare.PENDING, None, None, None)

                # NOTE: Here we're attempting to scale the provided comparison magnitude
                # threshold by the larger inter-quartile range (a measure of dispersion,
                # simply computed as the 75th percentile minus the 25th percentile). The
                # reason we're doing this is so that we can scale the tolerance
                # according to the noise inherent in the measurements -- i.e. more noisy
                # measurements will require a larger difference for us to consider
                # statistically significant.
                values_for_a = tuple(itertools.chain(*results_by_change[a]))
                values_for_b = tuple(itertools.chain(*results_by_change[b]))

                if not values_for_a:
                    return None
                if not values_for_b:
                    return None

                max_iqr = max(
                        scipy.stats.iqr(values_for_a), scipy.stats.iqr(values_for_b), 0.001)
                comparison_magnitude = analysis_options.comparison_magnitude
                if comparison_magnitude == 0.0: comparison_magnitude = 1.0
                comparison_magnitude /= max_iqr
                attempts = (len(values_for_a) + len(values_for_b)) // 2
                result = compare.compare(values_for_a, values_for_b, attempts,
                                       'performance', comparison_magnitude)
                return result

            def DetectChange(change_a, change_b):
                # We return None if the comparison determines that the result is
                # inconclusive. This is required by the exploration.speculate contract.
                comparison = Compare(change_a, change_b)
                if comparison.result == compare.UNKNOWN:
                    return None
                return comparison.result == compare.DIFFERENT

            changes_to_refine = []

            def CollectChangesToRefine(a, b):
                # Here we're collecting changes that need refinement, which
                # happens when two changes when compared yield the "unknown"
                # result.
                attempts_for_a = sum(state_by_change[a].values())
                attempts_for_b = sum(state_by_change[b].values())

                # Grow the attempts of both changes by 50% every time when
                # increasing attempt counts. This number is arbitrary, and we
                # should probably use something like a Fibonacci sequence when
                # scaling attempt counts.
                max_attempts = analysis_options.max_attempts
                if max_attempts == 0: max_attempts = 100
                new_attempts_size_a = min(
                        math.ceil(attempts_for_a * 1.5), max_attempts)
                new_attempts_size_b = min(
                        math.ceil(attempts_for_b * 1.5), max_attempts)

                # Only refine if the new attempt sizes are not large enough.
                if new_attempts_size_a > attempts_for_a:
                    changes_to_refine.append((a, new_attempts_size_a))
                if new_attempts_size_b > attempts_for_b:
                    changes_to_refine.append((b, new_attempts_size_b))

            def FindMidpoint(a, b):
                # Here we use the (very simple) midpoint finding algorithm given
                # that we already have the full range of commits to bisect
                # through.
                a_index = change_index[a]
                b_index = change_index[b]
                subrange = all_changes[a_index:b_index + 1]
                return None if len(subrange) <= 2 else subrange[len(subrange) // 2]

            # We have a striding iterable, which will give us the before, current, and
            # after for a given index in the iterable.
            def SlidingTriple(iterable):
                """s -> (None, s0, s1), (s0, s1, s2), (s1, s2, s3), ..."""
                p, c, n = itertools.tee(iterable, 3)
                p = itertools.chain([None], p)
                n = itertools.chain(itertools.islice(n, 1, None), [None])
                return zip(p, c, n)

            # This is a comparison between values at a change and the values at
            # the previous change and the next change.
            comparisons = [
                    PrevNextComparison(prev=Compare(p, c), next=Compare(c, n))
                    for (p, c, n) in SlidingTriple(ordered_changes)]

            # Collect the result values for each change with values.
            result_values = [
                    list(itertools.chain(*results_by_change.get(change, [])))
                    for change in ordered_changes
            ]
            results_for_changes = [
                    ResultForChange(result_values=rv, comparisons=c)
                    for (rv, c) in zip(result_values, comparisons)]

            if results_for_changes != [
                    ResultForChange.from_proto(change_result)
                    for change_result in task_payload.output.change_results]:

                del task_payload.output.change_results[:]
                task_payload.output.change_results.extend([
                        change_result.to_proto()
                        for change_result in results_for_changes])
                encoded_payload = any_pb2.Any()
                encoded_payload.Pack(task_payload)
                actions.append(
                    updates.UpdateTaskAction(self.datastore_client,
                                             self.job,
                                             task.id,
                                             payload=encoded_payload))

            if len(ordered_changes) < 2:
                # We do not have enough data yet to determine whether we should do
                # anything.
                return actions

            additional_changes = exploration.speculate(
                    ordered_changes,
                    change_detected=DetectChange,
                    on_unknown=CollectChangesToRefine,
                    midpoint=FindMidpoint,
                    levels=_DEFAULT_SPECULATION_LEVELS)

            # At this point we can collect the actions to extend the task graph based
            # on the results of the speculation, only if the changes don't have any
            # more associated pending/ongoing work.
            min_attempts = analysis_options.min_attempts
            if min_attempts == 0: min_attempts = 10
            additional_changes = list(additional_changes)
            new_actions = [
                    RefineExplorationAction(self.datastore_client, self.job,
                                            task, change, new_size)
                    for change, new_size in itertools.chain(
                            [(c, min_attempts) for _, c in additional_changes],
                            [(c, a) for c, a in changes_to_refine],
                    )
                    if not bool({'pending', 'ongoing'} & set(state_by_change[change]))
            ]
            actions += new_actions

            # Here we collect the points where we've found the changes.
            def Pairwise(iterable):
                """s -> (s0, s1), (s1, s2), (s2, s3), ..."""
                a, b = itertools.tee(iterable)
                next(b, None)
                return zip(a, b)
            culprits_before = len(task_payload.output.culprits)
            del task_payload.output.culprits[:]
            for a, b in Pairwise(ordered_changes):
                if not DetectChange(a, b): continue
                task_payload.output.culprits.add(from_=a.to_proto(),
                                                 to=b.to_proto())
            encoded_payload = any_pb2.Any()
            encoded_payload.Pack(task_payload)

            can_complete = not bool(
                    set(changes_by_state) - {'failed', 'completed'})
            if not actions and can_complete:
                # Mark this operation complete, storing the differences we can
                # compute.
                logging.debug('Returning CompleteExplorationAction')
                actions = [CompleteExplorationAction(
                        self.datastore_client, self.job, task,
                        encoded_payload)]
            elif len(task_payload.output.culprits) != culprits_before:
                # The operation isn't complete, but we have updated the set of
                # culprits found so far, so record that.
                actions.append(
                    updates.UpdateTaskAction(self.datastore_client,
                                             self.job,
                                             task.id,
                                             payload=encoded_payload))
            return actions

    def _read_values_payloads(self, task, context)-> Iterable[Tuple[
            str, result_reader_payload_pb2.ResultReaderPayload]]:
        deps = set(task.dependencies)
        for dep_id, task_context in context.items():
            if dep_id in deps:
                yield task_context.state, self.unpack(
                        result_reader_payload_pb2.ResultReaderPayload,
                        task_context.payload)


class Evaluator(combinators.FilteringEvaluator):
    def __init__(self, job, datastore_client):
        super(Evaluator, self).__init__(
            predicate=predicates.All(
                predicates.TaskTypeEq('find_culprit'),
                predicates.Not(predicates.TaskStateIn({'completed', 'failed'}))),
            delegate=FindCulprit(datastore_client, job))


@dataclasses.dataclass
class PrevNextComparison:
    prev: compare.ComparisonResult
    next: compare.ComparisonResult

    def to_proto(self) -> _PayloadOutput.ResultForChange.PrevNextComparison:
        return _PayloadOutput.ResultForChange.PrevNextComparison(
                prev=self.prev.to_proto() if self.prev is not None else None,
                next=self.next.to_proto() if self.next is not None else None)

    @classmethod
    def from_proto(cls, proto: _PayloadOutput.ResultForChange.PrevNextComparison):
        return cls(prev=compare.ComparisonResult.from_proto(proto.prev),
                   next=compare.ComparisonResult.from_proto(proto.next))


@dataclasses.dataclass
class ResultForChange:
    result_values: List[float]
    comparisons: PrevNextComparison

    def to_proto(self) -> _PayloadOutput.ResultForChange:
        return _PayloadOutput.ResultForChange(
                result_values=self.result_values,
                comparisons=self.comparisons.to_proto())

    @classmethod
    def from_proto(cls, proto: _PayloadOutput.ResultForChange):
        return cls(
                result_values=proto.result_values,
                comparisons=PrevNextComparison.from_proto(proto.comparisons))
