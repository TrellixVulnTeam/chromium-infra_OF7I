// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, property, internalProperty} from 'lit-element';
import React from 'react';
import ReactDOM from 'react-dom';

import {AutocompleteChangeDetails, AutocompleteChangeReason}
  from '@material-ui/core/Autocomplete';
import {createMuiTheme} from '@material-ui/core/styles';
import ThemeProvider from '@material-ui/core/styles/ThemeProvider';

import {connectStore} from 'reducers/base.js';
import * as projectV0 from 'reducers/projectV0.js';
import * as userV0 from 'reducers/userV0.js';
import {userRefsToDisplayNames} from 'shared/convertersV0.js';
import {arrayDifference} from 'shared/helpers.js';

import {ReactAutocomplete} from 'react/ReactAutocomplete.tsx';

type Vocabulary = 'component' | 'label' | 'member' | 'owner' | 'project' | '';


/**
 * A normal text input enhanced by a panel of suggested options.
 * `<mr-react-autocomplete>` wraps a React implementation of autocomplete
 * in a web component, suitable for embedding in a LitElement component
 * hierarchy. All parents must not use Shadow DOM. The supported autocomplete
 * option types are defined in type Vocabulary.
 */
export class MrReactAutocomplete extends connectStore(LitElement) {
  // Required properties passed in from the parent element.
  /** The `<input id>` attribute. Called "label" to avoid name conflicts. */
  @property() label: string = '';
  /** The autocomplete option type. See type Vocabulary for the full list. */
  @property() vocabularyName: Vocabulary = '';

  // Optional properties passed in from the parent element.
  /** The value (or values, if `multiple === true`). */
  @property() value?: string | string[] = undefined;
  /** Values that show up as disabled chips. */
  @property() fixedValues: string[] = [];
  /** True for chip input that takes multiple values, false for single input. */
  @property() multiple: boolean = false;
  /** Placeholder for the form input. */
  @property() placeholder?: string = '';
  /** Callback for input value changes. */
  @property() onChange: (
    event: React.SyntheticEvent,
    newValue: string | string[] | null,
    reason: AutocompleteChangeReason,
    details?: AutocompleteChangeDetails
  ) => void = () => {};

  // Internal state properties from the Redux store.
  @internalProperty() protected _components:
    Map<string, ComponentDef> = new Map();
  @internalProperty() protected _labels: Map<string, LabelDef> = new Map();
  @internalProperty() protected _members:
    {userRefs?: UserRef[], groupRefs?: UserRef[]} = {};
  @internalProperty() protected _projects:
    {contributorTo?: string[], memberOf?: string[], ownerOf?: string[]} = {};

  /** @override */
  createRenderRoot(): LitElement {
    return this;
  }

  /** @override */
  updated(changedProperties: Map<string | number | symbol, unknown>): void {
    super.updated(changedProperties);

    const theme = createMuiTheme({
      palette: {
        primary: {
          // Same as var(--chops-primary-accent-color).
          main: '#1976d2',
        },
      },
      typography: {fontSize: 11.375},
    });
    const element = <ThemeProvider theme={theme}>
      <ReactAutocomplete
        label={this.label}
        options={this._options()}
        value={this.value}
        fixedValues={this.fixedValues}
        multiple={this.multiple}
        placeholder={this.placeholder}
        onChange={this.onChange}
        getOptionDescription={this._getOptionDescription.bind(this)}
        getOptionLabel={(option: string) => option}
      />
    </ThemeProvider>;
    ReactDOM.render(element, this);
  }

  /** @override */
  stateChanged(state: any): void {
    super.stateChanged(state);

    this._components = projectV0.componentsMap(state);
    this._labels = projectV0.labelDefMap(state);
    this._members = projectV0.viewedVisibleMembers(state);
    this._projects = userV0.projects(state);
  }

  /**
   * Computes which description belongs to given autocomplete option.
   * Different data is shown depending on the autocomplete vocabulary.
   * @param option The option to find a description for.
   * @return The description for the option.
   */
  _getOptionDescription(option: string): string {
    switch (this.vocabularyName) {
      case 'component': {
        const component = this._components.get(option);
        return component && component.docstring || '';
      } case 'label': {
        const label = this._labels.get(option.toLowerCase());
        return label && label.docstring || '';
      } default: {
        return '';
      }
    }
  }

  /**
   * Computes the set of options used by the autocomplete instance.
   * @return Array of strings that the user can try to match.
   */
  _options(): string[] {
    switch (this.vocabularyName) {
      case 'component': {
        return [...this._components.keys()];
      } case 'label': {
        // The label map keys are lowercase. Use the LabelDef label name instead.
        return [...this._labels.values()].map((labelDef: LabelDef) => labelDef.label);
      } case 'member': {
        const {userRefs = []} = this._members;
        const users = userRefsToDisplayNames(userRefs);
        return users;
      } case 'owner': {
        const {userRefs = [], groupRefs = []} = this._members;
        const users = userRefsToDisplayNames(userRefs);
        const groups = userRefsToDisplayNames(groupRefs);
        // Remove groups from the list of all members.
        return arrayDifference(users, groups);
      } case 'project': {
        const {ownerOf = [], memberOf = [], contributorTo = []} = this._projects;
        return [...ownerOf, ...memberOf, ...contributorTo];
      } default: {
        throw new Error(`Unknown vocabulary name: ${this.vocabularyName}`);
      }
    }
  }
}
customElements.define('mr-react-autocomplete', MrReactAutocomplete);
