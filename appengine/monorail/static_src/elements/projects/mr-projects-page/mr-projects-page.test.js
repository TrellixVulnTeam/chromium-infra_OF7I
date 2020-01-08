// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrProjectsPage} from './mr-projects-page';

let element;

describe('mr-projects-page', () => {
  beforeEach(() => {
    element = document.createElement('mr-projects-page');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrProjectsPage);
  });

  it('renders user projects', async () => {
    sinon.stub(element, 'myProjects').get(() => [
      {name: 'chromium', summary: 'Best project ever'},
      {name: 'infra', summary: 'Make it work'},
    ]);

    await element.updateComplete;

    const projects = element.shadowRoot.querySelectorAll(
        '.my-projects > .project');

    assert.equal(projects.length, 2);
    assert.include(projects[0].querySelector('h3').textContent, 'chromium');
    assert.include(projects[0].textContent, 'Best project ever');

    assert.include(projects[1].querySelector('h3').textContent, 'infra');
    assert.include(projects[1].textContent, 'Make it work');
  });

  it('renders other projects', async () => {
    sinon.stub(element, 'otherProjects').get(() => [
      {name: 'test', summary: 'whatevs'},
      {name: 'lit-element', summary: 'hello world'},
    ]);

    await element.updateComplete;

    const projects = element.shadowRoot.querySelectorAll(
        '.other-projects > .project');

    assert.equal(projects.length, 2);
    assert.include(projects[0].querySelector('h3').textContent, 'test');
    assert.include(projects[0].textContent, 'whatevs');

    assert.include(projects[1].querySelector('h3').textContent, 'lit-element');
    assert.include(projects[1].textContent, 'hello world');
  });
});
