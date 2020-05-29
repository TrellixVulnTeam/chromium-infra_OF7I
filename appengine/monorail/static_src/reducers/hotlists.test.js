// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as hotlists from './hotlists.js';
import * as example from 'shared/test/constants-hotlists.js';
import * as exampleIssues from 'shared/test/constants-issueV0.js';
import * as exampleUsers from 'shared/test/constants-users.js';

import {prpcClient} from 'prpc-client-instance.js';

let dispatch;

describe('hotlist reducers', () => {
  it('root reducer initial state', () => {
    const actual = hotlists.reducer(undefined, {type: null});
    const expected = {
      name: null,
      byName: {},
      hotlistItems: {},
      requests: {
        deleteHotlist: {error: null, requesting: false},
        fetch: {error: null, requesting: false},
        fetchItems: {error: null, requesting: false},
        removeEditors: {error: null, requesting: false},
        removeItems: {error: null, requesting: false},
        rerankItems: {error: null, requesting: false},
        update: {error: null, requesting: false},
      },
    };
    assert.deepEqual(actual, expected);
  });

  it('name updates on SELECT', () => {
    const action = {type: hotlists.SELECT, name: example.NAME};
    const actual = hotlists.nameReducer(null, action);
    assert.deepEqual(actual, example.NAME);
  });

  it('byName updates on RECEIVE_HOTLIST', () => {
    const action = {type: hotlists.RECEIVE_HOTLIST, hotlist: example.HOTLIST};
    const actual = hotlists.byNameReducer({}, action);
    assert.deepEqual(actual, example.BY_NAME);
  });

  it('byName fills in missing fields on RECEIVE_HOTLIST', () => {
    const action = {
      type: hotlists.RECEIVE_HOTLIST,
      hotlist: {name: example.NAME},
    };
    const actual = hotlists.byNameReducer({}, action);

    const hotlist = {
      name: example.NAME,
      defaultColumns: hotlists.DEFAULT_COLUMNS,
      editors: [],
    };
    assert.deepEqual(actual, {[example.NAME]: hotlist});
  });

  it('hotlistItems updates on FETCH_ITEMS_SUCCESS', () => {
    const action = {
      type: hotlists.FETCH_ITEMS_SUCCESS,
      name: example.NAME,
      items: [example.HOTLIST_ITEM],
    };
    const actual = hotlists.hotlistItemsReducer({}, action);
    assert.deepEqual(actual, example.HOTLIST_ITEMS);
  });
});

describe('hotlist selectors', () => {
  it('name', () => {
    const state = {hotlists: {name: example.NAME}};
    assert.deepEqual(hotlists.name(state), example.NAME);
  });

  it('byName', () => {
    const state = {hotlists: {byName: example.BY_NAME}};
    assert.deepEqual(hotlists.byName(state), example.BY_NAME);
  });

  it('hotlistItems', () => {
    const state = {hotlists: {hotlistItems: example.HOTLIST_ITEMS}};
    assert.deepEqual(hotlists.hotlistItems(state), example.HOTLIST_ITEMS);
  });

  describe('viewedHotlist', () => {
    it('normal case', () => {
      const state = {hotlists: {name: example.NAME, byName: example.BY_NAME}};
      assert.deepEqual(hotlists.viewedHotlist(state), example.HOTLIST);
    });

    it('no name', () => {
      const state = {hotlists: {name: null, byName: example.BY_NAME}};
      assert.deepEqual(hotlists.viewedHotlist(state), null);
    });

    it('hotlist not found', () => {
      const state = {hotlists: {name: example.NAME, byName: {}}};
      assert.deepEqual(hotlists.viewedHotlist(state), null);
    });
  });

  describe('viewedHotlistOwner', () => {
    it('normal case', () => {
      const state = {
        hotlists: {name: example.NAME, byName: example.BY_NAME},
        users: {byName: exampleUsers.BY_NAME},
      };
      assert.deepEqual(hotlists.viewedHotlistOwner(state), exampleUsers.USER);
    });

    it('no hotlist', () => {
      const state = {hotlists: {}, users: {}};
      assert.deepEqual(hotlists.viewedHotlistOwner(state), null);
    });
  });

  describe('viewedHotlistEditors', () => {
    it('normal case', () => {
      const state = {
        hotlists: {
          name: example.NAME,
          byName: {[example.NAME]: {
            ...example.HOTLIST,
            editors: [exampleUsers.NAME, exampleUsers.NAME_2],
          }},
        },
        users: {byName: exampleUsers.BY_NAME},
      };

      const editors = [exampleUsers.USER, exampleUsers.USER_2];
      assert.deepEqual(hotlists.viewedHotlistEditors(state), editors);
    });

    it('no user data', () => {
      const editors = [exampleUsers.NAME, exampleUsers.NAME_2];
      const state = {
        hotlists: {
          name: example.NAME,
          byName: {[example.NAME]: {...example.HOTLIST, editors}},
        },
        users: {byName: {}},
      };
      assert.deepEqual(hotlists.viewedHotlistEditors(state), [null, null]);
    });

    it('no hotlist', () => {
      const state = {hotlists: {}, users: {}};
      assert.deepEqual(hotlists.viewedHotlistEditors(state), null);
    });
  });

  describe('viewedHotlistItems', () => {
    it('normal case', () => {
      const state = {hotlists: {
        name: example.NAME,
        hotlistItems: example.HOTLIST_ITEMS,
      }};
      const actual = hotlists.viewedHotlistItems(state);
      assert.deepEqual(actual, [example.HOTLIST_ITEM]);
    });

    it('no name', () => {
      const state = {hotlists: {
        name: null,
        hotlistItems: example.HOTLIST_ITEMS,
      }};
      assert.deepEqual(hotlists.viewedHotlistItems(state), []);
    });

    it('hotlist not found', () => {
      const state = {hotlists: {name: example.NAME, hotlistItems: {}}};
      assert.deepEqual(hotlists.viewedHotlistItems(state), []);
    });
  });

  describe('viewedHotlistIssues', () => {
    it('normal case', () => {
      const state = {
        hotlists: {
          name: example.NAME,
          hotlistItems: example.HOTLIST_ITEMS,
        },
        issue: {
          issuesByRefString: {
            [exampleIssues.ISSUE_REF_STRING]: exampleIssues.ISSUE,
          },
        },
        users: {byName: {[exampleUsers.NAME]: exampleUsers.USER}},
      };
      const actual = hotlists.viewedHotlistIssues(state);
      assert.deepEqual(actual, [example.HOTLIST_ISSUE]);
    });

    it('no issue', () => {
      const state = {
        hotlists: {
          name: example.NAME,
          hotlistItems: example.HOTLIST_ITEMS,
        },
        issue: {
          issuesByRefString: {
            [exampleIssues.ISSUE_OTHER_PROJECT_REF_STRING]: exampleIssues.ISSUE,
          },
        },
        users: {byName: {}},
      };
      assert.deepEqual(hotlists.viewedHotlistIssues(state), []);
    });
  });

  describe('viewedHotlistPermissions', () => {
    it('normal case', () => {
      const permissions = [hotlists.ADMINISTER, hotlists.EDIT];
      const state = {
        hotlists: {name: example.NAME, byName: example.BY_NAME},
        permissions: {byName: {[example.NAME]: {permissions}}},
      };
      assert.deepEqual(hotlists.viewedHotlistPermissions(state), permissions);
    });

    it('no issue', () => {
      const state = {hotlists: {}, permissions: {}};
      assert.deepEqual(hotlists.viewedHotlistPermissions(state), []);
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
    const actual = hotlists.select(example.NAME);
    const expected = {type: hotlists.SELECT, name: example.NAME};
    assert.deepEqual(actual, expected);
  });

  describe('deleteHotlist', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve({}));

      await hotlists.deleteHotlist(example.NAME)(dispatch);

      sinon.assert.calledWith(dispatch, {type: hotlists.DELETE_START});

      const args = {name: example.NAME};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Hotlists', 'DeleteHotlist', args);

      sinon.assert.calledWith(dispatch, {type: hotlists.DELETE_SUCCESS});
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlists.deleteHotlist(example.NAME)(dispatch);

      const action = {
        type: hotlists.DELETE_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('fetch', () => {
    it('success', async () => {
      const hotlist = example.HOTLIST;
      prpcClient.call.returns(Promise.resolve(hotlist));

      await hotlists.fetch(example.NAME)(dispatch);

      const args = {name: example.NAME};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Hotlists', 'GetHotlist', args);

      sinon.assert.calledWith(dispatch, {type: hotlists.FETCH_START});
      sinon.assert.calledWith(dispatch, {type: hotlists.FETCH_SUCCESS});
      sinon.assert.calledWith(
          dispatch, {type: hotlists.RECEIVE_HOTLIST, hotlist});
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlists.fetch(example.NAME)(dispatch);

      const action = {type: hotlists.FETCH_FAILURE, error: sinon.match.any};
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('fetchItems', () => {
    it('success', async () => {
      const response = {items: [example.HOTLIST_ITEM]};
      prpcClient.call.returns(Promise.resolve(response));

      const returnValue = await hotlists.fetchItems(example.NAME)(dispatch);
      assert.deepEqual(returnValue, [{...example.HOTLIST_ITEM, rank: 0}]);

      const args = {parent: example.NAME, orderBy: 'rank'};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Hotlists', 'ListHotlistItems', args);

      sinon.assert.calledWith(dispatch, {type: hotlists.FETCH_ITEMS_START});
      const action = {
        type: hotlists.FETCH_ITEMS_SUCCESS,
        name: example.NAME,
        items: [{...example.HOTLIST_ITEM, rank: 0}],
      };
      sinon.assert.calledWith(dispatch, action);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlists.fetchItems(example.NAME)(dispatch);

      const action = {
        type: hotlists.FETCH_ITEMS_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });

    it('success with empty hotlist', async () => {
      const response = {items: []};
      prpcClient.call.returns(Promise.resolve(response));

      const returnValue = await hotlists.fetchItems(example.NAME)(dispatch);
      assert.deepEqual(returnValue, []);

      sinon.assert.calledWith(dispatch, {type: hotlists.FETCH_ITEMS_START});

      const args = {parent: example.NAME, orderBy: 'rank'};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Hotlists', 'ListHotlistItems', args);

      const action = {
        type: hotlists.FETCH_ITEMS_SUCCESS,
        name: example.NAME,
        items: [],
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('removeEditors', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve({}));

      const editors = [exampleUsers.NAME];
      await hotlists.removeEditors(example.NAME, editors)(dispatch);

      const args = {name: example.NAME, editors};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Hotlists',
          'RemoveHotlistEditors', args);

      sinon.assert.calledWith(dispatch, {type: hotlists.REMOVE_EDITORS_START});
      const action = {type: hotlists.REMOVE_EDITORS_SUCCESS};
      sinon.assert.calledWith(dispatch, action);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlists.removeEditors(example.NAME, [])(dispatch);

      const action = {
        type: hotlists.REMOVE_EDITORS_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('removeItems', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve({}));

      const issues = [exampleIssues.NAME];
      await hotlists.removeItems(example.NAME, issues)(dispatch);

      const args = {parent: example.NAME, issues};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Hotlists',
          'RemoveHotlistItems', args);

      sinon.assert.calledWith(dispatch, {type: hotlists.REMOVE_ITEMS_START});
      sinon.assert.calledWith(dispatch, {type: hotlists.REMOVE_ITEMS_SUCCESS});
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlists.removeItems(example.NAME, [])(dispatch);

      const action = {
        type: hotlists.REMOVE_ITEMS_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('rerankItems', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve({}));

      const items = [example.HOTLIST_ITEM_NAME];
      await hotlists.rerankItems(example.NAME, items, 0)(dispatch);

      const args = {
        name: example.NAME,
        hotlistItems: items,
        targetPosition: 0,
      };
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Hotlists',
          'RerankHotlistItems', args);

      sinon.assert.calledWith(dispatch, {type: hotlists.RERANK_ITEMS_START});
      sinon.assert.calledWith(dispatch, {type: hotlists.RERANK_ITEMS_SUCCESS});
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await hotlists.rerankItems(example.NAME, [], 0)(dispatch);

      const action = {
        type: hotlists.RERANK_ITEMS_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('update', () => {
    it('success', async () => {
      const hotlistOnlyWithUpdates = {
        displayName: example.HOTLIST.displayName + 'foo',
        summary: example.HOTLIST.summary + 'abc',
      };
      const hotlist = {...example.HOTLIST, ...hotlistOnlyWithUpdates};
      prpcClient.call.returns(Promise.resolve(hotlist));

      await hotlists.update(
          example.HOTLIST.name, hotlistOnlyWithUpdates)(dispatch);

      const hotlistArg = {
        ...hotlistOnlyWithUpdates,
        name: example.HOTLIST.name,
      };
      const fieldMask = 'displayName,summary';
      const args = {hotlist: hotlistArg, updateMask: fieldMask};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Hotlists', 'UpdateHotlist', args);

      sinon.assert.calledWith(dispatch, {type: hotlists.UPDATE_START});
      sinon.assert.calledWith(dispatch, {type: hotlists.UPDATE_SUCCESS});
      sinon.assert.calledWith(
          dispatch, {type: hotlists.RECEIVE_HOTLIST, hotlist});
    });

    it('failure', async () => {
      prpcClient.call.throws();
      const hotlistOnlyWithUpdates = {
        displayName: example.HOTLIST.displayName + 'foo',
        summary: example.HOTLIST.summary + 'abc',
      };
      await hotlists.update(
          example.HOTLIST.name, hotlistOnlyWithUpdates)(dispatch);

      const action = {
        type: hotlists.UPDATE_FAILURE,
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

      const name = await hotlists.getHotlistName('foo@bar.com', 'hotlist');
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

      assert.isNull(await hotlists.getHotlistName('foo@bar.com', 'hotlist'));
    });
  });
});
