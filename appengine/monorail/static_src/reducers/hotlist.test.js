// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import * as hotlist from './hotlist.js';
import {hotlistExample, hotlistRefExample,
  hotlistRefStringExample} from 'shared/converters-hotlist.test.js';
import {prpcClient} from 'prpc-client-instance.js';

let dispatch;

const hotlistsExample = {[hotlistRefStringExample]: hotlistExample};

describe('hotlist', () => {
  describe('reducers', () => {
    it('root reducer initial state', () => {
      const actual = hotlist.reducer(undefined, {type: null});
      const expected = {
        hotlists: {},
        hotlistRef: null,
        requests: {
          fetch: {
            error: null,
            requesting: false,
          },
        },
      };
      assert.deepEqual(actual, expected);
    });

    it('hotlists reducer updates on FETCH_SUCCESS', () => {
      const action = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
      const actual = hotlist.hotlistsReducer({}, action);
      assert.deepEqual(actual, hotlistsExample);
    });

    it('hotlistRef reducer updates on SELECT_HOTLIST', () => {
      const action = {
        type: hotlist.SELECT_HOTLIST,
        hotlistRef: hotlistRefExample,
      };
      const actual = hotlist.hotlistRefReducer({}, action);
      assert.deepEqual(actual, hotlistRefExample);
    });
  });

  describe('selectors', () => {
    it('hotlists', () => {
      const state = {hotlist: {hotlists: hotlistsExample}};
      assert.deepEqual(hotlist.hotlists(state), hotlistsExample);
    });

    it('hotlistRef', () => {
      const state = {hotlist: {hotlistRef: hotlistRefExample}};
      assert.deepEqual(hotlist.hotlistRef(state), hotlistRefExample);
    });

    describe('hotlist', () => {
      it('normal case', () => {
        const state = {hotlist: {
          hotlists: hotlistsExample,
          hotlistRef: hotlistRefExample,
        }};
        assert.deepEqual(hotlist.hotlist(state), hotlistExample);
      });

      it('no hotlistRef', () => {
        const state = {hotlist: {hotlists: hotlistsExample, hotlistRef: null}};
        assert.deepEqual(hotlist.hotlist(state), null);
      });

      it('hotlist not found', () => {
        const state = {hotlist: {hotlists: {}, hotlistRef: hotlistRefExample}};
        assert.deepEqual(hotlist.hotlist(state), null);
      });
    });
  });

  describe('action creators', () => {
    beforeEach(() => {
      sinon.stub(prpcClient, 'call');
      dispatch = sinon.stub();
    });

    afterEach(() => {
      prpcClient.call.restore();
    });

    it('selectHotlist', () => {
      hotlist.selectHotlist('example@example.com', 'Hotlist-Name')(dispatch);

      const hotlistRef = {
        owner: {
          displayName: 'example@example.com',
        },
        name: 'Hotlist-Name',
      };
      const action = {type: hotlist.SELECT_HOTLIST, hotlistRef};
      sinon.assert.calledWith(dispatch, action);
    });

    describe('fetch', () => {
      it('success', async () => {
        prpcClient.call.returns(Promise.resolve({hotlist: hotlistExample}));

        const action = hotlist.fetch(hotlistRefExample);
        await(action(dispatch));

        sinon.assert.calledWith(dispatch, {type: hotlist.FETCH_START});

        const returnValue = {hotlistRef: hotlistRefExample};
        sinon.assert.calledWith(
            prpcClient.call, 'monorail.Features', 'GetHotlist', returnValue);

        const args = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
        sinon.assert.calledWith(dispatch, args);
      });

      it('failure', async () => {
        prpcClient.call.throws();

        const action = hotlist.fetch(hotlistRefExample);
        await(action(dispatch));

        const args = {type: hotlist.FETCH_FAILURE, error: sinon.match.any};
        sinon.assert.calledWith(dispatch, args);
      });
    });
  });
});
