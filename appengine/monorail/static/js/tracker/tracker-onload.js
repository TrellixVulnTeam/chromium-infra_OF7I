/* Copyright 2016 The Chromium Authors. All Rights Reserved.
 *
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file or at
 * https://developers.google.com/open-source/licenses/bsd
 */
/* eslint-disable camelcase */
/* eslint-disable no-unused-vars */


/**
 * This file contains the Monorail onload() function that is called
 * when each EZT page loads.
 */


/**
 * This code is run on every DIT page load.  It registers a handler
 * for autocomplete on four different types of text fields based on the
 * name of that text field.
 */
function TKR_onload() {
  TKR_install_ac();
  _PC_Install();
  TKR_allColumnNames = _allColumnNames;
  TKR_labelFieldIDPrefix = _lfidprefix;
  TKR_allOrigLabels = _allOrigLabels;
  TKR_initialFormValues = TKR_currentFormValues();
}

// External names for functions that are called directly from HTML.
// JSCompiler does not rename functions that begin with an underscore.
// They are not defined with "var" because we want them to be global.

// TODO(jrobbins): the underscore names could be shortened by a
// cross-file search-and-replace script in our build process.

_selectAllIssues = TKR_selectAllIssues;
_selectNoneIssues = TKR_selectNoneIssues;

_toggleRows = TKR_toggleRows;
_toggleColumn = TKR_toggleColumn;
_toggleColumnUpdate = TKR_toggleColumnUpdate;
_addGroupBy = TKR_addGroupBy;
_addcol = TKR_addColumn;
_checkRangeSelect = TKR_checkRangeSelect;
_makeIssueLink = TKR_makeIssueLink;

_onload = TKR_onload;

_handleListActions = TKR_handleListActions;
_handleDetailActions = TKR_handleDetailActions;

_loadStatusSelect = TKR_loadStatusSelect;
_fetchUserProjects = TKR_fetchUserProjects;
_setACOptions = TKR_setUpAutoCompleteStore;
_openIssueUpdateForm = TKR_openIssueUpdateForm;
_addAttachmentFields = TKR_addAttachmentFields;
_ignoreWidgetIfOpIsClear = TKR_ignoreWidgetIfOpIsClear;

_acstore = _AC_SimpleStore;
_accomp = _AC_Completion;
_acreg = _ac_register;

_formatContextQueryArgs = TKR_formatContextQueryArgs;
_ctxArgs = '';
_ctxCan = undefined;
_ctxQuery = undefined;
_ctxSortspec = undefined;
_ctxGroupBy = undefined;
_ctxDefaultColspec = undefined;
_ctxStart = undefined;
_ctxNum = undefined;
_ctxResultsPerPage = undefined;

_filterTo = TKR_filterTo;
_sortUp = TKR_sortUp;
_sortDown = TKR_sortDown;

_closeAllPopups = TKR_closeAllPopups;
_closeSubmenus = TKR_closeSubmenus;
_showRight = TKR_showRight;
_showBelow = TKR_showBelow;
_highlightRow = TKR_highlightRow;

_setFieldIDs = TKR_setFieldIDs;
_selectTemplate = TKR_selectTemplate;
_saveTemplate = TKR_saveTemplate;
_newTemplate = TKR_newTemplate;
_deleteTemplate = TKR_deleteTemplate;
_switchTemplate = TKR_switchTemplate;
_templateNames = TKR_templateNames;

_confirmNovelStatus = TKR_confirmNovelStatus;
_confirmNovelLabel = TKR_confirmNovelLabel;
_vallab = TKR_validateLabel;
_exposeExistingLabelFields = TKR_exposeExistingLabelFields;
_confirmDiscardEntry = TKR_confirmDiscardEntry;
_confirmDiscardUpdate = TKR_confirmDiscardUpdate;
_lfidprefix = undefined;
_allOrigLabels = undefined;
_checkPlusOne = TKR_checkPlusOne;
_checkUnrestrict = TKR_checkUnrestrict;

_clearOnFirstEvent = TKR_clearOnFirstEvent;
_forceProperTableWidth = TKR_forceProperTableWidth;

_initialFormValues = TKR_initialFormValues;
_currentFormValues = TKR_currentFormValues;

_acof = _ac_onfocus;
_acmo = _ac_mouseover;
_acse = _ac_select;
_acrob = _ac_ob;

// Variables that are given values in the HTML file.
_allColumnNames = [];

_go = TKR_go;
_getColspec = TKR_getColspecElement;

// Make the document actually listen for click events, otherwise the
// event handlers above would never get called.
if (document.captureEvents) document.captureEvents(Event.CLICK);

_setupKibblesOnEntryPage = TKR_setupKibblesOnEntryPage;
_setupKibblesOnListPage = TKR_setupKibblesOnListPage;

_checkFieldNameOnServer = TKR_checkFieldNameOnServer;
_checkLeafName = TKR_checkLeafName;

_addMultiFieldValueWidget = TKR_addMultiFieldValueWidget;
_removeMultiFieldValueWidget = TKR_removeMultiFieldValueWidget;
_trimCommas = TKR_trimCommas;

_initDragAndDrop = TKR_initDragAndDrop;
