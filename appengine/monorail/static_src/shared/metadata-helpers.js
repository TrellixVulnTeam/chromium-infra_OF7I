// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// TODO(crbug.com/monorail/4549): Remove this hardcoded data once backend custom
// field grouping is implemented.
export const HARDCODED_FIELD_GROUPS = [
  {
    groupName: 'Feature Team',
    fieldNames: ['PM', 'Tech Lead', 'Tech-Lead', 'TechLead', 'TL',
      'Team', 'UX', 'TE'],
    applicableType: 'FLT-Launch',
  },
  {
    groupName: 'Docs',
    fieldNames: ['PRD', 'DD', 'Design Doc', 'Design-Doc',
      'DesignDoc', 'Mocks', 'Test Plan', 'Test-Plan', 'TestPlan',
      'Metrics'],
    applicableType: 'FLT-Launch',
  },
];

export const fieldGroupMap = (fieldGroupsArg, issueType) => {
  const fieldGroups = groupsForType(fieldGroupsArg, issueType);
  return fieldGroups.reduce((acc, group) => {
    return group.fieldNames.reduce((acc, fieldName) => {
      acc[fieldName] = group.groupName;
      return acc;
    }, acc);
  }, {});
};

/**
 * Get all values for a field, given an issue's fieldValueMap.
 * @param {Map.<string, Array<string>>} fieldValueMap Map where keys are
 *   lowercase fieldNames and values are fieldValue strings.
 * @param {string} fieldName The name of the field to look up.
 * @param {string=} phaseName Name of the phase the field is attached to,
 *   if applicable.
 * @return {Array<string>} The values of the field.
 */
export const valuesForField = (fieldValueMap, fieldName, phaseName) => {
  if (!fieldValueMap) return [];
  return fieldValueMap.get(
      fieldValueMapKey(fieldName, phaseName)) || [];
};

/**
 * Get just one value for a field. Convenient in some cases for
 * fields that are not multi-valued.
 * @param {Map.<string, Array<string>>} fieldValueMap Map where keys are
 *   lowercase fieldNames and values are fieldValue strings.
 * @param {string} fieldName The name of the field to look up.
 * @param {string=} phaseName Name of the phase the field is attached to,
 *   if applicable.
 * @return {string} The value of the field.
 */
export function valueForField(fieldValueMap, fieldName, phaseName) {
  const values = valuesForField(fieldValueMap, fieldName, phaseName);
  return values.length ? values[0] : undefined;
}

/**
 * Helper to generate Map keys for FieldValueMaps in a standard format.
 * @param {string} fieldName Name of the field the value is tied to.
 * @param {string=} phaseName Name of the phase the value is tied to.
 * @return {string}
 */
export const fieldValueMapKey = (fieldName, phaseName) => {
  const key = [fieldName];
  if (phaseName) {
    key.push(phaseName);
  }
  return key.join(' ').toLowerCase();
};

export const groupsForType = (fieldGroups, issueType) => {
  return fieldGroups.filter((group) => {
    if (!group.applicableType) return true;
    return issueType && group.applicableType.toLowerCase() ===
      issueType.toLowerCase();
  });
};

export const fieldDefsWithGroup = (fieldDefs, fieldGroupsArg, issueType) => {
  const fieldGroups = groupsForType(fieldGroupsArg, issueType);
  if (!fieldDefs) return [];
  const groups = [];
  fieldGroups.forEach((group) => {
    const groupFields = [];
    group.fieldNames.forEach((name) => {
      const fd = fieldDefs.find(
          (fd) => (fd.fieldRef.fieldName == name));
      if (fd) {
        groupFields.push(fd);
      }
    });
    if (groupFields.length > 0) {
      groups.push({
        groupName: group.groupName,
        fieldDefs: groupFields,
      });
    }
  });
  return groups;
};

export const fieldDefsWithoutGroup = (fieldDefs, fieldGroups, issueType) => {
  if (!fieldDefs) return [];
  const map = fieldGroupMap(fieldGroups, issueType);
  return fieldDefs.filter((fd) => {
    return !(fd.fieldRef.fieldName in map);
  });
};
