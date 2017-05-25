(function() {
  'use strict';

  Polymer({
    is: 'som-tree-status',

    properties: {
      tree: {
        type: String,
        observer: 'refresh',
      },
      _hasError: {
        type: Boolean,
        computed: '_computeHasError(_hasStatusApp, _statusErrorJson)',
        value: false,
      },
      _hasStatusApp: {
        type: Boolean,
        computed: '_computeHasStatusApp(tree, _statusApps)',
      },
      _hideNotice: {
        type: Boolean,
        computed: '_computeHideNotice(_hasStatusApp, _hasError)',
        value: true,
      },
      _statusApps: {
        type: Object,
        value: {
          'chromium': 'https://chromium-status.appspot.com',
          'chromeos': 'https://chromiumos-status.appspot.com',
          'trooper': 'https://infra-status.appspot.com/',
        },
      },
      _statusErrorJson: Object,
      _statusJson: Object,
      _statusUrl: {
        type: String,
        computed: '_computeStatusUrl(tree, _statusApps)',
      },
      // Processed JSON data
      _email: {
        type: String,
        computed: '_computeEmail(_statusJson)',
      },
      _message: {
        type: String,
        computed: '_computeMessage(_statusJson)',
      },
      _status: {
        type: String,
        computed: '_computeStatus(_statusJson)',
      },
      _time: {
        type: String,
        computed: '_computeTime(_statusJson)',
      },
      _username: {
        type: String,
        computed: '_computeUsername(_email)',
      },
    },

    refresh: function() {
      if (!this._hasStatusApp) {
        return;
      }
      this.$.treeStatusAjax.generateRequest();
    },

    _computeHasError: function(hasStatusApp, json) {
      return hasStatusApp && !!json && Object.keys(json).length > 0;
    },

    _computeHasStatusApp: function(tree, statusApps) {
      return tree in statusApps;
    },

    _computeHideNotice: function(hasStatusApp, hasError) {
      return !hasStatusApp || hasError;
    },

    _computeStatusUrl: function(tree, statusApps) {
      if (!this._hasStatusApp) {
        return '';
      }
      return statusApps[tree];
    },

    // Processing JSON data for display
    _computeEmail(json) {
      if (!json || !json.username) {
        return '';
      }
      return json.username;
    },

    _computeStatus(json) {
      if (!json) {
        return '';
      }
      return json.general_state;
    },

    _computeMessage(json) {
      if (!json || !json.message) {
        return 'Unknown';
      }
      return json.message;
    },

    _computeTime(json) {
      if (!json || !json.date) {
        return 'Unknown';
      }
      return json.date + ' GMT';
    },

    _computeUsername(email) {
      let cutoff = email.indexOf('@');
      if (cutoff < 0) {
        return 'Unknown';
      }
      return email.substring(0, cutoff);
    },
  });
})();
