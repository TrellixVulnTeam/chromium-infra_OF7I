// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css, state, TemplateResult } from 'lit-element';
import { DateTime } from 'luxon';
import '@material/mwc-button';
import '@material/mwc-circular-progress';

/**
 * ReclusteringProgressIndicator displays the progress Weetbix is making
 * re-clustering test results to reflect current algorithms and
 * the current rule.
 */
@customElement('reclustering-progress-indicator')
export class ReclusteringProgressIndicator extends LitElement {
    @property()
    project: string;

    @property()
    // Whether the cluster for which the indicator is being shown is
    // defined by a failure association rule.
    hasRule: boolean | undefined;

    @property()
    // The last updated time of the rule which defines the cluster (if any).
    // This should be set if hasRule is true.
    ruleLastUpdated: string | undefined;

    @state()
    progress : ReclusteringProgress | undefined;

    @state()
    lastRefreshed : DateTime | undefined;

    @state()
    // Whether the indicator should be displayed. If re-clustering
    // is not complete, this will be set to true. It will only ever
    // be set to false if re-clustering is complete and the user
    // reloads cluster analysis.
    show: boolean;

    // The last progress shown on the UI.
    progressPerMille: number;

    // The ID returned by window.setInterval. Used to manage the timer
    // used to periodically poll for status updates.
    interval: number;

    connectedCallback() {
        super.connectedCallback();

        this.interval = window.setInterval(() => {
            this.timerTick();
        }, 5000);

        this.show = false;

        this.fetch();
    }

    disconnectedCallback() {
        super.disconnectedCallback();
        window.clearInterval(this.interval);
    }

    // tickerTick is called periodically. Its purpose is to obtain the
    // latest re-clustering progress if progress is not complete.
    timerTick() {
        // Only fetch updates if the page is being shown. This avoids
        // creating server load for no appreciable UX improvement.
        if (document.visibilityState == 'visible' && this.progressPerMille < 1000) {
            this.fetch();
        }
    }

    render() {
        if (this.progress === undefined ||
            this.hasRule === undefined ||
            (this.hasRule && !this.ruleLastUpdated)) {
            // Still loading.
            return html``;
        }

        let reclusteringTarget = "updated clustering algorithms";
        let progressPerMille = this.progressToLatestAlgorithms(this.progress);
        if (this.hasRule) {
            const ruleProgress = this.progressToRulesVersion(this.progress, this.ruleLastUpdated);
            if (ruleProgress < progressPerMille) {
                reclusteringTarget = "the latest rule definition";
                progressPerMille = ruleProgress;
            }
        }
        this.progressPerMille = progressPerMille;

        if (progressPerMille >= 1000 && !this.show) {
            return html``;
        }

        // Once shown, keep showing.
        this.show = true;

        let progressText = "task queued";
        if (progressPerMille >= 0) {
            progressText = (progressPerMille / 10).toFixed(1) + "%"
        }

        var content: TemplateResult
        if (progressPerMille < 1000) {
            content = html`
            <span class="progress-description" data-cy="reclustering-progress-description">
                Weetbix is re-clustering test results to reflect ${reclusteringTarget} (${progressText}). Cluster impact may be out-of-date.
                <span class="last-updated">
                    Last update ${this.lastRefreshed.toLocaleString(DateTime.TIME_WITH_SECONDS)}.
                </span>
            <span>`
        } else {
            content = html`
            <span class="progress-description" data-cy="reclustering-progress-description">
                Weetbix has finished re-clustering test results. Updated cluster impact is now available.
            </span>
            <mwc-button outlined @click=${this.refreshAnalysis}>
                View Updated Impact
            </mwc-button>`
        }

        return html`
        <div class="progress-box">
            <mwc-circular-progress
                ?indeterminate=${progressPerMille<0}
                progress="${Math.max(0, progressPerMille/1000)}">
            </mwc-circular-progress>
            ${content}
        </div>
        `;
    }

    async fetch() {
        const response = await fetch(`/api/projects/${encodeURIComponent(this.project)}/reclusteringProgress`);
        const progress = await response.json();

        this.lastRefreshed = DateTime.now();
        this.progress = progress;
    }

    refreshAnalysis() {
        this.fireRefreshAnalysis();
        this.show = false;
    }

    fireRefreshAnalysis() {
        const event = new CustomEvent<RefreshAnalysisEvent>('refreshanalysis', {
            detail: {
            },
        });
        this.dispatchEvent(event)
    }

    progressToLatestAlgorithms(p : ReclusteringProgress) : number {
        return this.progressTo(p, (t : ReclusteringTarget) => {
            return t.algorithmsVersion >= this.progress.latestAlgorithmsVersion
        });
    }

    progressToRulesVersion(p : ReclusteringProgress, rulesVersion : string) : number {
        const d = DateTime.fromISO(rulesVersion)
        return this.progressTo(p, (t : ReclusteringTarget) => {
            return DateTime.fromISO(t.rulesVersion) >= d
        });
    }

    // progressTo returns the progress to completing a re-clustering run
    // satisfying the given re-clustering target, expressed as a predicate.
    // If re-clustering has started, the returned value is value from 0 to
    // 1000. If the run is pending, the value -1 is returned.
    progressTo(p : ReclusteringProgress, predicate : (t : ReclusteringTarget) => boolean) : number {
        if (predicate(p.last)) {
            // Completed
            return 1000;
        }
        if (predicate(p.next)) {
            return p.progressPerMille;
        }
        // Run not yet started (e.g. because we are still finishing a previous
        // re-clustering).
        return -1;
    }

    static styles = [css`
        .progress-box {
            display: flex;
            background-color: var(--light-active-color);
            padding: 5px;
            align-items: center;
        }
        .progress-description {
            padding: 0px 10px;
        }
        .last-updated {
            padding: 0px;
            font-size: var(--font-size-small);
            color: var(--greyed-out-text-color);
        }
    `];
}

// RefreshAnalysisEvent is an event that is triggered when the user requests
// cluster analysis to be updated.
export interface RefreshAnalysisEvent {
}

// ReclusteringTarget captures the rules and algorithms a re-clustering run
// is re-clustering to.
interface ReclusteringTarget {
    rulesVersion: string; // RFC 3339 encoded date/time.
    algorithmsVersion: number;
}

// ReclusteringProgress captures the progress re-clustering a
// given LUCI project's test results with a specific rules
// version and/or algorithms version.
interface ReclusteringProgress {
    // ProgressPerMille is the progress of the current re-clustering run,
    // measured in thousandths (per mille).
    progressPerMille: number;
    // LatestAlgorithmsVersion is the latest version of algorithms known to
    // Weetbix.
    latestAlgorithmsVersion: number;
    // Next is the goal of the current re-clustering run. (For which
    // ProgressPerMille is specified.)
    next: ReclusteringTarget;
    // Last is the goal of the last completed re-clustering run.
    last: ReclusteringTarget;
}
