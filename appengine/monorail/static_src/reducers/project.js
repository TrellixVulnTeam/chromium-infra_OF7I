// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {combineReducers} from 'redux';
import {createSelector} from 'reselect';
import {createReducer, createRequestReducer} from './redux-helpers.js';
import {fieldTypes, SITEWIDE_DEFAULT_COLUMNS, defaultIssueFieldMap,
  parseColSpec, stringValuesForIssueField} from 'shared/issue-fields.js';
import {hasPrefix, removePrefix} from 'shared/helpers.js';
import {fieldNameToLabelPrefix,
  labelNameToLabelPrefix} from 'shared/converters.js';
import {prpcClient} from 'prpc-client-instance.js';
import 'shared/typedef.js';

// Actions
export const SELECT = 'project/SELECT';

const FETCH_CONFIG_START = 'project/FETCH_CONFIG_START';
export const FETCH_CONFIG_SUCCESS = 'project/FETCH_CONFIG_SUCCESS';
const FETCH_CONFIG_FAILURE = 'project/FETCH_CONFIG_FAILURE';

export const FETCH_PRESENTATION_CONFIG_START =
  'project/FETCH_PRESENTATION_CONFIG_START';
export const FETCH_PRESENTATION_CONFIG_SUCCESS =
  'project/FETCH_PRESENTATION_CONFIG_SUCCESS';
export const FETCH_PRESENTATION_CONFIG_FAILURE =
  'project/FETCH_PRESENTATION_CONFIG_FAILURE';

export const FETCH_VISIBLE_MEMBERS_START =
  'project/FETCH_VISIBLE_MEMBERS_START';
export const FETCH_VISIBLE_MEMBERS_SUCCESS =
  'project/FETCH_VISIBLE_MEMBERS_SUCCESS';
export const FETCH_VISIBLE_MEMBERS_FAILURE =
  'project/FETCH_VISIBLE_MEMBERS_FAILURE';

const FETCH_TEMPLATES_START = 'project/FETCH_TEMPLATES_START';
export const FETCH_TEMPLATES_SUCCESS = 'project/FETCH_TEMPLATES_SUCCESS';
const FETCH_TEMPLATES_FAILURE = 'project/FETCH_TEMPLATES_FAILURE';

const FETCH_FIELDS_LIST_START = 'project/FETCH_FIELDS_LIST_START';
export const FETCH_FIELDS_LIST_SUCCESS = 'project/FETCH_FIELDS_LIST_SUCCESS';
const FETCH_FIELDS_LIST_FAILURE = 'project/FECTH_FIELDS_LIST_FAILURE';

/* State Shape
{
  name: string,

  configs: Object.<string, Config>,
  presentationConfigs: Object.<string, PresentationConfig>,
  visibleMembers:
      Object.<string, {userRefs: Array<UserRef>, groupRefs: Array<UserRef>}>,
  templates: Object.<string, Array<TemplateDef>>,

  requests: {
    fetchConfig: ReduxRequestState,
    fetchFields: ReduxRequestState,
    fetchMembers: ReduxRequestState,
    fetchTemplates: ReduxRequestState,
  },
}
*/

// Reducers
export const nameReducer = createReducer(null, {
  [SELECT]: (_state, {projectName}) => projectName,
});

export const configsReducer = createReducer({}, {
  [FETCH_CONFIG_SUCCESS]: (state, {projectName, config}) => ({
    ...state,
    [projectName]: config,
  }),
  [FETCH_FIELDS_LIST_SUCCESS]: (state, {projectName, fieldDefs}) => ({
    ...state,
    [projectName]: {
      ...state[projectName],
      fieldDefs: fieldDefs,
    },
  }),
});

export const presentationConfigsReducer = createReducer({}, {
  [FETCH_PRESENTATION_CONFIG_SUCCESS]:
    (state, {projectName, presentationConfig}) => ({
      ...state,
      [projectName]: presentationConfig,
    }),
});

export const visibleMembersReducer = createReducer({}, {
  [FETCH_VISIBLE_MEMBERS_SUCCESS]: (state, {projectName, visibleMembers}) => ({
    ...state,
    [projectName]: visibleMembers,
  }),
});

export const templatesReducer = createReducer({}, {
  [FETCH_TEMPLATES_SUCCESS]: (state, {projectName, templates}) => ({
    ...state,
    [projectName]: templates,
  }),
});

const requestsReducer = combineReducers({
  fetchConfig: createRequestReducer(
      FETCH_CONFIG_START, FETCH_CONFIG_SUCCESS, FETCH_CONFIG_FAILURE),
  fetchMembers: createRequestReducer(
      FETCH_VISIBLE_MEMBERS_START,
      FETCH_VISIBLE_MEMBERS_SUCCESS,
      FETCH_VISIBLE_MEMBERS_FAILURE),
  fetchTemplates: createRequestReducer(
      FETCH_TEMPLATES_START, FETCH_TEMPLATES_SUCCESS, FETCH_TEMPLATES_FAILURE),
  fetchFields: createRequestReducer(
      FETCH_FIELDS_LIST_START,
      FETCH_FIELDS_LIST_SUCCESS,
      FETCH_FIELDS_LIST_FAILURE),
});

export const reducer = combineReducers({
  name: nameReducer,
  configs: configsReducer,
  presentationConfigs: presentationConfigsReducer,
  visibleMembers: visibleMembersReducer,
  templates: templatesReducer,
  requests: requestsReducer,
});

// Selectors
export const project = (state) => state.project || {};

export const viewedProjectName =
  createSelector(project, (project) => project.name || null);

export const configs =
  createSelector(project, (project) => project.configs || {});
export const presentationConfigs =
  createSelector(project, (project) => project.presentationConfigs || {});
export const visibleMembers =
  createSelector(project, (project) => project.visibleMembers || {});
export const templates =
  createSelector(project, (project) => project.templates || {});

export const viewedConfig = createSelector(
    [viewedProjectName, configs],
    (projectName, configs) => configs[projectName] || {});
export const viewedPresentationConfig = createSelector(
    [viewedProjectName, presentationConfigs],
    (projectName, configs) => configs[projectName] || {});
export const viewedVisibleMembers = createSelector(
    [viewedProjectName, visibleMembers],
    (projectName, visibleMembers) => visibleMembers[projectName] || {});
export const viewedTemplates = createSelector(
    [viewedProjectName, templates],
    (projectName, templates) => templates[projectName] || []);

/**
 * Get the default columns for the currently viewed project.
 */
export const defaultColumns = createSelector(viewedPresentationConfig,
    ({defaultColSpec}) =>{
      if (defaultColSpec) {
        return parseColSpec(defaultColSpec);
      }
      return SITEWIDE_DEFAULT_COLUMNS;
    });


/**
 * Get the default query for the currently viewed project.
 */
export const defaultQuery = createSelector(viewedPresentationConfig,
    (config) => config.defaultQuery || '');

// Look up components by path.
export const componentsMap = createSelector(
    viewedConfig,
    (config) => {
      if (!config || !config.componentDefs) return new Map();
      const acc = new Map();
      for (const v of config.componentDefs) {
        acc.set(v.path, v);
      }
      return acc;
    },
);

export const fieldDefs = createSelector(
    viewedConfig, (config) => ((config && config.fieldDefs) || []),
);

export const fieldDefMap = createSelector(
    fieldDefs, (fieldDefs) => {
      const map = new Map();
      fieldDefs.forEach((fd) => {
        map.set(fd.fieldRef.fieldName.toLowerCase(), fd);
      });
      return map;
    },
);

export const labelDefs = createSelector(
    viewedConfig, (config) => ((config && config.labelDefs) || []),
);

// labelDefs stored in an easily findable format with label names as keys.
export const labelDefMap = createSelector(
    labelDefs, (labelDefs) => {
      const map = new Map();
      labelDefs.forEach((ld) => {
        map.set(ld.label.toLowerCase(), ld);
      });
      return map;
    },
);

// Find the options that exist for a given label prefix.
export const labelPrefixOptions = createSelector(
    labelDefs, (labelDefs) => {
      const prefixMap = new Map();
      labelDefs.forEach((ld) => {
        const prefix = labelNameToLabelPrefix(ld.label).toLowerCase();

        if (prefixMap.has(prefix)) {
          prefixMap.get(prefix).push(ld.label);
        } else {
          prefixMap.set(prefix, [ld.label]);
        }
      });

      return prefixMap;
    },
);

// Some labels are implicitly used as custom fields in the grid and list view.
// Make this an Array to keep casing in tact. Only labels with more than one
// option are included, to reduce noise.
export const labelPrefixFields = createSelector(
    labelPrefixOptions, (map) => {
      const prefixes = [];

      map.forEach((options) => {
      // Ignore label prefixes with only one value.
        if (options.length > 1) {
        // Pick the first label defined to set the casing for the prefix value.
        // This shouldn't be too important of a decision because most labels
        // with shared prefixes should use the same casing across labels.
          prefixes.push(labelNameToLabelPrefix(options[0]));
        }
      });

      return prefixes;
    },
);

// Wrap label prefixes in a Set for fast lookup.
export const labelPrefixSet = createSelector(
    labelPrefixFields, (fields) => new Set(fields.map(
        (field) => field.toLowerCase())),
);

export const enumFieldDefs = createSelector(
    fieldDefs,
    (fieldDefs) => {
      return fieldDefs.filter(
          (fd) => fd.fieldRef.type === fieldTypes.ENUM_TYPE);
    },
);

/**
 * A selector that builds a function that's used to compute the value of
 * a given field name on a given issue. This function abstracts the difference
 * between custom fields, built-in fields, and implicit fields created
 * from labels and considers these values in the context of the current
 * project configuration.
 * @param {Object} state Current Redux state.
 * @return {function(Issue, string): Array<string>} A function that processes a
 *   given issue and field name to find the string value for that field, in
 *   the issue.
 */
export const extractFieldValuesFromIssue = createSelector(viewedProjectName,
    labelPrefixSet, fieldDefMap,
    (projectName, labelPrefixSet, fieldDefMap) => {
      return (issue, fieldName) => stringValuesForIssueField(issue, fieldName,
          projectName, fieldDefMap, labelPrefixSet);
    },
);

/**
 * A selector that builds a function that's used to compute the type of a given
 * field name.
 * @param {Object} state Current Redux state.
 * @return {function(string): string}
 */
export const extractTypeForFieldName = createSelector(fieldDefMap,
    (fieldDefMap) => {
      return (fieldName) => {
        const key = fieldName.toLowerCase();

        // If the field is a built in field. Default fields have precedence
        // over custom fields.
        if (defaultIssueFieldMap.hasOwnProperty(key)) {
          return defaultIssueFieldMap[key].type;
        }

        // If the field is a custom field. Custom fields have precedence
        // over label prefixes.
        if (fieldDefMap.has(key)) {
          return fieldDefMap.get(key).fieldRef.type;
        }

        // Default to STR_TYPE, including for label fields.
        return fieldTypes.STR_TYPE;
      };
    },
);

export const optionsPerEnumField = createSelector(
    enumFieldDefs,
    labelDefs,
    (fieldDefs, labelDefs) => {
      const map = new Map(fieldDefs.map(
          (fd) => [fd.fieldRef.fieldName.toLowerCase(), []]));
      labelDefs.forEach((ld) => {
        const labelName = ld.label;

        const fd = fieldDefs.find((fd) => hasPrefix(
            labelName, fieldNameToLabelPrefix(fd.fieldRef.fieldName)));
        if (fd) {
          const key = fd.fieldRef.fieldName.toLowerCase();
          map.get(key).push({
            ...ld,
            optionName: removePrefix(labelName,
                fieldNameToLabelPrefix(fd.fieldRef.fieldName)),
          });
        }
      });
      return map;
    },
);

export const fieldDefsForPhases = createSelector(
    fieldDefs,
    (fieldDefs) => {
      if (!fieldDefs) return [];
      return fieldDefs.filter((f) => f.isPhaseField);
    },
);

export const fieldDefsByApprovalName = createSelector(
    fieldDefs,
    (fieldDefs) => {
      if (!fieldDefs) return new Map();
      const acc = new Map();
      for (const fd of fieldDefs) {
        if (fd.fieldRef && fd.fieldRef.approvalName) {
          if (acc.has(fd.fieldRef.approvalName)) {
            acc.get(fd.fieldRef.approvalName).push(fd);
          } else {
            acc.set(fd.fieldRef.approvalName, [fd]);
          }
        }
      }
      return acc;
    },
);

export const fetchingConfig = (state) => {
  return state.project.requests.fetchConfig.requesting;
};

// Action Creators
/**
 * Action creator to set the currently viewed Project.
 * @param {string} projectName The name of the Project to select.
 * @return {function(function): void}
 */
export const select = (projectName) => {
  return (dispatch) => dispatch({type: SELECT, projectName});
};

export const fetch = (projectName) => async (dispatch) => {
  dispatch(fetchConfig(projectName));
  // TODO(zhangtiff): Split up GetConfig into multiple calls to
  // GetLabelOptions, ListComponents, etc.
  // dispatch(fetchFields(projectName));
  dispatch(fetchPresentationConfig(projectName));
  dispatch(fetchVisibleMembers(projectName));
  dispatch(fetchTemplates(projectName));
};

const fetchConfig = (projectName) => async (dispatch) => {
  dispatch({type: FETCH_CONFIG_START});

  const getConfig = prpcClient.call(
      'monorail.Projects', 'GetConfig', {projectName});

  try {
    const resp = await getConfig;
    dispatch({type: FETCH_CONFIG_SUCCESS, projectName, config: resp});
  } catch (error) {
    dispatch({type: FETCH_CONFIG_FAILURE, error});
  }
};

export const fetchPresentationConfig = (projectName) => async (dispatch) => {
  dispatch({type: FETCH_PRESENTATION_CONFIG_START});

  try {
    const presentationConfig = await prpcClient.call(
        'monorail.Projects', 'GetPresentationConfig', {projectName});
    dispatch({
      type: FETCH_PRESENTATION_CONFIG_SUCCESS,
      projectName,
      presentationConfig,
    });
  } catch (error) {
    dispatch({type: FETCH_PRESENTATION_CONFIG_FAILURE, error});
  }
};

export const fetchVisibleMembers = (projectName) => async (dispatch) => {
  dispatch({type: FETCH_VISIBLE_MEMBERS_START});

  try {
    const visibleMembers = await prpcClient.call(
        'monorail.Projects', 'GetVisibleMembers', {projectName});
    dispatch({
      type: FETCH_VISIBLE_MEMBERS_SUCCESS,
      projectName,
      visibleMembers,
    });
  } catch (error) {
    dispatch({type: FETCH_VISIBLE_MEMBERS_FAILURE, error});
  }
};

const fetchTemplates = (projectName) => async (dispatch) => {
  dispatch({type: FETCH_TEMPLATES_START});

  const listTemplates = prpcClient.call(
      'monorail.Projects', 'ListProjectTemplates', {projectName});

  // TODO(zhangtiff): Remove (see above TODO).
  if (!listTemplates) return;

  try {
    const resp = await listTemplates;
    dispatch({
      type: FETCH_TEMPLATES_SUCCESS,
      projectName,
      templates: resp.templates,
    });
  } catch (error) {
    dispatch({type: FETCH_TEMPLATES_FAILURE, error});
  }
};

export const fetchFields = (projectName) => async (dispatch) => {
  dispatch({type: FETCH_FIELDS_LIST_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Projects', 'ListFields', {
          projectName: projectName,
          includeUserChoices: true,
        },
    );
    const fieldDefs = (resp.fieldDefs || []);
    dispatch({type: FETCH_FIELDS_LIST_SUCCESS, projectName, fieldDefs});
  } catch (error) {
    dispatch({type: FETCH_FIELDS_LIST_FAILURE});
  }
};
