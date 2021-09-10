// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { customElement, html, LitElement, property } from 'lit-element';
import './element/bug_cluster_table.ts';
import './element/cluster_table.ts';


// MonorailTest excersises the monorail API in the server and displays an
// issue summary to verify it is working.
// This component is only temporary, so keeping it here rather than creating
// a file to hold it.
@customElement('monorail-test')
export class MonorailTest extends LitElement {
    @property()
    issue: Issue | undefined;

    connectedCallback() {
        super.connectedCallback()
        fetch("/api/monorailtest").then(r => r.json()).then(issue => this.issue = issue);
    }

    render() {
        if (this.issue === undefined) {
            return html`Loading...`;
        }
        return html`<p>Issue summary: ${this.issue.summary}</p>`;
    }
}

// Issue is a part of the monorail issue data sent from the server.
interface Issue {
    summary: string;
}
