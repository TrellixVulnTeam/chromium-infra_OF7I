// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview This file collects helpers for managing various canonical
 * formats used within Monorail's frontend. When converting between common
 * Objects, for example, it's recommended to use the helpers in this file
 * to ensure consistency across conversions.
 */

import qs from 'qs';

import {equalsIgnoreCase} from './helpers.js';
import {fromShortlink} from 'shared/federated.js';
import {UserInputError} from 'shared/errors.js';
import './typedef.js';

// A Regex defining the canonical String format used in Monorail for allowing
// users to input structured localId and projectName values in free text inputs.
// Match: projectName:localId format where projectName is optional.
// ie: "monorail:1234" or "1234".
const ISSUE_ID_REGEX = /(?:([a-z0-9-]+):)?(\d+)/i;

// RFC 2821-compliant email address regex used by the server when validating
// email addresses.
// eslint-disable-next-line max-len
const RFC_2821_EMAIL_REGEX = /^[-a-zA-Z0-9!#$%&'*+\/=?^_`{|}~]+(?:[.][-a-zA-Z0-9!#$%&'*+\/=?^_`{|}~]+)*@(?:(?:[0-9a-zA-Z](?:[-]*[0-9a-zA-Z]+)*)(?:\.[0-9a-zA-Z](?:[-]*[0-9a-zA-Z]+)*)*)\.(?:[a-zA-Z]{2,9})$/;

/**
 * Converts a displayName into a canonical UserRef Object format.
 *
 * @param {String} displayName The user's email address, used as a display name.
 * @return {UserRef} UserRef formatted object that contains a
 *   user's displayName.
 */
export function displayNameToUserRef(displayName) {
  if (displayName && !RFC_2821_EMAIL_REGEX.test(displayName)) {
    throw new UserInputError(
        `Invalid email address: ${displayName}`);
  }
  return {displayName};
}

/**
 * Converts an Object into a standard UserRef Object with only a displayName
 * and userId. Used for cases when we need to use only the data required to
 * identify a unique user, such as when requesting information related to a user
 * through the API.
 *
 * @param {User} user An Object representing a user, in the JSON format
 *   returned by the pRPC API.
 * @return {UserRef} UserRef style Object.
 */
export function userToUserRef(user) {
  if (!user) return {};
  const {userId, displayName} = user;
  return {userId, displayName};
}

/**
 * Convert a UserRef style Object to a userId string.
 *
 * @param {UserRef} userRef Object expected to contain a userId key.
 * @return {Number} the unique ID of the user.
 */
export function userRefToId(userRef) {
  return userRef && userRef.userId;
}

/**
 * Extracts the displayName property from a UserRef Object.
 *
 * @param {UserRef} userRef UserRef Object uniquely identifying a user.
 * @return {String} The user's display name (email address).
 */
export function userRefToDisplayName(userRef) {
  return userRef && userRef.displayName;
}

/**
 * Converts an Array of UserRefs to an Array of display name Strings.
 *
 * @param {Array<UserRef>} userRefs Array of UserRefs.
 * @return {Array<String>} Array of display names.
 */
export function userRefsToDisplayNames(userRefs) {
  if (!userRefs) return [];
  return userRefs.map(userRefToDisplayName);
}

/**
 * Takes an Array of UserRefs and keeps only UserRefs where ID
 * is known.
 *
 * @param {Array<UserRef>} userRefs Array of UserRefs.
 * @return {Array<UserRef>} Filtered Array IDs guaranteed.
 */
export function userRefsWithIds(userRefs) {
  if (!userRefs) return [];
  return userRefs.filter((u) => u.userId);
}

/**
 * Takes an Array of UserRefs and returns displayNames for
 * only those refs with IDs specified.
 *
 * @param {Array<UserRef>} userRefs Array of UserRefs.
 * @return {Array<String>} Array of user displayNames.
 */
export function filteredUserDisplayNames(userRefs) {
  if (!userRefs) return [];
  return userRefsToDisplayNames(userRefsWithIds(userRefs));
}

/**
 * Takes in the name of a label and turns it into a LabelRef Object.
 *
 * @param {String} label The name of a label.
 * @return {LabelRef}
 */
export function labelStringToRef(label) {
  return {label};
}

/**
 * Takes in the name of a label and turns it into a LabelRef Object.
 *
 * @param {LabelRef} labelRef
 * @return {String} The name of the label.
 */
export function labelRefToString(labelRef) {
  if (!labelRef) return;
  return labelRef.label;
}

/**
 * Converts an Array of LabelRef Objects to label name Strings.
 *
 * @param {Array<LabelRef>} labelRefs Array of LabelRef Objects.
 * @return {Array<String>} Array of label names.
 */
export function labelRefsToStrings(labelRefs) {
  if (!labelRefs) return [];
  return labelRefs.map(labelRefToString);
}

/**
 * Filters a list of labels into a list of only labels with one word.
 *
 * @param {Array<LabelRef>} labelRefs
 * @return {Array<LabelRef>} only the LabelRefs that do not have multiple words.
 */
export function labelRefsToOneWordLabels(labelRefs) {
  if (!labelRefs) return [];
  return labelRefs.filter(({label}) => {
    return isOneWordLabel(label);
  });
}

/**
 * Checks whether a particular label is one word.
 *
 * @param {String} label the name of the label being checked.
 * @return {Boolean} Whether the label is one word or not.
 */
export function isOneWordLabel(label = '') {
  const words = label.split('-');
  return words.length === 1;
}

/**
 * Converts a custom field name in to the prefix format used in
 * enum type field values. Monorail defines the enum options for
 * a custom field as labels.
 *
 * @param {String} fieldName Name of a custom field.
 * @return {String} The label prefixes for enum choices
 *   associated with the field.
 */
export function fieldNameToLabelPrefix(fieldName) {
  return `${fieldName.toLowerCase()}-`;
}

/**
 * Truncates a label to include only the label's prefix, delimited
 * by '-'. Labels that share the same prefix are implciitly treated like
 * enum fields in certain parts of Monorail's UI.
 *
 * @param {String} label The name of the label.
 * @return {String} The label's prefix.
 */
export function labelNameToLabelPrefix(label) {
  if (!label) return;
  return label.split('-')[0];
}

/**
 * Extracts just the name of the status from a StatusRef Object.
 *
 * @param {StatusRef} statusRef
 * @return {String} The name of the status.
 */
export function statusRefToString(statusRef) {
  return statusRef.status;
}

/**
 * Extracts the name of multiple statuses from multiple StatusRef Objects.
 *
 * @param {Array<StatusRef>} statusRefs
 * @return {Array<String>} The names of the statuses inputted.
 */
export function statusRefsToStrings(statusRefs) {
  return statusRefs.map(statusRefToString);
}

/**
 * Takes the name of a component and converts it into a ComponentRef
 * Object.
 *
 * @param {String} path Name of the component.
 * @return {ComponentRef}
 */
export function componentStringToRef(path) {
  return {path};
}

/**
 * Extracts just the name of a component from a ComponentRef.
 *
 * @param {ComponentRef} componentRef
 * @return {String} The name of the component.
 */
export function componentRefToString(componentRef) {
  return componentRef && componentRef.path;
}

/**
 * Extracts the names of multiple components from multiple refs.
 *
 * @param {Array<ComponentRef>} componentRefs
 * @return {Array<String>} Array of component names.
 */
export function componentRefsToStrings(componentRefs) {
  if (!componentRefs) return [];
  return componentRefs.map(componentRefToString);
}

/**
 * Takes a String with a project name and issue ID in Monorail's canonical
 * IssueRef format and converts it into an IssueRef Object.
 *
 * @param {String} defaultProjectName The implied projectName if none is
 *   specified.
 * @param {IssueRefString} idStr A String of the format projectName:1234, a
 *   standard issue ID input format used across Monorail.
 * @return {IssueRef}
 * @throws {UserInputError} If the IssueRef string is invalidly formatted.
 */
export function issueStringToRef(defaultProjectName, idStr) {
  if (!idStr) return {};

  // If the string includes a slash, it's an external tracker ref.
  if (idStr.includes('/')) {
    return {extIdentifier: idStr};
  }

  const matches = idStr.match(ISSUE_ID_REGEX);
  if (!matches) {
    throw new UserInputError(
        `Invalid issue ref: ${idStr}. Expected [projectName:]issueId.`);
  }
  const projectName = matches[1] ? matches[1] : defaultProjectName;
  const localId = Number.parseInt(matches[2]);
  return {localId, projectName};
}

/**
 * Takes a String with a project name and issue ID in Monorail's canonical
 * IssueRef format and converts it into an IssueRef Object.
 *
 * @param {String} projectName The current project the user is viewing.
 * @param {Number} localId The ID of the issue the user is viewing.
 * @param {IssueRefString} idStr A String of the format projectName:1234, a
 *   standard issue ID input format used across Monorail.
 * @return {IssueRef}
 * @throws {UserInputError} If the IssueRef string is invalidly formatted
 *   or if the issue is equivalent to the linked issue.
 */
// TODO(zhangtiff): Consider simplifying this helper function to only validate
// that an issue does not block itself rather than also doing string parsing.
export function issueStringToBlockingRef(projectName, localId, idStr) {
  const result = issueStringToRef(projectName, idStr);
  if (result.projectName === projectName && result.localId === localId) {
    throw new UserInputError(
        `Invalid issue ref: ${idStr
        }. Cannot merge or block an issue on itself.`);
  }
  return result;
}

/**
 * Converts an IssueRef into a canonical String format. ie: "project:1234"
 *
 * @param {IssueRef} ref
 * @param {String} projectName The current project context. The
 *   generated String excludes the projectName if it matches the
 *   project the user is currently viewing, to create simpler
 *   issue ID links.
 * @return {IssueRefString} A String representing the pieces of an IssueRef.
 */
export function issueRefToString(ref, projectName) {
  if (!ref) return '';

  if (ref.hasOwnProperty('extIdentifier')) {
    return ref.extIdentifier;
  }

  if (projectName && projectName.length &&
      equalsIgnoreCase(ref.projectName, projectName)) {
    return `${ref.localId}`;
  }
  return `${ref.projectName}:${ref.localId}`;
}

/**
 * Converts a full Issue Object into only the pieces of its data needed
 * to define an IssueRef. Useful for cases when we don't want to send excess
 * information to ifentify an Issue.
 *
 * @param {Issue} issue A full Issue Object.
 * @return {IssueRef} Just the ID part of the Issue Object.
 */
export function issueToIssueRef(issue) {
  if (!issue) return {};

  return {localId: issue.localId,
    projectName: issue.projectName};
}

/**
 * Converts a full Issue Object into an IssueRefString
 *
 * @param {Issue} issue A full Issue Object.
 * @param {String} projectName The default project the String should assume.
 * @return {IssueRefString} A String with all the data needed to
 *   construct an IssueRef.
 */
export function issueToIssueRefString(issue, projectName) {
  if (!issue) return '';

  const ref = issueToIssueRef(issue);
  return issueRefToString(ref, projectName);
}

/**
 * Creates a link to a particular issue specified in an IssueRef.
 *
 * @param {IssueRef} ref The issue that the generated URL will point to.
 * @param {Object} queryParams The URL params for the URL.
 * @return {String} The URL for the issue's page as a relative path.
 */
export function issueRefToUrl(ref, queryParams = {}) {
  if (!ref) return '';

  if (ref.extIdentifier) {
    const extRef = fromShortlink(ref.extIdentifier);
    if (!extRef) {
      console.error(`No tracker found for reference: ${ref.extIdentifier}`);
      return '';
    }
    return extRef.toURL();
  }

  let paramString = '';
  if (Object.keys(queryParams).length) {
    delete queryParams.id;

    paramString = `&${qs.stringify(queryParams)}`;
  }

  return `/p/${ref.projectName}/issues/detail?id=${ref.localId}${paramString}`;
}

/**
 * Converts multiple IssueRef Objects into Strings in the canonical IssueRef
 * String form expeced by Monorail.
 *
 * @param {Array<IssueRef>} arr Array of IssueRefs to convert to Strings.
 * @param {String} projectName The default project name.
 * @return {Array<IssueRefString>} Array of Strings where each entry is
 *   represents one IssueRef.
 */
export function issueRefsToStrings(arr, projectName) {
  if (!arr || !arr.length) return [];
  return arr.map((ref) => issueRefToString(ref, projectName));
}

/**
 * Since Monorail stores issue descriptions and description updates as comments,
 * this function exists to filter a list of comments to get only those comments
 * that are marked as descriptions.
 *
 * @param {Array<IssueComment>} comments List of many comments, usually all
 *   comments associated with an issue.
 * @return {Array<IssueComment>} List of only the comments that are
 *   descriptions.
 */
export function commentListToDescriptionList(comments) {
  if (!comments) return [];
  // First comment is always a description, even if it doesn't have a
  // descriptionNum.
  return comments.filter((c, i) => !i || c.descriptionNum);
}

/**
 * Wraps a String value for a field and a FieldRef into a FieldValue
 * Object.
 *
 * @param {FieldRef} fieldRef A reference to the custom field that this
 *   value is tied to.
 * @param {String} value The value associated with the FieldRef.
 * @return {FieldValue}
 */
export function valueToFieldValue(fieldRef, value) {
  return {fieldRef, value};
}
