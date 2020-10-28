// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/* eslint-disable no-unused-vars */

/**
 * Creates a ComponentDef.
 * @param {string} projectName The resource name of the parent project.
 * @param {string} value The name of the component
 *     e.g. "Triage" or "Triage>Security".
 * @param {string=} docstring Short description of the ComponentDef.
 * @param {Array<string>=} admins Array of User resource names to set as admins.
 * @param {Array<string>=} ccs Array of User resources names to set as auto-ccs.
 * @param {Array<string>=} labels Array of labels.
 * @return {ComponentDef}
 */
function createComponentDef(
    projectName, value, docstring, admins, ccs, labels) {
  const componentDef = {
    'value': value,
    'docstring': docstring,
  };
  if (admins) {
    componentDef['admins'] = admins;
  }
  if (ccs) {
    componentDef['ccs'] = ccs;
  }
  if (labels) {
    componentDef['labels'] = labels;
  }
  const message = {
    'parent': projectName,
    'componentDef': componentDef,
  };
  const url = URL + 'monorail.v3.Projects/CreateComponentDef';
  return run_(url, message);
}

/**
 * Deletes a ComponentDef.
 * @param {string} componentName Resource name of the ComponentDef to delete.
 * @return {EmptyProto}
 */
function deleteComponentDef(componentName) {
  const message = {
    'name': componentName,
  };
  const url = URL + 'monorail.v3.Projects/DeleteComponentDef';
  return run_(url, message);
}
