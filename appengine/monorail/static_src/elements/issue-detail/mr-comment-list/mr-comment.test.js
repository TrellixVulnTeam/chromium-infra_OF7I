// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrComment} from './mr-comment.js';


let element;

describe('mr-comment', () => {
  beforeEach(() => {
    element = document.createElement('mr-comment');
    element.comment = {
      canFlag: true,
      localId: 898395,
      canDelete: true,
      projectName: 'chromium',
      commenter: {
        displayName: 'user@example.com',
        userId: '12345',
      },
      content: 'foo',
      sequenceNum: 3,
      timestamp: 1549319989,
    };
    document.body.appendChild(element);

    // Stub RAF to execute immediately.
    sinon.stub(window, 'requestAnimationFrame').callsFake((func) => func());
  });

  afterEach(() => {
    document.body.removeChild(element);
    window.requestAnimationFrame.restore();
  });

  it('initializes', () => {
    assert.instanceOf(element, MrComment);
  });

  it('scrolls to comment', async () => {
    sinon.stub(element, 'scrollIntoView');

    element.highlighted = true;
    await element.updateComplete;

    assert.isTrue(element.scrollIntoView.calledOnce);

    element.scrollIntoView.restore();
  });

  it('comment header renders self link to comment', async () => {
    element.comment = {
      localId: 1,
      projectName: 'test',
      sequenceNum: 2,
    };

    await element.updateComplete;

    const link = element.shadowRoot.querySelector('.comment-link');

    assert.equal(link.textContent, 'Comment 2');
    assert.include(link.href, '?id=1#c2');
  });

  it('renders issue links for Blockedon issue amendments', async () => {
    element.comment = {
      projectName: 'test',
      amendments: [
        {
          fieldName: 'Blockedon',
          newOrDeltaValue: '-2 3',
        },
      ],
    };

    await element.updateComplete;

    const links = element.shadowRoot.querySelectorAll('mr-issue-link');

    assert.equal(links.length, 2);

    assert.equal(links[0].text, '-2');
    assert.deepEqual(links[0].href, '/p/test/issues/detail?id=2');

    assert.equal(links[1].text, '3');
    assert.deepEqual(links[1].href, '/p/test/issues/detail?id=3');
  });

  it('renders issue links for Blocking issue amendments', async () => {
    element.comment = {
      projectName: 'test',
      amendments: [
        {
          fieldName: 'Blocking',
          newOrDeltaValue: '-2 3',
        },
      ],
    };

    await element.updateComplete;

    const links = element.shadowRoot.querySelectorAll('mr-issue-link');

    assert.equal(links.length, 2);

    assert.equal(links[0].text, '-2');
    assert.deepEqual(links[0].href, '/p/test/issues/detail?id=2');

    assert.equal(links[1].text, '3');
    assert.deepEqual(links[1].href, '/p/test/issues/detail?id=3');
  });

  it('renders issue links for Mergedinto issue amendments', async () => {
    element.comment = {
      projectName: 'test',
      amendments: [
        {
          fieldName: 'Mergedinto',
          newOrDeltaValue: '-2 3',
        },
      ],
    };

    await element.updateComplete;

    const links = element.shadowRoot.querySelectorAll('mr-issue-link');

    assert.equal(links.length, 2);

    assert.equal(links[0].text, '-2');
    assert.deepEqual(links[0].href, '/p/test/issues/detail?id=2');

    assert.equal(links[1].text, '3');
    assert.deepEqual(links[1].href, '/p/test/issues/detail?id=3');
  });
});
