/// <reference path="../../bower_components/polymer/types/polymer.d.ts" />

import { customElement, property } from 'polymer-decorators/src/decorators';

@customElement('page-element')
export class PageElement extends Polymer.Element {

  @property({type: String})
  protected _pageError = '';

  @property({type: String})
  protected _pageErrorDetails = '';

  public load(path: string, queryParams: {}, data?: {}): void {
    throw new Error('No implementation in abstract class');
  }

  public showPageError(error: string, details?: string): void {
    this._pageError = error;
    if (details) {
      this._pageErrorDetails = details;
    }
  }
}
