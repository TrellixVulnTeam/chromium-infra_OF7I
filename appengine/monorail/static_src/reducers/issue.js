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
import {issueRefToString, issueToIssueRefString} from 'shared/converters.js';
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
const SET_ISSUE_REF = 'SET_ISSUE_REF';

export const FETCH_START = 'FETCH_START';
export const FETCH_SUCCESS = 'FETCH_SUCCESS';
export const FETCH_FAILURE = 'FETCH_FAILURE';

const FETCH_HOTLISTS_START = 'FETCH_HOTLISTS_START';
const FETCH_HOTLISTS_SUCCESS = 'FETCH_HOTLISTS_SUCCESS';
const FETCH_HOTLISTS_FAILURE = 'FETCH_HOTLISTS_FAILURE';

const FETCH_ISSUE_LIST_START = 'FETCH_ISSUE_LIST_START';
const FETCH_ISSUE_LIST_UPDATE = 'FETCH_ISSUE_LIST_UPDATE';
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

  issueRef: IssueRef,
  currentIssue: Issue,

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

// TODO(crbug.com/monorail/6882): Finish converting all other issue
//   actions to use this format.
/**
 * Adds issues fetched by a ListIssues request to the Redux store in a
 * normalized format.
 * @param {Object.<IssueRefString, Issue>} state Redux state.
 * @param {AnyAction} action
 * @param {Array<Issue>} action.issues The list of issues that was fetched.
 */
export const issuesByRefStringReducer = createReducer({}, {
  [FETCH_ISSUE_LIST_UPDATE]: (state, {issues}) => {
    const newState = {...state};

    issues.forEach((issue) => {
      const issueRefString = issueToIssueRefString(issue);

      newState[issueRefString] = {
        ...newState[issueRefString],
        ...issue,
      };
    });

    return newState;
  },
});

/**
 * Sets a reference for the issue that the user is currently viewing.
 * Note that this only handles the issue's numeric localId, not projectName.
 * Project name is inferred from separate state to reference the current project
 * the user is viewing.
 * @param {number} state Name of the currently viewed issue localId.
 * @param {AnyAction} action
 * @param {number} action.localId The updated localId to view.
 * @return {number}
 */
const localIdReducer = createReducer(0, {
  [SET_ISSUE_REF]: (state, {localId}) => localId || state,
});

/**
 * Changes the project that the user is viewing based on the viewed issue ref.
 * @param {string} state Name of the currently viewed project.
 * @param {AnyAction} action
 * @param {string} action.projectName Name of the new project to view.
 * @return {string}
 */
const projectNameReducer = createReducer('', {
  [SET_ISSUE_REF]: (state, {projectName}) => projectName || state,
});

/**
 * Updates data in the store for the issue the user is currently viewing. This
 * reducer handles many different possible actions that cause issue data to be
 * updated.
 * @param {Issue} state The issue data being mutated.
 * @param {AnyAction} action
 * @param {Issue=} action.issue New Issue Object to override values with.
 * @param {number=} action.starCount Number of stars the issue has. This changes
 *   when a user stars an issue and needs to be updated.
 * @param {ApprovalDef=} action.approval A new approval to update the issue
 *   with.
 * @return {Issue}
 */
const currentIssueReducer = createReducer({}, {
  [FETCH_SUCCESS]: (_state, {issue}) => issue,
  [STAR_SUCCESS]: (state, {starCount}) => {
    return {...state, starCount};
  },
  [CONVERT_SUCCESS]: (_state, {issue}) => issue,
  [UPDATE_SUCCESS]: (_state, {issue}) => issue,
  [UPDATE_APPROVAL_SUCCESS]: (state, {approval}) => {
    return updateApprovalValues(state, approval);
  },
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
    return starredIssueRefs.reduce((obj, issueRef) => ({
      ...obj, [issueRefToString(issueRef)]: true}), {});
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
  issueRef: combineReducers({
    localId: localIdReducer,
    projectName: projectNameReducer,
  }),

  issuesByRefString: issuesByRefStringReducer,

  currentIssue: currentIssueReducer,

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

export const viewedIssueRef = (state) => state.issue.issueRef;

// TODO(zhangtiff): Eventually Monorail's Redux state will store
// multiple issues, and this selector will have to find the viewed
// issue based on a viewed issue ref.
export const viewedIssue = (state) => state.issue.currentIssue;

export const comments = (state) => state.issue.comments;
export const commentsLoaded = (state) => state.issue.commentsLoaded;

const _commentReferences = (state) => state.issue.commentReferences;
export const commentReferences = createSelector(_commentReferences,
    (commentReferences) => objectToMap(commentReferences));

export const hotlists = (state) => state.issue.hotlists;

const issuesByRefString = (state) => state.issue.issuesByRefString;
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

export const issueLoaded = (state) => state.issue.issueLoaded;
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
export const setIssueRef = (localId, projectName) => {
  return {type: SET_ISSUE_REF, localId, projectName};
};

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

export const fetchIssuePageData = (message) => async (dispatch) => {
  dispatch(fetchComments(message));
  dispatch(fetch(message));
  dispatch(fetchPermissions(message));
  dispatch(fetchIsStarred(message));
};

export const fetch = (message) => async (dispatch) => {
  dispatch({type: FETCH_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'GetIssue', message,
    );
    const movedToRef = resp.movedToRef;
    const issue = {...resp.issue};
    if (movedToRef) {
      issue.movedToRef = movedToRef;
    }

    dispatch({type: FETCH_SUCCESS, issue});

    if (!issue.isDeleted && !movedToRef) {
      dispatch(fetchRelatedIssues(issue));
      dispatch(fetchHotlists(message.issueRef));
      dispatch(fetchReferencedUsers(issue));
      dispatch(user.fetchProjects([issue.reporterRef]));
    }
  } catch (error) {
    dispatch({type: FETCH_FAILURE, error});
  }
};

export const fetchHotlists = (issue) => async (dispatch) => {
  dispatch({type: FETCH_HOTLISTS_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Features', 'ListHotlistsByIssue', {issue});

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
 * @param {Object} params Options for which issues to fetch.
 * @param {string} [params.q] The query string for the search.
 * @param {string} [params.can] The ID of the canned query for the search.
 * @param {string} [params.groupby] The spec of which fields to group by.
 * @param {string} [params.sort] The spec of which fields to sort by.
 * @param {string} projectName The project to fetch issues from.
 * @param {Object} pagination Object with info on how many issues to fetch and
 *   how many issues to offset.
 * @param {number} maxCalls The maximum number of API calls to make. Combined
 *   with pagination.maxItems, this defines the maximum number of issues this
 *   method can fetch.
 * @return {Function}
 */
export const fetchIssueList =
  (params, projectName, pagination = {}, maxCalls = 1) => async (dispatch) => {
    let updateData = {};
    const promises = [];
    const issuesByRequest = [];
    let issueLimit;
    let totalIssues;
    let totalCalls;
    const itemsPerCall = (pagination.maxItems || 1000);

    const can = Number.parseInt(params.can) || undefined;

    dispatch({type: FETCH_ISSUE_LIST_START});

    // initial api call made to determine total number of issues matching
    // the query.
    try {
      // TODO(zhangtiff): Refactor this action creator when adding issue
      // list pagination.
      const resp = await prpcClient.call(
          'monorail.Issues', 'ListIssues', {
            query: params.q,
            cannedQuery: can,
            projectNames: [projectName],
            pagination: pagination,
            groupBySpec: params.groupby,
            sortSpec: params.sort,
          });

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
                query: params.q,
                cannedQuery: can,
                projectNames: [projectName],
                pagination: {start: i * itemsPerCall, maxItems: itemsPerCall},
                groupBySpec: params.groupby,
                sortSpec: params.sort,
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

export const fetchPermissions = (message) => async (dispatch) => {
  dispatch({type: FETCH_PERMISSIONS_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'ListIssuePermissions', message,
    );

    dispatch({type: FETCH_PERMISSIONS_SUCCESS, permissions: resp.permissions});
  } catch (error) {
    dispatch({type: FETCH_PERMISSIONS_FAILURE, error});
  };
};

export const fetchComments = (message) => async (dispatch) => {
  dispatch({type: FETCH_COMMENTS_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'ListComments', message);

    dispatch({type: FETCH_COMMENTS_SUCCESS, comments: resp.comments});
    dispatch(fetchCommentReferences(
        resp.comments, message.issueRef.projectName));

    const commenterRefs = (resp.comments || []).map(
        (comment) => comment.commenter);
    dispatch(user.fetchProjects(commenterRefs));
  } catch (error) {
    dispatch({type: FETCH_COMMENTS_FAILURE, error});
  };
};

export const fetchIsStarred = (message) => async (dispatch) => {
  dispatch({type: FETCH_IS_STARRED_START});
  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'IsIssueStarred', message,
    );

    dispatch({
      type: FETCH_IS_STARRED_SUCCESS,
      starred: resp.isStarred,
      issueRef: message.issueRef,
    });
  } catch (error) {
    dispatch({type: FETCH_IS_STARRED_FAILURE, error});
  };
};

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

export const presubmit = (message) => async (dispatch) => {
  dispatch({type: PRESUBMIT_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'PresubmitIssue', message);

    dispatch({type: PRESUBMIT_SUCCESS, presubmitResponse: resp});
  } catch (error) {
    dispatch({type: PRESUBMIT_FAILURE, error: error});
  }
};

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
  dispatch({type: UPDATE_APPROVAL_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'UpdateApproval', message);

    dispatch({type: UPDATE_APPROVAL_SUCCESS, approval: resp.approval});
    const baseMessage = {issueRef: message.issueRef};
    dispatch(fetch(baseMessage));
    dispatch(fetchComments(baseMessage));
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
    const fetchCommentsMessage = {issueRef: message.issueRef};
    dispatch(fetchComments(fetchCommentsMessage));
    dispatch(fetchRelatedIssues(resp.issue));
    dispatch(fetchReferencedUsers(resp.issue));
  } catch (error) {
    dispatch({type: UPDATE_FAILURE, error: error});
  };
};

export const convert = (message) => async (dispatch) => {
  dispatch({type: CONVERT_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Issues', 'ConvertIssueApprovalsTemplate', message);

    dispatch({type: CONVERT_SUCCESS, issue: resp.issue});
    const fetchCommentsMessage = {issueRef: message.issueRef};
    dispatch(fetchComments(fetchCommentsMessage));
  } catch (error) {
    dispatch({type: CONVERT_FAILURE, error: error});
  };
};
