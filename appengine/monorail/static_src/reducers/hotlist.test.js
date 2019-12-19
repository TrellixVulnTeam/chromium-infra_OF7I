// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import * as hotlist from './hotlist.js';
import * as example from 'shared/test/constants-hotlist.js';
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
      const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
      const actual = hotlist.hotlistsReducer({}, action);
      assert.deepEqual(actual, example.HOTLISTS);
    });

    it('hotlistItems updates on FETCH_ITEMS_SUCCESS', () => {
      const action = {
        type: hotlist.FETCH_ITEMS_SUCCESS,
        hotlistRef: example.HOTLIST_REF,
        items: [example.HOTLIST_ITEM],
      };
      const actual = hotlist.hotlistItemsReducer({}, action);
      assert.deepEqual(actual, example.HOTLIST_ITEMS);
    });

    describe('hotlistRef', () => {
      it('updates on SELECT', () => {
        const action = {type: hotlist.SELECT, hotlistRef: example.HOTLIST_REF};
        const actual = hotlist.hotlistRefReducer(null, action);
        assert.deepEqual(actual, example.HOTLIST_REF);
      });

      it('updates on FETCH_SUCCESS', () => {
        const state = {owner: {userId: 12345678}, name: 'Hotlist-Name'};
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
        const actual = hotlist.hotlistRefReducer(state, action);
        assert.deepEqual(actual, example.HOTLIST_REF);
      });

      it('doesn\'t update on FETCH_SUCCESS if different hotlist name', () => {
        const state = {owner: {userId: 12345678}, name: 'Another-Hotlist-Name'};
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
        assert.deepEqual(hotlist.hotlistRefReducer(state, action), state);
      });

      it('doesn\'t update on FETCH_SUCCESS if different hotlist owner', () => {
        const state = {owner: {userId: 87654321}, name: 'Hotlist-Name'};
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
        assert.deepEqual(hotlist.hotlistRefReducer(state, action), state);
      });

      it('doesn\'t update on FETCH_SUCCESS if refs are exactly equal', () => {
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
        const actual = hotlist.hotlistRefReducer(example.HOTLIST_REF, action);
        assert.deepEqual(actual, example.HOTLIST_REF);
      });

      it('doesn\'t update on FETCH_SUCCESS if null', () => {
        const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
        assert.deepEqual(hotlist.hotlistRefReducer(null, action), null);
      });
    });
  });

  describe('selectors', () => {
    it('hotlists', () => {
      const state = {hotlist: {hotlists: example.HOTLISTS}};
      assert.deepEqual(hotlist.hotlists(state), example.HOTLISTS);
    });

    it('hotlistItems', () => {
      const state = {hotlist: {hotlistItems: example.HOTLIST_ITEMS}};
      assert.deepEqual(hotlist.hotlistItems(state), example.HOTLIST_ITEMS);
    });

    it('hotlistRef', () => {
      const state = {hotlist: {hotlistRef: example.HOTLIST_REF}};
      assert.deepEqual(hotlist.hotlistRef(state), example.HOTLIST_REF);
    });

    describe('viewedHotlist', () => {
      it('normal case', () => {
        const state = {hotlist: {
          hotlists: example.HOTLISTS,
          hotlistRef: example.HOTLIST_REF,
        }};
        assert.deepEqual(hotlist.viewedHotlist(state), example.HOTLIST);
      });

      it('no hotlistRef', () => {
        const state = {hotlist: {hotlists: example.HOTLISTS, hotlistRef: null}};
        assert.deepEqual(hotlist.viewedHotlist(state), null);
      });

      it('hotlist not found', () => {
        const state = {hotlist: {
          hotlists: {},
          hotlistRef: example.HOTLIST_REF,
        }};
        assert.deepEqual(hotlist.viewedHotlist(state), null);
      });
    });

    describe('viewedHotlistItems', () => {
      it('normal case', () => {
        const state = {hotlist: {
          hotlistItems: example.HOTLIST_ITEMS,
          hotlistRef: example.HOTLIST_REF,
        }};
        const actual = hotlist.viewedHotlistItems(state);
        assert.deepEqual(actual, [example.HOTLIST_ITEM]);
      });

      it('no hotlistRef', () => {
        const state = {hotlist: {
          hotlistItems: example.HOTLIST_ITEMS,
          hotlistRef: null,
        }};
        assert.deepEqual(hotlist.viewedHotlistItems(state), []);
      });

      it('hotlist not found', () => {
        const state = {hotlist: {
          hotlistItems: {},
          hotlistRef: example.HOTLIST_REF,
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
      hotlist.select(example.HOTLIST_REF)(dispatch);
      const action = {type: hotlist.SELECT, hotlistRef: example.HOTLIST_REF};
      sinon.assert.calledWith(dispatch, action);
    });

    describe('fetch', () => {
      it('success', async () => {
        prpcClient.call.returns(Promise.resolve({hotlist: example.HOTLIST}));

        await hotlist.fetch(example.HOTLIST_REF)(dispatch);

        sinon.assert.calledWith(dispatch, {type: hotlist.FETCH_START});

        const args = {hotlistRef: example.HOTLIST_REF};
        sinon.assert.calledWith(
            prpcClient.call, 'monorail.Features', 'GetHotlist', args);

        const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
        sinon.assert.calledWith(dispatch, action);
      });

      it('failure', async () => {
        prpcClient.call.throws();

        await hotlist.fetch(example.HOTLIST_REF)(dispatch);

        const action = {type: hotlist.FETCH_FAILURE, error: sinon.match.any};
        sinon.assert.calledWith(dispatch, action);
      });
    });

    describe('fetchItems', () => {
      it('success', async () => {
        const response = {items: [example.HOTLIST_ITEM]};
        prpcClient.call.returns(Promise.resolve(response));

        await hotlist.fetchItems(example.HOTLIST_REF)(dispatch);

        sinon.assert.calledWith(dispatch, {type: hotlist.FETCH_ITEMS_START});

        const args = {hotlistRef: example.HOTLIST_REF};
        sinon.assert.calledWith(
            prpcClient.call, 'monorail.Features', 'ListHotlistItems', args);

        const action = {
          type: hotlist.FETCH_ITEMS_SUCCESS,
          hotlistRef: example.HOTLIST_REF,
          items: [example.HOTLIST_ITEM],
        };
        sinon.assert.calledWith(dispatch, action);
      });

      it('failure', async () => {
        prpcClient.call.throws();

        await hotlist.fetchItems(example.HOTLIST_REF)(dispatch);

        const action = {
          type: hotlist.FETCH_ITEMS_FAILURE,
          error: sinon.match.any,
        };
        sinon.assert.calledWith(dispatch, action);
      });
    });
  });
});
