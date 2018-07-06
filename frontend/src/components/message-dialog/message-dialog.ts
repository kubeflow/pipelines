import { customElement, property } from 'polymer-decorators/src/decorators';

import 'iron-collapse/iron-collapse.html';
import 'iron-icon/iron-icon.html';
import 'neon-animation/web-animations.html';
import 'paper-dialog/paper-dialog.html';
import 'polymer/polymer.html';
import './message-dialog.html';

@customElement('message-dialog')
export class MessageDialog extends Polymer.Element {

  @property({ type: String })
  public message = '';

  @property({ type: String })
  public details = '';

  public open(): void {
    (this.$.dialog as PaperDialogElement).open();
  }

  public close(): void {
    (this.$.dialog as PaperDialogElement).close();
  }

  public toggleDetails(): void {
    (this.$.detailsCollapse as IronCollapseElement).toggle();
    (this.$.dialog as PaperDialogElement).classList.toggle('expanded');
    (this.$.dialog as PaperDialogElement).notifyResize();
  }
}
