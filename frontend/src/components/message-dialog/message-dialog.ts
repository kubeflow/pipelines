import { customElement, property } from 'polymer-decorators/src/decorators';

import 'paper-dialog/paper-dialog.html';
import 'polymer/polymer.html';
import './message-dialog.html';

@customElement('message-dialog')
export class MessageDialog extends Polymer.Element {

  @property({type: String})
  message = '';

  open(): void {
    (this.$.dialog as PaperDialogElement).open();
  }

  close(): void {
    (this.$.dialog as PaperDialogElement).close();
  }
}
