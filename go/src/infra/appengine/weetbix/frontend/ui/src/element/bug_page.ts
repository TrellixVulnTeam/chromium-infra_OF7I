// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css, state } from 'lit-element';
import { RouterLocation, Router } from '@vaadin/router';
import { GrpcError, RpcCode } from '@chopsui/prpc-client';

import { LookupBugRequest, getRulesService, parseRuleName } from '../services/rules';
import { linkToRule } from '../urlHandling/links';

// BugPage handles the bug endpoint:
// /b/<bugtracker>/<bugid>
// Where bugtracker is either 'b' for buganizer or a monorail project name.
// It redirects to the page for the rule associated with the bug (if any).
@customElement('bug-page')
export class BugPage extends LitElement {
    @property({ attribute: false })
    location!: RouterLocation;

    @property()
    system: string = '';

    @property()
    id: string = '';

    @state()
    error: any;

    onBeforeEnter(location: RouterLocation) {
        // Take the first parameter value only.
        const bugTracker = typeof location.params.bugTracker == 'string' ? location.params.bugTracker : location.params.bugTracker[0];
        const id = typeof location.params.id == 'string' ? location.params.id : location.params.id[0];
        this.setBug(bugTracker, id);
    }

    connectedCallback() {
        super.connectedCallback();

        this.fetch();
    }

    setBug(tracker: string, id: string) {
        if (tracker == 'b') {
            this.system = 'buganizer';
            this.id = id;
        } else {
            this.system = 'monorail';
            this.id = tracker + '/' + id;
        }
    }

    async fetch() : Promise<void> {
        const service = getRulesService();
        try {
            const request: LookupBugRequest = {
                system: this.system,
                id: this.id,
            }
            const response = await service.lookupBug(request);
            const ruleKey = parseRuleName(response.rule);
            const link = linkToRule(ruleKey.project, ruleKey.ruleId);
            Router.go(link);
        } catch (e) {
            this.error = e;
        }
    }

    render() {
        return html`<div id="container">${this.message()}</div>`
    }

    message(): string {
        if (!this.error) {
            return `Loading...`;
        }

        if (this.error instanceof GrpcError) {
            if (this.error.code == RpcCode.NOT_FOUND) {
                return `No rule found matching the specified bug (${this.system}:${this.id}).`;
            }
            return `Error finding rule for bug (${this.system}:${this.id}): ${this.error.description.trim()}.`;
        }
        return `${this.error}`;
    }

    static styles = [css`
        #container {
            margin: 20px 14px;
        }
    `];
}
