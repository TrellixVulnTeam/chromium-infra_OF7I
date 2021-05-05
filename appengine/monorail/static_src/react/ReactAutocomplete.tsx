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

interface AutocompleteProps<T> {
  label: string;
  options: T[];
  value?: Value<T, boolean, false, true>;
  fixedValues?: T[];
  multiple?: boolean;
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
 */
export function ReactAutocomplete<T>(
  {
    label, options, value = undefined, fixedValues = [], multiple = false,
    onChange = () => {}, getOptionDescription = () => '',
    getOptionLabel = (o) => String(o)
  }: AutocompleteProps<T>
): React.ReactNode {
  value = value || (multiple ? [] : '');
  return <Autocomplete
    id={label}
    autoHighlight
    filterOptions={_filterOptions(getOptionDescription)}
    filterSelectedOptions
    freeSolo
    getOptionLabel={getOptionLabel}
    multiple={multiple}
    onChange={_onChange(fixedValues, multiple, onChange)}
    options={options}
    renderInput={_renderInput()}
    renderOption={_renderOption(getOptionDescription, getOptionLabel)}
    renderTags={_renderTags(fixedValues, getOptionLabel)}
    style={{width: 'var(--mr-edit-field-width)'}}
    value={multiple ? [...fixedValues, ...value] : value}
  />;
}

/**
 * Modifies the default option matching behavior to match on all Regex word
 * boundaries and to match on both label and description.
 * @param getOptionDescription
 */
function _filterOptions<T>(getOptionDescription: (option: T) => string) {
  return (
    options: T[],
    {inputValue, getOptionLabel}: FilterOptionsState<T>
  ): T[] => {
    const regex = _matchRegex(inputValue);
    const predicate = (option: T) => {
      return getOptionLabel(option).match(regex) ||
        getOptionDescription(option).match(regex);
    }
    return options.filter(predicate);
  }
}

/**
 *
 * @param fixedValues
 * @param multiple
 * @param onChange
 * @param setValue
 * @returns
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
      newValue = [
        ...fixedValues,
        ...newValue.filter((option: T) => fixedValues.indexOf(option) === -1),
      ];
    }

    // Propagate onChange callback.
    onChange(event, newValue, reason, details);
  }
}

function _renderInput() {
  return (params: AutocompleteRenderInputParams): React.ReactNode =>
    <TextField {...params} variant="standard" size="small" />;
}

function _renderOption<T>(
  getOptionDescription: (option: T) => string,
  getOptionLabel: (option: T) => string
) {
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
    >
      <span style={{display: 'inline-block', minWidth: '30%'}}>
        {optionTemplate}
      </span>
      {description &&
        <span style={{display: 'inline-block'}}>
          {descriptionTemplate}
        </span>
      }
    </li>;
  };
}

/**
 * Ensures that fixedValues are disabled.
 * @param fixedValues Undeleteable values in an issue usually set by filter
 *   rules.
 * @param getOptionLabel
 * @returns
 */
function _renderTags<T>(
  fixedValues: T[], getOptionLabel: (option: T) => string
) {
  return (
    value: T[],
    getTagProps: AutocompleteRenderGetTagProps
  ): React.ReactNode => {
    return value.map((option, index) => <Chip
      {...getTagProps({index})}
      label={getOptionLabel(option)}
      disabled={fixedValues.includes(option)}
      size="small"
    />);
  }
}

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
