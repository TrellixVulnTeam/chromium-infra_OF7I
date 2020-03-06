// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { assert } from 'chai';
import sinon, { SinonStub } from 'sinon';

import { authInitializedPromise, init } from './chops-signin';

function clearGapiHandlers() {
  getGapiHandlers().forEach(key => delete window[key]);
}

function getGapiHandlers() {
  return Object.keys(window).filter(key => key.startsWith('gapi0'));
}

beforeEach(() => {
  window.gapi = (sinon.stub() as unknown) as typeof window.gapi;
  window.gapi.load = sinon.stub();
  const authStub: Partial<gapi.auth2.GoogleAuth> = new Promise(resolve =>
    resolve()
  );
  authStub.currentUser = (sinon.stub() as unknown) as typeof authStub.currentUser;
  authStub.currentUser!.listen = sinon.stub();
  window.gapi.auth2 = ({
    init: sinon.stub().returns(authStub),
  } as unknown) as typeof window.gapi.auth2;

  sinon.stub(document.head, 'appendChild');
  sinon.stub(document.head, 'removeChild');

  clearGapiHandlers();
});

afterEach(() => {
  ((window.gapi as unknown) as SinonStub).reset();
  ((document.head.appendChild as unknown) as SinonStub).restore();
  ((document.head.removeChild as unknown) as SinonStub).restore();
});

test('init by default loads the auth2 library', () => {
  init('rutabaga');

  // Call the first (and should be only) global gapi load handler.
  window[getGapiHandlers()[0]]();

  sinon.assert.calledOnce(window.gapi.load as SinonStub);
  assert.equal((window.gapi.load as SinonStub).args[0][0], 'auth2');
});

test('init allows passing one extra library to load', () => {
  init('rutabaga', ['client']);

  // Call the first (and should be only) global gapi load handler.
  window[getGapiHandlers()[0]]();

  sinon.assert.calledOnce(window.gapi.load as SinonStub);
  assert.equal((window.gapi.load as SinonStub).args[0][0], 'auth2:client');
});

test('init allows passing multiple extra library to load', () => {
  init('rutabaga', ['client', 'rutabaga']);

  // Call the first (and should be only) global gapi load handler.
  window[getGapiHandlers()[0]]();

  sinon.assert.calledOnce(window.gapi.load as SinonStub);
  assert.equal(
    (window.gapi.load as SinonStub).args[0][0],
    'auth2:client:rutabaga'
  );
});

test('onAuthLoaded callback passes email scope by default', () => {
  init('rutabaga');

  // Call the first (and should be only) global gapi load handler.
  window[getGapiHandlers()[0]]();

  // Call onAuthLoaded handler.
  (window.gapi.load as SinonStub).args[0][1]();

  // Get first call to auth2.init.
  const actualConfig = (window.gapi.auth2.init as SinonStub).args[0][0];
  assert.equal(actualConfig.client_id, 'rutabaga');
  assert.equal(actualConfig.scope, 'email');
});

test('onAuthLoaded callback allows passing additional scopes', () => {
  init('rutabaga', [], ['https://scope']);

  // Call the first (and should be only) global gapi load handler.
  window[getGapiHandlers()[0]]();

  // Call onAuthLoaded handler.
  (window.gapi.load as SinonStub).args[0][1]();

  // Get first call to auth2.init.
  const actualConfig = (window.gapi.auth2.init as SinonStub).args[0][0];
  assert.equal(actualConfig.client_id, 'rutabaga');
  assert.equal(actualConfig.scope, 'email https://scope');
});

test('authInitializedPromise resolves when fulfilled', async () => {
  init('rutabaga');

  setTimeout(() => {
    // Call the first (and should be only) global gapi load handler.
    window[getGapiHandlers()[0]]();
    // Call onAuthLoaded handler.
    (window.gapi.load as SinonStub).args[0][1]();
  }, 10);

  // Test it doesn't spin forever.
  await authInitializedPromise;
});
