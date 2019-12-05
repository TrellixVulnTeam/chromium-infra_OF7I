// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {prpcClient} from 'prpc-client-instance.js';
import * as project from './project.js';
import {fieldTypes, SITEWIDE_DEFAULT_COLUMNS} from 'shared/issue-fields.js';

describe('project reducers', () => {
  it('visibleMembersReducer', () => {
    assert.deepEqual(project.visibleMembersReducer({}, {
      type: project.FETCH_VISIBLE_MEMBERS_SUCCESS,
      visibleMembers: {userRefs: [{userId: '123'}]},
    }), {userRefs: [{userId: '123'}]});

    const initialState = {
      groupRefs: [{userId: '543'}],
    };

    // Overrides existing state.
    assert.deepEqual(project.visibleMembersReducer(initialState, {
      type: project.FETCH_VISIBLE_MEMBERS_SUCCESS,
      visibleMembers: {userRefs: [{userId: '123'}]},
    }), {userRefs: [{userId: '123'}]});

    // Unrelated action does not affect state.
    assert.deepEqual(project.visibleMembersReducer(initialState, {
      type: 'no-op',
      visibleMembers: {userRefs: [{userId: '123'}]},
    }), initialState);
  });
});

describe('project selectors', () => {
  it('visibleMembers', () => {
    assert.deepEqual(project.visibleMembers({}), {});
    assert.deepEqual(project.visibleMembers({project: {}}), {});
    assert.deepEqual(project.visibleMembers({project: {
      visibleMembers: {
        userRefs: [{displayName: 'test@example.com', userId: '123'}],
        groupRefs: [],
      },
    }}), {
      userRefs: [{displayName: 'test@example.com', userId: '123'}],
      groupRefs: [],
    });
  });

  it('presentationConfig', () => {
    assert.deepEqual(project.presentationConfig({}), {});
    assert.deepEqual(project.presentationConfig({project: {}}), {});
    assert.deepEqual(project.presentationConfig({project: {
      presentationConfig: {
        projectThumbnailUrl: 'test.png',
      },
    }}), {
      projectThumbnailUrl: 'test.png',
    });
  });

  it('defaultColumns', () => {
    assert.deepEqual(project.defaultColumns({}), SITEWIDE_DEFAULT_COLUMNS);
    assert.deepEqual(project.defaultColumns({project: {}}),
        SITEWIDE_DEFAULT_COLUMNS);
    assert.deepEqual(project.defaultColumns({project: {
      presentationConfig: {},
    }}), SITEWIDE_DEFAULT_COLUMNS);
    assert.deepEqual(project.defaultColumns({project: {
      presentationConfig: {defaultColSpec: 'ID+Summary+AllLabels'},
    }}), ['ID', 'Summary', 'AllLabels']);
  });

  it('defaultQuery', () => {
    assert.deepEqual(project.defaultQuery({}), '');
    assert.deepEqual(project.defaultQuery({project: {}}), '');
    assert.deepEqual(project.defaultQuery({project: {
      presentationConfig: {
        defaultQuery: 'owner:me',
      },
    }}), 'owner:me');
  });

  it('fieldDefs', () => {
    assert.deepEqual(project.fieldDefs({project: {}}), []);
    assert.deepEqual(project.fieldDefs({project: {config: {}}}), []);
    assert.deepEqual(project.fieldDefs({
      project: {config: {fieldDefs: [{fieldRef: {fieldName: 'test'}}]}},
    }), [{fieldRef: {fieldName: 'test'}}]);
  });

  it('labelDefMap', () => {
    assert.deepEqual(project.labelDefMap({project: {}}), new Map());
    assert.deepEqual(project.labelDefMap({project: {config: {}}}), new Map());
    assert.deepEqual(project.labelDefMap({
      project: {config: {
        labelDefs: [
          {label: 'One'},
          {label: 'tWo'},
          {label: 'hello-world', docstring: 'hmmm'},
        ],
      }},
    }), new Map([
      ['one', {label: 'One'}],
      ['two', {label: 'tWo'}],
      ['hello-world', {label: 'hello-world', docstring: 'hmmm'}],
    ]));
  });

  it('labelPrefixOptions', () => {
    assert.deepEqual(project.labelPrefixOptions({project: {}}), new Map());
    assert.deepEqual(project.labelPrefixOptions({project: {config: {}}}),
        new Map());
    assert.deepEqual(project.labelPrefixOptions({
      project: {config: {
        labelDefs: [
          {label: 'One'},
          {label: 'tWo'},
          {label: 'tWo-options'},
          {label: 'hello-world', docstring: 'hmmm'},
          {label: 'hello-me', docstring: 'hmmm'},
        ],
      }},
    }), new Map([
      ['two', ['tWo', 'tWo-options']],
      ['hello', ['hello-world', 'hello-me']],
    ]));
  });

  it('labelPrefixFields', () => {
    assert.deepEqual(project.labelPrefixFields({project: {}}), []);
    assert.deepEqual(project.labelPrefixFields({project: {config: {}}}), []);
    assert.deepEqual(project.labelPrefixFields({
      project: {config: {
        labelDefs: [
          {label: 'One'},
          {label: 'tWo'},
          {label: 'tWo-options'},
          {label: 'hello-world', docstring: 'hmmm'},
          {label: 'hello-me', docstring: 'hmmm'},
        ],
      }},
    }), ['tWo', 'hello']);
  });

  it('enumFieldDefs', () => {
    assert.deepEqual(project.enumFieldDefs({project: {}}), []);
    assert.deepEqual(project.enumFieldDefs({project: {config: {}}}), []);
    assert.deepEqual(project.enumFieldDefs({
      project: {config: {fieldDefs: [
        {fieldRef: {fieldName: 'test'}},
        {fieldRef: {fieldName: 'enum', type: fieldTypes.ENUM_TYPE}},
        {fieldRef: {fieldName: 'ignore', type: fieldTypes.DATE_TYPE}},
      ]}},
    }), [{fieldRef: {fieldName: 'enum', type: fieldTypes.ENUM_TYPE}}]);
  });

  it('optionsPerEnumField', () => {
    assert.deepEqual(project.optionsPerEnumField({project: {}}), new Map());
    assert.deepEqual(project.optionsPerEnumField({
      project: {config: {
        fieldDefs: [
          {fieldRef: {fieldName: 'ignore', type: fieldTypes.DATE_TYPE}},
          {fieldRef: {fieldName: 'eNum', type: fieldTypes.ENUM_TYPE}},
        ],
        labelDefs: [
          {label: 'enum-one'},
          {label: 'ENUM-tWo'},
          {label: 'not-enum-three'},
        ],
      }},
    }), new Map([
      ['enum', [
        {label: 'enum-one', optionName: 'one'},
        {label: 'ENUM-tWo', optionName: 'tWo'},
      ]],
    ]));
  });

  describe('extractFieldValuesFromIssue', () => {
    let clock;
    let issue;
    let fieldExtractor;


    describe('built-in fields', () => {
      beforeEach(() => {
        // Built-in fields will always act the same, regardless of
        // project config.
        fieldExtractor = project.extractFieldValuesFromIssue({});

        // Set clock to some specified date for relative time.
        const initialTime = 365 * 24 * 60 * 60;

        issue = {
          localId: 33,
          projectName: 'chromium',
          summary: 'Test summary',
          attachmentCount: 22,
          starCount: 2,
          componentRefs: [{path: 'Infra'}, {path: 'Monorail>UI'}],
          blockedOnIssueRefs: [{localId: 30, projectName: 'chromium'}],
          blockingIssueRefs: [{localId: 60, projectName: 'chromium'}],
          labelRefs: [{label: 'Restrict-View-Google'}, {label: 'Type-Defect'}],
          reporterRef: {displayName: 'test@example.com'},
          ccRefs: [{displayName: 'test@example.com'}],
          ownerRef: {displayName: 'owner@example.com'},
          closedTimestamp: initialTime - 120, // 2 minutes ago
          modifiedTimestamp: initialTime - 60, // a minute ago
          openedTimestamp: initialTime - 24 * 60 * 60, // a day ago
          componentModifiedTimestamp: initialTime - 60, // a minute ago
          statusModifiedTimestamp: initialTime - 60, // a minute ago
          ownerModifiedTimestamp: initialTime - 60, // a minute ago
          statusRef: {status: 'Duplicate'},
          mergedIntoIssueRef: {localId: 31, projectName: 'chromium'},
        };

        clock = sinon.useFakeTimers({
          now: new Date(initialTime * 1000),
          shouldAdvanceTime: false,
        });
      });

      afterEach(() => {
        clock.restore();
      });

      it('computes strings for ID', () => {
        const fieldName = 'ID';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['chromium:33']);
      });

      it('computes strings for Project', () => {
        const fieldName = 'Project';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['chromium']);
      });

      it('computes strings for Attachments', () => {
        const fieldName = 'Attachments';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['22']);
      });

      it('computes strings for AllLabels', () => {
        const fieldName = 'AllLabels';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['Restrict-View-Google', 'Type-Defect']);
      });

      it('computes strings for Blocked when issue is blocked', () => {
        const fieldName = 'Blocked';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['Yes']);
      });

      it('computes strings for Blocked when issue is not blocked', () => {
        const fieldName = 'Blocked';
        issue.blockedOnIssueRefs = [];

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['No']);
      });

      it('computes strings for BlockedOn', () => {
        const fieldName = 'BlockedOn';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['chromium:30']);
      });

      it('computes strings for Blocking', () => {
        const fieldName = 'Blocking';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['chromium:60']);
      });

      it('computes strings for CC', () => {
        const fieldName = 'CC';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['test@example.com']);
      });

      it('computes strings for Closed', () => {
        const fieldName = 'Closed';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['2 minutes ago']);
      });

      it('computes strings for Component', () => {
        const fieldName = 'Component';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['Infra', 'Monorail>UI']);
      });

      it('computes strings for ComponentModified', () => {
        const fieldName = 'ComponentModified';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['a minute ago']);
      });

      it('computes strings for MergedInto', () => {
        const fieldName = 'MergedInto';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['chromium:31']);
      });

      it('computes strings for Modified', () => {
        const fieldName = 'Modified';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['a minute ago']);
      });

      it('computes strings for Reporter', () => {
        const fieldName = 'Reporter';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['test@example.com']);
      });

      it('computes strings for Stars', () => {
        const fieldName = 'Stars';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['2']);
      });

      it('computes strings for Status', () => {
        const fieldName = 'Status';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['Duplicate']);
      });

      it('computes strings for StatusModified', () => {
        const fieldName = 'StatusModified';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['a minute ago']);
      });

      it('computes strings for Summary', () => {
        const fieldName = 'Summary';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['Test summary']);
      });

      it('computes strings for Type', () => {
        const fieldName = 'Type';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['Defect']);
      });

      it('computes strings for Owner', () => {
        const fieldName = 'Owner';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['owner@example.com']);
      });

      it('computes strings for OwnerModified', () => {
        const fieldName = 'OwnerModified';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['a minute ago']);
      });

      it('computes strings for Opened', () => {
        const fieldName = 'Opened';

        assert.deepEqual(fieldExtractor(issue, fieldName),
            ['a day ago']);
      });
    });

    describe('custom approval fields', () => {
      beforeEach(() => {
        const fieldDefs = [
          {fieldRef: {type: 'APPROVAL_TYPE', fieldName: 'Goose-Approval'}},
          {fieldRef: {type: 'APPROVAL_TYPE', fieldName: 'Chicken-Approval'}},
          {fieldRef: {type: 'APPROVAL_TYPE', fieldName: 'Dodo-Approval'}},
        ];
        fieldExtractor = project.extractFieldValuesFromIssue({
          project: {
            config: {
              projectName: 'chromium',
              fieldDefs,
            },
          },
        });

        issue = {
          localId: 33,
          projectName: 'bird',
          approvalValues: [
            {fieldRef: {type: 'APPROVAL_TYPE', fieldName: 'Goose-Approval'},
              approverRefs: []},
            {fieldRef: {type: 'APPROVAL_TYPE', fieldName: 'Chicken-Approval'},
              status: 'APPROVED'},
            {fieldRef: {type: 'APPROVAL_TYPE', fieldName: 'Dodo-Approval'},
              status: 'NEED_INFO', approverRefs: [
                {displayName: 'kiwi@bird.test'},
                {displayName: 'mini-dino@bird.test'},
              ],
            },
          ],
        };
      });

      it('handles approval approver columns', () => {
        assert.deepEqual(fieldExtractor(issue, 'goose-approval-approver'), []);
        assert.deepEqual(fieldExtractor(issue, 'chicken-approval-approver'),
            []);
        assert.deepEqual(fieldExtractor(issue, 'dodo-approval-approver'),
            ['kiwi@bird.test', 'mini-dino@bird.test']);
      });

      it('handles approval value columns', () => {
        assert.deepEqual(fieldExtractor(issue, 'goose-approval'), ['NotSet']);
        assert.deepEqual(fieldExtractor(issue, 'chicken-approval'),
            ['Approved']);
        assert.deepEqual(fieldExtractor(issue, 'dodo-approval'),
            ['NeedInfo']);
      });
    });

    describe('custom fields', () => {
      beforeEach(() => {
        const fieldDefs = [
          {fieldRef: {type: 'STR_TYPE', fieldName: 'aString'}},
          {fieldRef: {type: 'ENUM_TYPE', fieldName: 'ENUM'}},
          {fieldRef: {type: 'INT_TYPE', fieldName: 'Cow-Number'},
            bool_is_phase_field: true, is_multivalued: true},
        ];
        // As a label prefix, aString conflicts with the custom field named
        // "aString". In this case, Monorail gives precedence to the
        // custom field.
        const labelDefs = [
          {label: 'aString-ignore'},
          {label: 'aString-two'},
        ];
        fieldExtractor = project.extractFieldValuesFromIssue({
          project: {
            config: {
              projectName: 'chromium',
              fieldDefs,
              labelDefs,
            },
          },
        });

        const fieldValues = [
          {fieldRef: {type: 'STR_TYPE', fieldName: 'aString'},
            value: 'test'},
          {fieldRef: {type: 'STR_TYPE', fieldName: 'aString'},
            value: 'test2'},
          {fieldRef: {type: 'ENUM_TYPE', fieldName: 'ENUM'},
            value: 'a-value'},
          {fieldRef: {type: 'INT_TYPE', fieldId: '6', fieldName: 'Cow-Number'},
            phaseRef: {phaseName: 'Cow-Phase'}, value: '55'},
          {fieldRef: {type: 'INT_TYPE', fieldId: '6', fieldName: 'Cow-Number'},
            phaseRef: {phaseName: 'Cow-Phase'}, value: '54'},
          {fieldRef: {type: 'INT_TYPE', fieldId: '6', fieldName: 'Cow-Number'},
            phaseRef: {phaseName: 'MilkCow-Phase'}, value: '56'},
        ];

        issue = {
          localId: 33,
          projectName: 'chromium',
          fieldValues,
        };
      });

      it('gets values for custom fields', () => {
        assert.deepEqual(fieldExtractor(issue, 'aString'), ['test', 'test2']);
        assert.deepEqual(fieldExtractor(issue, 'enum'), ['a-value']);
        assert.deepEqual(fieldExtractor(issue, 'cow-phase.cow-number'),
            ['55', '54']);
        assert.deepEqual(fieldExtractor(issue, 'milkcow-phase.cow-number'),
            ['56']);
      });

      it('custom fields get precedence over label fields', () => {
        issue.labelRefs = [{label: 'aString-ignore'}];
        assert.deepEqual(fieldExtractor(issue, 'aString'),
            ['test', 'test2']);
      });
    });

    describe('label prefix fields', () => {
      beforeEach(() => {
        issue = {
          localId: 33,
          projectName: 'chromium',
          labelRefs: [
            {label: 'test-label'},
            {label: 'test-label-2'},
            {label: 'ignore-me'},
            {label: 'Milestone-UI'},
            {label: 'Milestone-Goodies'},
          ],
        };

        fieldExtractor = project.extractFieldValuesFromIssue({
          project: {
            config: {
              projectName: 'chromium',
              labelDefs: [
                {label: 'test-1'},
                {label: 'test-2'},
                {label: 'milestone-1'},
                {label: 'milestone-2'},
              ],
            },
          },
        });
      });

      it('gets values for label prefixes', () => {
        assert.deepEqual(fieldExtractor(issue, 'test'), ['label', 'label-2']);
        assert.deepEqual(fieldExtractor(issue, 'Milestone'), ['UI', 'Goodies']);
      });
    });
  });

  it('fieldDefsByApprovalName', () => {
    assert.deepEqual(project.fieldDefsByApprovalName({project: {}}), new Map());

    assert.deepEqual(project.fieldDefsByApprovalName({project: {config: {
      fieldDefs: [
        {fieldRef: {fieldName: 'test', type: fieldTypes.INT_TYPE}},
        {fieldRef: {fieldName: 'ignoreMe', type: fieldTypes.APPROVAL_TYPE}},
        {fieldRef: {fieldName: 'yay', approvalName: 'ThisIsAnApproval'}},
        {fieldRef: {fieldName: 'ImAField', approvalName: 'ThisIsAnApproval'}},
        {fieldRef: {fieldName: 'TalkToALawyer', approvalName: 'Legal'}},
      ],
    }}}), new Map([
      ['ThisIsAnApproval', [
        {fieldRef: {fieldName: 'yay', approvalName: 'ThisIsAnApproval'}},
        {fieldRef: {fieldName: 'ImAField', approvalName: 'ThisIsAnApproval'}},
      ]],
      ['Legal', [
        {fieldRef: {fieldName: 'TalkToALawyer', approvalName: 'Legal'}},
      ]],
    ]));
  });
});

let dispatch;

describe('project action creators', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');

    dispatch = sinon.stub();
  });

  afterEach(() => {
    prpcClient.call.restore();
  });

  it('fetchPresentationConfig', async () => {
    const action = project.fetchPresentationConfig('chromium');

    prpcClient.call.returns(Promise.resolve({projectThumbnailUrl: 'test'}));

    await action(dispatch);

    sinon.assert.calledWith(dispatch,
        {type: project.FETCH_PRESENTATION_CONFIG_START});

    sinon.assert.calledWith(
        prpcClient.call,
        'monorail.Projects',
        'GetPresentationConfig',
        {projectName: 'chromium'});

    sinon.assert.calledWith(dispatch, {
      type: project.FETCH_PRESENTATION_CONFIG_SUCCESS,
      presentationConfig: {projectThumbnailUrl: 'test'},
    });
  });

  it('fetchVisibleMembers', async () => {
    const action = project.fetchVisibleMembers('chromium');

    prpcClient.call.returns(Promise.resolve({userRefs: [{userId: '123'}]}));

    await action(dispatch);

    sinon.assert.calledWith(dispatch,
        {type: project.FETCH_VISIBLE_MEMBERS_START});

    sinon.assert.calledWith(
        prpcClient.call,
        'monorail.Projects',
        'GetVisibleMembers',
        {projectName: 'chromium'});

    sinon.assert.calledWith(dispatch, {
      type: project.FETCH_VISIBLE_MEMBERS_SUCCESS,
      visibleMembers: {userRefs: [{userId: '123'}]},
    });
  });
});
