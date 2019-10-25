// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrMetadata} from './mr-metadata.js';

import {EMPTY_FIELD_VALUE} from 'shared/issue-fields.js';

let element;

describe('mr-metadata', () => {
  beforeEach(() => {
    element = document.createElement('mr-metadata');
    document.body.appendChild(element);

    element.issueRef = {projectName: 'proj'};
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrMetadata);
  });

  it('has table role set', () => {
    assert.equal(element.getAttribute('role'), 'table');
  });

  describe('default issue fields', () => {
    it('renders empty Owner', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-owner');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'Owner:');
      assert.equal(dataElement.textContent.trim(), EMPTY_FIELD_VALUE);
    });

    it('renders populated Owner', async () => {
      element.owner = {displayName: 'test@example.com'};

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-owner');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('mr-user-link');

      assert.equal(labelElement.textContent, 'Owner:');
      assert.include(dataElement.shadowRoot.textContent.trim(),
          'test@example.com');
    });

    it('renders empty CC', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-cc');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'CC:');
      assert.equal(dataElement.textContent.trim(), EMPTY_FIELD_VALUE);
    });

    it('renders multiple CCed users', async () => {
      element.cc = [
        {displayName: 'test@example.com'},
        {displayName: 'hello@example.com'},
      ];

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-cc');
      const labelElement = tr.querySelector('th');
      const dataElements = tr.querySelectorAll('mr-user-link');

      assert.equal(labelElement.textContent, 'CC:');
      assert.include(dataElements[0].shadowRoot.textContent.trim(),
          'test@example.com');
      assert.include(dataElements[1].shadowRoot.textContent.trim(),
          'hello@example.com');
    });

    it('renders empty Status', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-status');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'Status:');
      assert.equal(dataElement.textContent.trim(), EMPTY_FIELD_VALUE);
    });

    it('renders populated Status', async () => {
      element.issueStatus = {status: 'Fixed', meansOpen: false};

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-status');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'Status:');
      assert.equal(dataElement.textContent.trim(), 'Fixed (Closed)');
    });

    it('hides empty MergedInto', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-mergedinto');
      assert.isNull(tr);
    });

    it('hides MergedInto when Status is not Duplicate', async () => {
      element.issueStatus = {status: 'test'};
      element.mergedInto = {projectName: 'chromium', localId: 22};

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-mergedinto');
      assert.isNull(tr);
    });

    it('shows MergedInto when Status is Duplicate', async () => {
      element.issueStatus = {status: 'Duplicate'};
      element.mergedInto = {projectName: 'chromium', localId: 22};

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-mergedinto');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('mr-issue-link');

      assert.equal(labelElement.textContent, 'MergedInto:');
      assert.equal(dataElement.shadowRoot.textContent.trim(),
          'Issue chromium:22');
    });

    it('renders empty Components', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-components');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'Components:');
      assert.equal(dataElement.textContent.trim(), EMPTY_FIELD_VALUE);
    });

    it('renders multiple Components', async () => {
      element.components = [
        {path: 'Test', docstring: 'i got docs'},
        {path: 'Test>Nothing'},
      ];

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-components');
      const labelElement = tr.querySelector('th');
      const dataElements = tr.querySelectorAll('td > a');

      assert.equal(labelElement.textContent, 'Components:');

      assert.equal(dataElements[0].textContent.trim(), 'Test');
      assert.equal(dataElements[0].title, 'Test = i got docs');

      assert.equal(dataElements[1].textContent.trim(), 'Test>Nothing');
      assert.equal(dataElements[1].title, 'Test>Nothing');
    });

    it('renders empty Modified', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-modified');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'Modified:');
      assert.equal(dataElement.textContent.trim(), EMPTY_FIELD_VALUE);
    });

    it('renders populated Modified', async () => {
      element.modifiedTimestamp = 1234;

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-modified');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('chops-timestamp');

      assert.equal(labelElement.textContent, 'Modified:');
      assert.equal(dataElement.timestamp, 1234);
    });
  });

  describe('approval fields', () => {
    beforeEach(() => {
      element.builtInFieldSpec = ['ApprovalStatus', 'Approvers', 'Setter',
        'cue.availability_msgs'];
    });

    it('renders empty ApprovalStatus', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-approvalstatus');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'Status:');
      assert.equal(dataElement.textContent.trim(), EMPTY_FIELD_VALUE);
    });

    it('renders populated ApprovalStatus', async () => {
      element.approvalStatus = 'Approved';

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-approvalstatus');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'Status:');
      assert.equal(dataElement.textContent.trim(), 'Approved');
    });

    it('renders empty Approvers', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-approvers');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'Approvers:');
      assert.equal(dataElement.textContent.trim(), EMPTY_FIELD_VALUE);
    });

    it('renders multiple Approvers', async () => {
      element.approvers = [
        {displayName: 'test@example.com'},
        {displayName: 'hello@example.com'},
      ];

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-approvers');
      const labelElement = tr.querySelector('th');
      const dataElements = tr.querySelectorAll('mr-user-link');

      assert.equal(labelElement.textContent, 'Approvers:');
      assert.include(dataElements[0].shadowRoot.textContent.trim(),
          'test@example.com');
      assert.include(dataElements[1].shadowRoot.textContent.trim(),
          'hello@example.com');
    });

    it('hides empty Setter', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-setter');

      assert.isNull(tr);
    });

    it('renders populated Setter', async () => {
      element.setter = {displayName: 'test@example.com'};

      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-setter');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('mr-user-link');

      assert.equal(labelElement.textContent, 'Setter:');
      assert.include(dataElement.shadowRoot.textContent.trim(),
          'test@example.com');
    });

    it('renders cue.availability_msgs', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector(
          'tr.cue-availability_msgs');
      const cueElement = tr.querySelector('mr-cue');

      assert.isDefined(cueElement);
    });
  });

  describe('custom config', () => {
    beforeEach(() => {
      element.builtInFieldSpec = ['owner', 'fakefield'];
    });

    it('owner still renders when lowercase', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-owner');
      const labelElement = tr.querySelector('th');
      const dataElement = tr.querySelector('td');

      assert.equal(labelElement.textContent, 'owner:');
      assert.equal(dataElement.textContent.trim(), EMPTY_FIELD_VALUE);
    });

    it('fakefield does not render', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.row-fakefield');

      assert.isNull(tr);
    });

    it('cue.availability_msgs does not render when not configured', async () => {
      await element.updateComplete;

      const tr = element.shadowRoot.querySelector('tr.cue-availability_msgs');

      assert.isNull(tr);
    });
  });
});
