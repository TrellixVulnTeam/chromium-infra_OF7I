/* Copyright 2016 The Chromium Authors. All Rights Reserved.
 *
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file or at
 * https://developers.google.com/open-source/licenses/bsd
 */
/* eslint-disable camelcase */
/* eslint-disable no-unused-vars */

/**
  * Sets up the legacy autocomplete editing widget on DOM elements that are
  * set to use it.
  */
function TKR_install_ac() {
  _ac_install();

  _ac_register(function(input, event) {
    if (input.id.startsWith('hotlists')) return TKR_hotlistsStore;
    if (input.id.startsWith('search')) return TKR_searchStore;
    if (input.id.startsWith('query_') || input.id.startsWith('predicate_')) {
      return TKR_projectQueryStore;
    }
    if (input.id.startsWith('cmd')) return TKR_quickEditStore;
    if (input.id.startsWith('labelPrefix')) return TKR_labelPrefixStore;
    if (input.id.startsWith('label')) return TKR_labelStore;
    if (input.dataset.acType === 'label') return TKR_labelMultiStore;
    if (input.id.startsWith('component') || input.dataset.acType === 'component') return TKR_componentListStore;
    if (input.id.startsWith('status')) return TKR_statusStore;
    if (input.id.startsWith('member') || input.dataset.acType === 'member') return TKR_memberListStore;

    if (input.id == 'admin_names_editor') return TKR_memberListStore;
    if (input.id.startsWith('owner')) return TKR_ownerStore;
    if (input.name == 'needs_perm' || input.name == 'grants_perm') {
      return TKR_customPermissionsStore;
    }
    if (input.id == 'owner_editor' || input.dataset.acType === 'owner') return TKR_ownerStore;
    if (input.className.indexOf('userautocomplete') != -1) {
      const customFieldIDStr = input.name;
      const uac = TKR_userAutocompleteStores[customFieldIDStr];
      if (uac) return uac;
      return TKR_ownerStore;
    }
    if (input.className.indexOf('autocomplete') != -1) {
      return TKR_autoCompleteStore;
    }
    if (input.id.startsWith('copy_to') || input.id.startsWith('move_to') ||
       input.id.startsWith('new_savedquery_projects') ||
       input.id.startsWith('savedquery_projects')) {
      return TKR_projectStore;
    }
  });
};
