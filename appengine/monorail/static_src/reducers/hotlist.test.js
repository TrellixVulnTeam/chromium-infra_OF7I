// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as hotlist from './hotlist.js';
import * as example from 'shared/test/constants-hotlist.js';
import * as exampleIssue from 'shared/test/constants-issue.js';

import {prpcClient} from 'prpc-client-instance.js';

let dispatch;

describe('hotlist reducers', () => {
  it('root reducer initial state', () => {
    const actual = hotlist.reducer(undefined, {type: null});
    const expected = {
      name: null,
      hotlists: {},
      hotlistItems: {},
      requests: {
        fetch: {
          error: null,
          requesting: false,
        },
        fetchItems: {
          error: null,
          requesting: false,
        },
        removeItems: {
          error: null,
          requesting: false,
        },
        rerankItems: {
          error: null,
          requesting: false,
        },
      },
    };
    assert.deepEqual(actual, expected);
  });

  it('name updates on SELECT', () => {
    const action = {type: hotlist.SELECT, name: example.NAME};
    const actual = hotlist.nameReducer(null, action);
    assert.deepEqual(actual, example.NAME);
  });

  it('hotlists updates on FETCH_SUCCESS', () => {
    const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
    const actual = hotlist.hotlistsReducer({}, action);
    assert.deepEqual(actual, example.HOTLISTS);
  });

  it('hotlistItems updates on FETCH_ITEMS_SUCCESS', () => {
    const action = {
      type: hotlist.FETCH_ITEMS_SUCCESS,
      name: example.NAME,
      items: [example.HOTLIST_ITEM],
    };
    const actual = hotlist.hotlistItemsReducer({}, action);
    assert.deepEqual(actual, example.HOTLIST_ITEMS);
  });
});

describe('hotlist selectors', () => {
  it('name', () => {
    const state = {hotlist: {name: example.NAME}};
    assert.deepEqual(hotlist.name(state), example.NAME);
  });

  it('hotlists', () => {
    const state = {hotlist: {hotlists: example.HOTLISTS}};
    assert.deepEqual(hotlist.hotlists(state), example.HOTLISTS);
  });

  it('hotlistItems', () => {
    const state = {hotlist: {hotlistItems: example.HOTLIST_ITEMS}};
    assert.deepEqual(hotlist.hotlistItems(state), example.HOTLIST_ITEMS);
  });

  describe('viewedHotlist', () => {
    it('normal case', () => {
      const state = {hotlist: {
        name: example.NAME,
        hotlists: example.HOTLISTS,
      }};
      assert.deepEqual(hotlist.viewedHotlist(state), example.HOTLIST);
    });

    it('no name', () => {
      const state = {hotlist: {name: null, hotlists: example.HOTLISTS}};
      assert.deepEqual(hotlist.viewedHotlist(state), null);
    });

    it('hotlist not found', () => {
      const state = {hotlist: {name: example.NAME, hotlists: {}}};
      assert.deepEqual(hotlist.viewedHotlist(state), null);
    });
  });

  describe('viewedHotlistItems', () => {
    it('normal case', () => {
      const state = {hotlist: {
        name: example.NAME,
        hotlistItems: example.HOTLIST_ITEMS,
      }};
      const actual = hotlist.viewedHotlistItems(state);
      assert.deepEqual(actual, [example.HOTLIST_ITEM]);
    });

    it('no name', () => {
      const state = {hotlist: {
        name: null,
        hotlistItems: example.HOTLIST_ITEMS,
      }};
      assert.deepEqual(hotlist.viewedHotlistItems(state), []);
    });

    it('hotlist not found', () => {
      const state = {hotlist: {name: example.NAME, hotlistItems: {}}};
      assert.deepEqual(hotlist.viewedHotlistItems(state), []);
    });
  });

  describe('viewedHotlistIssues', () => {
    it('normal case', () => {
      const state = {
        hotlist: {
          name: example.NAME,
          hotlistItems: example.HOTLIST_ITEMS,
        },
        issue: {
          issuesByRefString: {
            [exampleIssue.ISSUE_REF_STRING]: exampleIssue.ISSUE,
          },
        },
      };
      const actual = hotlist.viewedHotlistIssues(state);
      assert.deepEqual(actual, [example.HOTLIST_ISSUE]);
    });

    it('no issue', () => {
      const state = {
        hotlist: {
          name: example.NAME,
          hotlistItems: example.HOTLIST_ITEMS,
        },
        issue: {
          issuesByRefString: {
            [exampleIssue.ISSUE_OTHER_PROJECT_REF_STRING]: exampleIssue.ISSUE,
          },
        },
      };
      assert.deepEqual(hotlist.viewedHotlistIssues(state), []);
    });
  });
});

describe('hotlist action creators', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');
    dispatch = sinon.stub();
  });

  afterEach(() => {
    prpcClient.call.restore();
  });

  it('select', () => {
    const actual = hotlist.select(example.NAME);
    const expected = {type: hotlist.SELECT, name: example.NAME};
    assert.deepEqual(actual, expected);
  });

  describe('fetch', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve(example.HOTLIST));

      await hotlist.fetch(example.NAME)(dispatch);

      sinon.assert.calledWith(dispatch, {type: hotlist.FETCH_START});

      const args = {name: example.NAME};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Hotlists', 'GetHotlist', args);

      const action = {type: hotlist.FETCH_SUCCESS, hotlist: example.HOTLIST};
      sinon.assert.calledWith(dispatch, action);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlist.fetch(example.NAME)(dispatch);

      const action = {type: hotlist.FETCH_FAILURE, error: sinon.match.any};
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('fetchItems', () => {
    it('success', async () => {
      const response = {items: [example.HOTLIST_ITEM]};
      prpcClient.call.returns(Promise.resolve(response));

      await hotlist.fetchItems(example.NAME)(dispatch);

      sinon.assert.calledWith(dispatch, {type: hotlist.FETCH_ITEMS_START});

      const args = {parent: example.NAME, orderBy: 'rank'};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Hotlists', 'ListHotlistItems', args);

      const action = {
        type: hotlist.FETCH_ITEMS_SUCCESS,
        name: example.NAME,
        items: [{...example.HOTLIST_ITEM, rank: 0}],
      };
      sinon.assert.calledWith(dispatch, action);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlist.fetchItems(example.NAME)(dispatch);

      const action = {
        type: hotlist.FETCH_ITEMS_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('removeItems', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve({}));

      const issues = [exampleIssue.NAME];
      await hotlist.removeItems(example.NAME, issues)(dispatch);

      sinon.assert.calledWith(dispatch, {type: hotlist.REMOVE_ITEMS_START});

      const args = {parent: example.NAME, issues};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Hotlists',
          'RemoveHotlistItems', args);

      sinon.assert.calledWith(dispatch, {type: hotlist.REMOVE_ITEMS_SUCCESS});
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlist.removeItems(example.NAME, [])(dispatch);

      const action = {
        type: hotlist.REMOVE_ITEMS_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('rerankItems', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve({}));

      const items = [example.HOTLIST_ITEM_NAME];
      await hotlist.rerankItems(example.NAME, items, 0)(dispatch);

      sinon.assert.calledWith(dispatch, {type: hotlist.RERANK_ITEMS_START});

      const args = {
        name: example.NAME,
        hotlistItems: items,
        targetPosition: 0,
      };
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Hotlists',
          'RerankHotlistItems', args);

      sinon.assert.calledWith(dispatch, {type: hotlist.RERANK_ITEMS_SUCCESS});
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlist.rerankItems(example.NAME, [], 0)(dispatch);

      const action = {
        type: hotlist.RERANK_ITEMS_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });
});

describe('helpers', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');
    dispatch = sinon.stub();
  });

  afterEach(() => {
    prpcClient.call.restore();
  });

  describe('getHotlistName', () => {
    it('success', async () => {
      const response = {hotlistId: '1234'};
      prpcClient.call.returns(Promise.resolve(response));

      const name = await hotlist.getHotlistName('foo@bar.com', 'hotlist');
      assert.deepEqual(name, 'hotlists/1234');

      const args = {hotlistRef: {
        owner: {displayName: 'foo@bar.com'},
        name: 'hotlist',
      }};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.Features', 'GetHotlistID', args);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      assert.isNull(await hotlist.getHotlistName('foo@bar.com', 'hotlist'));
    });
  });
});
