// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {prpcClient} from 'prpc-client-instance.js';
import {stateUpdated} from 'reducers/base.js';
import {users} from 'reducers/users.js';
import {stars} from 'reducers/stars.js';
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

  describe('project grouping', () => {
    beforeEach(() => {
      element._projects = [
        {name: 'projects/chromium', displayName: 'chromium',
          summary: 'Best project ever'},
        {name: 'projects/infra', displayName: 'infra',
          summary: 'Make it work'},
        {name: 'projects/test', displayName: 'test',
          summary: 'Hmm'},
        {name: 'projects/a-project', displayName: 'a-project',
          summary: 'I am Monkeyrail'},
      ];
      element._roleByProjectName = {
        'projects/chromium': 'Owner',
        'projects/infra': 'Committer',
      };
      element._isFetchingProjects = false;
    });

    it('myProjects filters out non-member projects', () => {
      assert.deepEqual(element.myProjects, [
        {name: 'projects/chromium', displayName: 'chromium',
          summary: 'Best project ever'},
        {name: 'projects/infra', displayName: 'infra',
          summary: 'Make it work'},
      ]);
    });

    it('otherProjects filters out member projects', () => {
      assert.deepEqual(element.otherProjects, [
        {name: 'projects/test', displayName: 'test',
          summary: 'Hmm'},
        {name: 'projects/a-project', displayName: 'a-project',
          summary: 'I am Monkeyrail'},
      ]);
    });

    it('renders user projects', async () => {
      await element.updateComplete;

      const projects = element.shadowRoot.querySelectorAll(
          '.my-projects > .project');

      assert.equal(projects.length, 2);
      assert.include(projects[0].querySelector('h3').textContent, 'chromium');
      assert.include(projects[0].textContent, 'Best project ever');
      assert.include(projects[0].querySelector('.subtitle').textContent,
          'Owner');

      assert.include(projects[1].querySelector('h3').textContent, 'infra');
      assert.include(projects[1].textContent, 'Make it work');
      assert.include(projects[1].querySelector('.subtitle').textContent,
          'Committer');
    });

    it('renders other projects', async () => {
      await element.updateComplete;

      const projects = element.shadowRoot.querySelectorAll(
          '.other-projects > .project');

      assert.equal(projects.length, 2);
      assert.include(projects[0].querySelector('h3').textContent, 'test');
      assert.include(projects[0].textContent, 'Hmm');

      assert.include(projects[1].querySelector('h3').textContent, 'a-project');
      assert.include(projects[1].textContent, 'I am Monkeyrail');
    });
  });
});

describe('mr-projects-page (connected)', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');
    sinon.spy(users, 'gatherProjectMemberships');
    sinon.spy(stars, 'listProjects');

    element = document.createElement('mr-projects-page');
  });

  afterEach(() => {
    if (document.body.contains(element)) {
      document.body.removeChild(element);
    }

    prpcClient.call.restore();
    users.gatherProjectMemberships.restore();
    stars.listProjects.restore();
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

  it('does not gather projects when user is logged out', async () => {
    document.body.appendChild(element);
    element._currentUser = '';

    await element.updateComplete;

    sinon.assert.notCalled(users.gatherProjectMemberships);
  });

  it('gathers user projects when user is logged in', async () => {
    document.body.appendChild(element);
    element._currentUser = 'users/1234';

    await element.updateComplete;

    sinon.assert.calledOnce(users.gatherProjectMemberships);
    sinon.assert.calledWith(users.gatherProjectMemberships, 'users/1234');
  });

  it('does not fetch stars user is logged out', async () => {
    document.body.appendChild(element);
    element._currentUser = '';

    await element.updateComplete;

    sinon.assert.notCalled(stars.listProjects);
  });

  it('fetches stars when user is logged in', async () => {
    document.body.appendChild(element);
    element._currentUser = 'users/1234';

    await element.updateComplete;

    sinon.assert.calledOnce(stars.listProjects);
    sinon.assert.calledWith(stars.listProjects, 'users/1234');
  });
});
