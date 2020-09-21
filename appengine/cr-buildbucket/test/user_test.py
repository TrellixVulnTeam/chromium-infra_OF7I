# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from components import auth
from parameterized import parameterized
from testing_utils import testing
import mock

from google.appengine.ext import ndb

from go.chromium.org.luci.buildbucket.proto import project_config_pb2
from test import test_util
from test.test_util import future
import config
import errors
import user

# Shortcuts
Bucket = project_config_pb2.Bucket
Acl = project_config_pb2.Acl


def perms_for_role(role):
  return frozenset(
      perm for perm, min_role in user.PERM_TO_MIN_ROLE.items()
      if role >= min_role
  )


class UserTest(testing.AppengineTestCase):

  def setUp(self):
    super(UserTest, self).setUp()
    self.current_identity = auth.Identity.from_bytes('user:a@example.com')
    self.patch(
        'components.auth.get_current_identity',
        autospec=True,
        side_effect=lambda: self.current_identity
    )
    user.clear_request_cache()

    self.patch('components.auth.is_admin', autospec=True, return_value=False)
    self.patch(
        'components.auth.should_enforce_realm_acl',
        autospec=True,
        return_value=False
    )
    self.patch(
        'components.auth.has_permission_dryrun',
        autospec=True,
        return_value=None
    )

    self.perms = {}  # realm -> [(group|identity, set of permissions it has)]

    def has_permission(perm, realms):
      assert isinstance(perm, auth.Permission)
      assert isinstance(realms, list)
      caller = auth.get_current_identity().to_bytes()
      for r in realms:
        for principal, granted in self.perms.get(r, []):
          applies = caller == principal or auth.is_group_member(principal)
          if applies and perm in granted:
            return True
      return False

    self.patch(
        'components.auth.has_permission',
        autospec=True,
        side_effect=has_permission
    )

    config.put_bucket(
        'p1', 'ignored-rev',
        Bucket(
            name='a',
            acls=[
                Acl(role=Acl.WRITER, group='a-writers'),
                Acl(role=Acl.READER, group='a-readers'),
            ]
        )
    )
    self.perms['p1:a'] = [
        ('a-writers', perms_for_role(Acl.WRITER)),
        ('a-readers', perms_for_role(Acl.READER)),
        ('project:p1', perms_for_role(Acl.SCHEDULER)),  # implicit
    ]

    config.put_bucket(
        'p2', 'ignored-rev',
        Bucket(
            name='b',
            acls=[
                Acl(role=Acl.WRITER, group='b-writers'),
                Acl(role=Acl.READER, group='b-readers'),
            ]
        )
    )
    self.perms['p2:b'] = [
        ('b-writers', perms_for_role(Acl.WRITER)),
        ('b-readers', perms_for_role(Acl.READER)),
        ('project:p2', perms_for_role(Acl.SCHEDULER)),  # implicit
    ]

    config.put_bucket(
        'p3', 'ignored-rev',
        Bucket(
            name='c',
            acls=[
                Acl(role=Acl.READER, group='c-readers'),
                Acl(role=Acl.READER, identity='user:a@example.com'),
                Acl(role=Acl.WRITER, group='c-writers'),
                Acl(role=Acl.READER, identity='project:p1'),
            ]
        )
    )
    self.perms['p3:c'] = [
        ('c-readers', perms_for_role(Acl.READER)),
        ('user:a@example.com', perms_for_role(Acl.READER)),
        ('c-writers', perms_for_role(Acl.WRITER)),
        ('project:p1', perms_for_role(Acl.READER)),
        ('project:p3', perms_for_role(Acl.SCHEDULER)),  # implicit
    ]

  def get_role(self, bucket_id):
    return user.get_role_async_deprecated(bucket_id).get_result()

  @mock.patch('components.auth.is_group_member', autospec=True)
  def test_get_role_deprecated(self, is_group_member):
    is_group_member.side_effect = lambda g, _=None: g == 'a-writers'

    self.assertEqual(self.get_role('p1/a'), Acl.WRITER)
    self.assertEqual(self.get_role('p2/a'), None)
    self.assertEqual(self.get_role('p3/c'), Acl.READER)
    self.assertEqual(self.get_role('p1/non.existing'), None)

    # Memcache test.
    user.clear_request_cache()
    self.assertEqual(self.get_role('p1/a'), Acl.WRITER)

  def test_get_role_admin_deprecated(self):
    auth.is_admin.return_value = True
    self.assertEqual(self.get_role('p1/a'), Acl.WRITER)
    self.assertEqual(self.get_role('p1/non.existing'), None)

  @mock.patch('components.auth.is_group_member', autospec=True)
  def test_get_role_for_project_deprecated(self, is_group_member):
    is_group_member.side_effect = lambda g, _=None: False

    self.current_identity = auth.Identity.from_bytes('project:p1')
    self.assertEqual(self.get_role('p1/a'), Acl.WRITER)  # implicit
    self.assertEqual(self.get_role('p2/b'), None)  # no roles at all
    self.assertEqual(self.get_role('p3/c'), Acl.READER)  # via explicit ACL

  @parameterized.expand([
      (user.PERM_BUILDS_GET, False, ['a-readers'], {'p1/a', 'p3/c'}),
      (user.PERM_BUILDS_ADD, False, ['b-writers'], {'p2/b'}),
      (user.PERM_BUILDS_GET, True, ['a-readers'], {'p1/a', 'p3/c'}),
      (user.PERM_BUILDS_ADD, True, ['b-writers'], {'p2/b'}),
  ])
  @mock.patch('components.auth.is_group_member', autospec=True)
  def test_buckets_by_perm_async(
      self, perm, use_realms, groups, expected, is_group_member
  ):
    auth.should_enforce_realm_acl.return_value = use_realms
    is_group_member.side_effect = lambda g, _=None: g in groups

    # Cold caches.
    buckets = user.buckets_by_perm_async(perm).get_result()
    self.assertEqual(buckets, expected)

    # Test coverage of ndb.Future caching.
    buckets = user.buckets_by_perm_async(perm).get_result()
    self.assertEqual(buckets, expected)

    # Memcache coverage.
    user.clear_request_cache()
    buckets = user.buckets_by_perm_async(perm).get_result()
    self.assertEqual(buckets, expected)

  @parameterized.expand([
      (False,),
      (True,),
  ])
  @mock.patch('components.auth.is_group_member', autospec=True)
  def test_buckets_by_perm_async_for_project(self, use_realms, is_group_member):
    auth.should_enforce_realm_acl.return_value = use_realms
    is_group_member.side_effect = lambda g, _=None: False
    self.current_identity = auth.Identity.from_bytes('project:p1')

    buckets = user.buckets_by_perm_async(user.PERM_BUILDS_GET).get_result()
    self.assertEqual(
        buckets,
        {
            'p1/a',  # implicit by being in the same project
            'p3/c',  # explicitly set in acls {...}, see setUp()
        }
    )

  def mock_role(self, role):
    self.patch('user.get_role_async_deprecated', return_value=future(role))

  def test_has_perm_deprecated(self):
    self.mock_role(Acl.READER)

    self.assertTrue(user.has_perm(user.PERM_BUILDS_GET, 'proj/bucket'))
    auth.has_permission_dryrun.assert_called_with(
        user.PERM_BUILDS_GET,
        ['proj:bucket'],
        expected_result=True,
        tracking_bug='crbug.com/1091604',
    )

    self.assertFalse(user.has_perm(user.PERM_BUILDS_CANCEL, 'proj/bucket'))
    self.assertFalse(user.has_perm(user.PERM_BUILDERS_SET_NUM, 'proj/bucket'))

    # Memcache coverage
    self.assertFalse(user.has_perm(user.PERM_BUILDERS_SET_NUM, 'proj/bucket'))

  def test_has_perm_no_roles_deprecated(self):
    self.mock_role(None)
    for perm in user.PERM_TO_MIN_ROLE:
      self.assertFalse(user.has_perm(perm, 'proj/bucket'))

  @mock.patch('components.auth.is_group_member', autospec=True)
  def test_has_perm(self, is_group_member):
    auth.should_enforce_realm_acl.return_value = True

    is_group_member.side_effect = lambda g: g == 'a-readers'
    self.assertTrue(user.has_perm(user.PERM_BUILDS_GET, 'p1/a'))
    self.assertFalse(user.has_perm(user.PERM_BUILDS_ADD, 'p1/a'))
    self.assertFalse(user.has_perm(user.PERM_BUILDS_GET, 'p2/b'))

    is_group_member.side_effect = lambda g: g == 'a-writers'
    self.assertTrue(user.has_perm(user.PERM_BUILDS_GET, 'p1/a'))
    self.assertTrue(user.has_perm(user.PERM_BUILDS_ADD, 'p1/a'))
    self.assertFalse(user.has_perm(user.PERM_BUILDS_GET, 'p2/b'))

  def test_has_perm_bad_input(self):
    with self.assertRaises(errors.InvalidInputError):
      bid = 'bad project id/bucket'
      user.has_perm(user.PERM_BUILDS_GET, bid)
    with self.assertRaises(errors.InvalidInputError):
      bid = 'project_id/bad bucket name'
      user.has_perm(user.PERM_BUILDS_GET, bid)

  @mock.patch('user.get_role_async_deprecated')
  def test_filter_buckets_by_perm(self, get_role_async):
    get_role_async.side_effect = lambda bid: future({
        'p/read': Acl.READER,
        'p/read-sched': Acl.SCHEDULER,
        'p/read-sched-write': Acl.WRITER,
    }.get(bid))

    all_buckets = ['p/read', 'p/read-sched', 'p/read-sched-write', 'p/unknown']

    filtered = user.filter_buckets_by_perm(user.PERM_BUILDS_GET, all_buckets)
    self.assertEqual(filtered, {'p/read', 'p/read-sched', 'p/read-sched-write'})

    filtered = user.filter_buckets_by_perm(user.PERM_BUILDS_ADD, all_buckets)
    self.assertEqual(filtered, {'p/read-sched', 'p/read-sched-write'})

    filtered = user.filter_buckets_by_perm(user.PERM_BUILDS_LEASE, all_buckets)
    self.assertEqual(filtered, {'p/read-sched-write'})

  def test_permitted_actions_async_no_roles(self):
    self.mock_role(None)
    self.assertEqual(
        user.permitted_actions_async('project/bucket').get_result(),
        (),
    )

  def test_permitted_actions_async_some_role(self):
    self.mock_role(Acl.READER)
    self.assertEqual(
        user.permitted_actions_async('project/bucket').get_result(),
        (
            user.Action.VIEW_BUILD, user.Action.SEARCH_BUILDS,
            user.Action.ACCESS_BUCKET
        ),
    )

  @mock.patch('user.auth.delegate_async', autospec=True)
  def test_delegate_async(self, delegate_async):
    delegate_async.return_value = future('token')
    token = user.delegate_async(
        'swarming.example.com', tag='buildbucket:bucket:x'
    ).get_result()
    self.assertEqual(token, 'token')
    delegate_async.assert_called_with(
        audience=[user.self_identity()],
        services=['https://swarming.example.com'],
        impersonate=auth.get_current_identity(),
        tags=['buildbucket:bucket:x'],
    )

  def test_parse_identity(self):
    self.assertEqual(
        user.parse_identity('user:a@example.com'),
        auth.Identity('user', 'a@example.com'),
    )
    self.assertEqual(
        auth.Identity('user', 'a@example.com'),
        auth.Identity('user', 'a@example.com'),
    )

    self.assertEqual(
        user.parse_identity('a@example.com'),
        auth.Identity('user', 'a@example.com'),
    )

    with self.assertRaises(errors.InvalidInputError):
      user.parse_identity('a:b')


class GetOrCreateCachedFutureTest(testing.AppengineTestCase):
  maxDiff = None

  def test_unfinished_future_in_different_context(self):
    # This test essentially asserts ndb behavior that we assume in
    # user._get_or_create_cached_future.

    ident1 = auth.Identity.from_bytes('user:1@example.com')
    ident2 = auth.Identity.from_bytes('user:2@example.com')

    # First define a correct async function that uses caching.
    log = []

    @ndb.tasklet
    def compute_async(x):
      log.append('compute_async(%r) started' % x)
      yield ndb.sleep(0.001)
      log.append('compute_async(%r) finishing' % x)
      raise ndb.Return(x)

    def compute_cached_async(x):
      log.append('compute_cached_async(%r)' % x)
      # Use different identities to make sure _get_or_create_cached_future is
      # OK with that.
      ident = ident1 if x % 2 else ident2
      return user._get_or_create_cached_future(
          ident, x, lambda: compute_async(x)
      )

    # Now call compute_cached_async a few times, but stop on the first result,
    # and exit the current ndb context leaving remaining futures unfinished.

    class Error(Exception):
      pass

    with self.assertRaises(Error):
      # This code is intentionally looks realistic.
      futures = [compute_cached_async(x) for x in xrange(5)]
      for f in futures:  # pragma: no branch
        f.get_result()
        log.append('got result')
        # Something bad happened during processing.
        raise Error()

    # Assert that only first compute_async finished.
    self.assertEqual(
        log,
        [
            'compute_cached_async(0)',
            'compute_cached_async(1)',
            'compute_cached_async(2)',
            'compute_cached_async(3)',
            'compute_cached_async(4)',
            'compute_async(0) started',
            'compute_async(1) started',
            'compute_async(2) started',
            'compute_async(3) started',
            'compute_async(4) started',
            'compute_async(0) finishing',
            'got result',
        ],
    )
    log[:] = []

    # Now we assert that waiting for another future, continues execution.
    self.assertEqual(compute_cached_async(3).get_result(), 3)
    self.assertEqual(
        log,
        [
            'compute_cached_async(3)',
            'compute_async(1) finishing',
            'compute_async(2) finishing',
            'compute_async(3) finishing',
        ],
    )
    # Note that compute_async(4) didin't finish.
