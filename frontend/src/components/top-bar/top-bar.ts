/// <reference path="../../../bower_components/polymer/types/polymer.d.ts" />

import Utils from '../../utils';

class TopBar extends Polymer.Element {
  static get is() {
    return 'top-bar';
  }
}

window.customElements.define(TopBar.is, TopBar);
