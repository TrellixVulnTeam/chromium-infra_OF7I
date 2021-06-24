/* Copyright 2019 The Chromium Authors. All rights reserved.
 * Use of this source code is governed by a BSD-style license that can be
 * found in the LICENSE file.
 */

/**
 * Form submission function for switching platforms.
 *
 * Allows for #platform_select to optionally populate
 * two fields of the form if '#' is present in the value.
 */
function switchPlatform() { // eslint-disable-line no-unused-vars
  const form = document.getElementById('platform_select_form');
  const select = document.getElementById('platform_select');
  const option = select.options[select.selectedIndex];
  if ('revision' in option) {
    const revisionField = document.createElement('input');
    revisionField.setAttribute('type', 'hidden');
    revisionField.setAttribute('name', 'revision');
    revisionField.setAttribute('value', option.revision);
    form.appendChild(revisionField);
  }
  form.submit();
}
