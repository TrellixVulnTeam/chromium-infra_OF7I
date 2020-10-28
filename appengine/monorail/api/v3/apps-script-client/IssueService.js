// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/* eslint-disable no-unused-vars */

const COMMENT_TYPE_DESCRIPTION = 'DESCRIPTION';

/**
 * Fetches the issue from Monorail.
 * @param {string} issueName The resource name of the issue.
 * @return {Issue}
 */
function getIssue(issueName) {
  const message = {'name': issueName};
  const url = URL + 'monorail.v3.Issues/GetIssue';
  return run_(url, message);
}

/**
 * Fetches all the given issues from Monorail.
 * @param {Array<string>} issueNames The resource names of the issues.
 * @return {Array<Issue>}
 */
function batchGetIssues(issueNames) {
  const message = {'names': issueNames};
  const url = URL + 'monorail.v3.Issues/BatchGetIssues';
  return run_(url, message);
}

/**
 * Fetches all the ApprovalValues that belong to the given issue.
 * @param {string} issueName The resource name of the issue.
 * @return {Array<ApprovalValue>}
 */
function listApprovalValues(issueName) {
  const message = {'parent': issueName};
  const url = URL + 'monorail.v3.Issues/ListApprovalValues';
  return run_(url, message);
}

/**
 * Calls SearchIssues with the given parameters.
 * @param {Array<string>} projectNames resource names of the projects to search.
 * @param {string} query The query to use to search.
 * @param {string} orderBy The issue fields to order issues by.
 * @param {Number} pageSize The maximum issues to return.
 * @param {string} pageToken The page token from the previous call.
 * @return {Array<SearchIssuesResponse>}
 */
function searchIssuesPagination_(
    projectNames, query, orderBy, pageSize, pageToken) {
  const message = {
    'projects': projectNames,
    'query': query,
    'orderBy': orderBy,
    'pageToken': pageToken};
  if (pageSize) {
    message['pageSize'] = pageSize;
  }
  const url = URL + 'monorail.v3.Issues/SearchIssues';
  return run_(url, message);
}

// TODO(crbug.com/monorail/7143): SearchIssues only accepts one project.
/**
 * Searches Monorail for issues using the given query.
 * NOTE: We currently only accept `projectNames` with one and only one project.
 * @param {Array<string>} projects Resource names of the projects to search
 *     within.
 * @param {string=} query The query to use to search.
 * @param {string=} orderBy The issue fields to order issues by,
 *    e.g. 'EstDays,Opened,-stars'
 * @return {Array<Issue>}
 */
function searchIssues(projects, query, orderBy) {
  const pageSize = 100;
  let pageToken;

  issues = [];

  do {
    const resp = searchIssuesPagination_(
        projects, query, orderBy, pageSize, pageToken);
    issues = issues.concat(resp.issues);
    pageToken = resp.nextPageToken;
  }
  while (pageToken);

  return issues;
}

/**
 * Calls ListComments with the given parameters.
 * @param {string} issueName Resource name of the issue.
 * @param {string} filter The approval filter query.
 * @param {Number} pageSize The maximum number of comments to return.
 * @param {string} pageToken The page token from the previous request.
 * @return {ListCommentsResponse}
 */
function listCommentsPagination_(issueName, filter, pageSize, pageToken) {
  const message = {
    'parent': issueName,
    'pageToken': pageToken,
    'filter': filter,
  };
  if (pageSize) {
    message['pageSize'] = pageSize;
  }
  const url = URL + 'monorail.v3.Issues/ListComments';
  return run_(url, message);
}

/**
 * Returns all comments and previous/current descriptions of an issue.
 * @param {string} issueName Resource name of the Issue.
 * @param {string=} filter The filter query filtering out comments.
 *     We only accept `approval = "<approvalDef resource name>""`.
 *     e.g. 'approval = "projects/chromium/approvalDefs/34"'
 * @return {Array<Comment>}
 */
function listComments(issueName, filter) {
  let pageToken;

  let comments = [];
  do {
    const resp = listCommentsPagination_(issueName, filter, '', pageToken);
    comments = comments.concat(resp.comments);
    pageToken = resp.nextPageToken;
  }
  while (pageToken);

  return comments;
}

/**
 * Gets the current description of an issue.
 * @param {string} issueName Resource name of the Issue.
 * @param {string=} filter The filter query filtering out comments.
 *     We only accept `approval = "<approvalDef resource name>""`.
 *     e.g. 'approval = "projects/chromium/approvalDefs/34"'
 * @return {Comment}
 */
function getCurrentDescription(issueName, filter) {
  const allComments = listComments(issueName, filter);
  for (let i = (allComments.length - 1); i > -1; i--) {
    if (allComments[i].type === COMMENT_TYPE_DESCRIPTION) {
      return allComments[i];
    }
  }
}

/**
 * Gets the first (non-description) comment of an issue.
 * @param {string} issueName Resource name of the Issue.
 * @param {string=} filter The filter query filtering out comments.
 *     We only accept `approval = "<approvalDef resource name>""`.
 *      e.g. 'approval = "projects/chromium/approvalDefs/34"'
 * @return {Comment}
 */
function getFirstComment(issueName, filter) {
  const allComments = listComments(issueName, filter);
  for (let i = 0; i < allComments.length; i++) {
    if (allComments[i].type !== COMMENT_TYPE_DESCRIPTION) {
      return allComments[i];
    }
  }
  return null;
}

/**
 * Gets the last (non-description) comment of an issue.
 * @param {string} issueName The resource name of the issue.
 * @param {string=} filter The filter query filtering out comments.
 *     We only accept `approval = "<approvalDef resource name>""`.
 *     e.g. 'approval = "projects/chromium/approvalDefs/34"'
 * @return {Issue}
 */
function getLastComment(issueName, filter) {
  const allComments = listComments(issueName, filter);
  for (let i = (allComments.length - 1); i > -1; i--) {
    if (allComments[i].type != COMMENT_TYPE_DESCRIPTION) {
      return allComments[i];
    }
  }
  return null;
}

/**
 * Checks if the given label exists in the issue.
 * @param {Issue} issue The issue to search within for the label.
 * @param {string} label The label to search for.
 * @return {boolean}
 */
function hasLabel(issue, label) {
  if (issue.labels) {
    const testLabel = label.toLowerCase();
    return issue.labels.some(({label}) => testLabel === label.toLowerCase());
  }
  return false;
}

/**
 * Checks if the issue has any labels matching the given regex.
 * @param {Issue} issue The issue to search within for matching labels.
 * @param {string} regex The regex pattern to use to search for labels.
 * @return {boolean}
 */
function hasLabelMatching(issue, regex) {
  if (issue.labels) {
    const re = new RegExp(regex, 'i');
    return issue.labels.some(({label}) => re.test(label));
  }
  return false;
}

/**
 * Returns all labels in the issue that match the given regex.
 * @param {Issue} issue The issue to search within for matching labels.
 * @param {string} regex The regex pattern to use to search for labels.
 * @return {Array<string>}
 */
function getLabelsMatching(issue, regex) {
  const labels = [];
  if (issue.labels) {
    const re = new RegExp(regex, 'i');
    for (let i = 0; i < issue.labels.length; i++) {
      if (re.test(issue.labels[i].label)) {
        labels.push(issue.labels[i].label);
      }
    }
  }
  return labels;
}

/**
 * Get the comment where the given label was added, if any.
 * @param {string} issueName The resource name of the issue.
 * @param {string} label The label that was remove.
 * @return {Comment}
 */
function getLabelSetComment(issueName, label) {
  const comments = listComments(issueName);
  for (let i = 0; i < comments.length; i++) {
    const comment = comments[i];
    if (comment.amendments) {
      for (let j = 0; j < comment.amendments.length; j++) {
        const amendment = comment.amendments[j];
        if (amendment['fieldName'] === 'Labels' &&
            amendment['newOrDeltaValue'].toLowerCase() === (
              label.toLocaleLowerCase())) {
          return comment;
        }
      }
    }
  }
  return null;
}

/**
 * Get the comment where the given label was removed, if any.
 * @param {string} issueName The resource name of the issue.
 * @param {string} label The label that was remove.
 * @return {Comment}
 */
function getLabelRemoveComment(issueName, label) {
  const comments = listComments(issueName);
  for (let i = 0; i < comments.length; i++) {
    const comment = comments[i];
    if (comment.amendments) {
      for (let j = 0; j < comment.amendments.length; j++) {
        const amendment = comment.amendments[j];
        if (amendment['fieldName'] === 'Labels' &&
            amendment[
                'newOrDeltaValue'].toLowerCase() === (
              '-' + label.toLocaleLowerCase())) {
          return comment;
        }
      }
    }
  }
  return null;
}

/**
 * Updates the issue to have the given label added.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue The issue to update.
 * @param {string} label The label to add.
 */
function addLabel(issue, label) {
  if (hasLabel(issue, label)) return;
  maybeCreateDelta_(issue);
  // Add the label to the issue's delta.labelsAdd.
  issue.delta.labelsAdd.push(label);
  // Add the label to the issue.
  issue.labels.push({label: label});
  // 'labels' added to updateMask in saveChanges().
}

/**
 * Updates the issue to have the given label removed from the issue.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue The issue to update.
 * @param {string} label The label to remove.
 */
function removeLabel(issue, label) {
  if (!hasLabel(issue, label)) return;
  maybeCreateDelta_(issue);
  // Add the label to the issue's delta.labelsRemove.
  issue.delta.labelsRemove.push(label);
  // Remove label from issue.
  for (let i = 0; i < issue.labels.length; i++) {
    if (issue.labels[i].label.toLowerCase() === label.toLowerCase()) {
      issue.labels.splice(i, 1);
      break;
    }
  }
}

/**
 * Sets the owner of the given issue.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {string} ownerName The resource name of the new owner,
 *     e.g. 'users/chicken@email.com'
*/
function setOwner(issue, ownerName) {
  maybeCreateDelta_(issue);
  issue.owner = {'user': ownerName};
  if (issue.delta.updateMask.indexOf('owner.user') === -1) {
    issue.delta.updateMask.push('owner.user');
  }
}

/**
 * Sets the summary of the given issue.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {string} summary The new summary of the issue.
*/
function setSummary(issue, summary) {
  maybeCreateDelta_(issue);
  issue.summary = summary;
  if (issue.delta.updateMask.indexOf('summary') === -1) {
    issue.delta.updateMask.push('summary');
  }
}

/**
 *Sets the status of the given issue.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {string} status The new status of the issue e.g. 'Available'.
*/
function setStatus(issue, status) {
  maybeCreateDelta_(issue);
  issue.status.status = status;
  if (issue.delta.updateMask.indexOf('status.status') === -1) {
    issue.delta.updateMask.push('status.status');
  }
}

/**
 * Sets the merged into issue for the given issue.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {IssueRef} mergedIntoRef IssueRef of the issue to merge into.
 */
function setMergedInto(issue, mergedIntoRef) {
  maybeCreateDelta_(issue);
  issue.mergedIntoIssueRef = mergedIntoRef;
  if (issue.delta.updateMask.indexOf('mergedIntoIssueRef') === -1) {
    issue.delta.updateMask.push('mergedIntoIssueRef');
  }
}

/**
 * Checks if target is found in source.
 * @param {IssueRef} target The IssueRef to look for.
 * @param {Array<IssueRef>} source the IssueRefs to look in.
 * @return {number} index of target in source, -1 if not found.
 */
function issueRefExists_(target, source) {
  for (let i = 0; i < source.length; i++) {
    if ((source[i].issue === target.issue || (!source[i].issue && !target.issue)
    ) && (source[i].extIdentifier === target.extIdentifier || (
      !source[i].extIdentifier && !target.extIdentifier))) {
      return i;
    }
  }
  return -1;
}

/**
 * Makes blocking issue ref changes.
 * blockingIssuesAdd are added before blockingIssuesRemove are removed.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {Array<IssueRef>} blockingIssuesAdd issues to add as blocking issues.
 * @param {Array<IssueRef>} blockingIssuesRemove issues to remove from blocking
 *     issues.
 */
function addBlockingIssueChanges(
    issue, blockingIssuesAdd, blockingIssuesRemove) {
  maybeCreateDelta_(issue);
  blockingIssuesAdd.forEach((addRef) => {
    const iInIssue = issueRefExists_(addRef, issue.blockingIssueRefs);
    if (iInIssue === -1) { // addRef not found in issue
      issue.blockingIssueRefs.push(addRef);
      issue.delta.blockingAdd.push(addRef);
      const iInDeltaRemove = issueRefExists_(
          addRef, issue.delta.blockingRemove);
      if (iInDeltaRemove != -1) {
        // Remove addRef from blckingRemove that may have been added earlier.
        issue.delta.blockingRemove.splice(iInDeltaRemove, 1);
      }
      // issue.delta.updateMask is updated in saveChanges()
    }
  });
  // Add blockingIssuesAdd to issue and issue.delta.blockingAdd if not in
  // issue.blockingIssues
  blockingIssuesRemove.forEach((removeRef) => {
    const iInIssue = issueRefExists_(removeRef, issue.blockingIssueRefs);
    if (iInIssue > -1) {
      issue.blockingIssueRefs.splice(iInIssue, 1);
      issue.delta.blockingRemove.push(removeRef);
      const iInDeltaAdd = issueRefExists_(removeRef, issue.delta.blockingAdd);
      if (iInDeltaAdd != -1) {
        issue.delta.blockingAdd.splice(iInDeltaAdd, 1);
      }
    }
  });
}

/**
 * Makes blocked-on issue ref changes.
 * blockedOnIssuesAdd are added before blockedOnIssuesRemove are removed.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {Array<IssueRef>} blockedOnIssuesAdd issues to add as blockedon
 *     issues.
 * @param {Array<IssueRef>} blockedOnIssuesRemove issues to remove from
 *     blockedon issues.
 */
function addBlockedOnIssueChanges(
    issue, blockedOnIssuesAdd, blockedOnIssuesRemove) {
  maybeCreateDelta_(issue);
  blockedOnIssuesAdd.forEach((addRef) => {
    const iInIssue = issueRefExists_(addRef, issue.blockedOnIssueRefs);
    if (iInIssue === -1) { // addRef not found in issue
      issue.blockedOnIssueRefs.push(addRef);
      issue.delta.blockedOnAdd.push(addRef);
      const iInDeltaRemove = issueRefExists_(
          addRef, issue.delta.blockedOnRemove);
      if (iInDeltaRemove != -1) {
        // Remove addRef from blckingRemove that may have been added earlier.
        issue.delta.blockedOnRemove.splice(iInDeltaRemove, 1);
      }
      // issue.delta.updateMask is updated in saveChanges()
    }
  });
  // Add blockedOnIssuesAdd to issue and issue.delta.blockedOnAdd if not in
  // issue.blockedOnIssues.
  blockedOnIssuesRemove.forEach((removeRef) => {
    const iInIssue = issueRefExists_(removeRef, issue.blockedOnIssueRefs);
    if (iInIssue > -1) {
      issue.blockedOnIssueRefs.splice(iInIssue, 1);
      issue.delta.blockedOnRemove.push(removeRef);
      const iInDeltaAdd = issueRefExists_(removeRef, issue.delta.blockedOnAdd);
      if (iInDeltaAdd != -1) {
        issue.delta.blockedOnAdd.splice(iInDeltaAdd, 1);
      }
    }
  });
}


/**
 * Looks for a component name in an Array of ComponentValues.
 * @param {string} compName Resource name of the Component to look for.
 * @param {Array<ComponentValue>} compArray List of ComponentValues.
 * @return {number} Index of compName in compArray, -1 if not found.
 */
function componentExists_(compName, compArray) {
  for (let i = 0; i < compArray.length; i++) {
    if (compArray[i].component === compName) {
      return i;
    }
  }
  return -1;
}

/**
 * Adds the component changes to the issue.
 * componentNamesAdd are added before componentNamesremove are removed.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {Array<string>} componentNamesAdd Array of component resource names.
 * @param {Array<string>} componentNamesRemove Array or component resource
 *     names.

*/
function addComponentChanges(issue, componentNamesAdd, componentNamesRemove) {
  maybeCreateDelta_(issue);
  componentNamesAdd.forEach((compName) => {
    const iInIssue = componentExists_(compName, issue.components);
    if (iInIssue === -1) { // compName is not in issue.
      issue.components.push({'component': compName});
      issue.delta.componentsAdd.push(compName);
      const iInDeltaRemove = issue.delta.componentsRemove.indexOf(compName);
      if (iInDeltaRemove != -1) {
        // Remove compName from issue.delta.componentsRemove that may have been
        // added before.
        issue.delta.componentsRemove.splice(iInDeltaRemove, 1);
      }
      // issue.delta.updateMask is updated in saveChanges()
    }
  });

  componentNamesRemove.forEach((compName) => {
    const iInIssue = componentExists_(compName, issue.components);
    if (iInIssue != -1) { // compName was found in issue.
      issue.components.splice(iInIssue, 1);
      issue.delta.componentsRemove.push(compName);
      const iInDeltaAdd = issue.delta.componentsAdd.indexOf(compName);
      if (iInDeltaAdd != -1) {
        // Remove compName from issue.delta.componentsAdd that may have been
        // added before.
        issue.delta.componentsAdd.splice(iInDeltaAdd, 1);
      }
    }
  });
}

/**
 * Checks if the fieldVal is found in fieldValsArray
 * @param {FieldValue} fieldVal the field to look for.
 * @param {Array<FieldValue>} fieldValsArray the Array to look within.
 * @return {number} the index of fieldVal in fieldValsArray, or -1 if not found.
 */
function fieldValueExists_(fieldVal, fieldValsArray) {
  for (let i = 0; i < fieldValsArray.length; i++) {
    const currFv = fieldValsArray[i];
    if (currFv.field === fieldVal.field && currFv.value === fieldVal.value && (
      currFv.phase === fieldVal.phase || (
        !currFv.phase && !fieldVal.phase))) {
      return i;
    }
  }
  return -1;
}

/**
 * Adds the FieldValue changes to the issue.
 * fieldValuesAdd are added before fieldValuesRemove are removed.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {Array<FieldValue>} fieldValuesAdd Array of FieldValues to add.
 * @param {Array<FieldValue>} fieldValuesRemove Array of FieldValues to remove.
*/
function addFieldValueChanges(issue, fieldValuesAdd, fieldValuesRemove) {
  maybeCreateDelta_(issue);
  fieldValuesAdd.forEach((fvAdd) => {
    const iInIssue = fieldValueExists_(fvAdd, issue.fieldValues);
    if (iInIssue === -1) { // fvAdd is not already in issue, so we can add it.
      issue.fieldValues.push(fvAdd);
      issue.delta.fieldValuesAdd.push(fvAdd);
      const iInDeltaRemove = fieldValueExists_(
          fvAdd, issue.delta.fieldValuesRemove);
      if (iInDeltaRemove != -1) {
        // fvAdd was added to fieldValuesRemove in a previous call.
        issue.delta.fieldValuesRemove.splice(iInDeltaRemove, 1);
      }
      // issue.delta.updateMask is updated in saveChanges()
    }
  });
  // issue.delta.updateMask is updated in saveChanges()
  fieldValuesRemove.forEach((fvRemove) => {
    const iInIssue = fieldValueExists_(fvRemove, issue.fieldValues);
    if (iInIssue != -1) { // fvRemove is in issue, so we can remove it.
      issue.fieldValues.splice(iInIssue, 1);
      issue.delta.fieldValuesRemove.push(fvRemove);
      const iInDeltaAdd = fieldValueExists_(
          fvRemove, issue.delta.fieldValuesAdd);
      if (iInDeltaAdd != -1) {
        // fvRemove was added to fieldValuesAdd in a previous call.
        issue.delta.fieldValuesAdd.splice(iInDeltaAdd, 1);
      }
    }
  });
}

/**
 * Checks for the existence of userName in userValues
 * @param {string} userName A user resource name to look for.
 * @param {Array<UserValue>} userValues UserValues to search through.
 * @return {number} Index of userName's UserValue in userValues or -1 if not
 *     found.
 */
function userValueExists_(userName, userValues) {
  for (let i = 0; i< userValues.length; i++) {
    if (userValues[i].user === userName) {
      return i;
    }
  }
  return -1;
}

/**
 * Adds the CC changes to the issue.
 * ccNamesAdd are added before ccNamesRemove are removed.
 * This method does not call Monorail's API to save this change.
 * Call saveChanges() to send all updates to Monorail.
 * @param {Issue} issue Issue to change.
 * @param {Array<string>} ccNamesAdd Array if user resource names.
 * @param {Array<string>} ccNamesRemove Array if user resource names.
*/
function addCcChanges(issue, ccNamesAdd, ccNamesRemove) {
  maybeCreateDelta_(issue);
  ccNamesAdd.forEach((ccName) => {
    const iInIssue = userValueExists_(ccName, issue.ccUsers);
    if (iInIssue === -1) { // User is not in issue, so we can add them.
      issue.ccUsers.push({'user': ccName});
      issue.delta.ccsAdd.push(ccName);
      const iInDeltaRemove = issue.delta.ccsRemove.indexOf(ccName);
      if (iInDeltaRemove != -1) {
        // ccName was added to ccsRemove in a previous call.
        issue.delta.ccsRemove.splice(iInDeltaRemove, 1);
      }
    }
  });
  ccNamesRemove.forEach((ccName) => {
    const iInIssue = userValueExists_(ccName, issue.ccUsers);
    if (iInIssue != -1) { // User is in issue, so we can remove it.
      issue.ccUsers.splice(iInIssue, 1);
      issue.delta.ccsRemove.push(ccName);
      const iInDeltaAdd = issue.delta.ccsAdd.indexOf(ccName);
      if (iInDeltaAdd != -1) {
        // ccName was added to delta.ccsAdd in a previous all.
        issue.delta.ccsAdd.splice(iInDeltaAdd, 1);
      }
    }
  });
}

/**
 * Set the pending comment of the issue.
 * @param {Issue} issue Issue whose comment we want to set.
 * @param {string} comment Comment that we want for the issue.
 */
function setComment(issue, comment) {
  maybeCreateDelta_(issue);
  issue.delta.comment = comment;
}

/**
 * Get the pending comment for the issue.
 * @param {Issue} issue Issue whose comment we want.
 * @return {string}
 */
function getPendingComment(issue) {
  if (issue.delta) {
    return issue.delta.comment;
  }
  return '';
}

/**
 * Adds to the existing pending comment
 * @param {Issue} issue Issue to update.
 * @param {string} comment The comment string to add to the existing one.
 */
function appendComment(issue, comment) {
  maybeCreateDelta_(issue);
  issue.delta.comment = issue.delta.comment.concat(comment);
}

/**
 * Sets up an issue for pending changes.
 * @param {Issue} issue The issue that needs to be updated.
 */
function maybeCreateDelta_(issue) {
  if (!issue.delta) {
    issue.delta = newIssueDelta_();
    if (!issue.components) {
      issue.components = [];
    };
    if (!issue.blockingIssueRefs) {
      issue.blockingIssueRefs = [];
    }
    if (!issue.blockedOnIssueRefs) {
      issue.blockedOnIssueRefs = [];
    }
    if (!issue.ccUsers) {
      issue.ccUsers = [];
    }
    if (!issue.labels) {
      issue.labels = [];
    }
    if (!issue.fieldValues) {
      issue.fieldValues = [];
    }
  }
}

/**
 * Creates an IssueDelta
 * @return {IssueDelta_}
 */
function newIssueDelta_() {
  return new IssueDelta_();
}

/** Used to track pending changes to an issue.*/
function IssueDelta_() {
  /** Array<string> */ this.updateMask = [];

  // User resource names.
  /** Array<string> */ this.ccsRemove = [];
  /** Array<string> */ this.ccsAdd = [];

  /** Array<IssueRef> */ this.blockedOnRemove = [];
  /** Array<IssueRef> */ this.blockedOnAdd = [];
  /** Array<IssueRef> */ this.blockingRemove = [];
  /** Array<IssueRef> */ this.blockingAdd = [];

  // Component resource names.
  /** Array<string> */ this.componentsRemove = [];
  /** Array<string> */ this.componentsAdd = [];

  // Label values, e.g. 'Security-Notify'.
  /** Array<string> */ this.labelsRemove = [];
  /** Array<string> */ this.labelsAdd = [];

  /** Array<FieldValue> */ this.fieldValuesRemove = [];
  /** Array<FieldValue> */ this.fieldValuesAdd = [];

  this.comment = '';
}

/**
 * Calls Monorail's API to update the issue.
 * @param {Issue} issue The issue to update where issue['delta'] is expected
 *     to exist.
 * @param {boolean} sendEmail True if the update should trigger email
 *     notifications.
 * @return {Issue}
 */
function saveChanges(issue, sendEmail) {
  if (!issue.delta) {
    throw new Error('No pending changes for issue.');
  }

  const modifyDelta = {
    'ccsRemove': issue.delta.ccsRemove,
    'blockedOnIssuesRemove': issue.delta.blockedOnRemove,
    'blockingIssuesRemove': issue.delta.blockingRemove,
    'componentsRemove': issue.delta.componentsRemove,
    'labelsRemove': issue.delta.labelsRemove,
    'fieldValsRemove': issue.delta.fieldValuesRemove,
    'issue': {
      'name': issue.name,
      'fieldValues': issue.delta.fieldValuesAdd,
      'blockedOnIssueRefs': issue.delta.blockedOnAdd,
      'blockingIssueRefs': issue.delta.blockingAdd,
      'mergedIntoIssueRef': issue.mergedIntoIssueRef,
      'summary': issue.summary,
      'status': issue.status,
      'owner': issue.owner,
      'labels': [],
      'ccUsers': [],
      'components': [],
    },
  };

  if (issue.delta.fieldValuesAdd.length > 0) {
    issue.delta.updateMask.push('fieldValues');
  }

  if (issue.delta.blockedOnAdd.length > 0) {
    issue.delta.updateMask.push('blockedOnIssueRefs');
  }

  if (issue.delta.blockingAdd.length > 0) {
    issue.delta.updateMask.push('blockingIssueRefs');
  }

  if (issue.delta.ccsAdd.length > 0) {
    issue.delta.updateMask.push('ccUsers');
  }
  issue.delta.ccsAdd.forEach((userResourceName) => {
    modifyDelta.issue['ccUsers'].push({'user': userResourceName});
  });

  if (issue.delta.labelsAdd.length > 0) {
    issue.delta.updateMask.push('labels');
  }
  issue.delta.labelsAdd.forEach((label) => {
    modifyDelta.issue['labels'].push({'label': label});
  });

  if (issue.delta.componentsAdd.length > 0) {
    issue.delta.updateMask.push('components');
  }
  issue.delta.componentsAdd.forEach((compResourceName) => {
    modifyDelta.issue['components'].push({'component': compResourceName});
  });

  modifyDelta['updateMask'] = issue.delta.updateMask.join();

  const message = {
    'deltas': [modifyDelta],
    'notifyType': sendEmail ? 'EMAIL' : 'NO_NOTIFICATION',
    'commentContent': issue.delta.comment,
  };

  const url = URL + 'monorail.v3.Issues/ModifyIssues';
  response = run_(url, message);
  if (!response.issues) {
    Logger.log('All changes Noop');
    return null;
  }
  issue = response.issues[0];
  return issue;
}

/**
 * Creates an Issue.
 * @param {string} projectName: Resource name of the parent project.
 * @param {string} summary: Summary of the issue.
 * @param {string} description: Description of the issue.
 * @param {string} status: Status of the issue, e.g. "Untriaged".
 * @param {boolean} sendEmail: True if this should trigger email notifications.
 * @param {string=} ownerName: Resource name of the issue owner.
 * @param {Array<string>=} ccNames: Resource names of the users to cc.
 * @param {Array<string>=} labels: Labels to add to the issue,
 *     e.g. "Restict-View-Google".
 * @param {Array<string>=} componentNames: Resource names of components to add.
 * @param {Array<FieldValue>=} fieldValues: FieldValues to add to the issue.
 * @param {Array<IssueRef>=} blockedOnRefs: IssueRefs for blocked on issues.
 * @param {Array<IssueRef>=} blockingRefs: IssueRefs for blocking issues.
 * @return {Issue}
 */
function makeIssue(
    projectName, summary, description, status, sendEmail, ownerName, ccNames,
    labels, componentNames, fieldValues, blockedOnRefs, blockingRefs) {
  const issue = {
    'summary': summary,
    'status': {'status': status},
    'ccUsers': [],
    'components': [],
    'labels': [],
  };

  if (ownerName) {
    issue['owner'] = {'user': ownerName};
  }

  if (ccNames) {
    ccNames.forEach((ccName) => {
      issue['ccUsers'].push({'user': ccName});
    });
  };

  if (labels) {
    labels.forEach((label) => {
      issue['labels'].push({'label': label});
    });
  };

  if (componentNames) {
    componentNames.forEach((componentName) => {
      issue['components'].push({'component': componentName});
    });
  };

  if (fieldValues) {
    issue['fieldValues'] = fieldValues;
  };

  if (blockedOnRefs) {
    issue['blockedOnIssueRefs'] = blockedOnRefs;
  };

  if (blockingRefs) {
    issue['blockingIssueRefs'] = blockingRefs;
  };

  const message = {
    'parent': projectName,
    'issue': issue,
    'description': description,
    'notifyType': sendEmail ? 'EMAIL': 'NO_NOTIFICATION',
  };
  const url = URL + 'monorail.v3.Issues/MakeIssue';
  return run_(url, message);
}
