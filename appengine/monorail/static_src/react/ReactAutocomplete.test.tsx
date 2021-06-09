// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import {assert} from 'chai';
import React from 'react';
import sinon from 'sinon';
import {fireEvent, render} from '@testing-library/react';

import {ReactAutocomplete, MAX_AUTOCOMPLETE_OPTIONS}
  from './ReactAutocomplete.tsx';

describe('ReactAutocomplete', () => {
  it('renders', async () => {
    const {container} = render(<ReactAutocomplete label="cool" options={[]} />);

    assert.isNotNull(container.querySelector('input'));
  });

  it('placeholder renders', async () => {
    const {container} = render(<ReactAutocomplete
      placeholder="penguins"
      options={['']}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    assert.strictEqual(input?.placeholder, 'penguins');
  });

  it('filterOptions empty input value', async () => {
    const {container} = render(<ReactAutocomplete
      label="cool"
      options={['option 1 label']}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    assert.strictEqual(input?.value, '');

    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
    assert.strictEqual(input?.value, '');
  });

  it('filterOptions truncates values', async () => {
    const options = [];

    // a0@test.com, a1@test.com, a2@test.com, ...
    for (let i = 0; i <= MAX_AUTOCOMPLETE_OPTIONS; i++) {
      options.push(`a${i}@test.com`);
    }

    const {container} = render(<ReactAutocomplete
      label="cool"
      options={options}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    fireEvent.change(input, {target: {value: 'a'}});

    const results = document.querySelectorAll('.autocomplete-option');

    assert.equal(results.length, MAX_AUTOCOMPLETE_OPTIONS);

    // Clean up autocomplete dropdown from the DOM for the next test.
    fireEvent.change(input, {target: {value: ''}});
    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
  });

  it('filterOptions label matching', async () => {
    const {container} = render(<ReactAutocomplete
      label="cool"
      options={['option 1 label']}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    assert.strictEqual(input?.value, '');

    fireEvent.change(input, {target: {value: 'lab'}});
    assert.strictEqual(input?.value, 'lab');

    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
    assert.strictEqual(input?.value, 'option 1 label');
  });

  it('filterOptions description matching', async () => {
    const {container} = render(<ReactAutocomplete
      label="cool"
      getOptionDescription={() => 'penguin apples'}
      options={['lol']}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    assert.strictEqual(input?.value, '');

    fireEvent.change(input, {target: {value: 'app'}});
    assert.strictEqual(input?.value, 'app');

    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
    assert.strictEqual(input?.value, 'lol');
  });

  it('filterOptions no match', async () => {
    const {container} = render(<ReactAutocomplete
      label="cool"
      options={[]}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    assert.strictEqual(input?.value, '');

    fireEvent.change(input, {target: {value: 'foobar'}});
    assert.strictEqual(input?.value, 'foobar');

    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
    assert.strictEqual(input?.value, 'foobar');
  });

  it('onChange callback is called', async () => {
    const onChangeStub = sinon.stub();

    const {container} = render(<ReactAutocomplete
      label="cool"
      options={[]}
      onChange={onChangeStub}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    sinon.assert.notCalled(onChangeStub);

    fireEvent.change(input, {target: {value: 'foobar'}});
    sinon.assert.notCalled(onChangeStub);

    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
    sinon.assert.calledOnce(onChangeStub);
  });

  it('onChange excludes fixed values', async () => {
    const onChangeStub = sinon.stub();

    const {container} = render(<ReactAutocomplete
      label="cool"
      options={['cute owl']}
      multiple={true}
      fixedValues={['immortal penguin']}
      onChange={onChangeStub}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    fireEvent.keyDown(input, {key: 'Backspace', code: 'Backspace'});
    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});

    sinon.assert.calledWith(onChangeStub, sinon.match.any, []);
  });

  it('_renderOption highlights matching text', async () => {
    const {container} = render(<ReactAutocomplete
      label="cool"
      options={['cute@owl.com']}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    fireEvent.change(input, {target: {value: 'ow'}});

    const option = document.querySelector('.autocomplete-option');
    const match = option?.querySelector('strong');

    assert.isNotNull(match);
    assert.equal(match?.innerText, 'ow');

    // Description is not rendered.
    assert.equal(option?.querySelectorAll('span').length, 1);
    assert.equal(option?.querySelectorAll('strong').length, 1);

    // Clean up autocomplete dropdown from the DOM for the next test.
    fireEvent.change(input, {target: {value: ''}});
    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
  });

  it('_renderOption highlights matching description', async () => {
    const {container} = render(<ReactAutocomplete
      label="cool"
      getOptionDescription={() => 'penguin of-doom'}
      options={['cute owl']}
    />);

    const input = container.querySelector('input');
    assert.isNotNull(input);
    if (!input) return;

    fireEvent.change(input, {target: {value: 'do'}});

    const option = document.querySelector('.autocomplete-option');
    const match = option?.querySelector('strong');

    assert.isNotNull(match);
    assert.equal(match?.innerText, 'do');

    assert.equal(option?.querySelectorAll('span').length, 2);
    assert.equal(option?.querySelectorAll('strong').length, 1);

    // Clean up autocomplete dropdown from the DOM for the next test.
    fireEvent.change(input, {target: {value: ''}});
    fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
  });

  it('_renderTags disables fixedValues', async () => {
    // TODO(crbug.com/monorail/9393): Add this test once we have a way to stub
    // out dependent components.
  });
});
