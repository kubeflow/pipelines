import { customElement, property } from 'polymer-decorators/src/decorators';

import 'iron-icon/iron-icon.html';
import 'paper-button/paper-button.html';
import 'polymer/polymer.html';

import './page-error.html';

@customElement('page-error')
export class PageError extends Polymer.Element {
  @property({ type: String })
  public error = '';

  @property({ type: Boolean })
  public showButton = true;

  protected _refresh(): void {
    location.reload();
  }
}
