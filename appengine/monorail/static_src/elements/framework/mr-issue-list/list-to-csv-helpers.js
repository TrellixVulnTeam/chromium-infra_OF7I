// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/** @const {string} CSV download link's data href prefix, RFC 4810 Section 3 */
export const CSV_DATA_HREF_PREFIX = 'data:text/csv;charset=utf-8,';

/**
 * Format array into plaintext csv
 * @param {Array<Array>} data
 * @return {string}
 */
export const convertListContentToCsv = (data) => {
  const result = data.reduce((acc, row) => {
    return `${acc}\r\n${row.map(preventCSVInjectionAndStringify).join(',')}`;
  }, '');
  // Remove leading /r and /n
  return result.slice(2);
};

/**
 * Prevent CSV injection, escape double quotes, and wrap with double quotes
 * See owasp.org/index.php/CSV_Injection
 * @param {string} cell
 * @return {string}
 */
export const preventCSVInjectionAndStringify = (cell) => {
  // Prepend all double quotes with another double quote, RFC 4810 Section 2.7
  let escaped = cell.replace(/"/g, '""');

  // prevent CSV injection: owasp.org/index.php/CSV_Injection
  if (cell[0] === '=' ||
      cell[0] === '+' ||
      cell[0] === '-' ||
      cell[0] === '@') {
    escaped = `'${escaped}`;
  }

  // Wrap cell with double quotes, RFC 4810 Section 2.7
  return `"${escaped}"`;
};

/**
 * Prepare data for csv download by converting array of array into csv string
 * @param {Array<Array<string>>} data
 * @param {Array<string>=} headers Column headers
 * @return {string} CSV formatted string
 */
export const prepareDataForDownload = (data, headers = []) => {
  const mainContent = [headers, ...data];

  return `${convertListContentToCsv(mainContent)}`;
};

/**
 * Constructs download link url from csv string data.
 * @param {string} data CSV data
 * @return {string}
 */
export const constructHref = (data = '') => {
  return `${CSV_DATA_HREF_PREFIX}${encodeURIComponent(data)}`;
};
