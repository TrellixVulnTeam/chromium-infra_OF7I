# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Task queue endpoints for creating and updating issues on issue tracker."""

import datetime
import logging
import webapp2

from google.appengine.api import taskqueue
from google.appengine.ext import ndb

from issue_tracker import issue_tracker_api, issue
from model.flake import FlakeUpdateSingleton, FlakeUpdate


FLAKY_RUNS_TEMPLATE = (
    'Detected %(new_flakes_count)d new flakes for test/step "%(name)s". To see '
    'the actual flakes, please visit %(flakes_url)s. This message was posted '
    'automatically by the chromium-try-flakes app.')
SUMMARY_TEMPLATE = '"%(name)s" is flaky'
DESCRIPTION_TEMPLATE = (
    '%(summary)s.\n\n'
    'This issue was created automatically by the chromium-try-flakes app. '
    'Please find the right owner to fix the respective test/step and assign '
    'this issue to them. If the step/test is infrastructure-related, please '
    'add Infra-Troopers label and change issue status to Untriaged.\n\n'
    'We have detected %(flakes_count)d recent flakes. List of all flakes can '
    'be found at %(flakes_url)s.')
REOPENED_DESCRIPTION_TEMPLATE = (
    '%(description)s\n\n'
    'This flaky test/step was previously tracked in issue %(old_issue)d.')
MAX_UPDATED_ISSUES_PER_DAY = 50
MAX_TIME_DIFFERENCE_SECONDS = 12 * 60 * 60
MIN_REQUIRED_FLAKY_RUNS = 5
FLAKES_URL_TEMPLATE = (
    'https://chromium-try-flakes.appspot.com/all_flake_occurrences?key=%s')


class ProcessIssue(webapp2.RequestHandler):
  @ndb.transactional
  def _get_flake_update_singleton_key(self):
    singleton_key = ndb.Key('FlakeUpdateSingleton', 'singleton')
    if not singleton_key.get():
      FlakeUpdateSingleton(key=singleton_key).put()
    return singleton_key

  @ndb.transactional
  def _increment_update_counter(self):
    FlakeUpdate(parent=self._get_flake_update_singleton_key()).put()

  @ndb.non_transactional
  def _time_difference(self, flaky_run):
    return (flaky_run.success_run.get().time_finished -
            flaky_run.failure_run_time_finished).total_seconds()

  @ndb.non_transactional
  def _is_same_day(self, flaky_run):
    time_since_finishing = (
        datetime.datetime.utcnow() - flaky_run.failure_run_time_finished)
    return time_since_finishing <= datetime.timedelta(days=1)

  @ndb.non_transactional
  def _get_new_flakes_count(self, flake):
    num_runs = len(flake.occurrences) - flake.num_reported_flaky_runs
    flaky_runs = ndb.get_multi(flake.occurrences[-num_runs:])
    return len([
      flaky_run for flaky_run in flaky_runs
      if self._is_same_day(flaky_run) and
         self._time_difference(flaky_run) <= MAX_TIME_DIFFERENCE_SECONDS])

  @ndb.transactional
  def _recreate_issue_for_flake(self, flake):
    """Updates a flake to re-create an issue and creates a respective task."""
    flake.old_issue_id = flake.issue_id
    flake.issue_id = 0
    taskqueue.add(url='/issues/process/%s' % flake.key.urlsafe(),
                  queue_name='issue-updates', transactional=True)

  @ndb.transactional
  def _update_issue(self, api, flake, new_flakes_count, now):
    """Updates an issue on the issue tracker."""
    flake_issue = api.getIssue(flake.issue_id)

    # Handle cases when an issue has been closed. We need to do this in a loop
    # because we might move onto another issue.
    seen_issues = set()
    while not flake_issue.open:
      if flake_issue.status == 'Duplicate':
        # If the issue was marked as duplicate, we update the issue ID stored in
        # datastore to the one it was merged into and continue working with the
        # new issue.
        seen_issues.add(flake_issue.id)
        if flake_issue.merged_into not in seen_issues:
          flake.issue_id = flake_issue.merged_into
          flake_issue = api.getIssue(flake.issue_id)
        else:
          logging.info('Detected issue duplication loop: %s. Re-creating an '
                       'issue for the flake %s.', seen_issues, flake.name)
          self._recreate_issue_for_flake(flake)
          return
      else:  # Fixed, WontFix, Verified, Archived, custom status
        # If the issue was closed, we do not update it. This allows changes made
        # to reduce flakiness to propagate and take effect. If after 3 days we
        # still see flakiness, we will create a new issue.
        if flake_issue.updated < now - datetime.timedelta(days=3):
          self._recreate_issue_for_flake(flake)
        return

    new_flaky_runs_msg = FLAKY_RUNS_TEMPLATE % {
        'name': flake.name,
        'new_flakes_count': new_flakes_count,
        'flakes_url': FLAKES_URL_TEMPLATE % flake.key.urlsafe()}
    api.update(flake_issue, comment=new_flaky_runs_msg)
    logging.info('Updated issue %d for flake %s with %d flake runs',
                 flake.issue_id, flake.name, new_flakes_count)
    flake.num_reported_flaky_runs = len(flake.occurrences)
    flake.issue_last_updated = now

  @ndb.transactional
  def _create_issue(self, api, flake, flakes_count):
    summary = SUMMARY_TEMPLATE % {'name': flake.name}
    description = DESCRIPTION_TEMPLATE % {
        'summary': summary,
        'flakes_url': FLAKES_URL_TEMPLATE % flake.key.urlsafe(),
        'flakes_count': flakes_count}
    if flake.old_issue_id:
      description = REOPENED_DESCRIPTION_TEMPLATE % {
          'description': description, 'old_issue': flake.old_issue_id}

    new_issue = issue.Issue({'summary': summary,
                             'description': description,
                             'status': 'Untriaged',
                             'labels': ['Type-Bug', 'Pri-1', 'Cr-Tests-Flaky',
                                        'Via-TryFlakes', 'Sheriff-Chromium']})
    flake_issue = api.create(new_issue)
    flake.issue_id = flake_issue.id
    flake.num_reported_flaky_runs = len(flake.occurrences)
    logging.info('Created a new issue %d for flake %s', flake.issue_id,
                 flake.name)

  @ndb.transactional(xg=True)  # pylint: disable=E1120
  def post(self, urlsafe_key):
    api = issue_tracker_api.IssueTrackerAPI('chromium', use_monorail=False)

    # Check if we should stop processing this issue because we've posted too
    # many updates to issue tracker today already.
    day_ago = datetime.datetime.utcnow() - datetime.timedelta(days=1)
    num_updates_last_day = FlakeUpdate.query(
        FlakeUpdate.time_updated > day_ago,
        ancestor=self._get_flake_update_singleton_key()).count()
    if num_updates_last_day >= MAX_UPDATED_ISSUES_PER_DAY:
      return

    now = datetime.datetime.utcnow()
    flake = ndb.Key(urlsafe=urlsafe_key).get()
    # Only update/file issues if there are new flaky runs.
    if flake.num_reported_flaky_runs == len(flake.occurrences):
      return

    # Retrieve flaky runs outside of the transaction, because we are not
    # planning to modify them and because there could be more of them than the
    # number of groups supported by cross-group transactions on AppEngine.
    new_flakes_count = self._get_new_flakes_count(flake)

    if new_flakes_count < MIN_REQUIRED_FLAKY_RUNS:
      return

    if flake.issue_id > 0:
      # Update issues at most once a day.
      if flake.issue_last_updated > now - datetime.timedelta(days=1):
        return

      self._update_issue(api, flake, new_flakes_count, now)
      self._increment_update_counter()
    else:
      self._create_issue(api, flake, new_flakes_count)
      # Don't update the issue just yet, this may fail, and we need the
      # transaction to succeed in order to avoid filing duplicate bugs.
      self._increment_update_counter()

    # Note that if transaction fails for some reason at this point, we may post
    # updates or create issues multiple times. On the other hand, this should be
    # extremely rare because we set the number of concurrently running tasks to
    # 1, therefore there should be no contention for updating this issue's
    # entity.
    flake.put()
