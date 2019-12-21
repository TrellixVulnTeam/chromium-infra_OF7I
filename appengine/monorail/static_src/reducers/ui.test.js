// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import * as ui from './ui.js';


describe('ui', () => {
  describe('reducers', () => {
    describe('snackbarsReducer', () => {
      it('adds snackbar', () => {
        let state = ui.snackbarsReducer([],
            {type: 'SHOW_SNACKBAR', id: 'one', text: 'A snackbar'});

        assert.deepEqual(state, [{id: 'one', text: 'A snackbar'}]);

        state = ui.snackbarsReducer(state,
            {type: 'SHOW_SNACKBAR', id: 'two', text: 'Another snack'});

        assert.deepEqual(state, [
          {id: 'one', text: 'A snackbar'},
          {id: 'two', text: 'Another snack'},
        ]);
      });

      it('removes snackbar', () => {
        let state = [
          {id: 'one', text: 'A snackbar'},
          {id: 'two', text: 'Another snack'},
        ];

        state = ui.snackbarsReducer(state,
            {type: 'HIDE_SNACKBAR', id: 'one'});

        assert.deepEqual(state, [
          {id: 'two', text: 'Another snack'},
        ]);

        state = ui.snackbarsReducer(state,
            {type: 'HIDE_SNACKBAR', id: 'two'});

        assert.deepEqual(state, []);
      });

      it('does not remove non-existent snackbar', () => {
        let state = [
          {id: 'one', text: 'A snackbar'},
          {id: 'two', text: 'Another snack'},
        ];

        state = ui.snackbarsReducer(state,
            {action: 'HIDE_SNACKBAR', id: 'whatever'});

        assert.deepEqual(state, [
          {id: 'one', text: 'A snackbar'},
          {id: 'two', text: 'Another snack'},
        ]);
      });
    });
  });

  describe('selectors', () => {
    it('snackbars', () => {
      assert.deepEqual(ui.snackbars({ui: {snackbars: []}}), []);
      assert.deepEqual(ui.snackbars({ui: {snackbars: [
        {text: 'Snackbar one', id: 'one'},
        {text: 'Snackbar two', id: 'two'},
      ]}}), [
        {text: 'Snackbar one', id: 'one'},
        {text: 'Snackbar two', id: 'two'},
      ]);
    });
  });

  describe('action creators', () => {
    describe('showSnackbar', () => {
      it('produces action', () => {
        const action = ui.showSnackbar('id', 'text');
        const dispatch = sinon.stub();

        action(dispatch);

        sinon.assert.calledWith(dispatch,
            {type: 'SHOW_SNACKBAR', text: 'text', id: 'id'});
      });

      it('hides snackbar after timeout', () => {
        const clock = sinon.useFakeTimers(0);

        const action = ui.showSnackbar('id', 'text', 1000);
        const dispatch = sinon.stub();

        action(dispatch);

        sinon.assert.neverCalledWith(dispatch,
            {type: 'HIDE_SNACKBAR', id: 'id'});

        clock.tick(1000);

        sinon.assert.calledWith(dispatch, {type: 'HIDE_SNACKBAR', id: 'id'});

        clock.restore();
      });

      it('does not setTimeout when no timeout specified', () => {
        sinon.stub(window, 'setTimeout');

        ui.showSnackbar('id', 'text', 0);

        sinon.assert.notCalled(window.setTimeout);

        window.setTimeout.restore();
      });
    });

    it('hideSnackbar produces action', () => {
      assert.deepEqual(ui.hideSnackbar('one'),
          {type: 'HIDE_SNACKBAR', id: 'one'});
    });
  });
});
