// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {
    css,
    customElement,
    html,
    LitElement,
    property,
    state,
    TemplateResult
} from 'lit-element';
import { Ref } from 'react';
import { NavigateFunction } from 'react-router-dom';

import { GrpcError } from '@chopsui/prpc-client';

import {
    getRulesService,
    LookupBugRequest,
    LookupBugResponse,
    parseRuleName
} from '../../../services/rules';
import { linkToRule } from '../../../tools/urlHandling/links';

// BugPage handles the bug endpoint:
// /b/<bugtracker>/<bugid>
// Where bugtracker is either 'b' for buganizer or a monorail project name.
// It redirects to the page for the rule associated with the bug (if any).
@customElement('bug-page')
export class BugPage extends LitElement {

    @property({ attribute: false })
    ref: Ref<BugPage> | null = null;

    @property()
    bugTracker = '';

    @property()
    bugId = '';

    navigate!: NavigateFunction;

    @property()
    system: string = '';

    @property()
    id: string = '';

    @state()
    error: any;

    @state()
    response: LookupBugResponse | null = null;

    connectedCallback() {
        super.connectedCallback();
        this.setBug(this.bugTracker, this.bugId);
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

    async fetch(): Promise<void> {
        const service = getRulesService();
        try {
            const request: LookupBugRequest = {
                system: this.system,
                id: this.id,
            }
            const response = await service.lookupBug(request);
            this.response = response;

            if (response.rules && response.rules.length === 1) {
                const ruleKey = parseRuleName(response.rules[0]);
                const link = linkToRule(ruleKey.project, ruleKey.ruleId);
                this.navigate(link);
            }
            this.requestUpdate();
        } catch (e) {
            this.error = e;
        }
    }

    render() {
        return html`<div id="container">${this.message()}</div>`
    }

    message(): TemplateResult {
        if (this.error) {
            if (this.error instanceof GrpcError) {
                return html`Error finding rule for bug (${this.system}:${this.id}): ${this.error.description.trim()}.`;
            }
            return html`${this.error}`;
        }
        if (this.response) {
            if (!this.response.rules) {
                return html`No rule found matching the specified bug (${this.system}:${this.id}).`;
            }

            const ruleLink = (ruleName: string): string => {
                const ruleKey = parseRuleName(ruleName);
                return linkToRule(ruleKey.project, ruleKey.ruleId);
            }

            return html`Multiple projects have rules matching the specified bug (${this.system}:${this.id}):
            <ul>
                ${this.response.rules.map(r => html`<li><a href="${ruleLink(r)}">${parseRuleName(r).project}</a></li>`)}
            </ul>
            `
        }
        return html`Loading...`;
    }

    static styles = [css`
        #container {
            margin: 20px 14px;
        }
    `];
}
