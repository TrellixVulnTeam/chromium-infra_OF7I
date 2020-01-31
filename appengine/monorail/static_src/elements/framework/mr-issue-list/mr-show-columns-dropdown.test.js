// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrShowColumnsDropdown} from './mr-show-columns-dropdown.js';

/** @type {MrShowColumnsDropdown} */
let element;

describe('mr-show-columns-dropdown', () => {
  beforeEach(() => {
    element = document.createElement('mr-show-columns-dropdown');
    document.body.appendChild(element);

    sinon.stub(element, '_baseUrl').returns('/p/chromium/issues/list');
    sinon.stub(element, '_page');
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrShowColumnsDropdown);
  });

  it('displaying columns (spa)', async () => {
    element.defaultFields = ['ID', 'Summary', 'AllLabels'];
    element.columns = ['ID'];
    element.issues = [
      {approvalValues: [{fieldRef: {fieldName: 'Approval-Name'}}]},
      {fieldValues: [
        {phaseRef: {phaseName: 'Phase'}, fieldRef: {fieldName: 'Field-Name'}},
        {fieldRef: {fieldName: 'Field-Name'}},
      ]},
      {labelRefs: [{label: 'Label-Name'}]},
    ];

    await element.updateComplete;

    const actual =
        element.items.map((item) => ({icon: item.icon, text: item.text}));
    const expected = [
      {icon: 'check', text: 'ID'},
      {icon: '', text: 'AllLabels'},
      {icon: '', text: 'Approval-Name'},
      {icon: '', text: 'Approval-Name-Approver'},
      {icon: '', text: 'Field-Name'},
      {icon: '', text: 'Label'},
      {icon: '', text: 'Phase.Field-Name'},
      {icon: '', text: 'Summary'},
    ];
    assert.deepEqual(actual, expected);
  });

  describe('displaying columns (ezt)', () => {
    it('sorts default column options', async () => {
      element.defaultFields = ['ID', 'Summary', 'AllLabels'];
      element.columns = [];

      // Re-compute menu items on update.
      await element.updateComplete;
      const options = element.items;

      assert.equal(options.length, 3);

      assert.equal(options[0].text.trim(), 'AllLabels');
      assert.equal(options[0].icon, '');

      assert.equal(options[1].text.trim(), 'ID');
      assert.equal(options[1].icon, '');

      assert.equal(options[2].text.trim(), 'Summary');
      assert.equal(options[2].icon, '');
    });

    it('sorts selected columns above unselected columns', async () => {
      element.defaultFields = ['ID', 'Summary', 'AllLabels'];
      element.columns = ['ID'];

      // Re-compute menu items on update.
      await element.updateComplete;
      const options = element.items;

      assert.equal(options.length, 3);

      assert.equal(options[0].text.trim(), 'ID');
      assert.equal(options[0].icon, 'check');

      assert.equal(options[1].text.trim(), 'AllLabels');
      assert.equal(options[1].icon, '');

      assert.equal(options[2].text.trim(), 'Summary');
      assert.equal(options[2].icon, '');
    });

    it('sorts field defs and label prefix column options', async () => {
      element.defaultFields = ['ID', 'Summary'];
      element.columns = [];
      element._fieldDefs = [
        {fieldRef: {fieldName: 'HelloWorld'}},
        {fieldRef: {fieldName: 'TestField'}},
      ];

      element._labelPrefixFields = ['Milestone', 'Priority'];

      // Re-compute menu items on update.
      await element.updateComplete;
      const options = element.items;

      assert.equal(options.length, 6);
      assert.equal(options[0].text.trim(), 'HelloWorld');
      assert.equal(options[0].icon, '');

      assert.equal(options[1].text.trim(), 'ID');
      assert.equal(options[1].icon, '');

      assert.equal(options[2].text.trim(), 'Milestone');
      assert.equal(options[2].icon, '');

      assert.equal(options[3].text.trim(), 'Priority');
      assert.equal(options[3].icon, '');

      assert.equal(options[4].text.trim(), 'Summary');
      assert.equal(options[4].icon, '');

      assert.equal(options[5].text.trim(), 'TestField');
      assert.equal(options[5].icon, '');
    });

    it('add approver fields for approval type fields', async () => {
      element.defaultFields = [];
      element.columns = [];
      element._fieldDefs = [
        {fieldRef: {fieldName: 'HelloWorld', type: 'APPROVAL_TYPE'}},
      ];

      // Re-compute menu items on update.
      await element.updateComplete;
      const options = element.items;

      assert.equal(options.length, 2);
      assert.equal(options[0].text.trim(), 'HelloWorld');
      assert.equal(options[0].icon, '');

      assert.equal(options[1].text.trim(), 'HelloWorld-Approver');
      assert.equal(options[1].icon, '');
    });

    it('phase field columns are correctly named', async () => {
      element.defaultFields = [];
      element.columns = [];
      element._fieldDefs = [
        {fieldRef: {fieldName: 'Number', type: 'INT_TYPE'}, isPhaseField: true},
        {fieldRef: {fieldName: 'Speak', type: 'STR_TYPE'}, isPhaseField: true},
      ];
      element.phaseNames = ['cow', 'chicken'];

      // Re-compute menu items on update.
      await element.updateComplete;
      const options = element.items;

      assert.equal(options.length, 4);
      assert.equal(options[0].text.trim(), 'chicken.Number');
      assert.equal(options[0].icon, '');

      assert.equal(options[1].text.trim(), 'chicken.Speak');
      assert.equal(options[1].icon, '');

      assert.equal(options[2].text.trim(), 'cow.Number');
      assert.equal(options[2].icon, '');

      assert.equal(options[3].text.trim(), 'cow.Speak');
      assert.equal(options[3].icon, '');
    });
  });

  describe('modifying columns', () => {
    it('clicking unset column adds a column', async () => {
      element.columns = ['ID', 'Summary'];
      element.defaultFields = ['ID', 'Summary', 'AllLabels'];
      element.queryParams = {};

      await element.updateComplete;
      element.clickItem(2);

      sinon.assert.calledWith(element._page,
          '/p/chromium/issues/list?colspec=ID%20Summary%20AllLabels');
    });

    it('clicking set column removes a column', async () => {
      element.columns = ['ID', 'Summary'];
      element.defaultFields = ['ID', 'Summary', 'AllLabels'];
      element.queryParams = {};

      await element.updateComplete;
      element.clickItem(0);

      sinon.assert.calledWith(element._page,
          '/p/chromium/issues/list?colspec=Summary');
    });
  });
});
