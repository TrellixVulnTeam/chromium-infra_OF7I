'use strict';

class SomDisableButton extends Polymer.Element {
  static get is() {
    return 'som-disable-button';
  }

  static get properties() {
    return {
      bugs: Array,
      testName: String,
      _wasPressed: {
        type: Boolean,
        value: false,
      },
    };
  }

  // When the button is clicked, copy a command to the clipboard which will
  // disable the test.
  _handleClick(evt) {
    let command = "tools/disable_tests/disable '" + this.testName + "'";
    if (this.bugs.length === 1) {
      // Add the bug ID if present. Only do it if there's exactly one. If there
      // are more than one we don't know which one to use.
      command += " -b " + this.bugs[0].id;
    }

    navigator.clipboard.writeText(command).catch(function(err) {
      console.log(err);
    });

    // Make a local binding so that the closure below can capture it. "this"
    // within the closure refers to something else.
    let _this = this;
    _this._wasPressed = true;
    setTimeout(function() { _this._wasPressed = false; }, 1000);
  }
}

customElements.define(SomDisableButton.is, SomDisableButton);
