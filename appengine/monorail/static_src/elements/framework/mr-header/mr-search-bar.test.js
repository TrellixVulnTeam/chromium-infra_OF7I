// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import {MrSearchBar} from './mr-search-bar.js';
import {prpcClient} from 'prpc-client-instance.js';
import {issueRefToUrl} from 'shared/converters.js';
import {clientLoggerFake} from 'shared/test-fakes.js';


window.CS_env = {
  token: 'foo-token',
};

let element;

describe('mr-search-bar', () => {
  beforeEach(() => {
    element = document.createElement('mr-search-bar');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrSearchBar);
  });

  it('render user saved queries', async () => {
    element.userDisplayName = 'test@user.com';
    element.userSavedQueries = [
      {name: 'test query', queryId: 101},
      {name: 'hello world', queryId: 202},
    ];

    await element.updateComplete;

    const queryOptions = element.shadowRoot.querySelectorAll(
        '.user-query');

    assert.equal(queryOptions.length, 2);

    assert.equal(queryOptions[0].value, '101');
    assert.equal(queryOptions[0].textContent, 'test query');

    assert.equal(queryOptions[1].value, '202');
    assert.equal(queryOptions[1].textContent, 'hello world');
  });

  it('render project saved queries', async () => {
    element.userDisplayName = 'test@user.com';
    element.projectSavedQueries = [
      {name: 'test query', queryId: 101},
      {name: 'hello world', queryId: 202},
    ];

    await element.updateComplete;

    const queryOptions = element.shadowRoot.querySelectorAll(
        '.project-query');

    assert.equal(queryOptions.length, 2);

    assert.equal(queryOptions[0].value, '101');
    assert.equal(queryOptions[0].textContent, 'test query');

    assert.equal(queryOptions[1].value, '202');
    assert.equal(queryOptions[1].textContent, 'hello world');
  });

  it('search input resets form value when initialQuery changes', async () => {
    element.initialQuery = 'first query';
    await element.updateComplete;

    const queryInput = element.shadowRoot.querySelector('#searchq');

    assert.equal(queryInput.value, 'first query');

    // Simulate a user typing something into the search form.
    queryInput.value = 'blah';

    element.initialQuery = 'second query';
    await element.updateComplete;

    // 'blah' disappears because the new initialQuery causes the form to
    // reset.
    assert.equal(queryInput.value, 'second query');
  });

  it('unrelated property changes do not reset query form', async () => {
    element.initialQuery = 'first query';
    await element.updateComplete;

    const queryInput = element.shadowRoot.querySelector('#searchq');

    assert.equal(queryInput.value, 'first query');

    // Simulate a user typing something into the search form.
    queryInput.value = 'blah';

    element.initialCan = '5';
    await element.updateComplete;

    assert.equal(queryInput.value, 'blah');
  });

  it('spell check is off for search bar', async () => {
    await element.updateComplete;
    const searchElement = element.shadowRoot.querySelector('#searchq');
    assert.equal(searchElement.getAttribute('spellcheck'), 'false');
  });

  describe('search form submit', () => {
    let prpcClientStub;
    beforeEach(() => {
      element.clientLogger = clientLoggerFake();

      element._page = sinon.stub();
      sinon.stub(window, 'open');

      element.projectName = 'chromium';
      prpcClientStub = sinon.stub(prpcClient, 'call');
    });

    afterEach(() => {
      window.open.restore();
      prpcClient.call.restore();
    });

    it('prevents default', async () => {
      await element.updateComplete;

      const form = element.shadowRoot.querySelector('form');

      // Note: HTMLFormElement's submit function does not run submit handlers
      // but clicking a submit buttons programmatically works.
      const event = new Event('submit');
      sinon.stub(event, 'preventDefault');
      form.dispatchEvent(event);

      sinon.assert.calledOnce(event.preventDefault);
    });

    it('uses initial values when no form changes', async () => {
      element.initialQuery = 'test query';
      element.currentCan = '3';

      await element.updateComplete;

      const form = element.shadowRoot.querySelector('form');

      form.dispatchEvent(new Event('submit'));

      sinon.assert.calledOnce(element._page);
      sinon.assert.calledWith(element._page,
          '/p/chromium/issues/list?q=test%20query&can=3');
    });

    it('adds form values to url', async () => {
      await element.updateComplete;

      const form = element.shadowRoot.querySelector('form');

      form.q.value = 'test';
      form.can.value = '1';

      form.dispatchEvent(new Event('submit'));

      sinon.assert.calledOnce(element._page);
      sinon.assert.calledWith(element._page,
          '/p/chromium/issues/list?q=test&can=1');
    });

    it('trims query', async () => {
      await element.updateComplete;

      const form = element.shadowRoot.querySelector('form');

      form.q.value = '  abc  ';
      form.can.value = '1';

      form.dispatchEvent(new Event('submit'));

      sinon.assert.calledOnce(element._page);
      sinon.assert.calledWith(element._page,
          '/p/chromium/issues/list?q=abc&can=1');
    });

    it('jumps to issue for digit-only query', async () => {
      prpcClientStub.returns(Promise.resolve({issue: 'hello world'}));

      await element.updateComplete;

      const form = element.shadowRoot.querySelector('form');

      form.q.value = '123';
      form.can.value = '1';

      form.dispatchEvent(new Event('submit'));

      await element._navigateToNext;

      const expected = issueRefToUrl('hello world', {q: '123', can: '1'});
      sinon.assert.calledWith(element._page, expected);
    });

    it('only keeps kept query params', async () => {
      element.queryParams = {fakeParam: 'test', x: 'Status'};
      element.keptParams = ['x'];

      await element.updateComplete;

      const form = element.shadowRoot.querySelector('form');

      form.dispatchEvent(new Event('submit'));

      sinon.assert.calledOnce(element._page);
      sinon.assert.calledWith(element._page,
          '/p/chromium/issues/list?x=Status&q=&can=2');
    });

    it('on shift+enter opens search in new tab', async () => {
      await element.updateComplete;

      const form = element.shadowRoot.querySelector('form');

      form.q.value = 'test';
      form.can.value = '1';

      // Dispatch event from an input in the form.
      form.q.dispatchEvent(new KeyboardEvent('keypress',
          {key: 'Enter', shiftKey: true, bubbles: true}));

      sinon.assert.calledOnce(window.open);
      sinon.assert.calledWith(window.open,
          '/p/chromium/issues/list?q=test&can=1', '_blank', 'noopener');
    });
  });
});
