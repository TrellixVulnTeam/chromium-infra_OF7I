'use strict';

class SomSettings extends Polymer.Element {

  static get is() {
    return 'som-settings';
  }

  static get properties() {
    return {
      collapseByDefault: {
        type: Boolean,
        notify: true,
      },
      defaultTree: {
        type: String,
        notify: true,
      },
    };
  }
}

customElements.define(SomSettings.is, SomSettings);
