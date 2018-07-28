import 'paper-icon-button/paper-icon-button.html';
import 'polymer/polymer-element.html';
import 'polymer/polymer.html';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';

import './table-viewer.html';

@customElement('table-viewer')
export class TableViewer extends Polymer.Element {

  @property({ type: Number })
  public pageSize = 20;

  @property({ type: Number })
  public pageIndex = 1;

  @property({ type: Array })
  public header: string[] = [];

  @property({ type: Array })
  public rows: string[][] = [];

  @property({ type: Number })
  protected _numPages = 0;

  @property({ type: Array })
  protected _pageRows: string[][] = [];

  @property({ type: Boolean })
  protected _hasNextPage = false;

  @property({ type: Boolean })
  protected _hasPrevPage = false;

  public get table(): HTMLTableElement {
    return this.$.table as HTMLTableElement;
  }

  public get pageIndexElement(): HTMLSpanElement {
    return this.$.pageIndex as HTMLSpanElement;
  }

  public get numPagesElement(): HTMLSpanElement {
    return this.$.numPages as HTMLSpanElement;
  }

  public get nextPageButton(): PaperButtonElement {
    return this.$.nextPage as PaperButtonElement;
  }

  public get prevPageButton(): PaperButtonElement {
    return this.$.prevPage as PaperButtonElement;
  }

  @observe('rows', 'pageSize')
  protected _rowsChanged(): void {
    this._numPages = Math.ceil(this.rows.length / this.pageSize);
    this._loadPage(1);
  }

  protected _loadPage(index: number): void {
    this.pageIndex = index;
    this._hasNextPage = this._hasPrevPage = true;
    if (this.pageIndex <= 1) {
      this.pageIndex = 1;
      this._hasPrevPage = false;
    } else if (this.pageIndex >= this._numPages) {
      this.pageIndex = this._numPages;
      this._hasNextPage = false;
    }
    const start = (this.pageIndex - 1) * this.pageSize;
    const end = start + this.pageSize;
    this._pageRows = this.rows.slice(start, end);
  }

  protected _nextPage(): void {
    this._loadPage(this.pageIndex + 1);
  }

  protected _prevPage(): void {
    this._loadPage(this.pageIndex - 1);
  }

}
