// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import * as hotlist from './hotlist.js';
import {hotlistRefExample, hotlistExample, hotlistItemExample, hotlistsExample,
  hotlistItemsExample} from 'shared/test/hotlist-constants.js';
import {prpcClient} from 'prpc-client-instance.js';

let dispatch;

describe('hotlist', () => {
  describe('reducers', () => {
    it('root reducer initial state', () => {
      const actual = hotlist.reducer(undefined, {type: null});
      const expected = {
        hotlists: {},
        hotlistItems: {},
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

    it('hotlists updates on FETCH_SUCCESS', () => {
      const action = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
      const actual = hotlist.hotlistsReducer({}, action);
      assert.deepEqual(actual, hotlistsExample);
    });

    it('hotlistItems updates on FETCH_ITEMS_SUCCESS', () => {
      const action = {
        type: hotlist.FETCH_ITEMS_SUCCESS,
        hotlistRef: hotlistRefExample,
        items: [hotlistItemExample],
      };
      const actual = hotlist.hotlistItemsReducer({}, action);
      assert.deepEqual(actual, hotlistItemsExample);
    });

    describe('hotlistRef', () => {
      it('updates on SELECT', () => {
        const action = {type: hotlist.SELECT, hotlistRef: hotlistRefExample};
        const actual = hotlist.hotlistRefReducer(null, action);
        assert.deepEqual(actual, hotlistRefExample);
      });

      it('updates on FETCH_SUCCESS', () => {
        const state = {owner: {userId: 12345678}, name: 'Hotlist-Name'};
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
        const actual = hotlist.hotlistRefReducer(state, action);
        assert.deepEqual(actual, hotlistRefExample);
      });

      it('doesn\'t update on FETCH_SUCCESS if different hotlist name', () => {
        const state = {owner: {userId: 12345678}, name: 'Another-Hotlist-Name'};
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
        assert.deepEqual(hotlist.hotlistRefReducer(state, action), state);
      });

      it('doesn\'t update on FETCH_SUCCESS if different hotlist owner', () => {
        const state = {owner: {userId: 87654321}, name: 'Hotlist-Name'};
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
        assert.deepEqual(hotlist.hotlistRefReducer(state, action), state);
      });

      it('doesn\'t update on FETCH_SUCCESS if refs are exactly equal', () => {
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
        const actual = hotlist.hotlistRefReducer(hotlistRefExample, action);
        assert.deepEqual(actual, hotlistRefExample);
      });

      it('doesn\'t update on FETCH_SUCCESS if null', () => {
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
        assert.deepEqual(hotlist.hotlistRefReducer(null, action), null);
      });
    });
  });

  describe('selectors', () => {
    it('hotlists', () => {
      const state = {hotlist: {hotlists: hotlistsExample}};
      assert.deepEqual(hotlist.hotlists(state), hotlistsExample);
    });

    it('hotlistItems', () => {
      const state = {hotlist: {hotlistItems: hotlistItemsExample}};
      assert.deepEqual(hotlist.hotlistItems(state), hotlistItemsExample);
    });

    it('hotlistRef', () => {
      const state = {hotlist: {hotlistRef: hotlistRefExample}};
      assert.deepEqual(hotlist.hotlistRef(state), hotlistRefExample);
    });

    describe('viewedHotlist', () => {
      it('normal case', () => {
        const state = {hotlist: {
          hotlists: hotlistsExample,
          hotlistRef: hotlistRefExample,
        }};
        assert.deepEqual(hotlist.viewedHotlist(state), hotlistExample);
      });

      it('no hotlistRef', () => {
        const state = {hotlist: {hotlists: hotlistsExample, hotlistRef: null}};
        assert.deepEqual(hotlist.viewedHotlist(state), null);
      });

      it('hotlist not found', () => {
        const state = {hotlist: {hotlists: {}, hotlistRef: hotlistRefExample}};
        assert.deepEqual(hotlist.viewedHotlist(state), null);
      });
    });

    describe('viewedHotlistItems', () => {
      it('normal case', () => {
        const state = {hotlist: {
          hotlistItems: hotlistItemsExample,
          hotlistRef: hotlistRefExample,
        }};
        const actual = hotlist.viewedHotlistItems(state);
        assert.deepEqual(actual, [hotlistItemExample]);
      });

      it('no hotlistRef', () => {
        const state = {hotlist: {
          hotlistItems: hotlistItemsExample,
          hotlistRef: null,
        }};
        assert.deepEqual(hotlist.viewedHotlistItems(state), []);
      });

      it('hotlist not found', () => {
        const state = {hotlist: {
          hotlistItems: {},
          hotlistRef: hotlistRefExample,
        }};
        assert.deepEqual(hotlist.viewedHotlistItems(state), []);
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

    it('select', () => {
      hotlist.select(hotlistRefExample)(dispatch);
      const action = {type: hotlist.SELECT, hotlistRef: hotlistRefExample};
      sinon.assert.calledWith(dispatch, action);
    });

    describe('fetch', () => {
      it('success', async () => {
        prpcClient.call.returns(Promise.resolve({hotlist: hotlistExample}));

        await hotlist.fetch(hotlistRefExample)(dispatch);

        sinon.assert.calledWith(dispatch, {type: hotlist.FETCH_START});

        const args = {hotlistRef: hotlistRefExample};
        sinon.assert.calledWith(
            prpcClient.call, 'monorail.Features', 'GetHotlist', args);

        const action = {type: hotlist.FETCH_SUCCESS, hotlist: hotlistExample};
        sinon.assert.calledWith(dispatch, action);
      });

      it('failure', async () => {
        prpcClient.call.throws();

        await hotlist.fetch(hotlistRefExample)(dispatch);

        const action = {type: hotlist.FETCH_FAILURE, error: sinon.match.any};
        sinon.assert.calledWith(dispatch, action);
      });
    });

    describe('fetchItems', () => {
      it('success', async () => {
        prpcClient.call.returns(Promise.resolve({items: [hotlistItemExample]}));

        await hotlist.fetchItems(hotlistRefExample)(dispatch);

        sinon.assert.calledWith(dispatch, {type: hotlist.FETCH_ITEMS_START});

        const args = {hotlistRef: hotlistRefExample};
        sinon.assert.calledWith(
            prpcClient.call, 'monorail.Features', 'ListHotlistItems', args);

        const action = {
          type: hotlist.FETCH_ITEMS_SUCCESS,
          hotlistRef: hotlistRefExample,
          items: [hotlistItemExample],
        };
        sinon.assert.calledWith(dispatch, action);
      });

      it('failure', async () => {
        prpcClient.call.throws();

        await hotlist.fetchItems(hotlistRefExample)(dispatch);

        const action = {
          type: hotlist.FETCH_ITEMS_FAILURE,
          error: sinon.match.any,
        };
        sinon.assert.calledWith(dispatch, action);
      });
    });
  });
});
