# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest

from chromeperf.pinpoint.models import exploration


def find_midpoint(a, b):
    offset = (b - a) // 2
    if offset == 0:
        return None
    return a + offset


def change_always_detected(*_):
    return True


def test_speculate_empty():
    results = exploration.speculate(
        [],
        change_detected=lambda *_: False,
        on_unknown=lambda *args: pytest.fail(f"on_unknown called with {args}"),
        midpoint=lambda *_: None,
        levels=100)
    assert results == []


def test_speculate_odd():
    changes = [1, 6]

    results = exploration.speculate(
        changes,
        change_detected=change_always_detected,
        on_unknown=lambda *args: pytest.fail(f"on_unknown called with {args}"),
        midpoint=find_midpoint,
        levels=2)
    for index, change in results:
        changes.insert(index, change)
    assert changes == [1, 2, 3, 4, 6]


def test_speculate_even():
    changes = [0, 100]

    results = exploration.speculate(
        changes,
        change_detected=change_always_detected,
        on_unknown=lambda *args: pytest.fail(f"on_unknown called with {args}"),
        midpoint=find_midpoint,
        levels=2)
    for index, change in results:
        changes.insert(index, change)
    assert changes == [0, 25, 50, 75, 100]


def test_speculate_unbalanced():
    changes = [0, 9, 100]

    results = exploration.speculate(
        changes,
        change_detected=change_always_detected,
        on_unknown=lambda *args: pytest.fail(f"on_unknown called with {args}"),
        midpoint=find_midpoint,
        levels=2)
    for index, change in results:
        changes.insert(index, change)
    assert changes == [0, 2, 4, 6, 9, 31, 54, 77, 100]


def test_speculate_iterations(mocker):
    on_unknown_mock = mocker.MagicMock()
    changes = [0, 10]

    results = exploration.speculate(
        changes,
        change_detected=change_always_detected,
        on_unknown=on_unknown_mock,
        midpoint=find_midpoint,
        levels=2)
    for index, change in results:
        changes.insert(index, change)
    assert changes == [0, 2, 5, 7, 10]

    # Run the bisection again and get the full range.
    results = exploration.speculate(
        changes,
        change_detected=change_always_detected,
        on_unknown=lambda *args: pytest.fail(f"on_unknown called with {args}"),
        midpoint=find_midpoint,
        levels=2)
    for index, change in results:
        changes.insert(index, change)
    assert changes == list(range(11))


def test_speculate_handle_unknown(mocker):
    on_unknown_mock = mocker.MagicMock()
    changes = [0, 5, 10]

    def change_unknown_detected(a, _):
        if a >= 5:
            return None
        return True

    results = exploration.speculate(
        changes,
        change_detected=change_unknown_detected,
        on_unknown=on_unknown_mock,
        midpoint=find_midpoint,
        levels=2)
    for index, change in results:
        changes.insert(index, change)
    assert on_unknown_mock.called
    assert changes == [0, 1, 2, 3, 5, 10]


def test_speculate_handle_change_never_detected():
    results = exploration.speculate(
        [0, 1000],
        change_detected=lambda *_: False,
        on_unknown=lambda *args: pytest.fail(f"on_unknown called with {args}"),
        midpoint=find_midpoint,
        levels=2)
    assert list(results) == []
