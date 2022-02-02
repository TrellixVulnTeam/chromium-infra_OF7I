// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css, state } from 'lit-element';
import { GrpcError, RpcCode } from '@chopsui/prpc-client';
import '@material/mwc-button';
import '@material/mwc-dialog';
import '@material/mwc-icon';
import '@material/mwc-list';
import '@material/mwc-list/mwc-list-item';
import { Select } from '@material/mwc-select';
import { Snackbar } from '@material/mwc-snackbar';
import { TextArea } from '@material/mwc-textarea';
import { TextField } from '@material/mwc-textfield';
import { BeforeEnterObserver, Router, RouterLocation } from '@vaadin/router';

import { getRulesService, ClusterId, CreateRuleRequest } from '../services/rules';
import { readProjectConfig, ProjectConfig } from '../libs/config';

/**
 * NewRulePage displays a page for creating a new rule in Weetbix.
 * This is implemented as a page and not a pop-up dialog, as it will make it
 * easier for external integrations that want to link to the new rule
 * page in Weetbix from a failure (e.g. from a failure in MILO).
 */
@customElement('new-rule-page')
export class NewRulePage extends LitElement implements BeforeEnterObserver {
    @property()
    project = '';

    @state()
    projectConfig : ProjectConfig | null = null;

    @state()
    validationMessage = '';

    @state()
    defaultRule = '';

    @state()
    sourceCluster : ClusterId = { algorithm: '', id: '' };

    @state()
    snackbarError = '';

    onBeforeEnter(location: RouterLocation) {
        // Take the first parameter value only.
        this.project = typeof location.params.project == 'string' ? location.params.project : location.params.project[0];
        let search = new URLSearchParams(location.search)
        let rule = search.get('rule')
        if (rule) {
            this.defaultRule = rule;
            console.log(rule);
        }
        let sourceClusterAlg = search.get('sourceAlg');
        let sourceClusterID = search.get('sourceId');
        if (sourceClusterAlg && sourceClusterID) {
            this.sourceCluster = {
                algorithm: sourceClusterAlg,
                id: sourceClusterID,
            }
        }
    }

    connectedCallback() {
        super.connectedCallback();

        this.validationMessage = '';
        this.snackbarError = '';

        this.fetch();
    }

    render() {
        return html`
        <div id="container">
            <h1>New Rule</h1>
            <div class="validation-error" data-cy="rule-definition-validation-error">${this.validationMessage}</div>
            <div class="label">Associated Bug <mwc-icon class="inline-icon" title="The bug corresponding to the specified failures.">help_outline</mwc-icon></div>
            <div id="bug">
                <mwc-select id="bug-system" required label="Bug Tracker" data-cy="bug-system-dropdown">
                    ${this.projectConfig != null ? html`<mwc-list-item value="monorail" selected>${this.projectConfig.monorail.displayPrefix}</mwc-list-item>` : null }
                </mwc-select>
                <mwc-textfield id="bug-number" pattern="[0-9]{1,16}" required label="Bug Number" data-cy="bug-number-textbox"></mwc-textfield>
            </div>
            <div class="label">Rule Definition <mwc-icon class="inline-icon" title="A rule describing the set of failures being associated. Rules follow a subset of BigQuery Standard SQL's boolean expression syntax.">help_outline</mwc-icon></div>
            <div class="info">
                E.g. reason LIKE "%something blew up%" or test = "mytest". Supported is AND, OR, =, <>, NOT, IN, LIKE, parentheses and <a href="https://cloud.google.com/bigquery/docs/reference/standard-sql/functions-and-operators#regexp_contains">REGEXP_CONTAINS</a>.
            </div>
            <mwc-textarea id="rule-definition" label="Definition" maxLength="4096" required data-cy="rule-definition-textbox" value=${this.defaultRule}></mwc-textarea>
            <mwc-button id="create-button" raised @click="${this.save}" data-cy="create-button">Create</mwc-button>
            <mwc-snackbar id="error-snackbar" labelText="${this.snackbarError}"></mwc-snackbar>
        </div>
        `;
    }

    async fetch() {
        if (!this.project) {
            throw new Error('new-rule-page element project property is required');
        }

        this.projectConfig = await readProjectConfig(this.project);
    }

    async save() {
        if (!this.projectConfig) {
            throw new Error('invariant violated: save cannot be called before projectConfig is loaded');
        }
        const ruleDefinition = this.shadowRoot!.getElementById('rule-definition') as TextArea;
        const bugSystemControl = this.shadowRoot!.getElementById('bug-system') as Select;
        const bugNumberField = this.shadowRoot!.getElementById('bug-number') as TextField;
        const bugSystem = bugSystemControl.selected?.value || '';

        let bugId = '';
        if (bugSystem == 'monorail') {
            bugId = this.projectConfig.monorail.project + '/' + bugNumberField.value;
        }

        this.validationMessage = '';

        const request: CreateRuleRequest = {
            parent: `projects/${this.project}`,
            rule: {
                bug: {
                    system: bugSystem,
                    id: bugId,
                },
                ruleDefinition: ruleDefinition.value,
                isActive: true,
                sourceCluster: this.sourceCluster,
            },
        }

        const service = getRulesService();
        try {
            const rule = await service.create(request);
            this.validationMessage = JSON.stringify(rule);
            const path = `/projects/${encodeURIComponent(rule.project)}/clusters/rules-v1/${encodeURIComponent(rule.ruleId)}`;
            Router.go(path);
        } catch (e) {
            let handled = false;
            if (e instanceof GrpcError) {
                if (e.code === RpcCode.INVALID_ARGUMENT) {
                    handled = true;
                    this.validationMessage = 'Validation error: ' + e.description.trim() + '.';
                }
            }
            if (!handled) {
                this.showSnackbar(e as string);
            }
        }
    }

    showSnackbar(error: string) {
        this.snackbarError = "Creating rule: " + error;

        // Let the snackbar manage its own closure after a delay.
        const snackbar = this.shadowRoot!.getElementById("error-snackbar") as Snackbar;
        snackbar.show();
    }

    static styles = [css`
        #container {
            margin: 20px 14px;
        }
        #bug {
            display: inline;
        }
        #rule-definition {
            width: 100%;
            height: 160px;
        }
        #create-button {
            margin-top: 10px;
            float: right;
        }
        h1 {
            font-size: 18px;
            font-weight: normal;
        }
        .inline-button {
            display: inline-block;
            vertical-align: middle;
        }
        .inline-icon {
            vertical-align: middle;
            font-size: 1.5em;
        }
        .title {
            margin-bottom: 0px;
        }
        .label {
            margin-top: 15px;
        }
        .info {
            color: var(--light-text-color);
        }
        .validation-error {
            margin-top: 10px;
            color: var(--mdc-theme-error, #b00020);
        }
        .definition-box {
            border: solid 1px var(--divider-color);
            background-color: var(--block-background-color);
            padding: 20px 14px;
            margin: 0px;
            display: inline-block;
            white-space: pre-wrap;
            overflow-wrap: anywhere;
        }
        mwc-textarea, mwc-textfield, mwc-select {
            margin: 5px 0px;
        }
    `];
}
