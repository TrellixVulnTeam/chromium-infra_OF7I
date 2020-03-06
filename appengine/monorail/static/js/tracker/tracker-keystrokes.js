/* Copyright 2016 The Chromium Authors. All Rights Reserved.
 *
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file or at
 * https://developers.google.com/open-source/licenses/bsd
 */

/**
 * This file contains JS functions that implement keystroke accelerators
 * for Monorail.
 */

/**
 * Array of HTML elements where the kibbles cursor can be.  E.g.,
 * the TR elements of an issue list, or the TR's for comments on an issue.
 */
let TKR_cursorStops;

/**
 * Integer index into TKR_cursorStops of the currently selected cursor
 * stop, or undefined if nothing has been selected yet.
 */
let TKR_selected = undefined;

/**
 * Register keystrokes that apply to all pages in the current component.
 * E.g., keystrokes that should work on every page under the "Issues" tab.
 * @param {string} listUrl Rooted URL of the artifact list.
 * @param {string} entryUrl Rooted URL of the artifact entry page.
 * @param {string} currentPageType One of 'list', 'entry', or 'detail'.
 */
function TKR_setupKibblesComponentKeys(listUrl, entryUrl, currentPageType) {
  if (currentPageType != 'list') {
    kibbles.keys.addKeyPressListener(
        'u', function() {
          TKR_go(listUrl);
        });
  }
}


/**
 * On the artifact list page, go to the artifact at the kibbles cursor.
 * @param {number} linkCellIndex row child that is expected to hold a link.
 */
function TKR_openArtifactAtCursor(linkCellIndex, newWindow) {
  if (TKR_selected >= 0 && TKR_selected < TKR_cursorStops.length) {
    window._goIssue(TKR_selected, newWindow);
  }
}


/**
 * On the artifact list page, toggle the checkbox for the artifact at
 * the kibbles cursor.
 * @param {number} cbCellIndex row child that is expected to hold a checkbox.
 */
function TKR_selectArtifactAtCursor(cbCellIndex) {
  if (TKR_selected >= 0 && TKR_selected < TKR_cursorStops.length) {
    const cell = TKR_cursorStops[TKR_selected].children[cbCellIndex];
    let cb = cell.firstChild;
    while (cb && cb.tagName != 'INPUT') {
      cb = cb.nextSibling;
    }
    if (cb) {
      cb.checked = cb.checked ? '' : 'checked';
      TKR_highlightRow(cb);
    }
  }
}

/**
 * On the artifact list page, toggle the star for the artifact at
 * the kibbles cursor.
 * @param {number} cbCellIndex row child that is expected to hold a checkbox
 *     and star widget.
 */
function TKR_toggleStarArtifactAtCursor(cbCellIndex) {
  if (TKR_selected >= 0 && TKR_selected < TKR_cursorStops.length) {
    const cell = TKR_cursorStops[TKR_selected].children[cbCellIndex];
    let starIcon = cell.firstChild;
    while (starIcon && starIcon.tagName != 'A') {
      starIcon = starIcon.nextSibling;
    }
    if (starIcon) {
      _TKR_toggleStar(
          starIcon, issueRefs[TKR_selected]['project_name'],
          issueRefs[TKR_selected]['id'], null, null);
    }
  }
}

/**
 * Updates the style on new stop and clears the style on the former stop.
 * @param {Object} newStop the cursor stop that the user is selecting now.
 * @param {Object} formerStop the old cursor stop, if any.
 */
function TKR_updateCursor(newStop, formerStop) {
  TKR_selected = undefined;
  if (formerStop) {
    formerStop.element.classList.remove('cursor_on');
    formerStop.element.classList.add('cursor_off');
  }
  if (newStop && newStop.element) {
    newStop.element.classList.remove('cursor_off');
    newStop.element.classList.add('cursor_on');
    TKR_selected = newStop.index;
  }
}


/**
 * Walk part of the page DOM to find elements that should be kibbles
 * cursor stops.  E.g., the rows of the issue list results table.
 * @return {Array} an array of html elements.
 */
function TKR_findCursorRows() {
  const rows = [];
  const cursorarea = document.getElementById('cursorarea');
  TKR_accumulateCursorRows(cursorarea, rows);
  return rows;
}


/**
 * Recusrively walk part of the page DOM to find elements that should
 * be kibbles cursor stops.  E.g., the rows of the issue list results
 * table.  The cursor stops are appended to the given rows array.
 * @param {Element} parent html element to start on.
 * @param {Array} rows  array of html TR or DIV elements, each cursor stop will
 *    be added to this array.
 */
function TKR_accumulateCursorRows(parent, rows) {
  for (let i = 0; i < parent.childNodes.length; i++) {
    const elem = parent.childNodes[i];
    const name = elem.tagName;
    if (name && (name == 'TR' || name == 'DIV')) {
      if (elem.className.indexOf('cursor') >= 0) {
        elem.cursorIndex = rows.length;
        rows.push(elem);
      }
    }
    TKR_accumulateCursorRows(elem, rows);
  }
}


/**
 * Initialize kibbles cursors stops for the current page.
 * @param {boolean} selectFirstStop True if the first stop should be
 *   selected before the user presses any keys.
 */
function TKR_setupKibblesCursorStops(selectFirstStop) {
  kibbles.skipper.addStopListener(
      kibbles.skipper.LISTENER_TYPE.PRE, TKR_updateCursor);

  // Set the 'offset' option to return the middle of the client area
  // an option can be a static value, or a callback
  kibbles.skipper.setOption('padding_top', 50);

  // Set the 'offset' option to return the middle of the client area
  // an option can be a static value, or a callback
  kibbles.skipper.setOption('padding_bottom', 50);

  // register our stops with skipper
  TKR_cursorStops = TKR_findCursorRows();
  for (let i = 0; i < TKR_cursorStops.length; i++) {
    const element = TKR_cursorStops[i];
    kibbles.skipper.append(element);

    if (element.className.indexOf('cursor_on') >= 0) {
      kibbles.skipper.setCurrentStop(i);
    }
  }
}


/**
 * Initialize kibbles keystrokes for an artifact entry page.
 * @param {string} listUrl Rooted URL of the artifact list.
 * @param {string} entryUrl Rooted URL of the artifact entry page.
 */
function TKR_setupKibblesOnEntryPage(listUrl, entryUrl) {
  TKR_setupKibblesComponentKeys(listUrl, entryUrl, 'entry');
}


/**
 * Initialize kibbles keystrokes for an artifact list page.
 * @param {string} listUrl Rooted URL of the artifact list.
 * @param {string} entryUrl Rooted URL of the artifact entry page.
 * @param {string} projectName Name of the current project.
 * @param {number} linkCellIndex table column that is expected to
 *   link to individual artifacts.
 * @param {number} opt_checkboxCellIndex table column that is expected
 *   to contain a selection checkbox.
 */
function TKR_setupKibblesOnListPage(
    listUrl, entryUrl, projectName, linkCellIndex,
    opt_checkboxCellIndex) {
  TKR_setupKibblesCursorStops(true);

  kibbles.skipper.addFwdKey('j');
  kibbles.skipper.addRevKey('k');

  if (opt_checkboxCellIndex != undefined) {
    const cbCellIndex = opt_checkboxCellIndex;
    kibbles.keys.addKeyPressListener(
        'x', function() {
          TKR_selectArtifactAtCursor(cbCellIndex);
        });
    kibbles.keys.addKeyPressListener(
        's',
        function() {
          TKR_toggleStarArtifactAtCursor(cbCellIndex);
        });
  }
  kibbles.keys.addKeyPressListener(
      'o', function() {
        TKR_openArtifactAtCursor(linkCellIndex, false);
      });
  kibbles.keys.addKeyPressListener(
      'O', function() {
        TKR_openArtifactAtCursor(linkCellIndex, true);
      });
  kibbles.keys.addKeyPressListener(
      'enter', function() {
        TKR_openArtifactAtCursor(linkCellIndex);
      });

  TKR_setupKibblesComponentKeys(listUrl, entryUrl, 'list');
}
