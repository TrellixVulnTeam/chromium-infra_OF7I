// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import { useSelector } from 'react-redux';
import { savePreferences, selectPreferences } from './preferencesSlice';

interface Props {
  children: React.ReactNode;
}

const PreferencesStore: React.FunctionComponent<Props> = (props: Props) => {
  // Using the selector here means that this code is run every time the store
  // is updated.  As a result, savePreferences is called, which is what keeps
  // the store and local storage in sync.
  const preferences = useSelector(selectPreferences);
  savePreferences(preferences);

  return <>{props.children}</>;
};

export default PreferencesStore;
