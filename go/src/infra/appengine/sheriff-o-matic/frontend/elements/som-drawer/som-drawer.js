'use strict';

// Refresh delay for on-call rotations in milliseconds.
// This does not need to refresh very frequently.
const drawerRefreshDelayMs = 60 * 60 * 1000;

const ROTATIONS = {
  'android': [
    {
      name: 'Android Sheriff',
      url: 'https://rota-ng.appspot.com/legacy/sheriff_android.json',
    },
  ],
  'chromeos': [
    {
      name: 'Moblab Peeler',
      url: 'https://rotation.googleplex.com/json?id=6383984776839168',
    },
    {
      name: 'Jetstream Sheriff',
      url: 'https://rotation.googleplex.com/json?id=5186988682510336',
    },
    {
      name: 'Morning Planner',
      url: 'https://rotation.googleplex.com/json?id=140009',
    },
  ],
  'chromium': [
    {
      name: 'Chromium Sheriff',
      url: 'https://rota-ng.appspot.com/legacy/sheriff.json',
    },
  ],
  'chromium.perf': [
    {
      name: 'Chromium Perf Sheriff',
      url: 'https://rota-ng.appspot.com/legacy/sheriff_perfbot.json',
    },
  ],
  'fuchsia': [
    {
      name: 'Fuchsia Build Cop',
      url: 'https://oncall.corp.google.com/tq-buildcop/json',
    },
    {
      name: 'Fuchsia Infra',
      url: 'https://oncall.corp.google.com/fuchsia-infra/json',
    },
    {
      name: 'Fuchsia E2E',
      url: 'https://rotation.googleplex.com/json?id=5683269937922048',
    },
  ],
};

class SomDrawer extends Polymer.Element {
  static get is() {
    return 'som-drawer';
  }

  static get properties() {
    return {
      _defaultTree: String,
      path: {
        type: String,
        notify: true,
      },
      _rotations: {
        type: Object,
        value: ROTATIONS,
      },
      _currentOncalls: {
        type: Array,
        value: null,
      },
      _staticPageList: {
        type: Array,
        computed: '_computeStaticPageList(staticPages)',
        value: function() {
          return [];
        },
      },
      tree: {
        type: Object,
        observer: '_treeChanged',
      },
      trees: Object,
      _treesList: {
        type: Array,
        computed: '_computeTreesList(trees)',
      },
      _trooperString: String,
      _troopers: {
        type: Array,
        computed: '_computeTroopers(_trooperString)',
        value: null,
      },
      // Settings.
      collapseByDefault: {
        type: Boolean,
        notify: true,
      },
    };
  }

  static get observers() {
    return [
      '_navigateToDefaultTree(path, trees, _defaultTree)',
    ];
  }

  created() {
    super.created();

    this.async(this._refreshAsync, drawerRefreshDelayMs);
  }

  _refresh() {
    this.$.fetchTrooper.generateRequest();
  }

  _refreshAsync() {
    this._refresh();
    this.async(this._refreshAsync, drawerRefreshDelayMs);
  }

  _isCros(tree) {
    return tree && tree.name === 'chromeos';
  }

  _isBrowserBranch(tree) {
    return tree && tree.name === 'chrome_browser_release';
  }

  _treeChanged(tree) {
    if (!(tree && this._rotations[tree.name])) {
      this.set('_currentOncalls', []);
      return;
    }

    this._currentOncalls = [];
    const self = this;
    this._rotations[tree.name].forEach(function(rotation, index) {
      self.push('_currentOncalls', {
        name: rotation.name,
        people: 'Loading...',
      });
      switch (rotation.url.split('/')[2]) {
        case 'rota-ng.appspot.com':
          fetch(rotation.url, {
            method: 'GET',
          }).then(function(response) {
            return response.json();
          }).then(function(response) {
            self.splice('_currentOncalls', index, 1, {
              name: rotation.name,
              people: response.emails.join(', '),
            });
          });
          break;
        case 'rotation.googleplex.com':
          fetch(rotation.url, {
            method: 'GET',
            credentials: 'include',
          }).then(function(response) {
            return response.json();
          }).then(function(response) {
            self.splice('_currentOncalls', index, 1, {
              name: rotation.name,
              people: response.primary,
            });
          });
          break;
        case 'oncall.corp.google.com':
          fetch(rotation.url, {
            method: 'GET',
            credentials: 'include',
          }).then(function(response) {
            return response.json();
          }).then(function(response) {
            const people = [];
            response.forEach(function(entry) {
              if (entry.person) {
                people.push(entry.person);
              }
            });
            self.splice('_currentOncalls', index, 1, {
              name: rotation.name,
              people: people.length ? people.join(', ') : 'None',
            });
          });
          break;
      }
    });
  }

  _computeStaticPageList(staticPages) {
    const pageList = [];
    for (let key = 0; key < staticPages; key++) {
      const page = staticPages[key];
      page.name = key;
      pageList.push(page);
    }
    return pageList;
  }

  _computeTreesList(trees) {
    return Object.values(trees);
  }

  _computeTroopers(trooperString) {
    if (!trooperString) {
      return [];
    }

    const troopers = trooperString.split(',');
    troopers[0] = troopers[0] + ' (primary)';
    if (troopers.length == 1) {
      return troopers;
    }
    troopers.slice(1).forEach(function(trooper, i) {
      troopers[i + 1] = trooper + ' (secondary)';
    });
    return troopers;
  }

  _formatDate(date) {
    return date.toISOString().substring(0, 10);
  }

  _formatDateShort(date) {
    return moment(date).format('MMM D');
  }

  _navigateToDefaultTree(path, trees, defaultTree) {
    // Not a huge fan of watching path while also changing it, but without
    // watching path, this fires before path has completely initialized,
    // causing the default page to be overwritten.
    if (path == '/') {
      if (defaultTree && defaultTree in trees) {
        this.path = '/' + defaultTree;
      }
    }
  }

  _onSelected(evt) {
    const pathIdentifier = evt.srcElement.value;
    this.path = '/' + pathIdentifier;

    if (pathIdentifier && pathIdentifier in this.trees) {
      this._defaultTree = pathIdentifier;
    }
  }

  toggleMenu(e) {
    const path = Polymer.dom(e).path;
    let target = null;
    let collapseId = null;

    for (let i = 0; i < path.length && !collapseId; i++) {
      target = path[i];
      collapseId = target.getAttribute('data-toggle-target');
    }

    const collapse = this.$[collapseId];
    collapse.opened = !collapse.opened;

    const icons = target.getElementsByClassName('toggle-icon');
    for (let i = 0; i < icons.length; i++) {
      icons[i].icon = collapse.opened ? 'remove' : 'add';
    }
  }
}

customElements.define(SomDrawer.is, SomDrawer);
