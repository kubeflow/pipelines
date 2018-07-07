import { customElement, property } from 'polymer-decorators/src/decorators';

import 'iron-icon/iron-icon.html';
import 'paper-dialog/paper-dialog.html';
import 'polymer/polymer.html';
import './popup-dialog.html';

export enum DialogResult {
  BUTTON1,
  BUTTON2,
  DISMISS,
}

@customElement('popup-dialog')
export class PopupDialog extends Polymer.Element {

  @property({ type: String })
  public title = '';

  @property({ type: String })
  public body = '';

  @property({ type: String })
  public button1 = '';

  @property({ type: String })
  public button2 = '';

  private _closeCallback: (result: DialogResult) => void;

  public get dialog(): PaperDialogElement {
    return this.$.dialog as PaperDialogElement;
  }

  public open(): Promise<DialogResult> {
    return new Promise<DialogResult>((resolve) => {
      if (resolve) {
        this._closeCallback = resolve;
      }
      this.dialog.addEventListener('iron-overlay-closed', (ev: any) => {
        if (ev.detail.canceled) {
          this._closeDialog(DialogResult.DISMISS);
        }
      });
      this.dialog.open();
    });
  }

  public close(): void {
    this.dialog.close();
  }

  public _openAndCallBack(callback: (_: any) => void): void {
    if (callback) {
      this._closeCallback = callback;
    }
    this.open();
  }

  protected _closedWithButton1(): void {
    this._closeDialog(DialogResult.BUTTON1);
  }

  protected _closedWithButton2(): void {
    this._closeDialog(DialogResult.BUTTON2);
  }

  private _closeDialog(result: DialogResult): void {
    if (this._closeCallback) {
      this._closeCallback(result);
    }
    this.dialog.close();
  }
}
