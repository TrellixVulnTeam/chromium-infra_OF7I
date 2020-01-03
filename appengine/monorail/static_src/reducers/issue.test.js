// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {createSelector} from 'reselect';
import {store, resetState} from './base.js';
import * as issue from './issue.js';
import {fieldTypes} from 'shared/issue-fields.js';
import {issueToIssueRef} from 'shared/converters.js';
import {prpcClient} from 'prpc-client-instance.js';
import {getSigninInstance} from 'shared/gapi-loader.js';

let prpcCall;
let dispatch;

describe('issue', () => {
  beforeEach(() => {
    store.dispatch(resetState());
  });
  describe('reducers', () => {
    describe('issueByRefReducer', () => {
      it('no-op on unmatching action', () => {
        const action = {
          type: 'FETCH_ISSUE_LIST_FAKE_ACTION',
          issues: [
            {localId: 1, projectName: 'chromium', summary: 'hello-world'},
          ],
        };
        assert.deepEqual(issue.issuesByRefStringReducer({}, action), {});

        assert.deepEqual(issue.issuesByRefStringReducer({
          ['chromium:1']: {localId: 1, projectName: 'chromium'},
        }, action), {
          ['chromium:1']: {localId: 1, projectName: 'chromium'},
        });
      });

      it('handles FETCH_ISSUE_LIST_UPDATE', () => {
        const newState = issue.issuesByRefStringReducer({}, {
          type: 'FETCH_ISSUE_LIST_UPDATE',
          issues: [
            {localId: 1, projectName: 'chromium', summary: 'hello-world'},
            {localId: 2, projectName: 'monorail', summary: 'Test'},
          ],
          totalResults: 2,
          progress: 1,
        });
        assert.deepEqual(newState, {
          'chromium:1': {localId: 1, projectName: 'chromium',
            summary: 'hello-world'},
          'monorail:2': {localId: 2, projectName: 'monorail',
            summary: 'Test'},
        });
      });
    });

    describe('issueListReducer', () => {
      it('no-op on unmatching action', () => {
        const action = {
          type: 'FETCH_ISSUE_LIST_FAKE_ACTION',
          issues: [
            {localId: 1, projectName: 'chromium', summary: 'hello-world'},
          ],
        };
        assert.deepEqual(issue.issueListReducer({}, action), {});

        assert.deepEqual(issue.issueListReducer({
          issueRefs: ['chromium:1'],
          totalResults: 1,
          progress: 1,
        }, action), {
          issueRefs: ['chromium:1'],
          totalResults: 1,
          progress: 1,
        });
      });

      it('handles FETCH_ISSUE_LIST_UPDATE', () => {
        const newState = issue.issueListReducer({}, {
          type: 'FETCH_ISSUE_LIST_UPDATE',
          issues: [
            {localId: 1, projectName: 'chromium', summary: 'hello-world'},
            {localId: 2, projectName: 'monorail', summary: 'Test'},
          ],
          totalResults: 2,
          progress: 1,
        });
        assert.deepEqual(newState, {
          issueRefs: ['chromium:1', 'monorail:2'],
          totalResults: 2,
          progress: 1,
        });
      });
    });

    describe('relatedIssuesReducer', () => {
      it('handles FETCH_RELATED_ISSUES_SUCCESS', () => {
        const newState = issue.relatedIssuesReducer({}, {
          type: 'FETCH_RELATED_ISSUES_SUCCESS',
          relatedIssues: {'rutabaga:1234': {}},
        });
        assert.deepEqual(newState, {'rutabaga:1234': {}});
      });

      describe('FETCH_FEDERATED_REFERENCES_SUCCESS', () => {
        it('returns early if data is missing', () => {
          const newState = issue.relatedIssuesReducer({'b/123': {}}, {
            type: 'FETCH_FEDERATED_REFERENCES_SUCCESS',
          });
          assert.deepEqual(newState, {'b/123': {}});
        });

        it('returns early if data is empty', () => {
          const newState = issue.relatedIssuesReducer({'b/123': {}}, {
            type: 'FETCH_FEDERATED_REFERENCES_SUCCESS',
            fedRefIssueRefs: [],
          });
          assert.deepEqual(newState, {'b/123': {}});
        });

        it('assigns each FedRef to the state', () => {
          const state = {
            'rutabaga:123': {},
            'rutabaga:345': {},
          };
          const newState = issue.relatedIssuesReducer(state, {
            type: 'FETCH_FEDERATED_REFERENCES_SUCCESS',
            fedRefIssueRefs: [
              {
                extIdentifier: 'b/987',
                summary: 'What is up',
                statusRef: {meansOpen: true},
              },
              {
                extIdentifier: 'b/765',
                summary: 'Rutabaga',
                statusRef: {meansOpen: false},
              },
            ],
          });
          assert.deepEqual(newState, {
            'rutabaga:123': {},
            'rutabaga:345': {},
            'b/987': {
              extIdentifier: 'b/987',
              summary: 'What is up',
              statusRef: {meansOpen: true},
            },
            'b/765': {
              extIdentifier: 'b/765',
              summary: 'Rutabaga',
              statusRef: {meansOpen: false},
            },
          });
        });
      });
    });
  });

  it('issue', () => {
    assert.deepEqual(issue.issue(wrapIssue()), {});
    assert.deepEqual(issue.issue(wrapIssue({localId: 100})), {localId: 100});
  });

  describe('issueList', () => {
    it('issueList', () => {
      const stateWithEmptyIssueList = {issue: {
        issueList: [],
      }};
      assert.deepEqual(issue.issueList(stateWithEmptyIssueList), []);

      const stateWithIssueList = {issue: {
        issuesByRefString: {
          'chromium:1': {localId: 1, projectName: 'chromium', summary: 'test'},
          'monorail:2': {localId: 2, projectName: 'monorail',
            summary: 'hello world'},
        },
        issueList: {
          issueRefs: ['chromium:1', 'monorail:2'],
        }}};
      assert.deepEqual(issue.issueList(stateWithIssueList),
          [
            {localId: 1, projectName: 'chromium', summary: 'test'},
            {localId: 2, projectName: 'monorail', summary: 'hello world'},
          ]);
    });

    it('is a selector', () => {
      issue.issueList.constructor === createSelector;
    });

    it('memoizes results: returns same reference', () => {
      const stateWithIssueList = {issue: {
        issuesByRefString: {
          'chromium:1': {localId: 1, projectName: 'chromium', summary: 'test'},
          'monorail:2': {localId: 2, projectName: 'monorail',
            summary: 'hello world'},
        },
        issueList: {
          issueRefs: ['chromium:1', 'monorail:2'],
        }}};
      const reference1 = issue.issueList(stateWithIssueList);
      const reference2 = issue.issueList(stateWithIssueList);

      assert.equal(typeof reference1, 'object');
      assert.equal(typeof reference2, 'object');
      assert.equal(reference1, reference2);
    });
  });

  it('fieldValues', () => {
    assert.isUndefined(issue.fieldValues(wrapIssue()));
    assert.deepEqual(issue.fieldValues(wrapIssue({
      fieldValues: [{value: 'v'}],
    })), [{value: 'v'}]);
  });

  it('type computes type from custom field', () => {
    assert.isUndefined(issue.type(wrapIssue()));
    assert.isUndefined(issue.type(wrapIssue({
      fieldValues: [{value: 'v'}],
    })));
    assert.deepEqual(issue.type(wrapIssue({
      fieldValues: [
        {fieldRef: {fieldName: 'IgnoreMe'}, value: 'v'},
        {fieldRef: {fieldName: 'Type'}, value: 'Defect'},
      ],
    })), 'Defect');
  });

  it('type computes type from label', () => {
    assert.deepEqual(issue.type(wrapIssue({
      labelRefs: [
        {label: 'Test'},
        {label: 'tYpE-FeatureRequest'},
      ],
    })), 'FeatureRequest');

    assert.deepEqual(issue.type(wrapIssue({
      fieldValues: [
        {fieldRef: {fieldName: 'IgnoreMe'}, value: 'v'},
      ],
      labelRefs: [
        {label: 'Test'},
        {label: 'Type-Defect'},
      ],
    })), 'Defect');
  });

  it('restrictions', () => {
    assert.deepEqual(issue.restrictions(wrapIssue()), {});
    assert.deepEqual(issue.restrictions(wrapIssue({labelRefs: []})), {});

    assert.deepEqual(issue.restrictions(wrapIssue({labelRefs: [
      {label: 'IgnoreThis'},
      {label: 'IgnoreThis2'},
    ]})), {});

    assert.deepEqual(issue.restrictions(wrapIssue({labelRefs: [
      {label: 'IgnoreThis'},
      {label: 'IgnoreThis2'},
      {label: 'Restrict-View-Google'},
      {label: 'Restrict-EditIssue-hello'},
      {label: 'Restrict-EditIssue-test'},
      {label: 'Restrict-AddIssueComment-HELLO'},
    ]})), {
      'view': ['Google'],
      'edit': ['hello', 'test'],
      'comment': ['HELLO'],
    });
  });

  it('isOpen', () => {
    assert.isFalse(issue.isOpen(wrapIssue()));
    assert.isTrue(issue.isOpen(wrapIssue({statusRef: {meansOpen: true}})));
    assert.isFalse(issue.isOpen(wrapIssue({statusRef: {meansOpen: false}})));
  });

  it('issueListPhaseNames', () => {
    const stateWithEmptyIssueList = {issue: {
      issueList: [],
    }};
    assert.deepEqual(issue.issueListPhaseNames(stateWithEmptyIssueList), []);
    const stateWithIssueList = {issue: {
      issuesByRefString: {
        '1': {localId: 1, phases: [{phaseRef: {phaseName: 'chicken-phase'}}]},
        '2': {localId: 2, phases: [
          {phaseRef: {phaseName: 'chicken-Phase'}},
          {phaseRef: {phaseName: 'cow-phase'}}],
        },
        '3': {localId: 3, phases: [
          {phaseRef: {phaseName: 'cow-Phase'}},
          {phaseRef: {phaseName: 'DOG-phase'}}],
        },
        '4': {localId: 4, phases: [
          {phaseRef: {phaseName: 'dog-phase'}},
        ]},
      },
      issueList: {
        issueRefs: ['1', '2', '3', '4'],
      }}};
    assert.deepEqual(issue.issueListPhaseNames(stateWithIssueList),
        ['chicken-phase', 'cow-phase', 'dog-phase']);
  });

  it('blockingIssues', () => {
    const relatedIssues = {
      ['proj:1']: {
        localId: 1,
        projectName: 'proj',
        labelRefs: [{label: 'label'}],
      },
      ['proj:3']: {
        localId: 3,
        projectName: 'proj',
        labelRefs: [],
      },
      ['chromium:332']: {
        localId: 332,
        projectName: 'chromium',
        labelRefs: [],
      },
    };
    const stateNoReferences = {issue: {
      currentIssue: {
        blockingIssueRefs: [{localId: 1, projectName: 'proj'}],
      },
      relatedIssues: {},
    }};
    assert.deepEqual(issue.blockingIssues(stateNoReferences),
        [{localId: 1, projectName: 'proj'}],
    );

    const stateNoIssues = {issue: {
      currentIssue: {
        blockingIssueRefs: [],
      },
      relatedIssues: relatedIssues,
    }};
    assert.deepEqual(issue.blockingIssues(stateNoIssues), []);

    const stateIssuesWithReferences = {issue: {
      currentIssue: {
        blockingIssueRefs: [
          {localId: 1, projectName: 'proj'},
          {localId: 332, projectName: 'chromium'},
        ],
      },
      relatedIssues: relatedIssues,
    }};
    assert.deepEqual(issue.blockingIssues(stateIssuesWithReferences),
        [
          {localId: 1, projectName: 'proj', labelRefs: [{label: 'label'}]},
          {localId: 332, projectName: 'chromium', labelRefs: []},
        ]);

    const stateIssuesWithFederatedReferences = {issue: {
      currentIssue: {
        blockingIssueRefs: [
          {localId: 1, projectName: 'proj'},
          {extIdentifier: 'b/1234'},
        ],
      },
      relatedIssues: relatedIssues,
    }};
    assert.deepEqual(issue.blockingIssues(stateIssuesWithFederatedReferences),
        [
          {localId: 1, projectName: 'proj', labelRefs: [{label: 'label'}]},
          {extIdentifier: 'b/1234'},
        ]);
  });

  it('blockedOnIssues', () => {
    const relatedIssues = {
      ['proj:1']: {
        localId: 1,
        projectName: 'proj',
        labelRefs: [{label: 'label'}],
      },
      ['proj:3']: {
        localId: 3,
        projectName: 'proj',
        labelRefs: [],
      },
      ['chromium:332']: {
        localId: 332,
        projectName: 'chromium',
        labelRefs: [],
      },
    };
    const stateNoReferences = {issue: {
      currentIssue: {
        blockedOnIssueRefs: [{localId: 1, projectName: 'proj'}],
      },
      relatedIssues: {},
    }};
    assert.deepEqual(issue.blockedOnIssues(stateNoReferences),
        [{localId: 1, projectName: 'proj'}],
    );

    const stateNoIssues = {issue: {
      currentIssue: {
        blockedOnIssueRefs: [],
      },
      relatedIssues: relatedIssues,
    }};
    assert.deepEqual(issue.blockedOnIssues(stateNoIssues), []);

    const stateIssuesWithReferences = {issue: {
      currentIssue: {
        blockedOnIssueRefs: [
          {localId: 1, projectName: 'proj'},
          {localId: 332, projectName: 'chromium'},
        ],
      },
      relatedIssues: relatedIssues,
    }};
    assert.deepEqual(issue.blockedOnIssues(stateIssuesWithReferences),
        [
          {localId: 1, projectName: 'proj', labelRefs: [{label: 'label'}]},
          {localId: 332, projectName: 'chromium', labelRefs: []},
        ]);

    const stateIssuesWithFederatedReferences = {issue: {
      currentIssue: {
        blockedOnIssueRefs: [
          {localId: 1, projectName: 'proj'},
          {extIdentifier: 'b/1234'},
        ],
      },
      relatedIssues: relatedIssues,
    }};
    assert.deepEqual(issue.blockedOnIssues(stateIssuesWithFederatedReferences),
        [
          {localId: 1, projectName: 'proj', labelRefs: [{label: 'label'}]},
          {extIdentifier: 'b/1234'},
        ]);
  });

  it('sortedBlockedOn', () => {
    const relatedIssues = {
      ['proj:1']: {
        localId: 1,
        projectName: 'proj',
        statusRef: {meansOpen: true},
      },
      ['proj:3']: {
        localId: 3,
        projectName: 'proj',
        statusRef: {meansOpen: false},
      },
      ['proj:4']: {
        localId: 4,
        projectName: 'proj',
        statusRef: {meansOpen: false},
      },
      ['proj:5']: {
        localId: 5,
        projectName: 'proj',
        statusRef: {meansOpen: false},
      },
      ['chromium:332']: {
        localId: 332,
        projectName: 'chromium',
        statusRef: {meansOpen: true},
      },
    };
    const stateNoReferences = {issue: {
      currentIssue: {
        blockedOnIssueRefs: [
          {localId: 3, projectName: 'proj'},
          {localId: 1, projectName: 'proj'},
        ],
      },
      relatedIssues: {},
    }};
    assert.deepEqual(issue.sortedBlockedOn(stateNoReferences), [
      {localId: 3, projectName: 'proj'},
      {localId: 1, projectName: 'proj'},
    ]);
    const stateReferences = {issue: {
      currentIssue: {
        blockedOnIssueRefs: [
          {localId: 3, projectName: 'proj'},
          {localId: 1, projectName: 'proj'},
        ],
      },
      relatedIssues: relatedIssues,
    }};
    assert.deepEqual(issue.sortedBlockedOn(stateReferences), [
      {localId: 1, projectName: 'proj', statusRef: {meansOpen: true}},
      {localId: 3, projectName: 'proj', statusRef: {meansOpen: false}},
    ]);
    const statePreservesArrayOrder = {issue: {
      currentIssue: {
        blockedOnIssueRefs: [
          {localId: 5, projectName: 'proj'}, // Closed
          {localId: 1, projectName: 'proj'}, // Open
          {localId: 4, projectName: 'proj'}, // Closed
          {localId: 3, projectName: 'proj'}, // Closed
          {localId: 332, projectName: 'chromium'}, // Open
        ],
      },
      relatedIssues: relatedIssues,
    }};
    assert.deepEqual(issue.sortedBlockedOn(statePreservesArrayOrder),
        [
          {localId: 1, projectName: 'proj', statusRef: {meansOpen: true}},
          {localId: 332, projectName: 'chromium', statusRef: {meansOpen: true}},
          {localId: 5, projectName: 'proj', statusRef: {meansOpen: false}},
          {localId: 4, projectName: 'proj', statusRef: {meansOpen: false}},
          {localId: 3, projectName: 'proj', statusRef: {meansOpen: false}},
        ],
    );
  });

  it('mergedInto', () => {
    assert.deepEqual(issue.mergedInto(wrapIssue()), {});
    assert.deepEqual(issue.mergedInto(wrapIssue({
      mergedIntoIssueRef: {localId: 22, projectName: 'proj'},
    })), {
      localId: 22,
      projectName: 'proj',
    });

    const merged = issue.mergedInto({
      issue: {
        currentIssue: {
          mergedIntoIssueRef: {localId: 22, projectName: 'proj'},
        },
        relatedIssues: {
          ['proj:22']: {localId: 22, projectName: 'proj', summary: 'test'},
        },
      },
    });
    assert.deepEqual(merged, {
      localId: 22,
      projectName: 'proj',
      summary: 'test',
    });
  });

  it('fieldValueMap', () => {
    assert.deepEqual(issue.fieldValueMap(wrapIssue()), new Map());
    assert.deepEqual(issue.fieldValueMap(wrapIssue({
      fieldValues: [],
    })), new Map());
    assert.deepEqual(issue.fieldValueMap(wrapIssue({
      fieldValues: [
        {fieldRef: {fieldName: 'hello'}, value: 'v1'},
        {fieldRef: {fieldName: 'hello'}, value: 'v2'},
        {fieldRef: {fieldName: 'world'}, value: 'v3'},
      ],
    })), new Map([
      ['hello', ['v1', 'v2']],
      ['world', ['v3']],
    ]));
  });

  it('fieldDefs filters fields by applicable type', () => {
    assert.deepEqual(issue.fieldDefs({
      project: {},
      ...wrapIssue(),
    }), []);

    assert.deepEqual(issue.fieldDefs({
      project: {
        name: 'chromium',
        configs: {
          chromium: {
            fieldDefs: [
              {fieldRef: {fieldName: 'intyInt', type: fieldTypes.INT_TYPE}},
              {fieldRef: {fieldName: 'enum', type: fieldTypes.ENUM_TYPE}},
              {
                fieldRef:
                  {fieldName: 'nonApplicable', type: fieldTypes.STR_TYPE},
                applicableType: 'None',
              },
              {fieldRef: {fieldName: 'defectsOnly', type: fieldTypes.STR_TYPE},
                applicableType: 'Defect'},
            ],
          },
        },
      },
      ...wrapIssue({
        fieldValues: [
          {fieldRef: {fieldName: 'Type'}, value: 'Defect'},
        ],
      }),
    }), [
      {fieldRef: {fieldName: 'intyInt', type: fieldTypes.INT_TYPE}},
      {fieldRef: {fieldName: 'enum', type: fieldTypes.ENUM_TYPE}},
      {fieldRef: {fieldName: 'defectsOnly', type: fieldTypes.STR_TYPE},
        applicableType: 'Defect'},
    ]);
  });

  it('fieldDefs skips approval fields for all issues', () => {
    assert.deepEqual(issue.fieldDefs({
      project: {
        name: 'chromium',
        configs: {
          chromium: {
            fieldDefs: [
              {fieldRef: {fieldName: 'test', type: fieldTypes.INT_TYPE}},
              {fieldRef:
                {fieldName: 'ignoreMe', type: fieldTypes.APPROVAL_TYPE}},
              {fieldRef:
                {fieldName: 'LookAway', approvalName: 'ThisIsAnApproval'}},
              {fieldRef: {fieldName: 'phaseField'}, isPhaseField: true},
            ],
          },
        },
      },
      ...wrapIssue(),
    }), [
      {fieldRef: {fieldName: 'test', type: fieldTypes.INT_TYPE}},
    ]);
  });

  it('fieldDefs includes non applicable fields when values defined', () => {
    assert.deepEqual(issue.fieldDefs({
      project: {
        name: 'chromium',
        configs: {
          chromium: {
            fieldDefs: [
              {
                fieldRef:
                  {fieldName: 'nonApplicable', type: fieldTypes.STR_TYPE},
                applicableType: 'None',
              },
            ],
          },
        },
      },
      ...wrapIssue({
        fieldValues: [
          {fieldRef: {fieldName: 'nonApplicable'}, value: 'v1'},
        ],
      }),
    }), [
      {fieldRef: {fieldName: 'nonApplicable', type: fieldTypes.STR_TYPE},
        applicableType: 'None'},
    ]);
  });

  describe('action creators', () => {
    beforeEach(() => {
      prpcCall = sinon.stub(prpcClient, 'call');
    });

    afterEach(() => {
      prpcCall.restore();
    });

    it('predictComponent sends prediction request', async () => {
      prpcCall.callsFake(() => {
        return {componentRef: {path: 'UI>Test'}};
      });

      const dispatch = sinon.stub();

      const action = issue.predictComponent('chromium',
          'test comments\nsummary');

      await action(dispatch);

      sinon.assert.calledOnce(prpcCall);

      sinon.assert.calledWith(prpcCall, 'monorail.Features',
          'PredictComponent', {
            projectName: 'chromium',
            text: 'test comments\nsummary',
          });

      sinon.assert.calledWith(dispatch, {type: 'PREDICT_COMPONENT_START'});
      sinon.assert.calledWith(dispatch, {
        type: 'PREDICT_COMPONENT_SUCCESS',
        component: 'UI>Test',
      });
    });

    it('fetchIssueList calls ListIssues', async () => {
      prpcCall.callsFake(() => {
        return {
          issues: [{localId: 1}, {localId: 2}, {localId: 3}],
          totalResults: 6,
        };
      });

      store.dispatch(issue.fetchIssueList({q: 'owner:me', can: '4'},
          'chromium'));

      sinon.assert.calledWith(prpcCall, 'monorail.Issues', 'ListIssues', {
        query: 'owner:me',
        cannedQuery: 4,
        projectNames: ['chromium'],
        pagination: {},
        groupBySpec: undefined,
        sortSpec: undefined,
      });
    });

    it('fetchIssueList does not set can when can is NaN', async () => {
      prpcCall.callsFake(() => ({}));

      store.dispatch(issue.fetchIssueList({q: 'owner:me',
        can: 'four-leaf-clover'}, 'chromium'));

      sinon.assert.calledWith(prpcCall, 'monorail.Issues', 'ListIssues', {
        query: 'owner:me',
        cannedQuery: undefined,
        projectNames: ['chromium'],
        pagination: {},
        groupBySpec: undefined,
        sortSpec: undefined,
      });
    });

    it('fetchIssueList makes several calls to ListIssues', async () => {
      prpcCall.callsFake(() => {
        return {
          issues: [{localId: 1}, {localId: 2}, {localId: 3}],
          totalResults: 6,
        };
      });

      const dispatch = sinon.stub();
      const action = issue.fetchIssueList({}, 'chromium', {maxItems: 3}, 2);
      await action(dispatch);

      sinon.assert.calledTwice(prpcCall);
      sinon.assert.calledWith(dispatch, {
        type: 'FETCH_ISSUE_LIST_UPDATE',
        issues:
          [{localId: 1}, {localId: 2}, {localId: 3},
            {localId: 1}, {localId: 2}, {localId: 3}],
        progress: 1,
        totalResults: 6,
      });
      sinon.assert.calledWith(dispatch, {type: 'FETCH_ISSUE_LIST_SUCCESS'});
    });

    it('fetchIssueList orders issues correctly', async () => {
      prpcCall.onFirstCall().returns({issues: [{localId: 1}], totalResults: 6});
      prpcCall.onSecondCall().returns({
        issues: [{localId: 2}],
        totalResults: 6});
      prpcCall.onThirdCall().returns({issues: [{localId: 3}], totalResults: 6});

      const dispatch = sinon.stub();
      const action = issue.fetchIssueList({}, 'chromium', {maxItems: 1}, 3);
      await action(dispatch);

      sinon.assert.calledWith(dispatch, {
        type: 'FETCH_ISSUE_LIST_UPDATE',
        issues: [{localId: 1}, {localId: 2}, {localId: 3}],
        progress: 1,
        totalResults: 6,
      });
      sinon.assert.calledWith(dispatch, {type: 'FETCH_ISSUE_LIST_SUCCESS'});
    });

    it('returns progress of 1 when no totalIssues', async () => {
      prpcCall.onFirstCall().returns({issues: [], totalResults: 0});

      const dispatch = sinon.stub();
      const action = issue.fetchIssueList({}, 'chromium', {maxItems: 1}, 1);
      await action(dispatch);

      sinon.assert.calledWith(dispatch, {
        type: 'FETCH_ISSUE_LIST_UPDATE',
        issues: [],
        progress: 1,
        totalResults: 0,
      });
      sinon.assert.calledWith(dispatch, {type: 'FETCH_ISSUE_LIST_SUCCESS'});
    });

    it('returns progress of 1 when totalIssues undefined', async () => {
      prpcCall.onFirstCall().returns({issues: []});

      const dispatch = sinon.stub();
      const action = issue.fetchIssueList({}, 'chromium', {maxItems: 1}, 1);
      await action(dispatch);

      sinon.assert.calledWith(dispatch, {
        type: 'FETCH_ISSUE_LIST_UPDATE',
        issues: [],
        progress: 1,
      });
      sinon.assert.calledWith(dispatch, {type: 'FETCH_ISSUE_LIST_SUCCESS'});
    });

    // TODO(kweng@) remove once crbug.com/monorail/6641 is fixed
    it('has sane default for empty response', async () => {
      prpcCall.onFirstCall().returns({});

      const dispatch = sinon.stub();
      const action = issue.fetchIssueList({}, 'chromium', {maxItems: 1}, 1);
      await action(dispatch);

      sinon.assert.calledWith(dispatch, {
        type: 'FETCH_ISSUE_LIST_UPDATE',
        issues: [],
        progress: 1,
        totalResults: 0,
      });
      sinon.assert.calledWith(dispatch, {type: 'FETCH_ISSUE_LIST_SUCCESS'});
    });

    describe('federated references', () => {
      beforeEach(() => {
        // Preload signinImpl with a fake for testing.
        getSigninInstance({
          init: sinon.stub(),
          getUserProfileAsync: () => (
            Promise.resolve({
              getEmail: sinon.stub().returns('rutabaga@google.com'),
            })
          ),
        });
        window.CS_env = {gapi_client_id: 'rutabaga'};
        const getStub = sinon.stub().returns({
          execute: (cb) => cb(response),
        });
        const response = {
          result: {
            resolvedTime: 12345,
            issueState: {
              title: 'Rutabaga title',
            },
          },
        };
        window.gapi = {
          client: {
            load: (_url, _version, cb) => cb(),
            corp_issuetracker: {issues: {get: getStub}},
          },
        };
      });

      afterEach(() => {
        delete window.CS_env;
        delete window.gapi;
      });

      describe('fetchFederatedReferences', () => {
        it('returns an empty map if no fedrefs found', async () => {
          const dispatch = sinon.stub();
          const testIssue = {};
          const action = issue.fetchFederatedReferences(testIssue);
          const result = await action(dispatch);

          assert.equal(dispatch.getCalls().length, 1);
          sinon.assert.calledWith(dispatch, {
            type: 'FETCH_FEDERATED_REFERENCES_START',
          });
          assert.isUndefined(result);
        });

        it('fetches from Buganizer API', async () => {
          const dispatch = sinon.stub();
          const testIssue = {
            danglingBlockingRefs: [
              {extIdentifier: 'b/123456'},
            ],
            danglingBlockedOnRefs: [
              {extIdentifier: 'b/654321'},
            ],
            mergedIntoIssueRef: {
              extIdentifier: 'b/987654',
            },
          };
          const action = issue.fetchFederatedReferences(testIssue);
          await action(dispatch);

          sinon.assert.calledWith(dispatch, {
            type: 'FETCH_FEDERATED_REFERENCES_START',
          });
          sinon.assert.calledWith(dispatch, {
            type: 'GAPI_LOGIN_SUCCESS',
            email: 'rutabaga@google.com',
          });
          sinon.assert.calledWith(dispatch, {
            type: 'FETCH_FEDERATED_REFERENCES_SUCCESS',
            fedRefIssueRefs: [
              {
                extIdentifier: 'b/123456',
                statusRef: {meansOpen: false},
                summary: 'Rutabaga title',
              },
              {
                extIdentifier: 'b/654321',
                statusRef: {meansOpen: false},
                summary: 'Rutabaga title',
              },
              {
                extIdentifier: 'b/987654',
                statusRef: {meansOpen: false},
                summary: 'Rutabaga title',
              },
            ],
          });
        });
      });

      describe('fetchRelatedIssues', () => {
        it('calls fetchFederatedReferences for mergedinto', async () => {
          const dispatch = sinon.stub();
          prpcCall.returns(Promise.resolve({openRefs: [], closedRefs: []}));
          const testIssue = {
            mergedIntoIssueRef: {
              extIdentifier: 'b/987654',
            },
          };
          const action = issue.fetchRelatedIssues(testIssue);
          await action(dispatch);

          // Important: mergedinto fedref is not passed to ListReferencedIssues.
          const expectedMessage = {issueRefs: []};
          sinon.assert.calledWith(prpcClient.call, 'monorail.Issues',
              'ListReferencedIssues', expectedMessage);

          sinon.assert.calledWith(dispatch, {
            type: 'FETCH_RELATED_ISSUES_START',
          });
          // No mergedInto refs returned, they're handled by
          // fetchFederatedReferences.
          sinon.assert.calledWith(dispatch, {
            type: 'FETCH_RELATED_ISSUES_SUCCESS',
            relatedIssues: {},
          });
        });
      });
    });
  });

  describe('starring issues', () => {
    describe('reducers', () => {
      it('FETCH_IS_STARRED_SUCCESS updates the starredIssues object', () => {
        const state = {};
        const newState = issue.starredIssuesReducer(state,
            {
              type: issue.FETCH_IS_STARRED_SUCCESS,
              starred: false,
              ref: {
                issueRef: {
                  projectName: 'proj',
                  localId: 1,
                },
              },
            },
        );
        assert.deepEqual(newState, {'proj:1': false});
      });

      it('FETCH_ISSUES_STARRED_SUCCESS updates the starredIssues object',
          () => {
            const state = {};
            const starredIssues = ['proj:1', 'proj:2'];
            const newState = issue.starredIssuesReducer(state,
                {type: issue.FETCH_ISSUES_STARRED_SUCCESS, starredIssues},
            );
            assert.deepEqual(newState, {'proj:1': true, 'proj:2': true});
          });

      it('STAR_SUCCESS updates the starredIssues object', () => {
        const state = {'proj:1': true, 'proj:2': false};
        const newState = issue.starredIssuesReducer(state,
            {type: issue.STAR_SUCCESS, starred: true, ref: 'proj:2'});
        assert.deepEqual(newState, {'proj:1': true, 'proj:2': true});
      });
    });

    describe('selectors', () => {
      it('starredIssues', () => {
        const state = {issue:
          {starredIssues: {'proj:1': true, 'proj:2': false}}};
        assert.deepEqual(issue.starredIssues(state), new Set(['proj:1']));
      });

      it('starringIssues', () => {
        const state = {issue: {
          requests: {
            starringIssues: {
              'proj:1': {requesting: true},
              'proj:2': {requestin: false, error: 'unknown error'},
            },
          },
        }};
        assert.deepEqual(issue.starringIssues(state), new Map([
          ['proj:1', {requesting: true}],
          ['proj:2', {requestin: false, error: 'unknown error'}],
        ]));
      });
    });

    describe('action creators', () => {
      beforeEach(() => {
        prpcCall = sinon.stub(prpcClient, 'call');

        dispatch = sinon.stub();
      });

      afterEach(() => {
        prpcCall.restore();
      });

      it('fetching if an issue is starred', async () => {
        const message = {projectName: 'proj', localId: 1};
        const action = issue.fetchIsStarred(message);

        prpcCall.returns(Promise.resolve({isStarred: true}));

        await action(dispatch);

        sinon.assert.calledWith(dispatch, {type: issue.FETCH_IS_STARRED_START});

        sinon.assert.calledWith(
            prpcClient.call, 'monorail.Issues',
            'IsIssueStarred', message,
        );

        sinon.assert.calledWith(dispatch, {
          type: issue.FETCH_IS_STARRED_SUCCESS,
          starred: true,
          ref: message,
        });
      });

      it('fetching starred issues', async () => {
        const returnedIssueRef = {projectName: 'proj', localId: 1};
        const starredIssueRefs = [returnedIssueRef];
        const action = issue.fetchStarredIssues();

        prpcCall.returns(Promise.resolve({starredIssueRefs}));

        await action(dispatch);

        sinon.assert.calledWith(dispatch, {type: 'FETCH_ISSUES_STARRED_START'});

        sinon.assert.calledWith(
            prpcClient.call, 'monorail.Issues',
            'ListStarredIssues', {},
        );

        sinon.assert.calledWith(dispatch, {
          type: issue.FETCH_ISSUES_STARRED_SUCCESS,
          starredIssues: ['proj:1'],
        });
      });

      it('star', async () => {
        const testIssue = {projectName: 'proj', localId: 1, starCount: 1};
        const issueRef = issueToIssueRef(testIssue);
        const action = issue.star(issueRef, false);

        prpcCall.returns(Promise.resolve(testIssue));

        await action(dispatch);

        sinon.assert.calledWith(dispatch, {
          type: issue.STAR_START,
          requestKey: 'proj:1',
        });

        sinon.assert.calledWith(
            prpcClient.call,
            'monorail.Issues', 'StarIssue',
            {issueRef, starred: false},
        );

        sinon.assert.calledWith(dispatch, {
          type: issue.STAR_SUCCESS,
          starCount: 1,
          ref: 'proj:1',
          starred: false,
          requestKey: 'proj:1',
        });
      });
    });
  });
});

/**
 * Return a container object wrapping input
 * @param {Object=} currentIssue
 * @return {Object}
 */
function wrapIssue(currentIssue) {
  return {issue: {currentIssue: {...currentIssue}}};
}
