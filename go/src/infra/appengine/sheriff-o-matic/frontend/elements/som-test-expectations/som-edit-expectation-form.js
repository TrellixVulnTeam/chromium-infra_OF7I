'use strict';

class SomEditExpectationForm extends Polymer.LegacyElementMixin(Polymer.Element) {

  static get is() {
    return 'som-edit-expectation-form';
  }

  static get properties() {
    return {
      expectation: {
        type: Object,
        value: null,
        observer: 'expectationChanged',
      },
      _editValue: {
        type: Object,
        value: function() { return {}; },
        notify: true,
      },
      expectationValues: {
        type: Array,
        value: [
          'Crash',
          'Failure',
          'Pass',
          'Slow',
          'Skip',
          'Timeout',
          'NeedsManualRebaseline',
        ],
      },
      modifierValues: {
        // Note: these values are defined in
        // third_party/WebKit/Tools/Scripts/webkitpy/layout_tests/models/test_expectations.py
        type: Array,
        value: [
          'Mac',
          'Mac10.9',
          'Mac10.10',
          'Mac10.11',
          'Mac10.12',
          'Retina',
          'Win',
          'Win7',
          'Win10',
          'Linux',
          'Android',
          'KitKat',
          'Release',
          'Debug',
        ],
      },
    };
  }

  expectationChanged(evt) {
    if (!this.expectation) {
      return;
    }
    // Make a copy of the expectation to edit in this form. Modify only
    // the copy, so we can cancel, or fire an edited event with old
    // and new values set in the details.
    this._editValue = JSON.parse(JSON.stringify(this.expectation));
  }

  _addBug(evt) {
    ga('send', 'event', this.nodeName.toLocaleLowerCase(), 'add-bug');
    const bug = this.$.newBug.value.trim();
    if (!this._isValidBugStr(bug)) {
      this._newBugError = 'Invalid bug';
      return;
    }

    this._newBugError = '';
    if (this._editValue.Bugs) {
      this.push('_editValue.Bugs', bug);
    } else {
      this.set('_editValue.Bugs', [bug]);
    }
    this.$.newBug.value = '';
  }

  _isValidBugStr(bugStr) {
    if (/^\d+$/.test(bugStr)) {
      return true;
    }

    const bugUrl = (/^https?:\/\//.test(bugStr)) ? bugStr : `https://${bugStr}`;
    let url;
    try {
      url = new URL(bugUrl);
    } catch (_) {
      return false;
    }
    const isValid =
      url.hostname === 'bugs.chromium.org' && /^\d+$/.test(url.searchParams.get('id')) || // eslint-disable-line max-len
      url.hostname === 'crbug.com' && /^\/\d+$/.test(url.pathname);
    return isValid;
  }

  _expects(item, val) {
    if (!item || !item.Expectations) {
      return false;
    }
    let ret = item.Expectations.some((v) => { return v == val; });
    return ret;
  }

  _removeBug(evt) {
    ga('send', 'event', this.nodeName.toLocaleLowerCase(), 'remove-bug');
    this.arrayDelete('_editValue.Bugs', evt.target.value);
  }

  _toggleExpectation(evt) {
    ga('send', 'event', this.nodeName.toLocaleLowerCase(),
        'toggle-expectation');
    if (!this._editValue.Expectations) {
      this._editValue.Expectations = [evt.target.value];
      return;
    }

    let pos = this._editValue.Expectations.indexOf(evt.target.value);
    if (pos == -1) {
      this._editValue.Expectations.push(evt.target.value);
      return;
    }

    this._editValue.Expectations =
        this._editValue.Expectations.filter((v, i) => { return pos != i; });
  }

  _hasModifier(item, val) {
    if (!item || !item.Modifiers) {
      return false;
    }
    let ret = item.Modifiers.some((v) => { return v == val; });
    return ret;
  }

  _toggleModifier(evt) {
    ga('send', 'event', this.nodeName.toLocaleLowerCase(), 'toggle-modifier');
    if (!this._editValue.Modifiers) {
      this._editValue.Modifiers = [evt.target.value];
      return;
    }

    let pos = this._editValue.Modifiers.indexOf(evt.target.value);
    if (pos == -1) {
      this._editValue.Modifiers.push(evt.target.value);
      return;
    }

    this._editValue.Modifiers =
        this._editValue.Modifiers.filter((v, i) => { return pos != i; });
  }

  _createChangeCL(evt) {
    this.fire('create-change-cl', {
        oldValue: this.expectation,
        newValue: this._editValue,
    });
  }

  _cancelChangeCL(evt) {
    // Reset form fields to the original values.
    this.expectationChanged();
    this.fire('cancel-change-cl');
  }
}

customElements.define(SomEditExpectationForm.is, SomEditExpectationForm);
