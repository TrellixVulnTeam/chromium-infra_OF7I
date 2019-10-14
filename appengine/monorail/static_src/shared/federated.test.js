// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {
  isShortlinkValid,
  fromShortlink,
  GoogleIssueTrackerIssue,
} from './federated.js';
import {getSigninInstance} from 'shared/gapi-loader.js';

describe('isShortlinkValid', () => {
  it('Returns true for valid links', () => {
    assert.isTrue(isShortlinkValid('b/1'));
    assert.isTrue(isShortlinkValid('b/12345678'));
  });

  it('Returns false for invalid links', () => {
    assert.isFalse(isShortlinkValid('b'));
    assert.isFalse(isShortlinkValid('b/'));
    assert.isFalse(isShortlinkValid('b//123456'));
    assert.isFalse(isShortlinkValid('b/123/123'));
    assert.isFalse(isShortlinkValid('b123/123'));
    assert.isFalse(isShortlinkValid('b/123a456'));
  });
});

describe('fromShortlink', () => {
  it('Returns an issue class for valid links', () => {
    assert.instanceOf(fromShortlink('b/1'), GoogleIssueTrackerIssue);
    assert.instanceOf(fromShortlink('b/12345678'), GoogleIssueTrackerIssue);
  });

  it('Returns null for invalid links', () => {
    assert.isNull(fromShortlink('b'));
    assert.isNull(fromShortlink('b/'));
    assert.isNull(fromShortlink('b//123456'));
    assert.isNull(fromShortlink('b/123/123'));
    assert.isNull(fromShortlink('b123/123'));
    assert.isNull(fromShortlink('b/123a456'));
  });
});

describe('GoogleIssueTrackerIssue', () => {
  describe('constructor', () => {
    it('Sets this.shortlink and this.issueID', () => {
      const shortlink = 'b/1234';
      const issue = new GoogleIssueTrackerIssue(shortlink);
      assert.equal(issue.shortlink, shortlink);
      assert.equal(issue.issueID, 1234);
    });

    it('Throws when given an invalid shortlink.', () => {
      assert.throws(() => {
        new GoogleIssueTrackerIssue('b/123/123');
      });
    });
  });

  describe('toURL', () => {
    it('Returns a valid URL.', () => {
      const issue = new GoogleIssueTrackerIssue('b/1234');
      assert.equal(issue.toURL(), 'https://issuetracker.google.com/issues/1234');
    });
  });

  describe('federated details', () => {
    let signinImpl;
    beforeEach(() => {
      window.CS_env = {gapi_client_id: 'rutabaga'};
      signinImpl = {
        init: sinon.stub(),
        getUserProfileAsync: () => (
          Promise.resolve({
            getEmail: sinon.stub().returns('rutabaga@google.com'),
          })
        ),
      };
      // Preload signinImpl with a fake for testing.
      getSigninInstance(signinImpl, true);
      delete window.__gapiLoadPromise;
    });

    afterEach(() => {
      delete window.CS_env;
    });

    describe('isOpen', () => {
      it('Fails open', () => {
        const issue = new GoogleIssueTrackerIssue('b/1234');
        assert.isTrue(issue.isOpen);
      });

      it('Is based on details.resolvedTime', () => {
        const issue = new GoogleIssueTrackerIssue('b/1234');
        issue._federatedDetails = {resolvedTime: 12345};
        assert.isFalse(issue.isOpen);

        issue._federatedDetails = {};
        assert.isTrue(issue.isOpen);
      });
    });

    describe('summary', () => {
      it('Returns null if not available', () => {
        const issue = new GoogleIssueTrackerIssue('b/1234');
        assert.isNull(issue.summary);
      });

      it('Returns the summary if available', () => {
        const issue = new GoogleIssueTrackerIssue('b/1234');
        issue._federatedDetails = {issueState: {title: 'Rutabaga title'}};
        assert.equal(issue.summary, 'Rutabaga title');
      });
    });

    describe('toIssueRef', () => {
      it('Returns an issue ref object', () => {
        const issue = new GoogleIssueTrackerIssue('b/1234');
        issue._federatedDetails = {
          resolvedTime: 12345,
          issueState: {
            title: 'A fedref issue title',
          },
        };

        assert.deepEqual(issue.toIssueRef(), {
          extIdentifier: 'b/1234',
          summary: 'A fedref issue title',
          statusRef: {meansOpen: false},
        });
      });
    });
  });
});
