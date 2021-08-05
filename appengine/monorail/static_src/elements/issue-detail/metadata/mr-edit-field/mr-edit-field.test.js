// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import userEvent from '@testing-library/user-event';

import {MrEditField} from './mr-edit-field.js';
import {fieldTypes} from 'shared/issue-fields.js';

import {enterInput} from 'shared/test/helpers.js';


let element;
let input;

xdescribe('mr-edit-field', () => {
  beforeEach(async () => {
    element = document.createElement('mr-edit-field');
    document.body.appendChild(element);

    element.label = 'testInput';
    await element.updateComplete;

    input = element.querySelector('#testInput');
  });

  afterEach(async () => {
    userEvent.clear(input);

    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrEditField);
  });

  it('reset input value', async () => {
    element.initialValues = [];
    await element.updateComplete;

    enterInput(input, 'jackalope');
    await element.updateComplete;

    assert.equal(element.value, 'jackalope');

    element.reset();
    await element.updateComplete;

    assert.equal(element.value, '');
  });

  it('input updates when initialValues change', async () => {
    element.initialValues = ['hello'];

    await element.updateComplete;

    assert.equal(element.value, 'hello');
  });

  it('initial value does not change after value set', async () => {
    element.initialValues = ['hello'];
    element.label = 'testInput';
    await element.updateComplete;

    input = element.querySelector('#testInput');

    enterInput(input, 'world');
    await element.updateComplete;

    assert.deepEqual(element.initialValues, ['hello']);
    assert.equal(element.value, 'world');
  });

  it('value updates when input is updated', async () => {
    element.initialValues = ['hello'];
    await element.updateComplete;

    enterInput(input, 'world');
    await element.updateComplete;

    assert.equal(element.value, 'world');
  });

  it('initial value does not change after user input', async () => {
    element.initialValues = ['hello'];
    await element.updateComplete;

    enterInput(input, 'jackalope');
    await element.updateComplete;

    assert.deepEqual(element.initialValues, ['hello']);
    assert.equal(element.value, 'jackalope');
  });

  it('get value after user input', async () => {
    element.initialValues = ['hello'];
    await element.updateComplete;

    enterInput(input, 'jackalope');
    await element.updateComplete;

    assert.equal(element.value, 'jackalope');
  });

  it('input value was added', async () => {
    // Simulate user input.
    await element.updateComplete;

    enterInput(input, 'jackalope');
    await element.updateComplete;

    assert.deepEqual(element.getValuesAdded(), ['jackalope']);
    assert.deepEqual(element.getValuesRemoved(), []);
  });

  it('input value was removed', async () => {
    await element.updateComplete;

    element.initialValues = ['hello'];
    await element.updateComplete;

    enterInput(input, '');
    await element.updateComplete;

    assert.deepEqual(element.getValuesAdded(), []);
    assert.deepEqual(element.getValuesRemoved(), ['hello']);
  });

  it('input value was changed', async () => {
    element.initialValues = ['hello'];
    await element.updateComplete;

    enterInput(input, 'world');
    await element.updateComplete;

    assert.deepEqual(element.getValuesAdded(), ['world']);
  });

  it('edit select updates value when initialValues change', async () => {
    element.multi = false;
    element.type = fieldTypes.ENUM_TYPE;

    element.options = [
      {optionName: 'hello'},
      {optionName: 'jackalope'},
      {optionName: 'text'},
    ];

    element.initialValues = ['hello'];

    await element.updateComplete;

    assert.equal(element.value, 'hello');

    const select = element.querySelector('select');
    userEvent.selectOptions(select, 'jackalope');

    // User input should not be overridden by the initialValue variable.
    assert.equal(element.value, 'jackalope');
    // Initial values should not change based on user input.
    assert.deepEqual(element.initialValues, ['hello']);

    element.initialValues = ['text'];
    await element.updateComplete;

    assert.equal(element.value, 'text');

    element.initialValues = [];
    await element.updateComplete;

    assert.deepEqual(element.value, '');
  });

  it('multi enum updates value on reset', async () => {
    element.multi = true;
    element.type = fieldTypes.ENUM_TYPE;
    element.options = [
      {optionName: 'hello'},
      {optionName: 'world'},
      {optionName: 'fake'},
    ];

    await element.updateComplete;

    element.initialValues = ['hello'];
    element.reset();
    await element.updateComplete;

    assert.deepEqual(element.values, ['hello']);

    const checkboxes = element.querySelector('mr-multi-checkbox');

    // User checks all boxes.
    checkboxes._inputRefs.forEach(
        (checkbox) => {
          checkbox.checked = true;
        },
    );
    checkboxes._changeHandler();

    await element.updateComplete;

    // User input should not be overridden by the initialValues variable.
    assert.deepEqual(element.values, ['hello', 'world', 'fake']);
    // Initial values should not change based on user input.
    assert.deepEqual(element.initialValues, ['hello']);

    element.initialValues = ['hello', 'world'];
    element.reset();
    await element.updateComplete;

    assert.deepEqual(element.values, ['hello', 'world']);
  });
});
