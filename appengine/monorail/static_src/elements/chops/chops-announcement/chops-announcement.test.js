// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import {assert} from 'chai';
import {ChopsAnnouncement, REFRESH_TIME_MS,
  XSSI_PREFIX} from './chops-announcement.js';
import sinon from 'sinon';

let element;
let clock;

describe('chops-announcement', () => {
  beforeEach(() => {
    element = document.createElement('chops-announcement');
    document.body.appendChild(element);

    clock = sinon.useFakeTimers({
      now: new Date(0),
      shouldAdvanceTime: false,
    });

    sinon.stub(window, 'fetch');
  });

  afterEach(() => {
    if (document.body.contains(element)) {
      document.body.removeChild(element);
    }

    clock.restore();

    window.fetch.restore();
  });

  it('initializes', () => {
    assert.instanceOf(element, ChopsAnnouncement);
  });

  it('does not request announcements when no service specified', async () => {
    sinon.stub(element, 'fetch');

    element.service = '';

    await element.updateComplete;

    sinon.assert.notCalled(element.fetch);
  });

  it('requests announcements when service is specified', async () => {
    sinon.stub(element, 'fetch');

    element.service = 'monorail';

    await element.updateComplete;

    sinon.assert.calledOnce(element.fetch);
  });

  it('refreshes announcements regularly', async () => {
    sinon.stub(element, 'fetch');

    element.service = 'monorail';

    await element.updateComplete;

    sinon.assert.calledOnce(element.fetch);

    clock.tick(REFRESH_TIME_MS);

    await element.updateComplete;

    sinon.assert.calledTwice(element.fetch);
  });

  it('stops refreshing when service removed', async () => {
    sinon.stub(element, 'fetch');

    element.service = 'monorail';

    await element.updateComplete;

    sinon.assert.calledOnce(element.fetch);

    element.service = '';

    await element.updateComplete;
    clock.tick(REFRESH_TIME_MS);
    await element.updateComplete;

    sinon.assert.calledOnce(element.fetch);
  });

  it('stops refreshing when element is disconnected', async () => {
    sinon.stub(element, 'fetch');

    element.service = 'monorail';

    await element.updateComplete;

    sinon.assert.calledOnce(element.fetch);

    document.body.removeChild(element);

    await element.updateComplete;
    clock.tick(REFRESH_TIME_MS);
    await element.updateComplete;

    sinon.assert.calledOnce(element.fetch);
  });

  it('renders error when thrown', async () => {
    sinon.stub(element, 'fetch');
    element.fetch.throws(() => Error('Something went wrong'));

    element.service = 'monorail';

    await element.updateComplete;

    // Fetch runs here.

    await element.updateComplete;

    assert.equal(element._error, 'Something went wrong');
    assert.include(element.shadowRoot.textContent, 'Something went wrong');
  });

  it('renders fetched announcement', async () => {
    sinon.stub(element, 'fetch');
    element.fetch.returns(
        {announcements: [{id: '1234', messageContent: 'test thing'}]});

    element.service = 'monorail';

    await element.updateComplete;

    // Fetch runs here.

    await element.updateComplete;

    assert.deepEqual(element._announcements,
        [{id: '1234', messageContent: 'test thing'}]);
    assert.include(element.shadowRoot.textContent, 'test thing');
  });

  it('fetch returns response data', async () => {
    const json = {announcements: [{id: '1234', messageContent: 'test thing'}]};
    const fakeResponse = XSSI_PREFIX + JSON.stringify(json);
    window.fetch.returns(new window.Response(fakeResponse));

    const resp = await element.fetch('monorail');

    assert.deepEqual(resp, json);
  });

  it('fetch errors when no XSSI prefix', async () => {
    const json = {announcements: [{id: '1234', messageContent: 'test thing'}]};
    const fakeResponse = JSON.stringify(json);
    window.fetch.returns(new window.Response(fakeResponse));

    try {
      await element.fetch('monorail');
    } catch (e) {
      assert.include(e.message, 'No XSSI prefix in announce response:');
    }
  });

  it('fetch errors when response is not okay', async () => {
    const json = {announcements: [{id: '1234', messageContent: 'test thing'}]};
    const fakeResponse = XSSI_PREFIX + JSON.stringify(json);
    window.fetch.returns(new window.Response(fakeResponse, {status: 500}));

    try {
      await element.fetch('monorail');
    } catch (e) {
      assert.include(e.message,
          'Something went wrong while fetching announcements');
    }
  });
});
