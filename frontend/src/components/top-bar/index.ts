import { Element as PolymerElement } from '@polymer/polymer/polymer-element';
import '../../../node_modules/@polymer/paper-button/paper-button';
import * as view from './top-bar.template.html';

export class MyTopBar extends PolymerElement {
  static get template() {
    return view;
  }
}
