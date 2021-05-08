// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, {ReactElement} from 'react';
import ReactDOM from 'react-dom';

/**
 * Base component for the issue filing wizard, wrapper for other components.
 * @return Issue wizard JSX.
 */
export function IssueWizard(): ReactElement {
  return (
    <p>Welcome to the new issue wizard</p>
  );
}

/**
 * Renders the issue filing wizard page.
 * @param mount HTMLElement that the React component should be
 *   added to.
 */
export function renderWizard(mount: HTMLElement): void {
  ReactDOM.render(<IssueWizard />, mount);
}
