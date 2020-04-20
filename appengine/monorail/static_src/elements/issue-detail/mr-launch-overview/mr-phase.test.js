// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import {MrPhase} from './mr-phase.js';


let element;

describe('mr-phase', () => {
  beforeEach(() => {
    element = document.createElement('mr-phase');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrPhase);
  });

  it('clicking edit button opens edit dialog', async () => {
    element.phaseName = 'Beta';

    await element.updateComplete;

    const editDialog = element.shadowRoot.querySelector('#editPhase');
    assert.isFalse(editDialog.opened);

    element.shadowRoot.querySelector('.phase-edit').click();

    await element.updateComplete;

    assert.isTrue(editDialog.opened);
  });

  it('discarding form changes closes dialog', async () => {
    await element.updateComplete;

    // Open the edit dialog.
    element.edit();
    const editDialog = element.shadowRoot.querySelector('#editPhase');
    const editForm = element.shadowRoot.querySelector('#metadataForm');

    await element.updateComplete;

    assert.isTrue(editDialog.opened);
    editForm.discard();

    await element.updateComplete;

    assert.isFalse(editDialog.opened);
  });

  describe('milestone fetching', () => {
    beforeEach(() => {
      sinon.stub(element, 'fetchMilestoneData');
    });

    it('_launchedMilestone extracts M-Launched for phase', () => {
      element._fieldValueMap = new Map([['m-launched beta', ['87']]]);
      element.phaseName = 'Beta';

      assert.equal(element._launchedMilestone, '87');
      assert.equal(element._approvedMilestone, undefined);
      assert.equal(element._targetMilestone, undefined);
    });

    it('_approvedMilestone extracts M-Approved for phase', () => {
      element._fieldValueMap = new Map([['m-approved beta', ['86']]]);
      element.phaseName = 'Beta';

      assert.equal(element._launchedMilestone, undefined);
      assert.equal(element._approvedMilestone, '86');
      assert.equal(element._targetMilestone, undefined);
    });

    it('_targetMilestone extracts M-Target for phase', () => {
      element._fieldValueMap = new Map([['m-target beta', ['85']]]);
      element.phaseName = 'Beta';

      assert.equal(element._launchedMilestone, undefined);
      assert.equal(element._approvedMilestone, undefined);
      assert.equal(element._targetMilestone, '85');
    });

    it('_milestoneToFetch returns empty when no relevant milestone', () => {
      element._fieldValueMap = new Map([['m-target beta', ['85']]]);
      element.phaseName = 'Stable';

      assert.equal(element._milestoneToFetch, '');
    });

    it('_milestoneToFetch selects highest milestone', () => {
      element._fieldValueMap = new Map([
        ['m-target beta', ['84']],
        ['m-approved beta', ['85']],
        ['m-launched beta', ['86']]]);
      element.phaseName = 'Beta';

      assert.equal(element._milestoneToFetch, '86');
    });

    it('does not fetch when no milestones specified', async () => {
      element.issue = {projectName: 'chromium', localId: 12};

      await element.updateComplete;

      sinon.assert.notCalled(element.fetchMilestoneData);
    });

    it('does not fetch when milestone to fetch is unchanged', async () => {
      element._milestoneData = {mstone: 86};
      element._fieldValueMap = new Map([['m-target beta', ['86']]]);
      element.phaseName = 'Beta';

      await element.updateComplete;

      sinon.assert.notCalled(element.fetchMilestoneData);
    });

    it('fetches when milestone found', async () => {
      element._milestoneData = {};
      element._fieldValueMap = new Map([['m-target beta', ['86']]]);
      element.phaseName = 'Beta';

      await element.updateComplete;

      sinon.assert.calledWith(element.fetchMilestoneData, '86');
    });

    it('re-fetches when new milestone found', async () => {
      element._milestoneData = {mstone: 86};
      element._fieldValueMap = new Map([
        ['m-target beta', ['86']],
        ['m-launched beta', ['87']]]);
      element.phaseName = 'Beta';

      await element.updateComplete;

      sinon.assert.calledWith(element.fetchMilestoneData, '87');
    });

    it('re-fetches only after last stale fetch finishes', async () => {
      element._milestoneData = {mstone: 84};
      element._fieldValueMap = new Map([['m-target beta', ['86']]]);
      element.phaseName = 'Beta';
      element._isFetchingMilestone = true;

      await element.updateComplete;

      sinon.assert.notCalled(element.fetchMilestoneData);

      // Previous in flight fetch finishes.
      element._milestoneData = {mstone: 85};
      element._isFetchingMilestone = false;

      await element.updateComplete;

      sinon.assert.calledWith(element.fetchMilestoneData, '86');
    });
  });
});
