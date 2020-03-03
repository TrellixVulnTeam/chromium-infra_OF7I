// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Issue actions, selectors, and reducers organized into
 * a single Redux "Duck" that manages updating and retrieving issue state
 * on the frontend.
 *
 * Reference: https://github.com/erikras/ducks-modular-redux
 */

import {combineReducers} from 'redux';
import {createSelector} from 'reselect';
import {autolink} from 'autolink.js';
import {fieldTypes, extractTypeForIssue,
  fieldValuesToMap} from 'shared/issue-fields.js';
import {removePrefix, objectToMap} from 'shared/helpers.js';
import {issueRefToString, issueToIssueRefString,
  issueStringToRef, issueNameToRefString} from 'shared/converters.js';
import {fromShortlink} from 'shared/federated.js';
import {createReducer, createRequestReducer,
  createKeyedRequestReducer} from './redux-helpers.js';
import * as project from './project.js';
import * as user from './user.js';
import {fieldValueMapKey} from 'shared/metadata-helpers.js';
import {prpcClient} from 'prpc-client-instance.js';
import loadGapi, {fetchGapiEmail} from 'shared/gapi-loader.js';
import 'shared/typedef.js';

/** @typedef {import('redux').AnyAction} AnyAction */

// Actions
export const VIEW_ISSUE = 'VIEW_ISSUE';

export const FETCH_START = 'issue/FETCH_START';
export const FETCH_SUCCESS = 'issue/FETCH_SUCCESS';
export const FETCH_FAILURE = 'issue/FETCH_FAILURE';

export const FETCH_ISSUES_START = 'issue/FETCH_ISSUES_START';
export const FETCH_ISSUES_SUCCESS = 'issue/FETCH_ISSUES_SUCCESS';
export const FETCH_ISSUES_FAILURE = 'issue/FETCH_ISSUES_FAILURE';

const FETCH_HOTLISTS_START = 'FETCH_HOTLISTS_START';
export const FETCH_HOTLISTS_SUCCESS = 'FETCH_HOTLISTS_SUCCESS';
const FETCH_HOTLISTS_FAILURE = 'FETCH_HOTLISTS_FAILURE';

const FETCH_ISSUE_LIST_START = 'FETCH_ISSUE_LIST_START';
export const FETCH_ISSUE_LIST_UPDATE = 'FETCH_ISSUE_LIST_UPDATE';
const FETCH_ISSUE_LIST_SUCCESS = 'FETCH_ISSUE_LIST_SUCCESS';
const FETCH_ISSUE_LIST_FAILURE = 'FETCH_ISSUE_LIST_FAILURE';

const FETCH_PERMISSIONS_START = 'FETCH_PERMISSIONS_START';
const FETCH_PERMISSIONS_SUCCESS = 'FETCH_PERMISSIONS_SUCCESS';
const FETCH_PERMISSIONS_FAILURE = 'FETCH_PERMISSIONS_FAILURE';

export const STAR_START = 'STAR_START';
export const STAR_SUCCESS = 'STAR_SUCCESS';
const STAR_FAILURE = 'STAR_FAILURE';

const PRESUBMIT_START = 'PRESUBMIT_START';
const PRESUBMIT_SUCCESS = 'PRESUBMIT_SUCCESS';
const PRESUBMIT_FAILURE = 'PRESUBMIT_FAILURE';

const PREDICT_COMPONENT_START = 'PREDICT_COMPONENT_START';
const PREDICT_COMPONENT_SUCCESS = 'PREDICT_COMPONENT_SUCCESS';
const PREDICT_COMPONENT_FAILURE = 'PREDICT_COMPONENT_FAILURE';

export const FETCH_IS_STARRED_START = 'FETCH_IS_STARRED_START';
export const FETCH_IS_STARRED_SUCCESS = 'FETCH_IS_STARRED_SUCCESS';
const FETCH_IS_STARRED_FAILURE = 'FETCH_IS_STARRED_FAILURE';

const FETCH_ISSUES_STARRED_START = 'FETCH_ISSUES_STARRED_START';
export const FETCH_ISSUES_STARRED_SUCCESS = 'FETCH_ISSUES_STARRED_SUCCESS';
const FETCH_ISSUES_STARRED_FAILURE = 'FETCH_ISSUES_STARRED_FAILURE';

const FETCH_COMMENTS_START = 'FETCH_COMMENTS_START';
export const FETCH_COMMENTS_SUCCESS = 'FETCH_COMMENTS_SUCCESS';
const FETCH_COMMENTS_FAILURE = 'FETCH_COMMENTS_FAILURE';

const FETCH_COMMENT_REFERENCES_START = 'FETCH_COMMENT_REFERENCES_START';
const FETCH_COMMENT_REFERENCES_SUCCESS = 'FETCH_COMMENT_REFERENCES_SUCCESS';
const FETCH_COMMENT_REFERENCES_FAILURE = 'FETCH_COMMENT_REFERENCES_FAILURE';

const FETCH_REFERENCED_USERS_START = 'FETCH_REFERENCED_USERS_START';
const FETCH_REFERENCED_USERS_SUCCESS = 'FETCH_REFERENCED_USERS_SUCCESS';
const FETCH_REFERENCED_USERS_FAILURE = 'FETCH_REFERENCED_USERS_FAILURE';

const FETCH_RELATED_ISSUES_START = 'FETCH_RELATED_ISSUES_START';
const FETCH_RELATED_ISSUES_SUCCESS = 'FETCH_RELATED_ISSUES_SUCCESS';
const FETCH_RELATED_ISSUES_FAILURE = 'FETCH_RELATED_ISSUES_FAILURE';

const FETCH_FEDERATED_REFERENCES_START = 'FETCH_FEDERATED_REFERENCES_START';
const FETCH_FEDERATED_REFERENCES_SUCCESS = 'FETCH_FEDERATED_REFERENCES_SUCCESS';
const FETCH_FEDERATED_REFERENCES_FAILURE = 'FETCH_FEDERATED_REFERENCES_FAILURE';

const CONVERT_START = 'CONVERT_START';
const CONVERT_SUCCESS = 'CONVERT_SUCCESS';
const CONVERT_FAILURE = 'CONVERT_FAILURE';

const UPDATE_START = 'UPDATE_START';
const UPDATE_SUCCESS = 'UPDATE_SUCCESS';
const UPDATE_FAILURE = 'UPDATE_FAILURE';

const UPDATE_APPROVAL_START = 'UPDATE_APPROVAL_START';
const UPDATE_APPROVAL_SUCCESS = 'UPDATE_APPROVAL_SUCCESS';
const UPDATE_APPROVAL_FAILURE = 'UPDATE_APPROVAL_FAILURE';

/* State Shape
{
  issuesByRefString: Object.<IssueRefString, Issue>,

  viewedIssueRef: IssueRefString,

  hotlists: Array<Hotlist>,
  issueList: {
    issueRefs: Array<IssueRefString>,
    progress: number,
    totalResults: number,
  }
  comments: Array<IssueComment>,
  commentReferences: Object,
  relatedIssues: Object,
  referencedUsers: Array<User>,
  starredIssues: Object.<IssueRefString, Boolean>,
  permissions: Array<string>,
  presubmitResponse: Object,
  predictedComponent: string,

  requests: {
    fetch: ReduxRequestState,
    fetchHotlists: ReduxRequestState,
    fetchIssueList: ReduxRequestState,
    fetchPermissions: ReduxRequestState,
    starringIssues: Object.<string, ReduxRequestState>,
    presubmit: ReduxRequestState,
    predictComponent: ReduxRequestState,
    fetchComments: ReduxRequestState,
    fetchCommentReferences: ReduxRequestState,
    fetchFederatedReferences: ReduxRequestState,
    fetchRelatedIssues: ReduxRequestState,
    fetchStarredIssues: ReduxRequestState,
    fetchStarredIssues: ReduxRequestState,
    convert: ReduxRequestState,
    update: ReduxRequestState,
    updateApproval: ReduxRequestState,
  },
}
*/

// Helpers for the reducers.

/**
 * Overrides local data for single approval on an Issue object with fresh data.
 * Note that while an Issue can have multiple approvals, this function only
 * refreshes data for a single approval.
 * @param {Issue} issue Issue Object being updated.
 * @param {ApprovalDef} approval A single approval to override in the issue.
 * @return {Issue} Issue with updated approval data.
 */
const updateApprovalValues = (issue, approval) => {
  if (!issue.approvalValues) return issue;
  const newApprovals = issue.approvalValues.map((item) => {
    if (item.fieldRef.fieldName === approval.fieldRef.fieldName) {
      // PhaseRef isn't populated on the response so we want to make sure
      // it doesn't overwrite the original phaseRef with {}.
      return {...approval, phaseRef: item.phaseRef};
    }
    return item;
  });
  return {...issue, approvalValues: newApprovals};
};

// Reducers

/**
 * Creates a new issuesByRefString Object with a single issue's data
 * edited.
 * @param {Object.<IssueRefString, Issue>} issuesByRefString
 * @param {Issue} issue The new issue data to add to the state.
 * @return {Object.<IssueRefString, Issue>}
 */
const updateSingleIssueInState = (issuesByRefString, issue) => {
  return {
    ...issuesByRefString,
    [issueToIssueRefString(issue)]: issue,
  };
};

// TODO(crbug.com/monorail/6882): Finish converting all other issue
//   actions to use this format.
/**
 * Adds issues fetched by a ListIssues request to the Redux store in a
 * normalized format.
 * @param {Object.<IssueRefString, Issue>} state Redux state.
 * @param {AnyAction} action
 * @param {Array<Issue>} action.issues The list of issues that was fetched.
 * @param {Issue=} action.issue The issue being updated.
 * @param {number=} action.starCount Number of stars the issue has. This changes
 *   when a user stars an issue and needs to be updated.
 * @param {ApprovalDef=} action.approval A new approval to update the issue
 *   with.
 * @param {IssueRef=} action.issueRef A specific IssueRef to update.
 */
export const issuesByRefStringReducer = createReducer({}, {
  [FETCH_ISSUE_LIST_UPDATE]: (state, {issues}) => {
    const newState = {...state};

    issues.forEach((issue) => {
      const refString = issueToIssueRefString(issue);
      newState[refString] = {...newState[refString], ...issue};
    });

    return newState;
  },
  [FETCH_SUCCESS]: (state, {issue}) => updateSingleIssueInState(state, issue),
  [FETCH_ISSUES_SUCCESS]: (state, {issues}) => {
    const newState = {...state};

    issues.forEach((issue) => {
      const refString = issueToIssueRefString(issue);
      newState[refString] = {...newState[refString], ...issue};
    });

    return newState;
  },
  [CONVERT_SUCCESS]: (state, {issue}) => updateSingleIssueInState(state, issue),
  [UPDATE_SUCCESS]: (state, {issue}) => updateSingleIssueInState(state, issue),
  [UPDATE_APPROVAL_SUCCESS]: (state, {issueRef, approval}) => {
    const issueRefString = issueToIssueRefString(issueRef);
    const originalIssue = state[issueRefString] || {};
    const newIssue = updateApprovalValues(originalIssue, approval);
    return {
      ...state,
      [issueRefString]: {
        ...newIssue,
      },
    };
  },
  [STAR_SUCCESS]: (state, {issueRef, starCount}) => {
    const issueRefString = issueToIssueRefString(issueRef);
    const originalIssue = state[issueRefString] || {};
    return {
      ...state,
      [issueRefString]: {
        ...originalIssue,
        starCount,
      },
    };
  },
});

/**
 * Sets a reference for the issue that the user is currently viewing.
 * @param {IssueRefString} state Currently viewed issue.
 * @param {AnyAction} action
 * @param {IssueRef} action.issueRef The updated localId to view.
 * @return {IssueRefString}
 */
const viewedIssueRefReducer = createReducer('', {
  [VIEW_ISSUE]: (state, {issueRef}) => issueRefToString(issueRef) || state,
});

/**
 * Reducer to manage updating the list of hotlists attached to an Issue.
 * @param {Array<Hotlist>} state List of issue hotlists.
 * @param {AnyAction} action
 * @param {Array<Hotlist>} action.hotlists New list of hotlists.
 * @return {Array<Hotlist>}
 */
const hotlistsReducer = createReducer([], {
  [FETCH_HOTLISTS_SUCCESS]: (_, {hotlists}) => hotlists,
});

/**
 * @typedef {Object} IssueListState
 * @property {Array<IssueRefString>} issues The list of issues being viewed,
 *   in a normalized form.
 * @property {number} progress The percentage of issues loaded. Used for
 *   incremental loading of issues in the grid view.
 * @property {number} totalResults The total number of issues matching the
 *   query.
 */

/**
 * Handles the state of the currently viewed issue list. This reducer
 * stores this data in normalized form.
 * @param {IssueListState} state
 * @param {AnyAction} action
 * @param {Array<Issue>} action.issues Issues that were fetched.
 * @param {number} state.progress New percentage of issues have been loaded.
 * @param {number} state.totalResults The total number of issues matching the
 *   query.
 * @return {IssueListState}
 */
export const issueListReducer = createReducer({}, {
  [FETCH_ISSUE_LIST_UPDATE]: (_state, {issues, progress, totalResults}) => ({
    issueRefs: issues.map(issueToIssueRefString), progress, totalResults,
  }),
});

/**
 * Updates the comments attached to the currently viewed issue.
 * @param {Array<IssueComment>} state The list of comments in an issue.
 * @param {AnyAction} action
 * @param {Array<IssueComment>} action.comments Fetched comments.
 * @return {Array<IssueComment>}
 */
const commentsReducer = createReducer([], {
  [FETCH_COMMENTS_SUCCESS]: (_state, {comments}) => comments,
});

// TODO(crbug.com/monorail/5953): Come up with some way to refactor
// autolink.js's reference code to allow avoiding duplicate lookups
// with data already in Redux state.
/**
 * For autolinking, this reducer stores the dereferenced data for bits
 * of data that were referenced in comments. For example, comments might
 * include user emails or IDs for other issues, and this state slice would
 * store the full Objects for that data.
 * @param {Array<CommentReference>} state
 * @param {AnyAction} action
 * @param {Array<CommentReference>} action.commentReferences New references
 *   to store.
 * @return {Array<CommentReference>}
 */
const commentReferencesReducer = createReducer({}, {
  [FETCH_COMMENTS_START]: (_state, _action) => ({}),
  [FETCH_COMMENT_REFERENCES_SUCCESS]: (_state, {commentReferences}) => {
    return commentReferences;
  },
});

/**
 * Handles state for related issues such as blocking and blocked on issues,
 * including federated references that could reference external issues outside
 * Monorail.
 * @param {Object.<IssueRefString, Issue>} state
 * @param {AnyAction} action
 * @param {Object.<IssueRefString, Issue>=} action.relatedIssues New related
 *   issues.
 * @param {Array<IssueRef>=} action.fedRefIssueRefs List of fetched federated
 *   issue references.
 * @return {Object.<IssueRefString, Issue>}
 */
export const relatedIssuesReducer = createReducer({}, {
  [FETCH_RELATED_ISSUES_SUCCESS]: (_state, {relatedIssues}) => relatedIssues,
  [FETCH_FEDERATED_REFERENCES_SUCCESS]: (state, {fedRefIssueRefs}) => {
    if (!fedRefIssueRefs) {
      return state;
    }

    const fedRefStates = {};
    fedRefIssueRefs.forEach((ref) => {
      fedRefStates[ref.extIdentifier] = ref;
    });

    // Return a new object, in Redux fashion.
    return Object.assign(fedRefStates, state);
  },
});

/**
 * Stores data for users referenced by issue. ie: Owner, CC, etc.
 * @param {Object.<string, User>} state
 * @param {AnyAction} action
 * @param {Object.<string, User>} action.referencedUsers
 * @return {Object.<string, User>}
 */
const referencedUsersReducer = createReducer({}, {
  [FETCH_REFERENCED_USERS_SUCCESS]: (_state, {referencedUsers}) =>
    referencedUsers,
});

/**
 * Handles updating state of all starred issues.
 * @param {Object.<IssueRefString, boolean>} state Set of starred issues,
 *   stored in a serializeable Object form.
 * @param {AnyAction} action
 * @param {IssueRef=} action.issueRef An issue with a star state being updated.
 * @param {boolean=} action.starred Whether the issue is starred or unstarred.
 * @param {Array<IssueRef>=} action.starredIssueRefs A list of starred issues.
 * @return {Object.<IssueRefString, boolean>}
 */
export const starredIssuesReducer = createReducer({}, {
  [STAR_SUCCESS]: (state, {issueRef, starred}) => {
    return {...state, [issueRefToString(issueRef)]: starred};
  },
  [FETCH_ISSUES_STARRED_SUCCESS]: (_state, {starredIssueRefs}) => {
    const normalizedStars = {};
    starredIssueRefs.forEach((issueRef) => {
      normalizedStars[issueRefToString(issueRef)] = true;
    });
    return normalizedStars;
  },
  [FETCH_IS_STARRED_SUCCESS]: (state, {issueRef, starred}) => {
    const refString = issueRefToString(issueRef);
    return {...state, [refString]: starred};
  },
});

/**
 * Adds the result of an IssuePresubmit response to the Redux store.
 * @param {Object} state Initial Redux state.
 * @param {AnyAction} action
 * @param {Object} action.presubmitResponse The issue
 *   presubmit response Object.
 * @return {Object}
 */
const presubmitResponseReducer = createReducer({}, {
  [PRESUBMIT_SUCCESS]: (_state, {presubmitResponse}) => presubmitResponse,
});

/**
 * To display the results of our ML component predictor, this reducer updates
 * the store with a recommended component name for the currently viewed issue.
 * @param {string} state The name of the component recommended by the
 *   prediction.
 * @param {AnyAction} action
 * @param {string} action.component Predicted component name.
 * @return {string}
 */
const predictedComponentReducer = createReducer('', {
  [PREDICT_COMPONENT_SUCCESS]: (_state, {component}) => component,
});

/**
 * Stores the user's permissions for a given issue.
 * @param {Array<string>} state Permission list. Each permission is a string
 *   with the name of the permission.
 * @param {AnyAction} action
 * @param {Array<string>} action.permissions The fetched permission data.
 * @return {Array<string>}
 */
const permissionsReducer = createReducer([], {
  [FETCH_PERMISSIONS_SUCCESS]: (_state, {permissions}) => permissions,
});

const requestsReducer = combineReducers({
  fetch: createRequestReducer(
      FETCH_START, FETCH_SUCCESS, FETCH_FAILURE),
  fetchIssues: createRequestReducer(
      FETCH_ISSUES_START, FETCH_ISSUES_SUCCESS, FETCH_ISSUES_FAILURE),
  fetchHotlists: createRequestReducer(
      FETCH_HOTLISTS_START, FETCH_HOTLISTS_SUCCESS, FETCH_HOTLISTS_FAILURE),
  fetchIssueList: createRequestReducer(
      FETCH_ISSUE_LIST_START,
      FETCH_ISSUE_LIST_SUCCESS,
      FETCH_ISSUE_LIST_FAILURE),
  fetchPermissions: createRequestReducer(
      FETCH_PERMISSIONS_START,
      FETCH_PERMISSIONS_SUCCESS,
      FETCH_PERMISSIONS_FAILURE),
  starringIssues: createKeyedRequestReducer(
      STAR_START, STAR_SUCCESS, STAR_FAILURE),
  presubmit: createRequestReducer(
      PRESUBMIT_START, PRESUBMIT_SUCCESS, PRESUBMIT_FAILURE),
  predictComponent: createRequestReducer(
      PREDICT_COMPONENT_START,
      PREDICT_COMPONENT_SUCCESS,
      PREDICT_COMPONENT_FAILURE),
  fetchComments: createRequestReducer(
      FETCH_COMMENTS_START, FETCH_COMMENTS_SUCCESS, FETCH_COMMENTS_FAILURE),
  fetchCommentReferences: createRequestReducer(
      FETCH_COMMENT_REFERENCES_START,
      FETCH_COMMENT_REFERENCES_SUCCESS,
      FETCH_COMMENT_REFERENCES_FAILURE),
  fetchFederatedReferences: createRequestReducer(
      FETCH_FEDERATED_REFERENCES_START,
      FETCH_FEDERATED_REFERENCES_SUCCESS,
      FETCH_FEDERATED_REFERENCES_FAILURE),
  fetchRelatedIssues: createRequestReducer(
      FETCH_RELATED_ISSUES_START,
      FETCH_RELATED_ISSUES_SUCCESS,
      FETCH_RELATED_ISSUES_FAILURE),
  fetchReferencedUsers: createRequestReducer(
      FETCH_REFERENCED_USERS_START,
      FETCH_REFERENCED_USERS_SUCCESS,
      FETCH_REFERENCED_USERS_FAILURE),
  fetchIsStarred: createRequestReducer(
      FETCH_IS_STARRED_START, FETCH_IS_STARRED_SUCCESS,
      FETCH_IS_STARRED_FAILURE),
  fetchStarredIssues: createRequestReducer(
      FETCH_ISSUES_STARRED_START, FETCH_ISSUES_STARRED_SUCCESS,
      FETCH_ISSUES_STARRED_FAILURE,
  ),
  convert: createRequestReducer(
      CONVERT_START, CONVERT_SUCCESS, CONVERT_FAILURE),
  update: createRequestReducer(
      UPDATE_START, UPDATE_SUCCESS, UPDATE_FAILURE),
  // TODO(zhangtiff): Update this to use createKeyedRequestReducer() instead, so
  // users can update multiple approvals at once.
  updateApproval: createRequestReducer(
      UPDATE_APPROVAL_START, UPDATE_APPROVAL_SUCCESS, UPDATE_APPROVAL_FAILURE),
});

export const reducer = combineReducers({
  viewedIssueRef: viewedIssueRefReducer,

  issuesByRefString: issuesByRefStringReducer,

  hotlists: hotlistsReducer,
  issueList: issueListReducer,
  comments: commentsReducer,
  commentReferences: commentReferencesReducer,
  relatedIssues: relatedIssuesReducer,
  referencedUsers: referencedUsersReducer,
  starredIssues: starredIssuesReducer,
  permissions: permissionsReducer,
  presubmitResponse: presubmitResponseReducer,
  predictedComponent: predictedComponentReducer,

  requests: requestsReducer,
});

// Selectors
const RESTRICT_VIEW_PREFIX = 'restrict-view-';
const RESTRICT_EDIT_PREFIX = 'restrict-editissue-';
const RESTRICT_COMMENT_PREFIX = 'restrict-addissuecomment-';

/**
 * Selector to retrieve all normalized Issue data in the Redux store,
 * keyed by IssueRefString.
 * @param {any} state
 * @return {Object.<IssueRefString, Issue>}
 */
const issuesByRefString = (state) => state.issue.issuesByRefString;

/**
 * Selector to return a function to retrieve an Issue from the Redux store.
 * @param {any} state
 * @return {function(string): ?Issue}
 */
export const issue = createSelector(issuesByRefString, (issuesByRefString) =>
  (name) => issuesByRefString[issueNameToRefString(name)]);

/**
 * Selector to return a function to retrieve a given Issue Object from
 * the Redux store.
 * @param {any} state
 * @return {function(IssueRefString, string): Issue}
 */
export const issueForRefString = createSelector(issuesByRefString,
    (issuesByRefString) => (issueRefString, projectName = undefined) => {
      // In some contexts, an issue ref string will omit a project name,
      // assuming the default project to be the project name. We never
      // omit the project name in strings used as keys, so we have to
      // make sure issue ref strings contain the project name.
      const ref = issueStringToRef(issueRefString, projectName);
      const refString = issueRefToString(ref);
      if (issuesByRefString.hasOwnProperty(refString)) {
        return issuesByRefString[refString];
      }
      return issueStringToRef(refString, projectName);
    });

/**
 * Selector to get a reference to the currently viewed issue, in string form.
 * @param {any} state
 * @return {IssueRefString}
 */
const viewedIssueRefString = (state) => state.issue.viewedIssueRef;

/**
 * Selector to get a reference to the currently viewed issue.
 * @param {any} state
 * @return {IssueRef}
 */
export const viewedIssueRef = createSelector(viewedIssueRefString,
    (viewedIssueRefString) => issueStringToRef(viewedIssueRefString));

/**
 * Selector to get the full Issue data for the currently viewed issue.
 * @param {any} state
 * @return {Issue}
 */
export const viewedIssue = createSelector(issuesByRefString,
    viewedIssueRefString,
    (issuesByRefString, viewedIssueRefString) =>
      issuesByRefString[viewedIssueRefString] || {});

export const comments = (state) => state.issue.comments;
export const commentsLoaded = (state) => state.issue.commentsLoaded;

const _commentReferences = (state) => state.issue.commentReferences;
export const commentReferences = createSelector(_commentReferences,
    (commentReferences) => objectToMap(commentReferences));

export const hotlists = (state) => state.issue.hotlists;

const stateIssueList = (state) => state.issue.issueList;
export const issueList = createSelector(
    issuesByRefString,
    stateIssueList,
    (issuesByRefString, stateIssueList) => {
      return (stateIssueList.issueRefs || []).map((issueRef) => {
        return issuesByRefString[issueRef];
      });
    },
);
export const totalIssues = (state) => state.issue.issueList.totalResults;
export const issueListProgress = (state) => state.issue.issueList.progress;
export const issueListPhaseNames = createSelector(issueList, (issueList) => {
  const phaseNamesSet = new Set();
  if (issueList) {
    issueList.forEach(({phases}) => {
      if (phases) {
        phases.forEach(({phaseRef: {phaseName}}) => {
          phaseNamesSet.add(phaseName.toLowerCase());
        });
      }
    });
  }
  return Array.from(phaseNamesSet);
});

/**
 * @param {any} state
 * @return {boolean} Whether the currently viewed issue list
 *   has loaded.
 */
export const issueListLoaded = createSelector(
    stateIssueList,
    (stateIssueList) => stateIssueList.issueRefs !== undefined);

export const permissions = (state) => state.issue.permissions;
export const presubmitResponse = (state) => state.issue.presubmitResponse;
export const predictedComponent = (state) => state.issue.predictedComponent;

const _relatedIssues = (state) => state.issue.relatedIssues || {};
export const relatedIssues = createSelector(_relatedIssues,
    (relatedIssues) => objectToMap(relatedIssues));

const _referencedUsers = (state) => state.issue.referencedUsers || {};
export const referencedUsers = createSelector(_referencedUsers,
    (referencedUsers) => objectToMap(referencedUsers));

export const isStarred = (state) => state.issue.isStarred;
export const _starredIssues = (state) => state.issue.starredIssues;

export const requests = (state) => state.issue.requests;

// Returns a Map of in flight StarIssues requests, keyed by issueRef.
export const starringIssues = createSelector(requests, (requests) =>
  objectToMap(requests.starringIssues));

export const starredIssues = createSelector(
    _starredIssues,
    (starredIssues) => {
      const stars = new Set();
      for (const [ref, starred] of Object.entries(starredIssues)) {
        if (starred) stars.add(ref);
      }
      return stars;
    },
);

// TODO(zhangtiff): Split up either comments or approvals into their own "duck".
export const commentsByApprovalName = createSelector(
    comments,
    (comments) => {
      const map = new Map();
      comments.forEach((comment) => {
        const key = (comment.approvalRef && comment.approvalRef.fieldName) ||
          '';
        if (map.has(key)) {
          map.get(key).push(comment);
        } else {
          map.set(key, [comment]);
        }
      });
      return map;
    },
);

export const fieldValues = createSelector(
    viewedIssue,
    (issue) => issue && issue.fieldValues,
);

export const labelRefs = createSelector(
    viewedIssue,
    (issue) => issue && issue.labelRefs,
);

export const type = createSelector(
    fieldValues,
    labelRefs,
    (fieldValues, labelRefs) => extractTypeForIssue(fieldValues, labelRefs),
);

export const restrictions = createSelector(
    labelRefs,
    (labelRefs) => {
      if (!labelRefs) return {};

      const restrictions = {};

      labelRefs.forEach((labelRef) => {
        const label = labelRef.label;
        const lowerCaseLabel = label.toLowerCase();

        if (lowerCaseLabel.startsWith(RESTRICT_VIEW_PREFIX)) {
          const permissionType = removePrefix(label, RESTRICT_VIEW_PREFIX);
          if (!('view' in restrictions)) {
            restrictions['view'] = [permissionType];
          } else {
            restrictions['view'].push(permissionType);
          }
        } else if (lowerCaseLabel.startsWith(RESTRICT_EDIT_PREFIX)) {
          const permissionType = removePrefix(label, RESTRICT_EDIT_PREFIX);
          if (!('edit' in restrictions)) {
            restrictions['edit'] = [permissionType];
          } else {
            restrictions['edit'].push(permissionType);
          }
        } else if (lowerCaseLabel.startsWith(RESTRICT_COMMENT_PREFIX)) {
          const permissionType = removePrefix(label, RESTRICT_COMMENT_PREFIX);
          if (!('comment' in restrictions)) {
            restrictions['comment'] = [permissionType];
          } else {
            restrictions['comment'].push(permissionType);
          }
        }
      });

      return restrictions;
    },
);

export const isOpen = createSelector(
    viewedIssue,
    (issue) => issue && issue.statusRef && issue.statusRef.meansOpen || false);

// Returns a function that, given an issue and its related issues,
// returns a combined list of issue ref strings including related issues,
// blocking or blocked on issues, and federated references.
const mapRefsWithRelated = (blocking) => {
  return (issue, relatedIssues) => {
    let refs = [];
    if (blocking) {
      if (issue.blockingIssueRefs) {
        refs = refs.concat(issue.blockingIssueRefs);
      }
      if (issue.danglingBlockingRefs) {
        refs = refs.concat(issue.danglingBlockingRefs);
      }
    } else {
      if (issue.blockedOnIssueRefs) {
        refs = refs.concat(issue.blockedOnIssueRefs);
      }
      if (issue.danglingBlockedOnRefs) {
        refs = refs.concat(issue.danglingBlockedOnRefs);
      }
    }

    // Note: relatedIssues is a Redux generated key for issues, not part of the
    // pRPC Issue object.
    if (issue.relatedIssues) {
      refs = refs.concat(issue.relatedIssues);
    }
    return refs.map((ref) => {
      const key = issueRefToString(ref);
      if (relatedIssues.has(key)) {
        return relatedIssues.get(key);
      }
      return ref;
    });
  };
};

export const blockingIssues = createSelector(
    viewedIssue, relatedIssues,
    mapRefsWithRelated(true),
);

export const blockedOnIssues = createSelector(
    viewedIssue, relatedIssues,
    mapRefsWithRelated(false),
);

export const mergedInto = createSelector(
    viewedIssue, relatedIssues,
    (issue, relatedIssues) => {
      if (!issue || !issue.mergedIntoIssueRef) return {};
      const key = issueRefToString(issue.mergedIntoIssueRef);
      if (relatedIssues && relatedIssues.has(key)) {
        return relatedIssues.get(key);
      }
      return issue.mergedIntoIssueRef;
    },
);

export const sortedBlockedOn = createSelector(
    blockedOnIssues,
    (blockedOn) => blockedOn.sort((a, b) => {
      const aIsOpen = a.statusRef && a.statusRef.meansOpen ? 1 : 0;
      const bIsOpen = b.statusRef && b.statusRef.meansOpen ? 1 : 0;
      return bIsOpen - aIsOpen;
    }),
);

// values (from issue.fieldValues) is an array with one entry per value.
// We want to turn this into a map of fieldNames -> values.
export const fieldValueMap = createSelector(
    fieldValues,
    (fieldValues) => fieldValuesToMap(fieldValues),
);

// Get the list of full componentDefs for the viewed issue.
export const components = createSelector(
    viewedIssue,
    project.componentsMap,
    (issue, components) => {
      if (!issue || !issue.componentRefs) return [];
      return issue.componentRefs.map(
          (comp) => components.get(comp.path) || comp);
    },
);

// Get custom fields that apply to a specific issue.
export const fieldDefs = createSelector(
    project.fieldDefs,
    type,
    fieldValueMap,
    (fieldDefs, type, fieldValues) => {
      if (!fieldDefs) return [];
      type = type || '';
      return fieldDefs.filter((f) => {
        const fieldValueKey = fieldValueMapKey(f.fieldRef.fieldName,
            f.phaseRef && f.phaseRef.phaseName);
        if (fieldValues && fieldValues.has(fieldValueKey)) {
        // Regardless of other checks, include a particular field def if the
        // issue has a value defined.
          return true;
        }
        // Skip approval type and phase fields here.
        if (f.fieldRef.approvalName ||
            f.fieldRef.type === fieldTypes.APPROVAL_TYPE ||
            f.isPhaseField) {
          return false;
        }

        // If this fieldDef belongs to only one type, filter out the field if
        // that type isn't the specified type.
        if (f.applicableType && type.toLowerCase() !==
            f.applicableType.toLowerCase()) {
          return false;
        }

        return true;
      });
    },
);

// Action Creators
/**
 * Tells Redux that the user has navigated to an issue page and is now
 * viewing a new issue.
 * @param {IssueRef} issueRef The issue that the user is viewing.
 * @return {AnyAction}
 */
export const viewIssue = (issueRef) => ({type: VIEW_ISSUE, issueRef});

export const fetchCommentReferences = (comments, projectName) => {
  return async (dispatch) => {
    dispatch({type: FETCH_COMMENT_REFERENCES_START});

    try {
      const refs = await autolink.getReferencedArtifacts(comments, projectName);
      const commentRefs = {};
      refs.forEach(({componentName, existingRefs}) => {
        commentRefs[componentName] = existingRefs;
      });
      dispatch({
        type: FETCH_COMMENT_REFERENCES_SUCCESS,
        commentReferences: commentRefs,
      });
    } catch (error) {
      dispatch({type: FETCH_COMMENT_REFERENCES_FAILURE, error});
    }
  };
};

export const fetchReferencedUsers = (issue) => async (dispatch) => {
  if (!issue) return;
  dispatch({type: FETCH_REFERENCED_USERS_START});

  // TODO(zhangtiff): Make this function account for custom fields
  // of type user.
  const userRefs = [...(issue.ccRefs || [])];
  if (issue.ownerRef) {
    userRefs.push(issue.ownerRef);
  }
  (issue.approvalValues || []).forEach((approval) => {
    userRefs.push(...(approval.approverRefs || []));
    if (approval.setterRef) {
      userRefs.push(approval.setterRef);
    }
  });

  try {
    const resp = await prpcClient.call(
        'monorail.Users', 'ListReferencedUsers', {userRefs});

    const referencedUsers = {};
    (resp.users || []).forEach((user) => {
      referencedUsers[user.displayName] = user;
    });
    dispatch({type: FETCH_REFERENCED_USERS_SUCCESS, referencedUsers});
  } catch (error) {
    dispatch({type: FETCH_REFERENCED_USERS_FAILURE, error});
  }
};

export const fetchFederatedReferences = (issue) => async (dispatch) => {
  dispatch({type: FETCH_FEDERATED_REFERENCES_START});

  // Concat all potential fedrefs together, convert from shortlink to classes,
  // then fire off a request to fetch the status of each.
  const fedRefs = []
      .concat(issue.danglingBlockingRefs || [])
      .concat(issue.danglingBlockedOnRefs || [])
      .concat(issue.mergedIntoIssueRef ? [issue.mergedIntoIssueRef] : [])
      .filter((ref) => ref && ref.extIdentifier)
      .map((ref) => fromShortlink(ref.extIdentifier))
      .filter((fedRef) => fedRef);

  // If no FedRefs, return empty Map.
  if (fedRefs.length === 0) {
    return;
  }

  try {
    // Load email separately since it might have changed.
    await loadGapi();
    const email = await fetchGapiEmail();

    // If already logged in, dispatch login success event.
    dispatch({
      type: user.GAPI_LOGIN_SUCCESS,
      email: email,
    });

    await Promise.all(fedRefs.map((fedRef) => fedRef.getFederatedDetails()));
    const fedRefIssueRefs = fedRefs.map((fedRef) => fedRef.toIssueRef());

    dispatch({
      type: FETCH_FEDERATED_REFERENCES_SUCCESS,
      fedRefIssueRefs: fedRefIssueRefs,
    });
  } catch (error) {
    dispatch({type: FETCH_FEDERATED_REFERENCES_FAILURE, error});
  }
};

// TODO(zhangtiff): Figure out if we can reduce request/response sizes by
// diffing issues to fetch against issues we already know about to avoid
// fetching duplicate info.
export const fetchRelatedIssues = (issue) => async (dispatch) => {
  if (!issue) return;
  dispatch({type: FETCH_RELATED_ISSUES_START});

  const refsToFetch = (issue.blockedOnIssueRefs || []).concat(
      issue.blockingIssueRefs || []);
  // Add mergedinto ref, exclude FedRefs which are fetched separately.
  if (issue.mergedIntoIssueRef && !issue.mergedIntoIssueRef.extIdentifier) {
    refsToFetch.push(issue.mergedIntoIssueRef);
  }

  const message = {
    issueRefs: refsToFetch,
  };
  try {
    // Fire off call to fetch FedRefs. Since it might take longer it is
    // handled by a separate reducer.
    dispatch(fetchFederatedReferences(issue));

    const resp = await prpcClient.call(
        'monorail.Issues', 'ListReferencedIssues', message);

    const relatedIssues = {};

    const openIssues = resp.openRefs || [];
    const closedIssues = resp.closedRefs || [];
    openIssues.forEach((issue) => {
      issue.statusRef.meansOpen = true;
      relatedIssues[issueRefToString(issue)] = issue;
    });
    closedIssues.forEach((issue) => {
      issue.statusRef.meansOpen = false;
      relatedIssues[issueRefToString(issue)] = issue;
    });
    dispatch({
      type: FETCH_RELATED_ISSUES_SUCCESS,
      relatedIssues: relatedIssues,
    });
  } catch (error) {
    dispatch({type: FETCH_RELATED_ISSUES_FAILURE, error});
  };
};

/**
 * Fetches issue data needed to display a detailed view of a single
 * issue. This function dispatches many actions to handle the fetching
 * of issue comments, permissions, star state, and more.
 * @param {IssueRef} issueRef The issue that the user is viewing.
 * @return {function(function): Promise<void>}
 */
export const fetchIssuePageData = (issueRef) => async (dispatch) => {
  dispatch(fetchComments(issueRef));
  dispatch(fetch(issueRef));
  dispatch(fetchPermissions(issueRef));
  dispatch(fetchIsStarred(issueRef));
};

/**
 * @param {IssueRef} issueRef Which issue to fetch.
 * @return {function(function): Promise<void>}
 */
export const fetch = (issueRef) => async (dispatch) => {
  dispatch({type: FETCH_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'GetIssue', {issueRef},
    );

    const movedToRef = resp.movedToRef;

    // The API can return deleted issue objects that don't have issueRef data
    // specified. For this case, we want to make sure a projectName and localId
    // are still provided to the frontend to ensure that keying issues still
    // works.
    const issue = {...issueRef, ...resp.issue};
    if (movedToRef) {
      issue.movedToRef = movedToRef;
    }

    dispatch({type: FETCH_SUCCESS, issue});

    if (!issue.isDeleted && !movedToRef) {
      dispatch(fetchRelatedIssues(issue));
      dispatch(fetchHotlists(issueRef));
      dispatch(fetchReferencedUsers(issue));
      dispatch(user.fetchProjects([issue.reporterRef]));
    }
  } catch (error) {
    dispatch({type: FETCH_FAILURE, error});
  }
};

/**
 * Action creator to fetch multiple Issues.
 * @param {Array<IssueRef>} issueRefs An Array of Issue references to fetch.
 * @return {function(function): Promise<void>}
 */
export const fetchIssues = (issueRefs) => async (dispatch) => {
  dispatch({type: FETCH_ISSUES_START});

  try {
    const {openRefs, closedRefs} = await prpcClient.call(
        'monorail.Issues', 'ListReferencedIssues', {issueRefs});
    const issues = [...openRefs || [], ...closedRefs || []];

    dispatch({type: FETCH_ISSUES_SUCCESS, issues});
  } catch (error) {
    dispatch({type: FETCH_ISSUES_FAILURE, error});
  }
};

/**
 * Gets the hotlists that a given issue is in.
 * @param {IssueRef} issueRef
 * @return {function(function): Promise<void>}
 */
export const fetchHotlists = (issueRef) => async (dispatch) => {
  dispatch({type: FETCH_HOTLISTS_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Features', 'ListHotlistsByIssue', {issue: issueRef});

    const hotlists = (resp.hotlists || []);
    hotlists.sort((hotlistA, hotlistB) => {
      return hotlistA.name.localeCompare(hotlistB.name);
    });
    dispatch({type: FETCH_HOTLISTS_SUCCESS, hotlists});
  } catch (error) {
    dispatch({type: FETCH_HOTLISTS_FAILURE, error});
  };
};

/**
 * Async action creator to fetch issues in the issue list and grid pages. This
 * action creator supports batching multiple async requests to support the grid
 * view's ability to load up to 6,000 issues in one page load.
 *
 * @param {string} projectName The project to fetch issues from.
 * @param {Object} params Options for which issues to fetch.
 * @param {string=} params.q The query string for the search.
 * @param {string=} params.can The ID of the canned query for the search.
 * @param {string=} params.groupby The spec of which fields to group by.
 * @param {string=} params.sort The spec of which fields to sort by.
 * @param {number=} params.start What cursor index to start at.
 * @param {number=} params.maxItems How many items to fetch per page.
 * @param {number=} params.maxCalls The maximum number of API calls to make.
 *   Combined with pagination.maxItems, this defines the maximum number of
 *   issues this method can fetch.
 * @return {function(function): Promise<void>}
 */
export const fetchIssueList =
  (projectName, {q = undefined, can = undefined, groupby = undefined,
    sort = undefined, start = undefined, maxItems = undefined,
    maxCalls = 1,
  }) => async (dispatch) => {
    let updateData = {};
    const promises = [];
    const issuesByRequest = [];
    let issueLimit;
    let totalIssues;
    let totalCalls;
    const itemsPerCall = maxItems || 1000;

    const cannedQuery = Number.parseInt(can) || undefined;

    const pagination = {
      ...(start && {start}),
      ...(maxItems && {maxItems}),
    };

    const message = {
      projectNames: [projectName],
      query: q,
      cannedQuery,
      groupBySpec: groupby,
      sortSpec: sort,
      pagination,
    };

    dispatch({type: FETCH_ISSUE_LIST_START});

    // initial api call made to determine total number of issues matching
    // the query.
    try {
      // TODO(zhangtiff): Refactor this action creator when adding issue
      // list pagination.
      const resp = await prpcClient.call(
          'monorail.Issues', 'ListIssues', message);

      // prpcClient is not actually a protobuf client and therefore not
      // hydrating default values. See crbug.com/monorail/6641
      // Until that is fixed, we have to explicitly define it.
      const defaultFetchListResponse = {totalResults: 0, issues: []};

      updateData =
        Object.entries(resp).length === 0 ?
          defaultFetchListResponse :
          resp;
      issuesByRequest[0] = updateData.issues;
      issueLimit = updateData.totalResults;

      // determine correct issues to load and number of calls to be made.
      if (issueLimit > (itemsPerCall * maxCalls)) {
        totalIssues = itemsPerCall * maxCalls;
        totalCalls = maxCalls - 1;
      } else {
        totalIssues = issueLimit;
        totalCalls = Math.ceil(issueLimit / itemsPerCall) - 1;
      }

      if (totalIssues) {
        updateData.progress = updateData.issues.length / totalIssues;
      } else {
        updateData.progress = 1;
      }

      dispatch({type: FETCH_ISSUE_LIST_UPDATE, ...updateData});

      // remaining api calls are made.
      for (let i = 1; i <= totalCalls; i++) {
        promises[i - 1] = (async () => {
          const resp = await prpcClient.call(
              'monorail.Issues', 'ListIssues', {
                ...message,
                pagination: {start: i * itemsPerCall, maxItems: itemsPerCall},
              });
          issuesByRequest[i] = (resp.issues || []);
          // sort the issues in the correct order.
          updateData.issues = [];
          issuesByRequest.forEach((issue) => {
            updateData.issues = updateData.issues.concat(issue);
          });
          updateData.progress = updateData.issues.length / totalIssues;
          dispatch({type: FETCH_ISSUE_LIST_UPDATE, ...updateData});
        })();
      }

      await Promise.all(promises);

      // TODO(zhangtiff): Try to delete FETCH_ISSUE_LIST_SUCCESS in favor of
      // just FETCH_ISSUE_LIST_UPDATE.
      dispatch({type: FETCH_ISSUE_LIST_SUCCESS});
    } catch (error) {
      dispatch({type: FETCH_ISSUE_LIST_FAILURE, error});
    };
  };

/**
 * Fetches the currently logged in user's permissions for a given issue.
 * @param {Issue} issueRef
 * @return {function(function): Promise<void>}
 */
export const fetchPermissions = (issueRef) => async (dispatch) => {
  dispatch({type: FETCH_PERMISSIONS_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'ListIssuePermissions', {issueRef},
    );

    dispatch({type: FETCH_PERMISSIONS_SUCCESS, permissions: resp.permissions});
  } catch (error) {
    dispatch({type: FETCH_PERMISSIONS_FAILURE, error});
  };
};

/**
 * Fetches comments for an issue. Note that issue descriptions are also
 * comments.
 * @param {IssueRef} issueRef
 * @return {function(function): Promise<void>}
 */
export const fetchComments = (issueRef) => async (dispatch) => {
  dispatch({type: FETCH_COMMENTS_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'ListComments', {issueRef});

    dispatch({type: FETCH_COMMENTS_SUCCESS, comments: resp.comments});
    dispatch(fetchCommentReferences(
        resp.comments, issueRef.projectName));

    const commenterRefs = (resp.comments || []).map(
        (comment) => comment.commenter);
    dispatch(user.fetchProjects(commenterRefs));
  } catch (error) {
    dispatch({type: FETCH_COMMENTS_FAILURE, error});
  };
};

/**
 * Gets whether the logged in user has starred a given issue.
 * @param {IssueRef} issueRef
 * @return {function(function): Promise<void>}
 */
export const fetchIsStarred = (issueRef) => async (dispatch) => {
  dispatch({type: FETCH_IS_STARRED_START});
  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'IsIssueStarred', {issueRef},
    );

    dispatch({
      type: FETCH_IS_STARRED_SUCCESS,
      starred: resp.isStarred,
      issueRef: issueRef,
    });
  } catch (error) {
    dispatch({type: FETCH_IS_STARRED_FAILURE, error});
  };
};

/**
 * Fetch all of a logged in user's starred issues.
 * @return {function(function): Promise<void>}
 */
export const fetchStarredIssues = () => async (dispatch) => {
  dispatch({type: FETCH_ISSUES_STARRED_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'ListStarredIssues', {},
    );
    dispatch({type: FETCH_ISSUES_STARRED_SUCCESS,
      starredIssueRefs: resp.starredIssueRefs});
  } catch (error) {
    dispatch({type: FETCH_ISSUES_STARRED_FAILURE, error});
  };
};

/**
 * Stars or unstars an issue.
 * @param {IssueRef} issueRef The issue to star.
 * @param {boolean} starred Whether to star or unstar.
 * @return {function(function): Promise<void>}
 */
export const star = (issueRef, starred) => async (dispatch) => {
  const requestKey = issueRefToString(issueRef);

  dispatch({type: STAR_START, requestKey});
  const message = {issueRef, starred};

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'StarIssue', message,
    );

    dispatch({
      type: STAR_SUCCESS,
      starCount: resp.starCount,
      issueRef,
      starred,
      requestKey,
    });
  } catch (error) {
    dispatch({type: STAR_FAILURE, error, requestKey});
  }
};

/**
 * Runs a presubmit request to find warnings to show the user before an issue
 * edit is saved.
 * @param {IssueRef} issueRef The issue being edited.
 * @param {IssueDelta} issueDelta The user's in flight changes to the issue.
 * @return {function(function): Promise<void>}
 */
export const presubmit = (issueRef, issueDelta) => async (dispatch) => {
  dispatch({type: PRESUBMIT_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'PresubmitIssue', {issueRef, issueDelta});

    dispatch({type: PRESUBMIT_SUCCESS, presubmitResponse: resp});
  } catch (error) {
    dispatch({type: PRESUBMIT_FAILURE, error: error});
  }
};

/**
 * Sends a request to run ML on a user's edits to guess what component
 * might fit an issue.
 * @param {string} projectName The project this check is happening in.
 * @param {string} text Text to run prediction against.
 * @return {function(function): Promise<void>}
 */
export const predictComponent = (projectName, text) => async (dispatch) => {
  dispatch({type: PREDICT_COMPONENT_START});

  const message = {
    projectName,
    text,
  };

  try {
    const response = await prpcClient.call(
        'monorail.Features', 'PredictComponent', message);
    const component = response.componentRef && response.componentRef.path ?
      response.componentRef.path : '';
    dispatch({type: PREDICT_COMPONENT_SUCCESS, component});
  } catch (error) {
    dispatch({type: PREDICT_COMPONENT_FAILURE, error: error});
  }
};

export const updateApproval = (message) => async (dispatch) => {
  const {issueRef} = message;
  dispatch({type: UPDATE_APPROVAL_START});

  try {
    const {approval} = await prpcClient.call(
        'monorail.Issues', 'UpdateApproval', message);

    dispatch({type: UPDATE_APPROVAL_SUCCESS, approval, issueRef});
    dispatch(fetch(issueRef));
    dispatch(fetchComments(issueRef));
  } catch (error) {
    dispatch({type: UPDATE_APPROVAL_FAILURE, error: error});
  };
};

export const update = (message) => async (dispatch) => {
  dispatch({type: UPDATE_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'UpdateIssue', message);

    dispatch({type: UPDATE_SUCCESS, issue: resp.issue});
    dispatch(fetchComments(message.issueRef));
    dispatch(fetchRelatedIssues(resp.issue));
    dispatch(fetchReferencedUsers(resp.issue));
  } catch (error) {
    dispatch({type: UPDATE_FAILURE, error: error});
  };
};

/**
 * Converts an issue from one template to another. This is used for changing
 * launch issues.
 * @param {IssueRef} issueRef
 * @param {Object} options
 * @param {string=} options.templateName
 * @param {string=} options.commentContent
 * @param {boolean=} options.sendEmail
 * @return {function(function): Promise<void>}
 */
export const convert = (issueRef, {templateName = '',
  commentContent = '', sendEmail = true},
) => async (dispatch) => {
  dispatch({type: CONVERT_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'ConvertIssueApprovalsTemplate',
        {issueRef, templateName, commentContent, sendEmail});

    dispatch({type: CONVERT_SUCCESS, issue: resp.issue});
    const fetchCommentsMessage = {issueRef};
    dispatch(fetchComments(fetchCommentsMessage));
  } catch (error) {
    dispatch({type: CONVERT_FAILURE, error: error});
  };
};
