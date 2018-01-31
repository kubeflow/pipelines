/// <reference path="../../../bower_components/polymer/types/polymer.d.ts" />

import Utils from '../../utils';

class TopBar extends Polymer.Element {
  static get is() {
    Utils.log();
    return 'top-bar';
  }
}

window.customElements.define(TopBar.is, TopBar);
