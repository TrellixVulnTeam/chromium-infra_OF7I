// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrComment} from './mr-comment.js';


let element;

/**
 * Testing helper to find if an Array of options has an option with some
 * text.
 * @param {Array<MenuItem>} options Dropdown options to look through.
 * @param {string} needle The text to search for.
 * @return {boolean} Whether the option exists or not.
 */
const hasOptionWithText = (options, needle) => {
  return options.some(({text}) => text === needle);
};

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

  describe('3-dot menu options', () => {
    it('allows showing deleted comment content', () => {
      element._isExpandedIfDeleted = false;

      // The comment is deleted.
      element.comment = {content: 'test', isDeleted: true, canDelete: true};
      assert.isTrue(hasOptionWithText(element._commentOptions,
          'Show comment content'));

      // The comment is spam.
      element.comment = {content: 'test', isSpam: true, canFlag: true};
      assert.isTrue(hasOptionWithText(element._commentOptions,
          'Show comment content'));
    });

    it('allows hiding deleted comment content', () => {
      element._isExpandedIfDeleted = true;

      // The comment is deleted.
      element.comment = {content: 'test', isDeleted: true, canDelete: true};
      assert.isTrue(hasOptionWithText(element._commentOptions,
          'Hide comment content'));

      // The comment is spam.
      element.comment = {content: 'test', isSpam: true, canFlag: true};
      assert.isTrue(hasOptionWithText(element._commentOptions,
          'Hide comment content'));
    });

    it('disallows showing deleted comment content', () => {
      // The comment is deleted.
      element.comment = {content: 'test', isDeleted: true, canDelete: false};
      assert.isFalse(hasOptionWithText(element._commentOptions,
          'Hide comment content'));

      // The comment is spam.
      element.comment = {content: 'test', isSpam: true, canFlag: false};
      assert.isFalse(hasOptionWithText(element._commentOptions,
          'Hide comment content'));
    });

    it('allows deleting comment', () => {
      element.comment = {content: 'test', isDeleted: false, canDelete: true};
      assert.isTrue(hasOptionWithText(element._commentOptions,
          'Delete comment'));
    });

    it('disallows deleting comment', () => {
      element.comment = {content: 'test', isDeleted: false, canDelete: false};
      assert.isFalse(hasOptionWithText(element._commentOptions,
          'Delete comment'));
    });

    it('allows undeleting comment', () => {
      element.comment = {content: 'test', isDeleted: true, canDelete: true};
      assert.isTrue(hasOptionWithText(element._commentOptions,
          'Undelete comment'));
    });

    it('disallows undeleting comment', () => {
      element.comment = {content: 'test', isDeleted: true, canDelete: false};
      assert.isFalse(hasOptionWithText(element._commentOptions,
          'Undelete comment'));
    });

    it('allows flagging comment as spam', () => {
      element.comment = {content: 'test', isSpam: false, canFlag: true};
      assert.isTrue(hasOptionWithText(element._commentOptions,
          'Flag comment'));
    });

    it('disallows flagging comment as spam', () => {
      element.comment = {content: 'test', isSpam: false, canFlag: false};
      assert.isFalse(hasOptionWithText(element._commentOptions,
          'Flag comment'));
    });

    it('allows unflagging comment as spam', () => {
      element.comment = {content: 'test', isSpam: true, canFlag: true};
      assert.isTrue(hasOptionWithText(element._commentOptions,
          'Unflag comment'));
    });

    it('disallows unflagging comment as spam', () => {
      element.comment = {content: 'test', isSpam: true, canFlag: false};
      assert.isFalse(hasOptionWithText(element._commentOptions,
          'Unflag comment'));
    });
  });
});
