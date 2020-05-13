// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {prpcClient} from 'prpc-client-instance.js';
import {stateUpdated} from 'reducers/base.js';
import {MrProjectsPage} from './mr-projects-page.js';

let element;

describe('mr-projects-page', () => {
  beforeEach(() => {
    element = document.createElement('mr-projects-page');
    document.body.appendChild(element);

    sinon.stub(element, 'stateChanged');
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrProjectsPage);
  });

  it('renders loading', async () => {
    element._isFetchingProjects = true;

    await element.updateComplete;

    assert.equal(element.shadowRoot.textContent.trim(), 'Loading...');
  });

  it('renders projects when refetching projects', async () => {
    element._isFetchingProjects = true;
    element._projects = [
      {name: 'projects/chromium', displayName: 'chromium',
        summary: 'Best project ever'},
    ];

    await element.updateComplete;

    const headers = element.shadowRoot.querySelectorAll('h2');

    assert.equal(headers.length, 1);
    assert.equal(headers[0].textContent.trim(), 'All projects');

    const projects = element.shadowRoot.querySelectorAll(
        '.all-projects > .project');
    assert.equal(projects.length, 1);

    assert.include(projects[0].querySelector('h3').textContent, 'chromium');
    assert.include(projects[0].textContent, 'Best project ever');
  });

  it('renders all projects when no user projects', async () => {
    element._isFetchingProjects = false;
    element._projects = [
      {name: 'projects/chromium', displayName: 'chromium',
        summary: 'Best project ever'},
      {name: 'projects/infra', displayName: 'infra',
        summary: 'Make it work'},
    ];

    await element.updateComplete;

    const headers = element.shadowRoot.querySelectorAll('h2');

    assert.equal(headers.length, 1);
    assert.equal(headers[0].textContent.trim(), 'All projects');

    const projects = element.shadowRoot.querySelectorAll(
        '.all-projects > .project');
    assert.equal(projects.length, 2);

    assert.include(projects[0].querySelector('h3').textContent, 'chromium');
    assert.include(projects[0].textContent, 'Best project ever');

    assert.include(projects[1].querySelector('h3').textContent, 'infra');
    assert.include(projects[1].textContent, 'Make it work');
  });

  it('renders no projects found', async () => {
    element._isFetchingProjects = false;
    sinon.stub(element, 'myProjects').get(() => []);
    sinon.stub(element, 'otherProjects').get(() => []);

    await element.updateComplete;

    assert.equal(element.shadowRoot.textContent.trim(), 'No projects found.');
  });

  it('renders user projects', async () => {
    element._isFetchingProjects = false;
    sinon.stub(element, 'myProjects').get(() => [
      {name: 'projects/chromium', displayName: 'chromium',
        summary: 'Best project ever'},
      {name: 'projects/infra', displayName: 'infra',
        summary: 'Make it work'},
    ]);
    sinon.stub(element, 'otherProjects').get(() => []);

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
    element._isFetchingProjects = false;
    sinon.stub(element, 'myProjects').get(() => [
      {name: 'projects/chromium', displayName: 'chromium',
        summary: 'Best project ever'},
    ]);
    sinon.stub(element, 'otherProjects').get(() => [
      {name: 'projects/test', displayName: 'test',
        summary: 'whatevs'},
      {name: 'projects/lit-element', displayName: 'lit-element',
        summary: 'hello world'},
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

describe('mr-projects-page (connected)', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');

    element = document.createElement('mr-projects-page');
  });

  afterEach(() => {
    if (document.body.contains(element)) {
      document.body.removeChild(element);
    }

    prpcClient.call.restore();
  });

  it('fetches projects when connected', async () => {
    const promise = Promise.resolve({
      projects: [{name: 'projects/proj', displayName: 'proj',
        summary: 'test'}],
    });
    prpcClient.call.returns(promise);

    assert.isFalse(element._isFetchingProjects);
    sinon.assert.notCalled(prpcClient.call);

    // Trigger connectedCallback().
    document.body.appendChild(element);
    await stateUpdated, element.updateComplete;

    sinon.assert.calledWith(prpcClient.call, 'monorail.v3.Projects',
        'ListProjects', {});

    assert.isFalse(element._isFetchingProjects);
    assert.deepEqual(element._projects,
        [{name: 'projects/proj', displayName: 'proj',
          summary: 'test'}]);
  });
});
