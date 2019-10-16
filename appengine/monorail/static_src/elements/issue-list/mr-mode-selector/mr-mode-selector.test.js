// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import sinon from 'sinon';
import {assert} from 'chai';
import {MrModeSelector} from './mr-mode-selector.js';

let element;

describe('mr-mode-selector', () => {
  beforeEach(() => {
    element = document.createElement('mr-mode-selector');
    document.body.appendChild(element);

    element._page = sinon.stub();
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrModeSelector);
  });

  it('renders links with projectName and queryParams', async () => {
    element.value = 'list';
    element.projectName = 'chromium';
    element.queryParams = {q: 'hello-world'};

    await element.updateComplete;

    const links = element.shadowRoot.querySelectorAll('a');

    assert.include(links[0].href, '/p/chromium/issues/list?q=hello-world');
    assert.include(links[1].href,
        '/p/chromium/issues/list?q=hello-world&mode=grid');
    assert.include(links[2].href,
        '/p/chromium/issues/list?q=hello-world&mode=chart');
  });
});
