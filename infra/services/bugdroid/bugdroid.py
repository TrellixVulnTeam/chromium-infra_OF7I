# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Update bug trackers with information from repository commits."""

import argparse
import datetime
import json
import logging
import os
import sys
import threading
import time
import urlparse

from infra.services.bugdroid import branch_utils
from infra.services.bugdroid import config_service
from infra.services.bugdroid import gerrit_poller
from infra.services.bugdroid import gitiles_poller
from infra.services.bugdroid import log_parser
from infra.services.bugdroid import monorail_client
from infra.services.bugdroid import poller_handlers
from infra.services.bugdroid import scm_helper

import infra_libs.logs
from infra_libs import ts_mon


class BugdroidGitPollerHandler(poller_handlers.BasePollerHandler):
  """Handler for updating bugs with information from commits."""

  bug_comments_metric = ts_mon.CounterMetric(
      'bugdroid/bug_comments',
      'Counter of comments added to bugs',
      [ts_mon.StringField('project'),
       ts_mon.StringField('status')])

  def __init__(self, monorail, logger, default_project,
               no_merge=None, public_bugs=True, test_mode=False,
               issues_labels=None, shorten_links=False, *args, **kwargs):
    self.monorail_client = monorail
    self.logger = logger
    self.default_project = default_project
    self.no_merge = no_merge or []
    self.public_bugs = public_bugs
    self.test_mode = test_mode
    self.shorten_links = shorten_links
    if issues_labels:
      self.issues_labels = dict((p.key, p.value) for p in issues_labels)
    else:
      self.issues_labels = {}
    super(BugdroidGitPollerHandler, self).__init__(*args, **kwargs)

  def _ApplyMergeMergedLabel(self, issue, branch):
    if not branch or not issue:
      return

    mstone = branch_utils.get_mstone(branch, False)

    # Apply "merge-merged-{branch}" and "merge-merged-{milestone}" labels
    merged_prefix = self.issues_labels.get('merge', 'merge-merged')
    merged_labels = ['%s-%s' % (merged_prefix, branch)]
    if mstone:
      merged_labels.append('%s-%s' % (merged_prefix, mstone))
    for label in merged_labels:
      issue.add_label(label)
      self.logger.debug('Adding %s', label)

    label = self.issues_labels.get('approved', 'merge-approved')
    if issue.has_label(label):
      issue.remove_label(label)
      self.logger.debug('Removing %s', label)

    if mstone:
      label = 'merge-approved-%s' % mstone
      if issue.has_label(label):
        issue.remove_label(label)
        self.logger.debug('Removing %s', label)

  def ProcessLogEntry(self, log_entry):
    affected_issues = log_parser.get_issues(
        log_entry, default_project=self.default_project)
    self.logger.info('Processing commit %s : bugs %s',
                     log_entry.revision, str(affected_issues['bugs']))
    if affected_issues['bugs']:
      comment = self._CreateMessage(log_entry)
      self.logger.debug(comment)

      for project, bugs in affected_issues['bugs'].iteritems():
        for bug in bugs:
          try:
            issue = self.monorail_client.get_issue(project, bug)
            issue.set_comment(comment[:24 * 1024])
            if bug in affected_issues['fixed'].get(project, set()):
              issue.mark_fixed()
            branch = scm_helper.GetBranch(log_entry)
            # Apply merge labels if this commit landed on a branch.
            if branch and not (log_entry.scm in ['git', 'gerrit'] and
                               scm_helper.GetBranch(log_entry, full=True) in
                               self.no_merge):
              self._ApplyMergeMergedLabel(issue, branch)
            self.logger.debug('Attempting to save issue: %d', issue.id)
            if not self.test_mode:
              self.monorail_client.update_issue(
                  project, issue, log_parser.should_send_email(log_entry.msg))
            else:
              self.logger.debug('Test mode, skipping')
          except Exception:
            self.bug_comments_metric.increment(
                {'project': project, 'status': 'failure'})
            raise
          else:
            self.bug_comments_metric.increment(
                {'project': project, 'status': 'success'})

  def _CreateMessage(self, log_entry):
    msg = ''
    msg += 'The following revision refers to this bug:\n'
    if log_entry.HasUrl():
      msg += '  %s\n\n' % log_entry.GetCommitUrl()
    msg += self._BuildLogSpecial(log_entry)
    return msg

  def _BuildLogSpecial(self, log_entry):
    """Generate git-log style message, with links to files in the Web UI."""
    rtn = 'commit %s\n' % log_entry.commit
    rtn += 'Author: %s <%s>\n' % (log_entry.author_name, log_entry.author_email)
    rtn += 'Date: %s\n' % log_entry.committer_date

    # Skip commit message and paths for non-public bugs.
    if not self.public_bugs:
      return rtn
    rtn += '\n%s\n' % log_entry.msg

    for path in log_entry.paths:
      if path.action == 'delete':
        # Use parent and copy_from_path for deletions, otherwise we get links
        # to https://.../<commit>//dev/null
        rtn += '[%s] %s\n' % (
            path.action, log_entry.GetPathUrl(
                path.copy_from_path, parent=True, shorten=self.shorten_links))
      else:
        rtn += '[%s] %s\n' % (
            path.action, log_entry.GetPathUrl(
                path.filename, shorten=self.shorten_links))
    return rtn


class Bugdroid(object):
  """App to setup and run repository pollers and bug updating handlers."""

  def __init__(
      self, configfile, credentials_db, run_once, datadir, dryrun=False):
    self.pollers = []
    self.credentials_db = credentials_db
    self.run_once = run_once
    self.dryrun = dryrun

    if not os.path.isdir(datadir):
      if os.path.exists(datadir):
        raise ConfigsException(
          'datadir "%s" is not a directory.' % datadir)
      os.makedirs(datadir)
    self.datadir = datadir

    logging.basicConfig(level=logging.DEBUG)

    configs = config_service.get_repos(self.credentials_db, configfile)

    if not configs.repos:
      raise ConfigsException('Failed to load poller configs. Aborting.')

    git_projects = []

    # 1. Create Monorail client.
    self.monorail_client = monorail_client.MonorailClient(self.credentials_db)

    # 2. Generate git pollers
    for config in configs.repos:
      t = config_service.decode_repo_type(config.repo_type)
      if t != 'git':
        continue
      if config.refs_regex and config.filter_regex:
        if len(config.refs_regex) != len(config.filter_regex):
          raise ConfigsException(
              'Config error (%s): "refs_regex" and "filter_regex" '
              'cannot have different numbers of items.' % config.repo_name)
      path = urlparse.urlparse(config.repo_url).path.lower()
      if path.startswith('/a/'):
        path = path[3:]
      else:
        path = path[1:]
      if path.endswith('.git'):
        path = path[:len(path)-4]
      git_projects.append(path)
      poller = self.InitPoller(config.repo_name, config)
      self.pollers.append(poller)

    logging.info('Git projects %s', git_projects)

    # 3. Generate gerrit pollers
    for config in configs.repos:
      t = config_service.decode_repo_type(config.repo_type)
      if t != 'gerrit':
        continue
      poller = self.InitPoller(config.repo_name, config, git_projects)
      self.pollers.append(poller)

  def InitPoller(self, name, config, git_projects=None):
    """Create a repository poller based on the given config."""

    t = config_service.decode_repo_type(config.repo_type)
    interval_minutes = 1
    default_project = config.default_project
    logger = logging.getLogger(name)
    if t == 'git':
      poller = gitiles_poller.GitilesPoller(
          config.repo_url,
          name,
          refs_regex=config.refs_regex,
          paths_regex=config.paths_regex,
          filter_regex=config.filter_regex,
          interval_in_minutes=interval_minutes,
          logger=logger,
          run_once=self.run_once,
          datadir=self.datadir,
          with_paths=not config.skip_paths)
    elif t == 'gerrit':
      poller = gerrit_poller.GerritPoller(
          config.repo_url,
          name,
          interval_in_minutes=interval_minutes,
          logger=logger,
          run_once=self.run_once,
          datadir=self.datadir,
          git_projects=git_projects)
    else:
      raise ConfigsException('Unknown poller type: %s' % t)

    poller.add_handler(BugdroidGitPollerHandler(
        monorail=self.monorail_client,
        logger=logger,
        default_project=default_project,
        public_bugs=config.public_bugs,
        test_mode=config.test_mode or self.dryrun,
        issues_labels=config.issues_labels,
        no_merge=config.no_merge_refs,
        shorten_links=config.shorten_links))
    poller.saved_config = config
    poller.daemon = True
    return poller

  def Execute(self):
    for poller in self.pollers:
      poller.logger.info('Starting Poller "%s".', poller.poller_id)
      poller.start()
    for poller in self.pollers:
      poller.join()


def inner_loop(opts):
  bugdroid = Bugdroid(
      opts.configfile, opts.credentials_db, True, opts.datadir, opts.dryrun)
  bugdroid.Execute()
  return True


class Error(Exception):
  pass


class ConfigsException(Error):
  """The repo configs cannot be used."""
