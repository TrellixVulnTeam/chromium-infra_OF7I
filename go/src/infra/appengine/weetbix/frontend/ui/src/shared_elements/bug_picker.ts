// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css, state } from 'lit-element';

import { readProjectConfig, ProjectConfig } from '../services/config';
import { Select } from '@material/mwc-select';
import { TextField } from '@material/mwc-textfield';
import '@material/mwc-list';
import '@material/mwc-list/mwc-list-item';

// BugPicker lists the failure association rules configured in Weetbix.
@customElement('bug-picker')
export class BugPicker extends LitElement {
    @property()
    project = '';

    // The bug tracking system. Valid values are 'monorail' or 'buganizer'.
    @property()
    bugSystem = '';

    // The bug ID within the bug tracking system.
    // For monorail, the scheme is '{monorail_project}/{bug_number}'.
    // For buganizer, the scheme is '{bug_number}'.
    @property()
    bugId = '';

    @state()
    projectConfig : ProjectConfig | null = null;

    // Implements the workaround for mwc-select inside of an mwc-dialog, as
    // described in
    // https://github.com/material-components/material-web/issues/832.
    @property({type: Boolean})
    material832Workaround = false;

    connectedCallback() {
        super.connectedCallback();
        this.fetch();
    }

    async fetch() {
        if (!this.project) {
            throw new Error('invariant violated: project must be set before fetch');
        }
        this.projectConfig = await readProjectConfig(this.project);
        if (this.bugSystem == '') {
            // Default the bug tracking system.
            this.setSystemMonorail();
        }
    }

    render() {
        let bugNumber = this.bugNumber();
        let monorailSystem = this.monorailSystem();

        return html`
            <mwc-select ?fixedMenuPosition=${this.material832Workaround} id="bug-system" required label="Bug Tracker" data-cy="bug-system-dropdown" @change=${this.onSystemChange} @closed=${this.onSelectClosed}>
                ${this.projectConfig != null ? html`<mwc-list-item value="monorail" .selected=${this.bugSystem == 'monorail' && monorailSystem == this.projectConfig.monorail.project}>${this.projectConfig.monorail.displayPrefix}</mwc-list-item>` : null }
            </mwc-select>
            <mwc-textfield id="bug-number" pattern="[0-9]{1,16}" required label="Bug Number" data-cy="bug-number-textbox" .value=${bugNumber} @change=${this.onNumberChange}></mwc-textfield>
        `;
    }

    monorailSystem(): string | null {
        if (this.bugId.indexOf('/') >= 0) {
            let parts = this.bugId.split('/');
            return parts[0];
        } else {
            return null;
        }
    }

    bugNumber(): string {
        if (this.bugId.indexOf('/') >= 0) {
            let parts = this.bugId.split('/');
            return parts[1];
        } else {
            return this.bugId;
        }
    }

    onSystemChange(event: Event) {
        let select = event.target as Select;

        // If no actual value is selected, do not register a change, as
        // doing so would wipe the existing system that was set.
        // We want to retain the existing selected value until options load
        // and an actual selection is made.
        if (!select.value) {
            return;
        }
        if (select.value == 'monorail') {
            this.setSystemMonorail();
        } else {
            // TODO: support buganizer.
            throw new Error('unknown bug system: ' + select.value)
        }
    }

    onSelectClosed(e: Event) {
        // Stop closure of mwc-select closing an mwc-dialog that this bug
        // picker may be enclosed inside.
        // https://github.com/material-components/material-web/issues/1150.
        e.stopPropagation();
    }

    setSystemMonorail() {
        if (!this.projectConfig) {
            throw new Error('invariant violated: projectConfig must be loaded before setting bug system');
        }
        this.bugSystem = 'monorail';
        this.bugId = `${this.projectConfig!.monorail.project}/${this.bugNumber()}`;
    }

    onNumberChange(event: Event) {
        let textfield = event.target as TextField;

        // Update the bug number, preserving whatever monorail system has
        // been set (if any). Do not check the value of the bug system
        // dropdown as the projectConfig may not have loaded.
        let monorailSystem = this.monorailSystem();
        if (monorailSystem != null) {
            this.bugId = `${monorailSystem}/${textfield.value}`;
        } else {
            this.bugId = textfield.value;
        }
    }

    static styles = [css`
        :host {
            display: inline-block;
        }
    `];
}
