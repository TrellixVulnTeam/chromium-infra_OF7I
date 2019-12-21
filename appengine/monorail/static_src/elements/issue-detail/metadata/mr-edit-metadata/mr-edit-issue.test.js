// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import sinon from 'sinon';
import {assert} from 'chai';
import {prpcClient} from 'prpc-client-instance.js';
import {MrEditIssue, allowRemovedRestrictions} from './mr-edit-issue.js';
import {clientLoggerFake} from 'shared/test-fakes.js';

let element;

describe('mr-edit-issue', () => {
  beforeEach(() => {
    element = document.createElement('mr-edit-issue');
    document.body.appendChild(element);
    sinon.stub(prpcClient, 'call');

    element.clientLogger = clientLoggerFake();
  });

  afterEach(() => {
    document.body.removeChild(element);
    prpcClient.call.restore();
  });

  it('initializes', () => {
    assert.instanceOf(element, MrEditIssue);
  });

  it('scrolls into view on #makechanges hash', async () => {
    await element.updateComplete;

    const header = element.shadowRoot.querySelector('#makechanges');
    sinon.stub(header, 'scrollIntoView');

    element.focusId = 'makechanges';
    await element.updateComplete;

    assert.isTrue(header.scrollIntoView.calledOnce);

    header.scrollIntoView.restore();
  });

  it('snows snackbar when editing finishes', async () => {
    sinon.stub(element, '_showCommentAddedSnackbar');

    element.updatingIssue = true;
    await element.updateComplete;

    sinon.assert.notCalled(element._showCommentAddedSnackbar);

    element.updatingIssue = false;
    await element.updateComplete;

    sinon.assert.calledOnce(element._showCommentAddedSnackbar);
  });

  it('shows current status even if not defined for project', async () => {
    await element.updateComplete;

    const editMetadata = element.shadowRoot.querySelector('mr-edit-metadata');
    assert.deepEqual(editMetadata.statuses, []);

    element.projectConfig = {statusDefs: [
      {status: 'hello'},
      {status: 'world'},
    ]};

    await editMetadata.updateComplete;

    assert.deepEqual(editMetadata.statuses, [
      {status: 'hello'},
      {status: 'world'},
    ]);

    element.issue = {
      statusRef: {status: 'hello'},
    };

    await editMetadata.updateComplete;

    assert.deepEqual(editMetadata.statuses, [
      {status: 'hello'},
      {status: 'world'},
    ]);

    element.issue = {
      statusRef: {status: 'weirdStatus'},
    };

    await editMetadata.updateComplete;

    assert.deepEqual(editMetadata.statuses, [
      {status: 'weirdStatus'},
      {status: 'hello'},
      {status: 'world'},
    ]);
  });

  it('ignores deprecated statuses, unless used on current issue', async () => {
    await element.updateComplete;

    const editMetadata = element.shadowRoot.querySelector('mr-edit-metadata');
    assert.deepEqual(editMetadata.statuses, []);

    element.projectConfig = {statusDefs: [
      {status: 'new'},
      {status: 'accepted', deprecated: false},
      {status: 'compiling', deprecated: true},
    ]};

    await editMetadata.updateComplete;

    assert.deepEqual(editMetadata.statuses, [
      {status: 'new'},
      {status: 'accepted', deprecated: false},
    ]);


    element.issue = {
      statusRef: {status: 'compiling'},
    };

    await editMetadata.updateComplete;

    assert.deepEqual(editMetadata.statuses, [
      {status: 'compiling'},
      {status: 'new'},
      {status: 'accepted', deprecated: false},
    ]);
  });

  it('filter out empty or deleted user owners', () => {
    assert.equal(
        element._ownerDisplayName({displayName: 'a_deleted_user'}),
        '');
    assert.equal(
        element._ownerDisplayName({
          displayName: 'test@example.com',
          userId: '1234',
        }),
        'test@example.com');
  });

  it('logs issue-update metrics', async () => {
    await element.updateComplete;

    const editMetadata = element.shadowRoot.querySelector('mr-edit-metadata');

    sinon.stub(editMetadata, 'delta').get(() => ({summary: 'test'}));

    await element.save();

    sinon.assert.calledOnce(element.clientLogger.logStart);
    sinon.assert.calledWith(element.clientLogger.logStart,
        'issue-update', 'computer-time');

    // Simulate a response updating the UI.
    element.issue = {summary: 'test'};

    await element.updateComplete;
    await element.updateComplete;

    sinon.assert.calledOnce(element.clientLogger.logEnd);
    sinon.assert.calledWith(element.clientLogger.logEnd,
        'issue-update', 'computer-time', 120 * 1000);
  });

  it('presubmits issue on change', async () => {
    element.issueRef = 'issueRef';

    await element.updateComplete;
    const editMetadata = element.shadowRoot.querySelector('mr-edit-metadata');
    editMetadata.dispatchEvent(new CustomEvent('change', {
      detail: {
        delta: {
          summary: 'Summary',
        },
      },
    }));

    sinon.assert.calledWith(prpcClient.call, 'monorail.Issues',
        'PresubmitIssue',
        {issueDelta: {summary: 'Summary'}, issueRef: 'issueRef'});
  });

  it('does not presubmit issue when no changes', () => {
    element._presubmitIssue({});

    sinon.assert.notCalled(prpcClient.call);
  });

  it('predicts components for chromium on form change', async () => {
    element.issueRef = {projectName: 'chromium'};
    element.comments = [{content: 'comments text'}];
    element.issue = {summary: 'summary'};

    await element.updateComplete;
    const editMetadata = element.shadowRoot.querySelector('mr-edit-metadata');
    editMetadata.dispatchEvent(new CustomEvent('change', {
      detail: {
        delta: {},
        commentContent: 'commentContent',
      },
    }));

    const expectedText = 'comments text\nsummary\ncommentContent';
    sinon.assert.calledWith(prpcClient.call, 'monorail.Features',
        'PredictComponent', {text: expectedText, projectName: 'chromium'});
  });

  it('does not predict components for other projects', () => {
    element.issueRef = {projectName: 'proj'};

    element._predictComponent({}, 'test');

    sinon.assert.notCalled(prpcClient.call);
  });

  it('predicts component using edited summary if one exists', () => {
    element.issueRef = {projectName: 'chromium'};
    element.comments = [{content: 'comments text'}];
    element.issue = {summary: 'old summary'};

    element._predictComponent({summary: 'new summary'}, 'new comment');

    const expectedText = 'comments text\nnew summary\nnew comment';
    sinon.assert.calledWith(prpcClient.call, 'monorail.Features',
        'PredictComponent', {text: expectedText, projectName: 'chromium'});
  });
});

describe('allowRemovedRestrictions', () => {
  beforeEach(() => {
    sinon.stub(window, 'confirm');
  });

  afterEach(() => {
    window.confirm.restore();
  });

  it('returns true if no restrictions removed', () => {
    assert.isTrue(allowRemovedRestrictions([
      {label: 'not-restricted'},
      {label: 'fine'},
    ]));
  });

  it('returns false if restrictions removed and confirmation denied', () => {
    window.confirm.returns(false);
    assert.isFalse(allowRemovedRestrictions([
      {label: 'not-restricted'},
      {label: 'restrict-view-people'},
    ]));
  });

  it('returns true if restrictions removed and confirmation accepted', () => {
    window.confirm.returns(true);
    assert.isTrue(allowRemovedRestrictions([
      {label: 'not-restricted'},
      {label: 'restrict-view-people'},
    ]));
  });
});
