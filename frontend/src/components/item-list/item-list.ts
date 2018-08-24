// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import 'iron-icons/av-icons.html';
import 'iron-icons/device-icons.html';
import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-dropdown-menu/paper-dropdown-menu.html';
import 'paper-icon-button/paper-icon-button.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import {
  ItemDblClickEvent,
  ListFormatChangeEvent,
  NewListPageEvent,
} from '../../model/events';

import './item-list.html';

export type ColumnType = Date|number|string|undefined;

export enum ColumnTypeName {
  DATE,
  NUMBER,
  STRING,
}

export class ItemListColumn {
  public name: string;
  public sortKey: string;
  public type: ColumnTypeName;
  public flex: number;

  constructor(name: string, type: ColumnTypeName, sortKey = '', flex = 1) {
    this.name = name;
    this.type = type;
    this.sortKey = sortKey;
    this.flex = flex;
  }
}

/** Fields that can be passed to the ItemListRow constructor. */
interface ItemListRowParameters {
  columns: ColumnType[];
  icon?: string;
  selected?: boolean;
}

/**
 * Object representing a row in the item list
 */
export class ItemListRow {
  public selected: boolean;
  public columns: ColumnType[];

  private _icon: string;

  constructor({columns, icon, selected}: ItemListRowParameters) {
    this.columns = columns;
    this.selected = selected || false;
    this._icon = icon || '';
  }

  /**
   * If the given icon is a link, its src attribute should be set to that link,
   * and the icon attribute should be empty. If instead it's an icon name,
   * these two attributes should be reversed.
   */
  get icon(): string { return this._hasLinkIcon() ? '' : this._icon; }
  get src(): string { return this._hasLinkIcon() ? this._icon : ''; }

  private _hasLinkIcon(): boolean {
    return this._icon.startsWith('http://') || this._icon.startsWith('https://');
  }
}

/**
 * Multi-column list element.
 * This element takes a list of column names, and a list of row objects,
 * each containing values for each of the columns, an icon name, and a selected
 * property. The items are displayed in a table form. Clicking an item selects
 * it and unselects all other items. Clicking the checkbox next to an item
 * allows for multi-selection. Shift and ctrl keys can also be used to select
 * multiple items.
 * Double clicking an item fires an 'ItemDblClickEvent' event with this item's index.
 * Selecting an item by single clicking it changes the selectedIndices
 * property. This also notifies the host, which can listen to the
 * selected-indices-changed event.
 * If the "hide-header" attribute is specified, the header is hidden.
 * If the "disable-selection" attribute is specified, the checkboxes are
 * hidden, and clicking items does not select them.
 */
@customElement('item-list')
export class ItemListElement extends Polymer.Element {

  /**
   * List of data rows, each implementing the row interface
   */
  @property({ type: Array })
  public rows: ItemListRow[] = [];

  /**
   * List of string data columns names
   */
  @property({ type: Array })
  public columns: ItemListColumn[] = [];

  /**
   * Whether to hide the header row
   */
  @property({ type: Boolean })
  public hideHeader = false;

  /**
   * Whether to disable item selection
   */
  @property({ type: Boolean })
  public disableSelection = false;

  /**
   * Whether to disable multi-selection
   */
  @property({ type: Boolean })
  public noMultiselect = false;

  /**
   * The list of currently selected indices
   */
  @property({ computed: '_computeSelectedIndices(rows.*)', notify: true, type: Array })
  public selectedIndices: number[] = [];

  @property({ type: String })
  public filterString = '';

  /**
   * If true, the ItemListElement component will handle filtering of its data.
   * If false, the ItemListElement will do no filtering, however, its parent component can still
   * listen for ListFormatChangeEvent events and handle them as desired.
   */
  @property({ type: Boolean })
  public filterLocally = false;

  /**
   * If true, the ItemListElement component will handle sorting of its data.
   * If false, the ItemListElement will do no sorting, however, its parent component can still
   * listen for ListFormatChangeEvent events and handle them as desired.
   */
  @property({ type: Boolean })
  public sortLocally = false;

  /**
   * Possible values for number of rows shown per page
   */
  @property({ type: Array })
  public pageSizes: number[] = [ 20, 50, 100 ];

  /**
   * Selected number of rows to show per page. Defaults to 20 items per page.
   */
  @property({ type: Number })
  public selectedPageSize = this.pageSizes[0];

  /**
   * Message to show if the list has no items
   */
  @property({ type: String })
  public emptyMessage = '';

  @property({ type: Boolean })
  protected _showFilterBox = false;

  @property({ computed: '_computeIsAllSelected(selectedIndices)', type: Boolean })
  protected _isAllSelected = false;

  @property({ computed: '_computeHideCheckboxes(disableSelection, noMultiselect)', type: Boolean })
  protected _hideCheckboxes = false;

  @property({
    computed: '_computeDisableNextPageButton(_currentPage, _maxPageNumber)',
    type: Boolean
  })
  protected _disableNextPageButton = false;

  @property({ type: Number })
  protected _currentPage = 0;

  @property({ type: Number })
  protected _maxPageNumber = Number.MAX_SAFE_INTEGER;

  private _currentSort = {
    asc: true,   // ascending or descending
    column: -1,  // index of current sort column
  };
  private _lastSelectedIndex = -1;

  private _pageTokens = [''];

  private _sortByColumn = '';

  public get emptyMessageSpan(): HTMLSpanElement {
    const root = this.shadowRoot as ShadowRoot;
    return root.querySelector('.empty-message') as HTMLSpanElement;
  }

  public get filterBox(): HTMLElement {
    return this.$.filterBox as HTMLElement;
  }

  public get filterToggleButton(): PaperIconButtonElement {
    return this.$.filterToggle as PaperIconButtonElement;
  }

  public get header(): HTMLElement {
    return this.$.header as HTMLElement;
  }

  public get list(): Polymer.DomRepeat {
    return this.$.list as Polymer.DomRepeat;
  }

  public get listContainer(): HTMLElement {
    return this.$.listContainer as HTMLElement;
  }

  public get nextPageButton(): PaperButtonElement {
    return this.$.nextPage as PaperButtonElement;
  }

  public get previousPageButton(): PaperButtonElement {
    return this.$.previousPage as PaperButtonElement;
  }

  public get selectAllCheckbox(): PaperCheckboxElement {
    return this.$.selectAllCheckbox as PaperCheckboxElement;
  }

  public reset(): void {
    this._pageTokens = [''];
    this._maxPageNumber = Number.MAX_SAFE_INTEGER;
    this._currentPage = 0;
  }

  public updateNextPageToken(nextPageToken: string): void {
    if (nextPageToken) {
      // If we're using the greatest yet known page, then the pageToken will be new.
      if (this._currentPage + 1 === this._pageTokens.length) {
        this._pageTokens.push(nextPageToken);
      }
    } else {
      this._maxPageNumber = this._currentPage;
    }
  }

  public ready(): void {
    super.ready();

    // Add box-shadow to header container on scroll
    const container = this.$.listContainer as HTMLDivElement;
    const headerContainer = this.$.headerContainer as HTMLDivElement;
    container.addEventListener('scroll', () => {
      const yOffset = Math.min(container.scrollTop / 20, 5);
      const shadow = '0px ' + yOffset + 'px 10px -5px #ccc';
      headerContainer.style.boxShadow = shadow;
    });

    // Ensure items begin sorted when not relying on the backend to handling sorting.
    if (this.sortLocally) {
      this._sortBy(0);
    }
  }

  /**
   * Returns the cell Div element at the specified row and column indexes, 1-based.
   */
  public getCellElement(rowIndex: number, colIndex: number): HTMLDivElement {
    const rowEl = this.$.listContainer.querySelector(`.row:nth-of-type(${rowIndex})`);
    if (!rowEl) {
      throw new Error('Could not find row ' + rowIndex);
    }
    // First column is taken up by the checkbox
    const colEl = rowEl.querySelector(`.column:nth-of-type(${colIndex + 1})`);
    if (!colEl) {
      throw new Error('Could not find column ' + colIndex);
    }
    return colEl as HTMLDivElement;
  }

  /**
   * Visible for testing.
   */
  public _sortBy(column: number): void {
    // If sort is requested on the current sort column, reverse the sort order.
    // Otherwise, set the current sort column to that.
    if (this._currentSort.column === column) {
      this._currentSort.asc = !this._currentSort.asc;
    } else {
      this._currentSort = {
        asc: true,
        column,
      };
    }

    if (this.sortLocally) {
      (this.$.list as Polymer.DomRepeat).sort = (a: ItemListRow, b: ItemListRow) => {
        // Bail out of sort if no columns have been set yet.
        if (!this.columns.length) {
          return;
        }
        if (a.columns[column] === b.columns[column]) {
          return 0;
        }
        let compResult = -1;
        if (this.columns[column].type === ColumnTypeName.STRING) {
          if ((a.columns[column] as string).toLowerCase() >
              (b.columns[column] as string).toLowerCase()) {
            compResult = 1;
          }
        } else if (this.columns[column].type === ColumnTypeName.NUMBER) {
          if ((a.columns[column] as number) > (b.columns[column] as number)) {
            compResult = 1;
          }
        } else if (this.columns[column].type === ColumnTypeName.DATE) {
          if ((a.columns[column] as Date) > (b.columns[column] as Date)) {
            compResult = 1;
          }
        }
        return this._currentSort.asc ? compResult : compResult * -1;
      };
    } else {
      // Allow backend to handle sorting
      this._sortByColumn = this.columns[column].sortKey;
      // Reset paging.
      this.reset();
      this.dispatchEvent(
          new ListFormatChangeEvent(
              this.filterString,
              this._currentSort.asc,
              this.selectedPageSize,
              this._sortByColumn));
    }
    this._updateSortIcons();
  }

  /**
   * Selects all displayed items in the list.
   *
   * Visible for tests.
   */
  public _selectAllDisplayedItems(): void {
    const allElements = this.$.listContainer.querySelectorAll('paper-item') as NodeList;
    allElements.forEach((_, i) => this._selectItemByDisplayIndex(i));
  }

  /**
   * Selects an item in the list using its display index. Note the item must be
   * visible in the rendered list.
   *
   * Visible for testing.
   *
   * @param index display index of item to select
   * @param single true if we are not being called in a bulk operation
   */
  public _selectItemByDisplayIndex(index: number, single?: boolean): void {
    const realIndex = this._displayIndexToRealIndex(index);
    this._selectItemByRealIndex(realIndex, single);
  }

  /**
   * Selects all items in the list.
   *
   * Visible for testing.
   */
  public _selectAll(): void {
    for (let i = 0; i < this.rows.length; ++i) {
      this._selectItemByRealIndex(i);
    }
  }

  /**
   * Selects an item in the list using its real index.
   *
   * Visible for testing.
   */
  public _selectItemByRealIndex(realIndex: number, single?: boolean): void {
    if (this.rows[realIndex].selected && !single) {
      return;   // Avoid lots of useless work when no change.
    }
    this.set('rows.' + realIndex + '.selected', true);
  }

  /**
   * Unselects an item in the list using its display index. Note the item must be
   * visible in the rendered list.
   *
   * Visible for testing.
   *
   * @param index display index of item to unselect
   * @param single true if we are not being called in a bulk operation
   */
  public _unselectItemByDisplayIndex(index: number, single?: boolean): void {
    const realIndex = this._displayIndexToRealIndex(index);
    this._unselectItemByRealIndex(realIndex, single);
  }

  /**
   * Custom render function to paint custom HTML, will be called for each cell
   * in the item list.
   * Note: The output of this function is inlined as innerHTML in the
   * dom-repeat template, so all user data must be sanitized before used in any
   * HTML passed in here.
   * @param value column value passed in from the dom-repeat template
   * @param colIndex index of column
   * @param rowIndex index of row. This isn't used in this default
   *    implementation, but is there to allow overloading functions to customize
   *    rendering per row.
   */
  public renderColumn(value: ColumnType, colIndex: number, rowIndex: number): string {
    if (this.columns[colIndex] && value) {
      if (this.columns[colIndex].type === ColumnTypeName.DATE) {
        return (value as Date).toLocaleString();
      } else {
        return value.toString();
      }
    } else {
      return '-';
    }
  }

  protected _columnButtonClicked(e: any): void {
    this._sortBy(e.model.itemsIndex);
  }

  @observe('rows', 'columns')
  protected _updateSortIcons(): void {
    // Make sure all elements have rendered.
    Polymer.flush();
    const iconEls = this.$.header.querySelectorAll('.sort-icon') as NodeListOf<HTMLElement>;
    if (iconEls.length && this._currentSort.column > -1) {
      iconEls.forEach((el: HTMLElement) => el.hidden = true);
      iconEls[this._currentSort.column].hidden = false;
      iconEls[this._currentSort.column].setAttribute('icon',
          this._currentSort.asc ? 'arrow-upward' : 'arrow-downward');
    }
  }

  protected _getColumnFlex(index: number): number {
    return this.columns.length ? this.columns[index].flex : 1;
  }

  protected _toggleFilter(): void {
    this._showFilterBox = !this._showFilterBox;

    // If the filter box is now visible, focus it.
    // If not, reset the filter to go back to showing the full list.
    if (this._showFilterBox) {
      (this.$.filterBox as HTMLElement).focus();
    } else {
      this.filterString = '';
    }
  }

  protected _computeFilter(filterString: string): Function | null {
    if (!filterString || !this.filterLocally) {
      // set filter to null to disable filtering
      return null;
    } else {
      // return a filter function for the current search string
      filterString = filterString.toLowerCase();
      return (item: ItemListRow) => {
        const strVal = this.renderColumn(item.columns[0], 0, -1);
        return strVal.toLowerCase().indexOf(filterString) > -1;
      };
    }
  }

  /**
   * Returns value for the computed property selectedIndices, which is the list
   * of indices of the currently selected items.
   */
  protected _computeSelectedIndices(): number[] {
    const selected: number[] = [];
    this.rows.forEach((row, i) => {
      if (row.selected) {
        selected.push(i);
      }
    });
    return selected;
  }

  /**
   * Returns the value for the computed property isAllSelected, which is whether
   * all items in the list are selected.
   */
  protected _computeIsAllSelected(): boolean {
    return this.rows.length > 0 && this.rows.length === this.selectedIndices.length;
  }

  /**
   * Returns the value for the computed property hideCheckboxes.
   */
  protected _computeHideCheckboxes(disableSelection: boolean, noMultiselect: boolean): boolean {
    return disableSelection || noMultiselect;
  }

  /**
   * Returns whether or not the nextPageButton should be disabled.
   */
  protected _computeDisableNextPageButton(currentPage: number, maxPageNumber: number): boolean {
    return currentPage === maxPageNumber;
  }

  /**
   * Unselects an item in the list using its real index.
   */
  protected _unselectItemByRealIndex(realIndex: number, single?: boolean): void {
    if (!this.rows[realIndex].selected && !single) {
      return;   // Avoid lots of useless work when no change.
    }
    this.set('rows.' + realIndex + '.selected', false);
  }

  /**
   * Unselects all displayed items in the list.
   */
  protected _unselectAllDisplayedItems(): void {
    const allElements = this.$.listContainer.querySelectorAll('paper-item') as NodeList;
    allElements.forEach((_, i) => this._unselectItemByDisplayIndex(i));
  }

  /**
   * Unselects all items in the list.
   */
  protected _unselectAll(): void {
    for (let i = 0; i < this.rows.length; ++i) {
      this._unselectItemByRealIndex(i);
    }
  }

  /**
   * Called when the select/unselect all checkbox checked value is changed.
   */
  protected _selectAllChanged(): void {
    if ((this.$.selectAllCheckbox as HTMLInputElement).checked === true) {
      this._selectAllDisplayedItems();
    } else {
      this._unselectAllDisplayedItems();
    }
  }

  /**
   * On row click, checks the click target, if it's the checkbox, adds it to
   * the selected rows, otherwise selects it only.
   * Note the distinction between displayIndex, which is the index of the item
   * in the rendered list, and realIndex, which is the index of the item in the
   * original list that was submitted to the item-list element. These might be
   * different when filtering or sorting.
   */
  protected _rowClicked(e: MouseEvent): void {
    if (this.disableSelection) {
      return;
    }
    const target = e.target as HTMLDivElement;
    const displayIndex = (this.$.list as Polymer.DomRepeat).indexForElement(target) || 0;
    const realIndex = this._displayIndexToRealIndex(displayIndex);

    // If shift key is pressed and we had saved the last selected index, select
    // all items from this index till the last selected.
    if (!this.noMultiselect && e.shiftKey && this._lastSelectedIndex !== -1 &&
        this.selectedIndices.length > 0) {
      this._unselectAll();
      const start = Math.min(this._lastSelectedIndex, displayIndex);
      const end = Math.max(this._lastSelectedIndex, displayIndex);
      for (let i = start; i <= end; ++i) {
        this._selectItemByDisplayIndex(i);
      }
    } else if (!this.noMultiselect && (e.ctrlKey || e.metaKey)) {
      // If ctrl (or Meta for MacOS) key is pressed, toggle its selection.
      if (this.rows[realIndex].selected === false) {
        this._selectItemByDisplayIndex(displayIndex, true);
      } else {
        this._unselectItemByDisplayIndex(displayIndex, true);
      }
    } else {
      // No modifier keys are pressed, proceed normally to select/unselect the item.
      // If the clicked element is the checkbox, the checkbox already toggles selection in
      // the UI, so change the item's selection state to match the checkbox's new value.
      // Otherwise, select this element, unselect all others.
      if (target.tagName === 'PAPER-CHECKBOX') {
        if (this.rows[realIndex].selected === false) {
          // Remove this element from the selected elements list if it's being unselected
          this._unselectItemByDisplayIndex(displayIndex, true);
        } else {
          // Add this element to the selected elements list if it's being selected,
          this._selectItemByDisplayIndex(displayIndex, true);
        }
      } else {
        this._unselectAll();
        this._selectItemByDisplayIndex(displayIndex, true);
      }
    }

    // Save this index to enable multi-selection using shift later.
    this._lastSelectedIndex = displayIndex;
  }

  /**
   * On row double click, fires an event with the clicked item's index.
   */
  protected _rowDoubleClicked(e: MouseEvent): void {
    const displayIndex =
      (this.$.list as Polymer.DomRepeat).indexForElement(e.target as HTMLElement) || 0;
    const realIndex = this._displayIndexToRealIndex(displayIndex);
    this.dispatchEvent(new ItemDblClickEvent(realIndex));
  }

  /**
   * SortBy is slightly more complicated, so it is handled within the _sortBy() function instead.
   */
  @observe('filterString', 'selectedPageSize')
  protected _listFormatChanged(): void {
    this.reset();
    this.dispatchEvent(
        new ListFormatChangeEvent(
            this.filterString, this._currentSort.asc, this.selectedPageSize, this._sortByColumn)
    );
  }

  protected _previousPage(): void {
    this._currentPage--;
    this._loadNewPage();
  }

  protected _nextPage(): void {
    if (this._currentPage < this._maxPageNumber) {
      this._currentPage++;
    }
    this._loadNewPage();
  }

  private _loadNewPage(): void {
    this.dispatchEvent(
        new NewListPageEvent(
            this.filterString,
            this._currentPage,
            this.selectedPageSize,
            this._pageTokens[this._currentPage],
            this._sortByColumn));
  }

  private _displayIndexToRealIndex(index: number): number {
    const element = this.$.listContainer.querySelector(
        'paper-item:nth-of-type(' + (index + 1 ) + ')') as HTMLElement;
    return ((this.$.list as Polymer.DomRepeat).modelForElement(element) as any).itemsIndex;
  }
}
