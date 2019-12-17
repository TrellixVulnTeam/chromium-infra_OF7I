// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import page from 'page';
import {ChopsChoiceButtons} from
  'elements/chops/chops-choice-buttons/chops-choice-buttons.js';
import {urlWithNewParams} from 'shared/helpers.js';

/**
 * Component for showing the chips to switch between List, Grid, and Chart modes
 * on the Monorail issue list page.
 * @extends {ChopsChoiceButtons}
 */
export class MrModeSelector extends ChopsChoiceButtons {
  /** @override */
  static get properties() {
    return {
      ...ChopsChoiceButtons.properties,
      queryParams: {type: Object},
      projectName: {type: String},
    };
  }

  /** @override */
  constructor() {
    super();

    this.queryParams = {};
    this.projectName = '';

    this._page = page;
  };

  /** @override */
  update(changedProperties) {
    if (changedProperties.has('queryParams') ||
        changedProperties.has('projectName')) {
      this.options = [
        {text: 'List', value: 'list', url: this._newListViewPath()},
        {text: 'Grid', value: 'grid', url: this._newListViewPath('grid')},
        {text: 'Chart', value: 'chart', url: this._newListViewPath('chart')},
      ];
    }
    super.update(changedProperties);
  }

  _newListViewPath(mode) {
    const basePath = `/p/${this.projectName}/issues/list`;
    const deletedParams = mode ? undefined : ['mode'];
    return urlWithNewParams(basePath, this.queryParams, {mode}, deletedParams);
  }
};

customElements.define('mr-mode-selector', MrModeSelector);
