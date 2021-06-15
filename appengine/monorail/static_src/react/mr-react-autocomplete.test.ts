// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import {MrReactAutocomplete} from './mr-react-autocomplete.tsx';

let element: MrReactAutocomplete;

describe('mr-react-autocomplete', () => {
  beforeEach(() => {
    element = document.createElement('mr-react-autocomplete');
    element.vocabularyName = 'member';
    document.body.appendChild(element);

    sinon.stub(element, 'stateChanged');
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrReactAutocomplete);
  });

  it('ReactDOM renders on update', async () => {
    element.value = 'Penguin Island';

    await element.updateComplete;

    const input = element.querySelector('input');

    assert.equal(input?.value, 'Penguin Island');
  });

  it('_getOptionDescription with component vocabulary gets docstring', () => {
    element.vocabularyName = 'component';
    element._components = new Map([['Infra>UI', {docstring: 'Test docs'}]]);
    element._labels = new Map([['M-84', {docstring: 'Test label docs'}]]);

    assert.equal(element._getOptionDescription('Infra>UI'), 'Test docs');
    assert.equal(element._getOptionDescription('M-84'), '');
    assert.equal(element._getOptionDescription('NoMatch'), '');
  });

  it('_getOptionDescription with label vocabulary gets docstring', () => {
    element.vocabularyName = 'label';
    element._components = new Map([['Infra>UI', {docstring: 'Test docs'}]]);
    element._labels = new Map([['m-84', {docstring: 'Test label docs'}]]);

    assert.equal(element._getOptionDescription('Infra>UI'), '');
    assert.equal(element._getOptionDescription('M-84'), 'Test label docs');
    assert.equal(element._getOptionDescription('NoMatch'), '');
  });

  it('_getOptionDescription with other vocabulary gets empty docstring', () => {
    element.vocabularyName = 'owner';
    element._components = new Map([['Infra>UI', {docstring: 'Test docs'}]]);
    element._labels = new Map([['M-84', {docstring: 'Test label docs'}]]);

    assert.equal(element._getOptionDescription('Infra>UI'), '');
    assert.equal(element._getOptionDescription('M-84'), '');
    assert.equal(element._getOptionDescription('NoMatch'), '');
  });

  it('_options gets component names', () => {
    element.vocabularyName = 'component';
    element._components = new Map([
      ['Infra>UI', {docstring: 'Test docs'}],
      ['Bird>Penguin', {docstring: 'Test docs'}],
    ]);

    assert.deepEqual(element._options(), ['Infra>UI', 'Bird>Penguin']);
  });

  it('_options gets label names', () => {
    element.vocabularyName = 'label';
    element._labels = new Map([
      ['M-84', {label: 'm-84', docstring: 'Test docs'}],
      ['Restrict-View-Bagel', {label: 'restrict-VieW-bAgEl', docstring: 'T'}],
    ]);

    assert.deepEqual(element._options(), ['m-84', 'restrict-VieW-bAgEl']);
  });

  it('_options gets member names with groups', () => {
    element.vocabularyName = 'member';
    element._members = {
      userRefs: [
        {displayName: 'penguin@island.com'},
        {displayName: 'google@monorail.com'},
        {displayName: 'group@birds.com'},
      ],
      groupRefs: [{displayName: 'group@birds.com'}],
    };

    assert.deepEqual(element._options(),
        ['penguin@island.com', 'google@monorail.com', 'group@birds.com']);
  });

  it('_options gets owner names without groups', () => {
    element.vocabularyName = 'owner';
    element._members = {
      userRefs: [
        {displayName: 'penguin@island.com'},
        {displayName: 'google@monorail.com'},
        {displayName: 'group@birds.com'},
      ],
      groupRefs: [{displayName: 'group@birds.com'}],
    };

    assert.deepEqual(element._options(),
        ['penguin@island.com', 'google@monorail.com']);
  });

  it('_options gets owner names without groups', () => {
    element.vocabularyName = 'project';
    element._projects = {
      ownerOf: ['penguins'],
      memberOf: ['birds'],
      contributorTo: ['canary', 'owl-island'],
    };

    assert.deepEqual(element._options(),
        ['penguins', 'birds', 'canary', 'owl-island']);
  });

  it('_options throws error on unknown vocabulary', () => {
    element.vocabularyName = 'whatever';

    assert.throws(element._options.bind(element),
        'Unknown vocabulary name: whatever');
  });
});
