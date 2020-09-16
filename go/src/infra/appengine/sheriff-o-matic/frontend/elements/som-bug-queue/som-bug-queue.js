'use strict';

const UNSET_PRIORITY = Number.MAX_SAFE_INTEGER;
const TREENAME_TO_PROJECT_MAPPING = {
  'fuchsia' /* tree name */: 'fuchsia' /* default bug project name*/,
  'chromium': 'chromium',
};

class SomBugQueue extends Polymer.Element {
  static get is() {
    return 'som-bug-queue';
  }

  static get properties() {
    return {
      bugQueueLabel: {
        type: String,
        observer: '_bugQueueLabelChanged',
      },
      bugs: {
        type: Array,
        notify: true,
        computed: '_computeBugs(_bugQueueJson, _uncachedBugsJson)',
      },
      treeDisplayName: String,
      _activeRequests: Object,
      _bugsByPriority: {
        type: Array,
        computed: '_computeBugsByPriority(bugs, _priorityField)',
      },
      _bugQueueJson: {
        type: Object,
        value: null,
      },
      _bugQueueJsonError: {
        type: Object,
        value: null,
      },
      _bugsLoaded: {
        type: Boolean,
        value: false,
      },
      _defaultOpenState: {
        type: Boolean,
        value: true,
      },
      _hideBugQueue: {
        type: Boolean,
        value: true,
        computed: '_computeHideBugQueue(bugQueueLabel)',
      },
      _priorityField: {
        type: String,
        computed: '_computePriorityField(_bugQueueJson)',
      },
      _showNoBugs: {
        type: Boolean,
        value: false,
        computed: '_computeShowNoBugs(bugs, _bugsLoaded, _bugQueueJsonError)',
      },
      _toggleSectionIcon: {
        type: String,
        computed: '_computeToggleSectionIcon(_opened)',
      },
      _opened: {
        type: Boolean,
        value: true,
      },
      _uncachedBugsJson: {
        type: Object,
        value: null,
      },
      _uncachedBugsJsonError: {
        type: Object,
        value: null,
      },
      _defaultBugProject: {
        type: String,
        computed: '_computeDefaultProjectIdFromTree(treeDisplayName)',
      },
    };
  }

  ready() {
    super.ready();

    // This is to expose the UNSET_PRIORITY constant for use in unit testing.
    this.UNSET_PRIORITY = UNSET_PRIORITY;
  }

  refresh() {
    if (this._hideBugQueue) {
      return;
    }

    if (this._activeRequests) {
      this._activeRequests.forEach((req) => {
        req.abort();
      });
    }

    const requests = [this.$.bugQueueAjax.generateRequest()];

    const promises = requests.map((r) => {
      return r.completes;
    });

    this._activeRequests = requests;
    Promise.all(promises).then(() => {
      this._bugsLoaded = true;
    });
  }

  _bugQueueLabelChanged() {
    this._bugQueueJson = null;
    this._bugQueueJsonError = null;

    this._uncachedBugsJson = null;
    this._uncachedBugsJsonError = null;

    this._bugsLoaded = false;

    this.refresh();
  }

  _computeBugs(bugQueueJson, uncachedBugsJson) {
    const hasBugJson = bugQueueJson && bugQueueJson.issues;

    const hasUncachedJson = uncachedBugsJson && uncachedBugsJson.issues;
    if (!hasBugJson && !hasUncachedJson) {
      return [];
    } else if (!hasUncachedJson) {
      return bugQueueJson.issues;
    }
    return uncachedBugsJson.issues;
  }

  _computeBugsByPriority(bugs, priorityField) {
    // update last updated time as relative time
    for (let i = 0; i < bugs.length; i++) {
      if (bugs[i].modify_time) {
        bugs[i].modify_time = moment(
            bugs[i].modify_time.seconds * 1000).fromNow();
      }
    }
    const buckets = bugs.reduce((function(obj, b) {
      const p = this._computePriority(b, priorityField);
      if (!(p in obj)) {
        obj[p] = [b];
      } else {
        obj[p].push(b);
      }
      return obj;
    }).bind(this),
    {});

    // Flatten the buckets into an array for use in dom-repeat.
    const result = Object.keys(buckets).sort().map(function(key) {
      return {'priority': key, 'bugs': buckets[key]};
    });
    return result;
  }

  _computePriorityField(bugQueueJson) {
    if (bugQueueJson && bugQueueJson['extras']) {
      return bugQueueJson['extras']['priority_field'];
    }
  }

  _computeHideBugQueue(bugQueueLabel) {
    // No loading or empty message is shown unless a bug queue exists.
    return !bugQueueLabel || bugQueueLabel === '' ||
      bugQueueLabel === 'Performance-Sheriff-BotHealth';
  }

  _computePriority(bug, priorityField) {
    if (!bug || !bug.field_values || !priorityField) {
      return this.UNSET_PRIORITY;
    }
    for (let i = 0; i < bug.field_values.length; i++) {
      const field = bug.field_values[i].field;
      if (field == priorityField) {
        const result = bug.field_values[i].value;
        return result !== NaN ? result : this.UNSET_PRIORITY;
      }
    }
    return this.UNSET_PRIORITY;
  }

  _computeProject(bug) {
    const match = bug.name.match(/^projects\/(.+)\/issues\/(.+)$/);
    if (match) {
      return match[1];
    }
  }

  _computeBugId(bug) {
    const match = bug.name.match(/^projects\/(.+)\/issues\/(.+)$/);
    if (match) {
      return match[2];
    }
  }

  _computeShowNoBugs(bugs, bugsLoaded, error) {
    // Show the "No bugs" message only when the queue is done loading
    return bugsLoaded && this._haveNoBugs(bugs) && this._haveNoErrors(error);
  }

  _filterBugLabels(labels, bugQueueLabel) {
    if (!labels) {
      return [];
    }
    bugQueueLabel = bugQueueLabel || '';
    return labels.map((label) => label.label).filter((label) => {
      return label.toLowerCase() != bugQueueLabel.toLowerCase();
    });
  }

  _haveNoBugs(bugs) {
    return !bugs || bugs.length == 0;
  }

  _haveNoErrors(error) {
    return !error;
  }

  _priorityText(pri) {
    if (this._validPriority(pri)) {
      return `Priority ${pri}`;
    }
    return 'No Priority';
  }

  _showBugsLoading(bugsLoaded, error) {
    return !bugsLoaded && this._haveNoErrors(error);
  }

  _validPriority(pri) {
    return pri != this.UNSET_PRIORITY;
  }

  // //////////////////// Collapsing by priority ///////////////////////////

  _computeCollapseId(pri) {
    return `collapsePri${pri}`;
  }

  _computeCollapseIcon(opened) {
    return opened ? 'remove' : 'add';
  }

  _collapseAll() {
    for (let i = 0; i < this._bugsByPriority.length; i++) {
      const pri = this._bugsByPriority[i].priority;
      const id = this._computeCollapseId(pri);
      const collapse = this.shadowRoot.querySelector('#' + id);

      collapse.opened = false;
      this.shadowRoot.querySelector('#toggleIconPri' + pri).icon =
        this._computeCollapseIcon(collapse.opened);
    }
  }

  _expandAll() {
    for (let i = 0; i < this._bugsByPriority.length; i++) {
      const pri = this._bugsByPriority[i].priority;
      const id = this._computeCollapseId(pri);
      const collapse = this.shadowRoot.querySelector('#' + id);

      collapse.opened = true;
      this.shadowRoot.querySelector('#toggleIconPri' + pri).icon =
        this._computeCollapseIcon(collapse.opened);
    }

    this._opened = true;
  }

  _togglePriorityCollapse(evt) {
    const i = evt.model.get('index');
    const pri = this._bugsByPriority[i].priority;
    const id = this._computeCollapseId(pri);
    const collapse = this.shadowRoot.querySelector('#' + id);
    if (!collapse) {
      console.error(id + ' is not a valid Id.');
    } else {
      collapse.toggle();

      this.shadowRoot.querySelector('#toggleIconPri' + pri).icon =
        this._computeCollapseIcon(collapse.opened);
    }
  }

  _computeDefaultProjectIdFromTree(treeName) {
    const projectName = TREENAME_TO_PROJECT_MAPPING[treeName.toLowerCase()];
    return projectName == undefined ? 'chromium' : projectName;
  }

  // //////////////////// Collapsing the section ///////////////////////////

  _computeToggleSectionIcon(opened) {
    return opened ? 'unfold-less' : 'unfold-more';
  }

  _toggleSection() {
    this._opened = !this._opened;
  }
}

customElements.define(SomBugQueue.is, SomBugQueue);
