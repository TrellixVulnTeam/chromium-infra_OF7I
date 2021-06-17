// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';

import {FilterOptionsState} from '@material-ui/core';
import Autocomplete, {
  AutocompleteChangeDetails, AutocompleteChangeReason,
  AutocompleteRenderGetTagProps, AutocompleteRenderInputParams,
  AutocompleteRenderOptionState,
} from '@material-ui/core/Autocomplete';
import Chip from '@material-ui/core/Chip';
import TextField from '@material-ui/core/TextField';
import {Value} from '@material-ui/core/useAutocomplete';

export const MAX_AUTOCOMPLETE_OPTIONS = 100;

interface AutocompleteProps<T> {
  label: string;
  options: T[];
  value?: Value<T, boolean, false, true>;
  fixedValues?: T[];
  multiple?: boolean;
  placeholder?: string;
  onChange?: (
    event: React.SyntheticEvent,
    value: Value<T, boolean, false, true>,
    reason: AutocompleteChangeReason,
    details?: AutocompleteChangeDetails<T>
  ) => void;
  getOptionDescription?: (option: T) => string;
  getOptionLabel?: (option: T) => string;
}

/**
 * A wrapper around Material UI Autocomplete that customizes and extends it for
 * Monorail's theme and options. Adds support for:
 * - Fixed values that render as disabled chips.
 * - Option descriptions that render alongside the option labels.
 * - Matching on word boundaries in both the labels and descriptions.
 * - Highlighting of the matching substrings.
 * @return Autocomplete instance with Monorail-specific properties set.
 */
export function ReactAutocomplete<T>(
  {
    label, options, value = undefined, fixedValues = [], multiple = false,
    placeholder = '', onChange = () => {}, getOptionDescription = () => '',
    getOptionLabel = (o) => String(o)
  }: AutocompleteProps<T>
): React.ReactNode {
  value = value || (multiple ? [] : '');

  return <Autocomplete
    id={label}
    autoHighlight
    autoSelect
    filterOptions={_filterOptions(getOptionDescription)}
    filterSelectedOptions={multiple}
    freeSolo
    getOptionLabel={getOptionLabel}
    multiple={multiple}
    onChange={_onChange(fixedValues, multiple, onChange)}
    onKeyDown={_onKeyDown}
    options={options}
    renderInput={_renderInput(placeholder)}
    renderOption={_renderOption(getOptionDescription, getOptionLabel)}
    renderTags={_renderTags(fixedValues, getOptionLabel)}
    style={{width: 'var(--mr-edit-field-width)'}}
    value={multiple ? [...fixedValues, ...value] : value}
  />;
}

/**
 * Modifies the default option matching behavior to match on all Regex word
 * boundaries and to match on both label and description.
 * @param getOptionDescription Function to get the description for an option.
 * @return The text for a given option.
 */
function _filterOptions<T>(getOptionDescription: (option: T) => string) {
  return (
    options: T[],
    {inputValue, getOptionLabel}: FilterOptionsState<T>
  ): T[] => {
    if (!inputValue.length) {
      return [];
    }
    const regex = _matchRegex(inputValue);
    const predicate = (option: T) => {
      return getOptionLabel(option).match(regex) ||
        getOptionDescription(option).match(regex);
    }
    options = options.filter(predicate).slice(0, MAX_AUTOCOMPLETE_OPTIONS);
    if (!options.includes(inputValue)) {
      // Include the option the user typed as a value, so they can select it.
      options.push(inputValue);
    }
    return options;
  }
}

/**
 * Computes an onChange handler for Autocomplete. Adds logic to make sure
 * fixedValues are preserved and wraps whatever onChange handler the parent
 * passed in.
 * @param fixedValues Values that display in the edit field but can't be
 *   edited by the user. Usually set by filter rules in Monorail.
 * @param multiple Whether this input takes multiple values or not.
 * @param onChange onChange property passed in by parent, used to sync value
 *   changes to parent.
 * @return Function that's run on Autocomplete changes.
 */
function _onChange<T, Multiple, DisableClearable, FreeSolo>(
  fixedValues: T[],
  multiple: Multiple,
  onChange: (
    event: React.SyntheticEvent,
    value: Value<T, Multiple, DisableClearable, FreeSolo>,
    reason: AutocompleteChangeReason,
    details?: AutocompleteChangeDetails<T>
  ) => void,
) {
  return (
    event: React.SyntheticEvent,
    newValue: Value<T, Multiple, DisableClearable, FreeSolo>,
    reason: AutocompleteChangeReason,
    details?: AutocompleteChangeDetails<T>
  ): void => {
    // Ensure that fixed values can't be removed.
    if (multiple) {
      newValue = newValue.filter((option: T) => !fixedValues.includes(option));
    }

    // Propagate onChange callback.
    onChange(event, newValue, reason, details);
  }
}

/**
 * Custom keydown handler.
 * @param e Keyboard event.
 */
function _onKeyDown(e: React.KeyboardEvent) {
  // Convert spaces to Enter events to allow users to type space to create new
  // chips.
  if (e.key === ' ') {
    e.key = 'Enter';
  }
}

/**
 * @param placeholder Placeholder text for the input.
 * @return A function that renders the input element used by
 *   ReactAutocomplete.
 */
function _renderInput(placeholder = ''):
    (params: AutocompleteRenderInputParams) => React.ReactNode {
  return (params: AutocompleteRenderInputParams): React.ReactNode =>
    <TextField
      {...params} variant="standard" size="small" placeholder={placeholder}
    />;
}

/**
 * Renders a single instance of an option for Autocomplete.
 * @param getOptionDescription Function to get the description text shown.
 * @param getOptionLabel Function to get the name of the option shown to the
 *   user.
 * @return ReactNode containing the JSX to be rendered.
 */
function _renderOption<T>(
  getOptionDescription: (option: T) => string,
  getOptionLabel: (option: T) => string
): React.ReactNode {
  return (
    props: React.HTMLAttributes<HTMLLIElement>,
    option: T,
    {inputValue}: AutocompleteRenderOptionState
  ): React.ReactNode => {
    // Render the option label.
    const label = getOptionLabel(option);
    const matchValue = label.match(_matchRegex(inputValue));
    let optionTemplate = <>{label}</>;
    if (matchValue) {
      // Highlight the matching text.
      optionTemplate = <>
        {matchValue[1]}
        <strong>{matchValue[2]}</strong>
        {matchValue[3]}
      </>;
    }

    // Render the option description.
    const description = getOptionDescription(option);
    const matchDescription =
      description && description.match(_matchRegex(inputValue));
    let descriptionTemplate = <>{description}</>;
    if (matchDescription) {
      // Highlight the matching text.
      descriptionTemplate = <>
        {matchDescription[1]}
        <strong>{matchDescription[2]}</strong>
        {matchDescription[3]}
      </>;
    }

    // Put the label and description together into one <li>.
    return <li
      {...props}
      className={`${props.className} autocomplete-option`}
      style={{display: 'flex', flexDirection: 'row', wordWrap: 'break-word'}}
    >
      <span style={{display: 'block', width: (description ? '40%' : '100%')}}>
        {optionTemplate}
      </span>
      {description &&
        <span style={{display: 'block', boxSizing: 'border-box',
            paddingLeft: '8px', width: '60%'}}>
          {descriptionTemplate}
        </span>
      }
    </li>;
  };
}

/**
 * Helper to render the Chips elements used by Autocomplete. Ensures that
 * fixedValues are disabled.
 * @param fixedValues Undeleteable values in an issue usually set by filter
 *   rules.
 * @param getOptionLabel Function to compute text for the option.
 * @return Function to render the ReactNode for all the chips.
 */
function _renderTags<T>(
  fixedValues: T[], getOptionLabel: (option: T) => string
) {
  return (
    value: T[],
    getTagProps: AutocompleteRenderGetTagProps
  ): React.ReactNode => {
    return value.map((option, index) => {
      const label = getOptionLabel(option);
      return <Chip
        {...getTagProps({index})}
        key={label}
        label={label}
        disabled={fixedValues.includes(option)}
        size="small"
      />;
    });
  }
}

/**
 * Generates a RegExp to match autocomplete values.
 * @param needle The string the user is searching for.
 * @return A RegExp to find matching values.
 */
function _matchRegex(needle: string): RegExp {
  // This code copied from ac.js.
  // Since we use needle to build a regular expression, we need to escape RE
  // characters. We match '-', '{', '$' and others in the needle and convert
  // them into "\-", "\{", "\$".
  const regexForRegexCharacters = /([\^*+\-\$\\\{\}\(\)\[\]\#?\.])/g;
  const modifiedPrefix = needle.replace(regexForRegexCharacters, '\\$1');

  // Match the modifiedPrefix anywhere as long as it is either at the very
  // beginning "Th" -> "The Hobbit", or comes immediately after a word separator
  // such as "Ga" -> "The-Great-Gatsby".
  const patternRegex = '^(.*\\W)?(' + modifiedPrefix + ')(.*)';
  return new RegExp(patternRegex, 'i' /* ignore case */);
}
