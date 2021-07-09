# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style liense that can be
# found in the LICENSE file.
"""Some utilities for concurrency."""

from collections import deque
import contextlib
import multiprocessing
import sys
import threading
import time
import traceback

from . import util


class Pool():
  """Multithreaded work pool. Dispatch tasks via `apply' to be run concurrently.

  This class is similar to the multiprocessing.Pool class, but for threads
  instead of processes. It queues up all work requests and then runs them all
  with a target count of active threads to run in parallel. There is no pooling
  of the threads themselves: one thread is created per task. Hence this isn't
  suitable for applications with a large or unbounded number of small tasks. But
  it works fine for our application of O(100) large IO-bound tasks.
  """

  class Thread(threading.Thread):
    """Pool-specific Thread subclass that communicates status to the pool."""

    def __init__(self, target, args, event, errors):
      super(Pool.Thread, self).__init__()

      self.target = target
      self.args = args
      self.event = event
      self.errors = errors
      self.active = True

    def run(self):
      try:
        self.target(*self.args)
      except:  # pylint: disable=bare-except
        # We disable the pylint check as we really do want to catch absolutely
        # everything here, and propagate back to the main thread.

        # Return an error string encapsulating the traceback and error message
        # from the exception we received, so that the pool can include it in its
        # error message.
        self.errors.append(''.join(traceback.format_exception(*sys.exc_info())))
      finally:
        # We set this member variable rather than relying on Thread.is_active to
        # avoid races where the main thread wakes up in between a worker thread
        # signalling the event and actually exiting.
        self.active = False
        # Signal the pool controller thread that a task has finished.
        self.event.set()

  class TaskException(Exception):
    pass

  def __init__(self, cpus=None):
    self.queued_threads = deque()
    self.active_threads = []
    self.target_active_thread_count = cpus or multiprocessing.cpu_count()
    self.errors = []
    self.event = threading.Event()

  def apply(self, f, args):
    """Add a task which runs f(*args) to the pool.

    This mirrors the multiprocessing.Pool API.
    """
    self.queued_threads.appendleft(
        Pool.Thread(f, args, self.event, self.errors))

  def run_all(self):
    """Run all enqueued tasks, blocking until all have finished.

    If a task throws an exception, it is propagated back to the thread that
    called `run_all', and no new tasks will be launched.
    """

    while True:
      if self.errors:
        error = self.errors[0]
        util.LOGGER.error('Build task failed, exiting')

        # We could consider waiting for all currently active threads to finish
        # before re-raising the error from the child, for a cleaner shutdown. In
        # practice this seems to be fine though.
        #
        # We could also consider retrying failed tasks to mitigate the effects
        # of rare race conditions.
        raise Pool.TaskException(error)

      # There's a possible race here where a thread finishes and sets the event
      # in-between us clearing it and executing the below code, but it's benign
      # as this loop body simply does nothing if all threads are still active.

      self.active_threads = [t for t in self.active_threads if t.active]

      # Start as many threads as necessary to bring the active thread count back
      # up to `target_active_thread_count'.
      new_threads_to_start = min(
          len(self.queued_threads),
          self.target_active_thread_count - len(self.active_threads))
      if new_threads_to_start > 0:
        for _ in range(new_threads_to_start):
          t = self.queued_threads.pop()
          self.active_threads.append(t)
          t.start()

      if not self.active_threads:
        # We've run out of tasks to run.
        break

      self.event.wait()

      # There's a possible race here where another thread also sets the event in
      # between the above wait finishing and us clearing it below. But this is
      # fine - we'll find all finished threads regardless of whether they were
      # the one to signal us.
      self.event.clear()


class KeyedLock():
  """A lock which blocks based on a dynamic set of keys.

  For any given key, this class acts the same as `Lock'. Useful for guarding
  access to resources tied to a particular identifier, for example filesystem
  paths.

  Note that a unique lock is created for each key, and they're never deleted
  until the whole KeyedLock is deleted. As such, this class should only be used
  when the set of keys is small and bounded.
  """

  def __init__(self):
    self.lock = threading.Lock()
    self.lock_dict = {}

  def get(self, key):
    # Lock around this "get-or-insert" operation, to ensure we create a unique
    # lock for every key.
    with self.lock:
      lock = self.lock_dict.get(key)
      if not lock:
        self.lock_dict[key] = lock = threading.Lock()

    return lock


class RWLock():
  """A simple single-writer, multi-reader lock, using a mutex and semaphore.

  This is used to synchronise access to resources where the users fall into two
  classes: reader and writer. Any number of readers can access the resource at
  once, but none can access it while there are any writers.
  """

  def __init__(self):
    self.read_lock = threading.Lock()
    self.rw_semaphore = threading.Semaphore()
    self.count = 0

  @contextlib.contextmanager
  def read(self):
    # Increment the count, while holding the read lock.
    with self.read_lock:
      self.count += 1
      # If we're the first reader, acquire the read-write semaphore. This has
      # the effect of blocking any writers while allowing other readers.
      if self.count == 1:
        self.rw_semaphore.acquire()

    # Critical section. Here we hold the read-write semaphore, if we're the
    # first reader, or nothing if we're a later reader. So during this block
    # we exclude any writers, but not any other readers.
    yield

    # Decrement the count, while holding the read lock.
    with self.read_lock:
      self.count -= 1
      # If we're the last reader, release the read-write semaphore. Now writers
      # will be able to access the resource. Here we see why this must be a
      # semaphore, and not a mutex. If there are multiple readers that finish in
      # a different sequence to which they started, then the final reader to
      # finish may not be the same as the first reader to start.
      if self.count == 0:
        self.rw_semaphore.release()

  def write(self):
    # Writes are simple - just acquire the read-write semaphore. This will block
    # if there are any concurrent readers or writers.
    return self.rw_semaphore

  def shared(self):
    return self.read()

  def exclusive(self):
    return self.write()


# This lock is used to make sure that no processes are spawned while a file is
# open for writing. It's acquired in 'shared' mode by anything opening files,
# and in 'exclusive' mode by System.run around spawning processes.
#
# On systems using `fork' to spawn new processes, the subprocess will
# inherit any open file handles from the parent. If a process is spawned while
# we have files open, that process will keep the file handle open much longer
# than it should, which can cause issues with:
#
# * Multiple processes trying to write to the same file, which Windows seems not
#   to like.
# * Keeping temp files open when we're trying to delete them.
# * Having binaries open in write mode when we try to execute them, which isn't
#   allowed.
PROCESS_SPAWN_LOCK = RWLock()
