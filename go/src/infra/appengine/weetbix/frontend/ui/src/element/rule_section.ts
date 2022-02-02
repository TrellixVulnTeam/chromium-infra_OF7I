// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css, state, TemplateResult } from 'lit-element';
import { DateTime } from 'luxon';
import { GrpcError, RpcCode } from '@chopsui/prpc-client';
import '@material/mwc-button';
import '@material/mwc-dialog';
import '@material/mwc-textfield';
import '@material/mwc-textarea';
import { TextArea } from '@material/mwc-textarea';
import '@material/mwc-textfield';
import '@material/mwc-snackbar';
import { Snackbar } from '@material/mwc-snackbar';
import '@material/mwc-icon'

import { getRulesService, Rule, UpdateRuleRequest } from '../services/rules';

/**
 * RuleSection displays a rule tracked by Weetbix.
 */
@customElement('rule-section')
export class RuleSection extends LitElement {
    @property()
    project = '';

    @property()
    ruleId = '';

    @state()
    rule: Rule | null = null;

    @state()
    editing = false;

    @state()
    validationMessage = '';

    @state()
    snackbarError = '';

    connectedCallback() {
        super.connectedCallback();

        this.editing = false;
        this.validationMessage = "";
        this.snackbarError = "";

        this.fetch();
    }

    render() {
        if (!this.rule) {
            return html`Loading...`;
        }
        const r = this.rule;
        const formatTime = (time: string): string => {
            let t = DateTime.fromISO(time);
            let d = DateTime.now().diff(t);
            if (d.as('seconds') < 60) {
                return "just now";
            }
            if (d.as('hours') < 24) {
                return t.toRelative()?.toLocaleLowerCase() || '';
            }
            return DateTime.fromISO(time).toLocaleString(DateTime.DATETIME_SHORT);
        }
        const formatTooltipTime = (time: string): string => {
            // Format date/time with full month name, e.g. "January" and Timezone,
            // to disambiguate date/time even if the user's locale has been set
            // incorrectly.
            return DateTime.fromISO(time).toLocaleString(DateTime.DATETIME_FULL_WITH_SECONDS);
        }
        const formatUser = (user: string): TemplateResult => {
            if (user == 'weetbix') {
                return html`Weetbix`;
            } else if (user.endsWith("@google.com")) {
                var ldap = user.substr(0, user.length - "@google.com".length)
                return html`<a href="http://who/${ldap}">${ldap}</a>`;
            } else {
                return html`${user}`;
            }
        }
        return html`
        <div>
            <div class="definition-box-container">
                <pre class="definition-box" data-cy="rule-definition">${r.ruleDefinition}</pre>
                <div class="definition-edit-button">
                    <mwc-button outlined @click="${this.edit}" data-cy="rule-definition-edit">Edit</mwc-button>
                </div>
            </div>
            <table>
                <tbody>
                    <tr>
                        <th>Type</th>
                        <td>Bug</td>
                    </tr>
                    <tr>
                        <th>Associated Bug</th>
                        <td><a href="${r.bug.url}">${r.bug.linkText}</a></td>
                    </tr>
                    <tr>
                        <th>Enabled <mwc-icon class="inline-icon" title="Enabled failure association rules are used to match failures. If a rule is no longer needed, it should be disabled.">help_outline</mwc-icon></th>
                        <td data-cy="rule-enabled">
                            ${r.isActive ? "Yes" : "No"}
                            <div class="inline-button">
                                <mwc-button outlined dense @click="${this.toggleActive}" data-cy="rule-enabled-toggle">${r.isActive ? "Disable" : "Enable"}</mwc-button>
                            </div>
                        </td>
                    </tr>
                    <tr>
                        <th>Source Cluster <mwc-icon class="inline-icon" title="The cluster this rule was originally created from.">help_outline</mwc-icon></th>
                        <td>
                            ${r.sourceCluster.algorithm && r.sourceCluster.id ?
                                html`<a href="/projects/${this.project}/clusters/${r.sourceCluster.algorithm}/${r.sourceCluster.id}">${r.sourceCluster.algorithm}/${r.sourceCluster.id}</a>` :
                                html`None`
                            }
                        </td>
                    </tr>
                </tbody>
            </table>
            <div class="audit">
                ${(r.lastUpdateTime != r.createTime) ?
                html`Last updated by <span class="user">${formatUser(r.lastUpdateUser)}</span> <span class="time" title="${formatTooltipTime(r.lastUpdateTime)}">${formatTime(r.lastUpdateTime)}</span>.` : html``}
                Created by <span class="user">${formatUser(r.createUser)}</span> <span class="time" title="${formatTooltipTime(r.createTime)}">${formatTime(r.createTime)}</span>.
            </div>
        </div>
        <mwc-dialog class="rule-edit-dialog" ?open="${this.editing}" @closed="${this.editClosed}">
            <div class="edit-title">Edit Rule Definition <mwc-icon class="inline-icon" title="Weetbix rule definitions describe the failures associated with a bug. Rules follow a subset of BigQuery Standard SQL's boolean expression syntax.">help_outline</mwc-icon></div>
            <div class="validation-error" data-cy="rule-definition-validation-error">${this.validationMessage}</div>
            <mwc-textarea id="rule-definition" label="Rule Definition" maxLength="4096" required data-cy="rule-definition-textbox"></mwc-textarea>
            <div>
                Supported is AND, OR, =, <>, NOT, IN, LIKE, parentheses and <a href="https://cloud.google.com/bigquery/docs/reference/standard-sql/functions-and-operators#regexp_contains">REGEXP_CONTAINS</a>.
                Valid identifiers are <em>test</em> and <em>reason</em>.
            </div>
            <mwc-button slot="primaryAction" @click="${this.save}" data-cy="rule-definition-save">Save</mwc-button>
            <mwc-button slot="secondaryAction" dialogAction="close" data-cy="rule-definition-cancel">Cancel</mwc-button>
        </mwc-dialog>
        <mwc-snackbar id="error-snackbar" labelText="${this.snackbarError}"></mwc-snackbar>
        `;
    }

    async fetch() {
        if (!this.ruleId) {
            throw new Error('rule-section element ruleID property is required');
        }
        const service = getRulesService();
        const rule = await service.get({
            name: `projects/${this.project}/rules/${this.ruleId}`
        })

        this.rule = rule || null;
        this.fireRuleChanged();
    }

    edit() {
        if (!this.rule) {
            throw new Error('invariant violated: edit cannot be called before rule is loaded');
        }
        const ruleDefinition = this.shadowRoot!.getElementById("rule-definition") as TextArea;
        ruleDefinition.value = this.rule.ruleDefinition;

        this.editing = true;
        this.validationMessage = "";
    }

    editClosed() {
        this.editing = false;
    }

    async save() {
        if (!this.rule) {
            throw new Error('invariant violated: save cannot be called before rule is loaded');
        }
        const ruleDefinition = this.shadowRoot!.getElementById("rule-definition") as TextArea;
        if (ruleDefinition.value == this.rule.ruleDefinition) {
            this.editing = false;
            return;
        }

        this.validationMessage = "";

        const request: UpdateRuleRequest = {
            rule: {
                name: this.rule.name,
                ruleDefinition: ruleDefinition.value,
            },
            updateMask: "ruleDefinition",
            etag: this.rule.etag,
        }

        try {
            await this.applyUpdate(request);
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

    async toggleActive() {
        if (!this.rule) {
            throw new Error('invariant violated: toggleActive cannot be called before rule is loaded');
        }
        const request: UpdateRuleRequest = {
            rule: {
                name: this.rule.name,
                isActive: !this.rule.isActive,
            },
            updateMask: "isActive",
            etag: this.rule.etag,
        }
        try {
            await this.applyUpdate(request);
        } catch (err) {
            this.showSnackbar(err as string);
        }
    }

    // applyUpdate tries to apply the given update to the rule. If the
    // update succeeds, this method returns nil. If a validation error
    // occurs, the validation message is returned.
    async applyUpdate(request: UpdateRuleRequest) : Promise<void> {
        const service = getRulesService();
        const rule = await service.update(request)
        this.rule = rule;
        this.editing = false;
        this.fireRuleChanged();
    }

    showSnackbar(error: string) {
        this.snackbarError = "Updating rule: " + error;

        // Let the snackbar manage its own closure after a delay.
        const snackbar = this.shadowRoot!.getElementById("error-snackbar") as Snackbar;
        snackbar.show();
    }

    fireRuleChanged() {
        if (!this.rule) {
            throw new Error('invariant violated: fireRuleChanged cannot be called before rule is loaded');
        }
        const event = new CustomEvent<RuleChangedEvent>('rulechanged', {
            detail: {
                lastUpdated: this.rule.lastUpdateTime
            },
        });
        this.dispatchEvent(event);
    }

    static styles = [css`
        .inline-button {
            display: inline-block;
            vertical-align: middle;
        }
        .inline-icon {
            vertical-align: middle;
            font-size: 1.5em;
        }
        .edit-title {
            margin-bottom: 10px;
        }
        .rule-edit-dialog {
            --mdc-dialog-min-width:1000px
        }
        .validation-error {
            color: var(--mdc-theme-error, #b00020);
        }
        #rule-definition {
            width: 100%;
            height: 160px;
        }
        .definition-box-container {
            display: flex;
            margin-bottom: 20px;
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
        .definition-edit-button {
            align-self: center;
            margin: 5px;
        }
        table {
            border-collapse: collapse;
            max-width: 100%;
        }
        th {
            font-weight: normal;
            color: var(--greyed-out-text-color);
            text-align: left;
        }
        td,th {
            padding: 4px;
            max-width: 80%;
        }
        mwc-textarea, mwc-textfield {
            margin: 5px 0px;
        }
        .audit {
            font-size: var(--font-size-small);
            color: var(--greyed-out-text-color);
        }
    `];
}

export interface RuleChangedEvent {
    lastUpdated: string; // RFC 3339 encoded date/time.
}
