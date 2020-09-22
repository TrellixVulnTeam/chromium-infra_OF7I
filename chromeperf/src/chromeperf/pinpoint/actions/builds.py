# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import dataclasses
import json
import logging
import urllib
import typing

from google.cloud import datastore
from google.protobuf import any_pb2
from google.protobuf import message

from chromeperf.engine import event as event_module
from chromeperf.engine import evaluator as evaluator_module
from chromeperf.pinpoint import errors
from chromeperf.pinpoint import find_isolate_task_payload_pb2
from chromeperf.pinpoint.actions import updates
from chromeperf.pinpoint.models import change as change_module
from chromeperf.pinpoint.models import commit as commit_module
from chromeperf.pinpoint.models import task as task_module
from chromeperf.services import buildbucket_service
from chromeperf.services import gerrit_service
from chromeperf.services import request

FAILURE_MAPPING = {'FAILURE': 'failed', 'CANCELLED': 'cancelled'}


def _request_build(datastore_client,
                   builder_name,
                   change,
                   bucket,
                   build_tags,
                   task=None):
    base_as_dict = change.base_commit.AsDict(datastore_client)
    review_url = base_as_dict.get('review_url')
    if not review_url:
        raise errors.BuildGerritUrlNotFound(str(change.base_commit))

    url_parts = urllib.parse.urlparse(review_url)
    base_review_url = urllib.parse.urlunsplit(
        (url_parts.scheme, url_parts.netloc, '', '', ''))

    patch = change_module.GerritPatch.FromUrl(review_url)

    change_info = gerrit_service.get_change(base_review_url, patch.change)

    commit_url_parts = urllib.parse.urlparse(base_as_dict['url'])

    # Note: The ordering here for buildbucket v1 api is important.
    # crbug.com/937392
    builder_tags = []
    if change.patch:
        builder_tags.append(change.patch.BuildsetTags())
    builder_tags.append('buildset:commit/gitiles/%s/%s/+/%s' %
                        (commit_url_parts.netloc, change_info['project'],
                         change.base_commit.git_hash))
    builder_tags.extend(['%s:%s' % (k, v) for k, v in build_tags.items()])

    deps_overrides = {dep.repository_url: dep.git_hash for dep in change.deps}
    parameters = {
        'builder_name': builder_name,
        'properties': {
            # We're making Pinpoint use incremental builds to amortise the cost
            # of rebuilding the object files. Clobber builds indicate that a
            # builder will clean out previous build artifacts instead of
            # re-using potentially already-built object files from a previous
            # checkout. Incremental builds will be much faster especially with
            # the help of goma.
            'clobber': False,
            'revision': change.base_commit.git_hash,
            'deps_revision_overrides': deps_overrides,
        },
    }

    if change.patch:
        parameters['properties'].update(change.patch.BuildParameters())

    logging.debug('bucket: %s', bucket)
    logging.debug('builder_tags: %s', builder_tags)
    logging.debug('parameters: %s', parameters)

    pubsub_callback = None
    if build_tags:
        # This means we have access to Pinpoint job details, we should provide
        # this information to the attempts to build.
        pubsub_callback = {
            # TODO(dberris): Consolidate constants in environment vars?
            'topic':
            'projects/chromeperf/topics/pinpoint-swarming-updates',
            'auth_token':
            'UNUSED',
            'user_data':
            json.dumps({
                'job_id': build_tags.get('pinpoint_job_id'),
                'task': {
                    'type':
                    'build',
                    'id':
                    build_tags.get('pinpoint_task_id')
                    if not task else task.id,
                }
            })
        }
        logging.debug('pubsub_callback: %s', pubsub_callback)

    return buildbucket_service.put(bucket, builder_tags, parameters,
                                   pubsub_callback)


def _build_tags_from_job(job):
    return collections.OrderedDict([
        ('pinpoint_job_id', job.job_id),
        ('pinpoint_user', job.user),
        ('pinpoint_url', job.url),
    ])


@dataclasses.dataclass
class ScheduleBuildBucketBuild(task_module.PayloadUnpackingMixin):
    """Action to schedule a build via BuildBucket.

    This action will schedule a build via the BuildBucket API, and ensure
    that Pinpoint is getting updates via PubSub on request completions.

    Side Effects:

        - The Action will update the Task payload to include the build
        request information and the response from the BuildBucket service,
        and set the state to 'ongoing' on success, 'failed' otherwise.

    """
    job: typing.Any
    task: task_module.Task
    change: change_module.Change
    datastore_client: datastore.Client

    @updates.log_transition_failures
    def __call__(self, context):
        # TODO(dberris): Maybe use a value in the context to check whether we
        # should bail?
        # TODO(dberris): Figure out whether we should make the "build proto"
        # more generic, instead of being specific to the evaluator.
        payload = self.unpack(
            find_isolate_task_payload_pb2.FindIsolateTaskPayload,
            self.task.payload,
        )
        payload.tries += 1
        self.task.payload.Pack(payload)
        updates.update_task(self.datastore_client,
                            self.job,
                            self.task.id,
                            new_state='ongoing',
                            payload=self.task.payload)
        result = _request_build(self.datastore_client, payload.builder,
                                self.change, payload.bucket,
                                _build_tags_from_job(self.job), self.task)
        build = result.get('build')
        if build:
            payload.buildbucket_build.id = build.get('id', '')
            payload.buildbucket_build.url = build.get('url', '')

        # TODO(dberris): Poll the ongoing build if the attempt to update fails,
        # if we have the data in payload?
        self.task.payload.Pack(payload)
        updates.update_task(self.datastore_client,
                            self.job,
                            self.task.id,
                            payload=self.task.payload)

    def __str__(self):
        return 'Build Action(job = %s, task = %s)' % (self.job.job_id,
                                                      self.task.id)


@dataclasses.dataclass
class UpdateBuildStatus(task_module.PayloadUnpackingMixin,
                        updates.ErrorAppendingMixin):
    datastore_client: datastore.Client
    job: typing.Any
    task: evaluator_module.NormalizedTask
    change: change_module.Change
    event: event_module.Event

    def _append_error(self, task_payload, reason: str, message: str):
        return self.update_task_with_error(
            datastore_client=self.datastore_client,
            job=self.job,
            task=self.task,
            payload=task_payload,
            reason=reason,
            message=message,
        )

    @updates.log_transition_failures
    def __call__(self, accumulator):
        # The task contains the buildbucket_result which we need to update by
        # polling the status of the id.
        build_update = self.unpack(find_isolate_task_payload_pb2.BuildUpdate,
                                   self.event.payload)
        task_payload = self.unpack(
            find_isolate_task_payload_pb2.FindIsolateTaskPayload,
            self.task.payload)
        if not task_payload.buildbucket_build.id:
            logging.error(
                'No build details in attempt to update build status; task = %s',
                self.task,
            )
            updates.update_task(
                self.datastore_client,
                self.job,
                self.task.id,
                new_state='failed',
            )
            return None

        try:
            buildbucket_build = buildbucket_service.get(
                task_payload.buildbucket_build.id).get(
                    'build',
                    {},
                )
            task_payload.buildbucket_build.status = buildbucket_build.get(
                'status',
                '',
            )
            task_payload.buildbucket_build.result = buildbucket_build.get(
                'result',
                '',
            )
            task_payload.buildbucket_build.url = buildbucket_build.get(
                'url',
                '',
            )
            task_payload.buildbucket_build.result_details_json = buildbucket_build.get(
                'result_details_json',
                '',
            )
        except request.RequestError as e:
            logging.error('Failed getting Buildbucket Job status: %s', e)
            return self._append_error(
                task_payload,
                reason=type(e).__name__,
                message=f'Service request error response: {e}',
            )

        logging.debug('buildbucket response: %s', buildbucket_build)

        # Update the buildbucket result.
        # Decide whether the build was successful or not.
        if task_payload.buildbucket_build.status != 'COMPLETED':
            # Skip this update.
            return None

        if not task_payload.buildbucket_build.result:
            logging.debug('Missing result field in response, bailing.')
            return self._append_error(
                task_payload,
                reason='InvalidResponse',
                message='Response is missing the "result" field.',
            )

        if task_payload.buildbucket_build.result in FAILURE_MAPPING:
            build = task_payload.buildbucket_build
            message = (f'BuildBucket job "{build.id}" '
                       f'failed with status "{build.result}')
            return self._append_error(
                task_payload,
                reason='BuildFailed',
                message=message,
            )

        # Parse the result and mark this task completed.
        if not task_payload.buildbucket_build.result_details_json:
            return self._append_error(
                task_payload,
                reason='BuildIsolateNotFound',
                message=f'Could not find isolate for build at {self.change}',
            )

        try:
            result_details = json.loads(
                task_payload.buildbucket_build.result_details_json)
        except ValueError as e:
            return self._append_error(
                task_payload,
                reason='BuildIsolateNotFound',
                message=f'Invalid JSON response: {e}',
            )

        if 'properties' not in result_details:
            return self._append_error(
                reason='BuildIsolateNotFound',
                message=
                f'Could not find result details for build at {self.change}',
            )

        properties = result_details['properties']

        # Validate whether the properties in the result include required data.
        required_keys = set(['isolate_server', 'got_revision_cp'])
        missing_keys = required_keys - set(properties)
        if missing_keys:
            return self._append_error(
                task_payload,
                reason='BuildIsolateNotFound',
                message=
                f'Properties in result missing required data: {missing_keys}',
            )

        commit_position = properties['got_revision_cp'].replace('@', '(at)')
        suffix = ('without_patch'
                  if 'patch_storage' not in properties else 'with_patch')
        key = '_'.join(('swarm_hashes', commit_position, suffix))

        if task_payload.target not in properties.get(key, {}):
            # TODO(dberris): Update the job state with an exception, or set of
            # failures.
            return self._append_error(
                task_payload,
                reason='BuildIsolateNotFound',
                message=f'Could not find isolate for build at {self.change}',
            )

        task_payload.isolate_server = properties['isolate_server']
        task_payload.isolate_hash = properties[key][task_payload.target]
        self.task.payload.Pack(task_payload)
        updates.update_task(
            self.datastore_client,
            self.job,
            self.task.id,
            new_state='completed',
            payload=self.task.payload,
        )

    def __str__(self):
        return (f'Update Build Action '
                f'<job = {self.job.job_id}, task = {self.task.id}>')
