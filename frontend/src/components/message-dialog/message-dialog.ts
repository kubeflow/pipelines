import { customElement, property } from 'polymer-decorators/src/decorators';

import 'paper-dialog/paper-dialog.html';
import 'polymer/polymer.html';
import './message-dialog.html';

@customElement('message-dialog')
export class MessageDialog extends Polymer.Element {

  @property({type: String})
  message = '';

  open() {
    (this.$.dialog as any).open();
  }

  close() {
    (this.$.dialog as any).close();
  }
}
