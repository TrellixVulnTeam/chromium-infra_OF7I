// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/** @const {string} CSV download link's data href prefix, RFC 4810 Section 3 */
export const CSV_DATA_HREF_PREFIX = 'data:text/csv;charset=utf-8,';

/**
 * CSV files are at risk for the PDF content sniffing by Acrobat Reader.
 * Prefix with over 1024 bytes of static content to avoid content sniffing.
 * @const {string}
 */
export const SNIFF_PREVENTION = '-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-='; // eslint-disable-line max-len

/**
 * Format array into plaintext csv
 * @param {Array<Array>} data
 * @return {string}
 */
export const convertListContentToCsv = (data) => {
  return data.reduce((acc, row) => {
    return `${acc}\n${row.map(preventCSVInjectionAndStringify).join(',')}`;
  }, '');
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
 * Prepare data for csv download:
 *  - Prepend sniffing prevention
 *  - Optionally prepend instructions on how to modify csv content
 *  - Convert array of array into csv format
 * @param {Array<Array<string>>} data
 * @param {Array<string>=} headers Column headers
 * @param {string=} prefix
 * @return {string} CSV formatted string
 */
export const prepareDataForDownload = (data, headers = [], prefix = '') => {
  const mainContent = [headers, ...data];

  return `${SNIFF_PREVENTION}${prefix && ('\n' + prefix)}` +
      convertListContentToCsv(mainContent);
};

/**
 * Constructs download link url from csv string data.
 * @param {string} data CSV data
 * @return {string}
 */
export const constructHref = (data = '') => {
  return `${CSV_DATA_HREF_PREFIX}${encodeURIComponent(data)}`;
};
