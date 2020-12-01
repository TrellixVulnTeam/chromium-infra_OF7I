// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import sinon from 'sinon';
import {assert} from 'chai';
import {MrGridControls} from './mr-grid-controls.js';
import {SERVER_LIST_ISSUES_LIMIT} from 'shared/constants.js';

let element;

describe('mr-grid-controls', () => {
  beforeEach(() => {
    element = document.createElement('mr-grid-controls');
    document.body.appendChild(element);

    element._page = sinon.stub();
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrGridControls);
  });

  it('selecting row updates y param', async () => {
    const stub = sinon.stub(element, '_changeUrlParams');

    await element.updateComplete;

    const dropdownRows = element.shadowRoot.querySelector('.row-selector');

    dropdownRows.selection = 'Status';
    dropdownRows.dispatchEvent(new Event('change'));
    sinon.assert.calledWith(stub, {y: 'Status'});
  });

  it('setting row to None deletes y param', async () => {
    element.queryParams = {y: 'Remove', x: 'Keep'};
    element.projectName = 'chromium';

    await element.updateComplete;

    const dropdownRows = element.shadowRoot.querySelector('.row-selector');

    dropdownRows.selection = 'None';
    dropdownRows.dispatchEvent(new Event('change'));

    sinon.assert.calledWith(element._page,
        '/p/chromium/issues/list?x=Keep');
  });

  it('selecting col updates x param', async () => {
    const stub = sinon.stub(element, '_changeUrlParams');
    await element.updateComplete;

    const dropdownCols = element.shadowRoot.querySelector('.col-selector');

    dropdownCols.selection = 'Blocking';
    dropdownCols.dispatchEvent(new Event('change'));
    sinon.assert.calledWith(stub, {x: 'Blocking'});
  });

  it('setting col to None deletes x param', async () => {
    element.queryParams = {y: 'Keep', x: 'Remove'};
    element.projectName = 'chromium';

    await element.updateComplete;

    const dropdownCols = element.shadowRoot.querySelector('.col-selector');

    dropdownCols.selection = 'None';
    dropdownCols.dispatchEvent(new Event('change'));

    sinon.assert.calledWith(element._page,
        '/p/chromium/issues/list?y=Keep');
  });

  it('cellOptions computes URLs with queryParams and projectName', async () => {
    element.projectName = 'chromium';
    element.queryParams = {q: 'hello-world'};

    assert.deepEqual(element.cellOptions, [
      {text: 'Tile', value: 'tiles',
        url: '/p/chromium/issues/list?q=hello-world'},
      {text: 'IDs', value: 'ids',
        url: '/p/chromium/issues/list?q=hello-world&cells=ids'},
      {text: 'Counts', value: 'counts',
        url: '/p/chromium/issues/list?q=hello-world&cells=counts'},
    ]);
  });

  describe('displays appropriate messaging for issue count', () => {
    it('for one issue', () => {
      element.issueCount = 1;
      assert.equal(element.totalIssuesDisplay, '1 issue shown');
    });

    it('for less than 100,000 issues', () => {
      element.issueCount = 50;
      assert.equal(element.totalIssuesDisplay, '50 issues shown');
    });

    it('for 100,000 issues or more', () => {
      element.issueCount = SERVER_LIST_ISSUES_LIMIT;
      assert.equal(element.totalIssuesDisplay, '100,000+ issues shown');
    });
  });
});
