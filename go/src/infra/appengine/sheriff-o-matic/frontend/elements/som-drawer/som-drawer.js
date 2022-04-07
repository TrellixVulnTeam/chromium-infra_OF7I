'use strict';

// Refresh delay for on-call rotations in milliseconds.
// This does not need to refresh very frequently.
const drawerRefreshDelayMs = 60 * 60 * 1000;

const ROTATIONS = {
  'android': [
    {
      name: 'Android Sheriff',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chrome-android-sheriff',
    },
  ],
  'chromeos': [
    {
      name: 'Infra Triage (West)',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-infra-triage',
    },
    {
      name: 'Infra Triage (East)',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:cros-infra-triage-apac',
    },
    {
      name: 'Sheriff (West)',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-sheriffs-west',
    },
    {
      name: 'Sheriff (East)',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-sheriffs-east',
    },
    {
      name: 'CI Bobby',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-ci-eng',
    },
    {
      name: 'Fleet Datacenter',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chrome-fleet-datacenter',
    },
    {
      name: 'Fleet Software',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chrome-fleet-software',
    },
    {
      name: 'Gardener',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-gardeners',
    },
    {
      name: 'Gardener (APAC)',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-apac-gardeners',
    },
    {
      name: 'ARC Constable (PST)',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-arc-constable-pst',
    },
    {
      name: 'ChromeOS Build Team',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-build-eng',
    },
    {
      name: 'ChromeOS Test Scheduling & Execution Team',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-ts-eng',
    },
    {
      name: 'ChromeOS Toolchain Team',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-toolchain',
    },
    {
      name: 'ARC Constable (non-PST)',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chromeos-arc-constable-nonpst',
    },
    {
      name: 'Jetstream Sheriff',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/grotation:chromeos-jetstream-sheriff',
    },
    {
      name: 'Morning Planner',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/grotation:chromeos-morning-planner',
    },
  ],
  'chromium': [
    {
      name: 'Chromium Sheriff',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chrome-build-sheriff',
    },
  ],
  'chrome_browser_release': [
    {
      name: 'Chrome Branch Sheriff',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chrome-branch-sheriff',
    },
  ],
  'chromium.perf': [
    {
      name: 'Chromium Perf Sheriff',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/grotation:chromium-perf-bot-sheriff',
    },
  ],
  'fuchsia': [
    {
      name: 'Fuchsia Build Gardener',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:fuchsia-build-gardener',
    },
    {
      name: 'Fuchsia Infra On-Call',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:fuchsia-infra',
    },
    {
      name: 'Fuchsia E2E On-Call',
      url: 'https://chrome-ops-rotation-proxy.appspot.com/current/grotation:fuchsia-e2e',
    },
  ],
};

const TROOPER_ROTATION = {
  name: 'Trooper',
  url: 'https://chrome-ops-rotation-proxy.appspot.com/current/oncallator:chrome-ops-client-infra',
  isTrooper: true,
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

  _refreshAsync() {
    this._refresh();
    this.async(this._refreshAsync, drawerRefreshDelayMs);
  }

  _treeChanged(tree) {
    // Copy the list of rotations and add the trooper to the start.
    let rotationsForTree = [];
    if (tree && this._rotations[tree.name]) {
      rotationsForTree = [...this._rotations[tree.name]];
    }
    rotationsForTree.unshift(TROOPER_ROTATION);

    this._currentOncalls = [];
    const self = this;
    rotationsForTree.forEach(function(rotation, index) {
      self.push('_currentOncalls', {
        name: rotation.name,
        people: 'Loading...',
        isTrooper: !!rotation.isTrooper,
      });
      fetch(rotation.url, {
        method: 'GET',
      }).then(function(response) {
        return response.json();
      }).then(function(response) {
        self.splice('_currentOncalls', index, 1, {
          name: rotation.name,
          people: response.emails.join(', '),
          isTrooper: !!rotation.isTrooper,
        });
      }).catch(function(reason) {
        self.splice('_currentOncalls', index, 1, {
          name: rotation.name,
          people: '' + reason,
          isTrooper: !!rotation.isTrooper,
        });
      });
    });
  }

  _computeStaticPageList(staticPages) {
    const pageList = [];
    for (const key in staticPages) {
      const page = staticPages[key];
      page.name = key;
      pageList.push(page);
    }
    return pageList;
  }

  _computeTreesList(trees) {
    return Object.values(trees);
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
