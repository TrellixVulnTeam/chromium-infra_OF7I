// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import sinon from 'sinon';
import {assert} from 'chai';

import {store, stateUpdated} from 'reducers/base.js';
import {prpcClient} from 'prpc-client-instance.js';
import * as sitewide from './sitewide.js';
import {SITEWIDE_DEFAULT_COLUMNS} from 'shared/issue-fields.js';

let prpcCall;

describe('sitewide selectors', () => {
  it('queryParams', () => {
    assert.deepEqual(sitewide.queryParams({}), {});
    assert.deepEqual(sitewide.queryParams({sitewide: {}}), {});
    assert.deepEqual(sitewide.queryParams({sitewide: {queryParams:
      {q: 'owner:me'}}}), {q: 'owner:me'});
  });

  describe('pageTitle', () => {
    it('defaults to Monorail when no data', () => {
      assert.equal(sitewide.pageTitle({}), 'Monorail');
      assert.equal(sitewide.pageTitle({sitewide: {}}), 'Monorail');
    });

    it('prepends local page title when one exists', () => {
      assert.equal(sitewide.pageTitle(
          {sitewide: {pageTitle: 'Issue Detail'}}), 'Issue Detail - Monorail');
    });

    it('shows data for view project', () => {
      assert.equal(sitewide.pageTitle({
        sitewide: {pageTitle: 'Page'},
        project: {
          config: {projectName: 'chromium'},
          presentationConfig: {projectSummary: 'Open source browser'},
        },
      }), 'Page - chromium - Open source browser - Monorail');
    });
  });

  describe('currentColumns', () => {
    it('defaults to sitewide default columns when no configuration', () => {
      assert.deepEqual(sitewide.currentColumns({}), SITEWIDE_DEFAULT_COLUMNS);
      assert.deepEqual(sitewide.currentColumns({project: {}}),
          SITEWIDE_DEFAULT_COLUMNS);
      assert.deepEqual(sitewide.currentColumns({project: {
        presentationConfig: {},
      }}), SITEWIDE_DEFAULT_COLUMNS);
    });

    it('uses project default columns', () => {
      assert.deepEqual(sitewide.currentColumns({project: {
        presentationConfig: {defaultColSpec: 'ID+Summary+AllLabels'},
      }}), ['ID', 'Summary', 'AllLabels']);
    });

    it('columns in URL query params override all defaults', () => {
      assert.deepEqual(sitewide.currentColumns({
        project: {
          presentationConfig: {defaultColSpec: 'ID+Summary+AllLabels'},
        },
        sitewide: {
          queryParams: {colspec: 'ID+Summary+ColumnName+Priority'},
        },
      }), ['ID', 'Summary', 'ColumnName', 'Priority']);
    });
  });

  describe('currentCan', () => {
    it('uses sitewide default can by default', () => {
      assert.deepEqual(sitewide.currentCan({}), '2');
    });

    it('URL params override default can', () => {
      assert.deepEqual(sitewide.currentCan({
        sitewide: {
          queryParams: {can: '3'},
        },
      }), '3');
    });

    it('undefined query param does not override default can', () => {
      assert.deepEqual(sitewide.currentCan({
        sitewide: {
          queryParams: {can: undefined},
        },
      }), '2');
    });
  });

  describe('currentQuery', () => {
    it('defaults to empty', () => {
      assert.deepEqual(sitewide.currentQuery({}), '');
      assert.deepEqual(sitewide.currentQuery({project: {}}), '');
    });

    it('uses project default when no params', () => {
      assert.deepEqual(sitewide.currentQuery({project: {
        presentationConfig: {
          defaultQuery: 'owner:me',
        },
      }}), 'owner:me');
    });

    it('URL query params override default query', () => {
      assert.deepEqual(sitewide.currentQuery({
        project: {
          presentationConfig: {
            defaultQuery: 'owner:me',
          },
        },
        sitewide: {
          queryParams: {q: 'component:Infra'},
        },
      }), 'component:Infra');
    });

    it('empty string in param overrides default project query', () => {
      assert.deepEqual(sitewide.currentQuery({
        project: {
          presentationConfig: {
            defaultQuery: 'owner:me',
          },
        },
        sitewide: {
          queryParams: {q: ''},
        },
      }), '');
    });

    it('undefined query param does not override default search', () => {
      assert.deepEqual(sitewide.currentQuery({
        project: {
          presentationConfig: {
            defaultQuery: 'owner:me',
          },
        },
        sitewide: {
          queryParams: {q: undefined},
        },
      }), 'owner:me');
    });
  });
});


describe('sitewide action creators', () => {
  beforeEach(() => {
    prpcCall = sinon.stub(prpcClient, 'call');
  });

  afterEach(() => {
    prpcClient.call.restore();
  });

  it('setQueryParams updates queryParams', async () => {
    store.dispatch(sitewide.setQueryParams({test: 'param'}));

    await stateUpdated;

    assert.deepEqual(sitewide.queryParams(store.getState()), {test: 'param'});
  });

  describe('getServerStatus', () => {
    it('gets server status', async () => {
      prpcCall.callsFake(() => {
        return {
          bannerMessage: 'Message',
          bannerTime: 1234,
          readOnly: true,
        };
      });

      store.dispatch(sitewide.getServerStatus());

      await stateUpdated;
      const state = store.getState();

      assert.deepEqual(sitewide.bannerMessage(state), 'Message');
      assert.deepEqual(sitewide.bannerTime(state), 1234);
      assert.isTrue(sitewide.readOnly(state));

      assert.deepEqual(sitewide.requests(state), {
        serverStatus: {
          error: null,
          requesting: false,
        },
      });
    });

    it('gets empty status', async () => {
      prpcCall.callsFake(() => {
        return {};
      });

      store.dispatch(sitewide.getServerStatus());

      await stateUpdated;
      const state = store.getState();

      assert.deepEqual(sitewide.bannerMessage(state), '');
      assert.deepEqual(sitewide.bannerTime(state), 0);
      assert.isFalse(sitewide.readOnly(state));

      assert.deepEqual(sitewide.requests(state), {
        serverStatus: {
          error: null,
          requesting: false,
        },
      });
    });

    it('fails', async () => {
      const error = new Error('error');
      prpcCall.callsFake(() => {
        throw error;
      });

      store.dispatch(sitewide.getServerStatus());

      await stateUpdated;
      const state = store.getState();

      assert.deepEqual(sitewide.bannerMessage(state), '');
      assert.deepEqual(sitewide.bannerTime(state), 0);
      assert.isFalse(sitewide.readOnly(state));

      assert.deepEqual(sitewide.requests(state), {
        serverStatus: {
          error: error,
          requesting: false,
        },
      });
    });
  });
});
