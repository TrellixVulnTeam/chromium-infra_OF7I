// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement } from 'lit-element';

// NotFoundPage is displayed if no routes match the page URL path.
@customElement('not-found-page')
export class NotFoundPage extends LitElement {
    render() {
        return html`Page not found`;
    }
}

