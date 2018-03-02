/*
 * Copyright 2017 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 */

type ColumnType = Date|number|string;

enum ColumnTypeName {
  DATE,
  NUMBER,
  STRING,
}

interface Column {
  name: string;
  type: ColumnTypeName;
}

/**
 * Mode definition for which items, out of all items that are capable
 * of showing details, actually show those details.
 */
enum InlineDetailsDisplayMode {
  NONE,               // Don't show any inline details elements
  SINGLE_SELECT,      // Show details only when a single item is selected
  MULTIPLE_SELECT,    // Show details for all selected items
  ALL                 // Show details for all items
}
// TODO: the current behavior of ALL does not show all of the details when
// the page first renders, but only displays the details when the
// selection state for an item changes. We might want to change this
// behavior so that it figures out in advance that it should show
// all the details.
/** Fields that can be passed to the ItemListRow constructor. */
interface ItemListRowParameters {
  columns: ColumnType[];
  createDetailsElement?: () => HTMLElement;
  icon?: string;
  selected?: boolean;
}

/**
 * Object representing a row in the item list
 */
class ItemListRow {
  public selected: boolean;
  public columns: ColumnType[];
  public canShowDetails: boolean;
  public showInlineDetails = false;

  private _createDetailsElement: (() => HTMLElement) | undefined;
  private _detailsElement: HTMLElement;
  private _icon: string;

  constructor(
      {columns, icon, selected, createDetailsElement}: ItemListRowParameters) {
    this.columns = columns;
    this.selected = selected || false;
    this.canShowDetails = (!!createDetailsElement);
    this._createDetailsElement = createDetailsElement;
    this._icon = icon || '';
  }

  /**
   * If the given icon is a link, its src attribute should be set to that link,
   * and the icon attribute should be empty. If instead it's an icon name,
   * these two attributes should be reversed.
   */
  get icon() { return this._hasLinkIcon() ? '' : this._icon; }
  get src() { return this._hasLinkIcon() ? this._icon : ''; }

  /**
   * Updates our showInlineDetails flag after a selection change.
   * If we are showing details for the first time for this row,
   * create the details element and add it to the DOM.
   */
  _updateInlineDetails(
      inlineDetailsMode: InlineDetailsDisplayMode,
      multipleSelected: boolean, rowDetailsElement: HTMLElement) {
    const oldShowInlineDetails = this.showInlineDetails;
    if (!this.canShowDetails) {
      // If we don't know how to dislay details element, then we never do
      this.showInlineDetails = false;
    } else if (inlineDetailsMode === InlineDetailsDisplayMode.NONE) {
      this.showInlineDetails = false;
    } else if (inlineDetailsMode === InlineDetailsDisplayMode.ALL) {
      this.showInlineDetails = true;
    } else if (!this.selected) {
      this.showInlineDetails = false;
    } else if (!multipleSelected) {
      this.showInlineDetails = true;
    } else if (inlineDetailsMode === InlineDetailsDisplayMode.MULTIPLE_SELECT) {
      this.showInlineDetails = true;
    } else {
      // Assume SINGLE_SELECT, but we know multiple items are selected
      this.showInlineDetails = false;
    }
    if (this.showInlineDetails) {
      if (!this._detailsElement) {
        this._addDetailsElement(rowDetailsElement);
      } else if (!oldShowInlineDetails) {
        const detailsElementAsAny = this._detailsElement as any;
        if (detailsElementAsAny.show) {
          detailsElementAsAny.show();
        }
      }
    }
  }

  // Create and add the details element for this item when we first display it.
  private _addDetailsElement(rowDetailsElement: HTMLElement) {
    if (!this._createDetailsElement) {
      return;
    }

    // The list can get reused when we switch to display a different directory,
    // so we need to clear potential old details.
    Utils.deleteAllChildren(rowDetailsElement);

    // Create and add the new details element
    this._detailsElement = this._createDetailsElement();
    rowDetailsElement.appendChild(this._detailsElement);
  }

  private _hasLinkIcon() {
    return this._icon.startsWith('http://') || this._icon.startsWith('https://');
  }
}

/**
 * CustomEvent that gets dispatched when an item is clicked or double clicked
 */
class ItemClickEvent extends CustomEvent<any> {
  detail: {
    index: number;
  };
}

/**
 * Multi-column list element.
 * This element takes a list of column names, and a list of row objects,
 * each containing values for each of the columns, an icon name, and a selected
 * property. The items are displayed in a table form. Clicking an item selects
 * it and unselects all other items. Clicking the checkbox next to an item
 * allows for multi-selection. Shift and ctrl keys can also be used to select
 * multiple items.
 * Double clicking an item fires an 'ItemClickEvent' event with this item's index.
 * Selecting an item by single clicking it changes the selectedIndices
 * property. This also notifies the host, which can listen to the
 * selected-indices-changed event.
 * If the "hide-header" attribute is specified, the header is hidden.
 * If the "disable-selection" attribute is specified, the checkboxes are
 * hidden, and clicking items does not select them.
 */
@Polymer.decorators.customElement('item-list')
class ItemListElement extends Polymer.Element {

  /**
   * List of data rows, each implementing the row interface
   */
  @Polymer.decorators.property({type: Array})
  public rows: ItemListRow[] = [];

  /**
   * List of string data columns names
   */
  @Polymer.decorators.property({type: Array})
  public columns: Column[] = [];

  /**
   * Whether to hide the header row
   */
  @Polymer.decorators.property({type: Boolean})
  public hideHeader = false;

  /**
   * Whether to disable item selection
   */
  @Polymer.decorators.property({type: Boolean})
  public disableSelection = false;

  /**
   * Whether to disable multi-selection
   */
  @Polymer.decorators.property({type: Boolean})
  public noMultiselect = false;

  /**
   * The list of currently selected indices
   */
  @Polymer.decorators.property({
      computed: '_computeSelectedIndices(rows.*)', notify: true, type: Array})
  public selectedIndices: number[] = [];

  /**
   * Display mode for inline details
   */
  @Polymer.decorators.property({type: Object})
  public inlineDetailsMode = InlineDetailsDisplayMode.NONE;

  @Polymer.decorators.property({type: Boolean})
  _showFilterBox = false;

  @Polymer.decorators.property({
      computed: '_computeIsAllSelected(selectedIndices)', type: Boolean})
  _isAllSelected: boolean;

  @Polymer.decorators.property({
      computed: '_computeHideCheckboxes(disableSelection, noMultiselect)', type: Boolean})
  _hideCheckboxes: boolean;

  _filterString: string;

  private _currentSort = {
    asc: true,   // ascending or descending
    column: -1,  // index of current sort column
  };
  private _lastSelectedIndex = -1;

  ready() {
    super.ready();

    // Add box-shadow to header container on scroll
    const container = this.$.listContainer as HTMLDivElement;
    const headerContainer = this.$.headerContainer as HTMLDivElement;
    container.addEventListener('scroll', () => {
      const yOffset = Math.min(container.scrollTop / 20, 5);
      const shadow = '0px ' + yOffset + 'px 10px -5px #ccc';
      headerContainer.style.boxShadow = shadow;
    });

    // Initial sort
    this._sortBy(0);
  }

  resetFilter() {
    this._filterString = '';
  }

  _formatColumnValue(value: ColumnType, i: number, columns: Column[]): string {
    if (columns[i]) {
      if (columns[i].type === ColumnTypeName.DATE) {
        return (value as Date).toLocaleString();
      } else {
        return value.toString();
      }
    } else {
      return '';
    }
  }

  _columnButtonClicked(e: any) {
    this._sortBy(e.model.itemsIndex);
  }

  _sortBy(column: number) {
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

    this.$.list.sort = (a: ItemListRow, b: ItemListRow) => {
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
    this._updateSortIcons();
  }

  @Polymer.decorators.observe(['rows', 'columns'])
  _updateSortIcons() {
    // Make sure all elements have rendered.
    Polymer.dom.flush();
    const iconEls = this.$.header.querySelectorAll('.sort-icon');
    if (iconEls.length && this._currentSort.column > -1) {
      iconEls.forEach((el: HTMLElement) => el.hidden = true);
      iconEls[this._currentSort.column].hidden = false;
      iconEls[this._currentSort.column].setAttribute('icon',
          this._currentSort.asc ? 'arrow-upward' : 'arrow-downward');
    }
  }

  _toggleFilter() {
    this._showFilterBox = !this._showFilterBox;

    // If the filter box is now visible, focus it.
    // If not, reset the filter to go back to showing the full list.
    if (this._showFilterBox) {
      this.$.filterBox.focus();
    } else {
      this._filterString = '';
    }
  }

  _computeFilter(filterString: string) {
    if (!filterString) {
      // set filter to null to disable filtering
      return null;
    } else {
      // return a filter function for the current search string
      filterString = filterString.toLowerCase();
      return (item: ItemListRow) => {
          const strVal = this._formatColumnValue(item.columns[0], 0, this.columns);
          return strVal.toLowerCase().indexOf(filterString) > -1;
      };
    }
  }

  /**
   * Returns value for the computed property selectedIndices, which is the list
   * of indices of the currently selected items.
   */
  _computeSelectedIndices() {
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
  _computeIsAllSelected() {
    return this.rows.length > 0 && this.rows.length === this.selectedIndices.length;
  }

  /**
   * Returns the value for the computed property hideCheckboxes.
   */
  _computeHideCheckboxes(disableSelection: boolean, noMultiselect: boolean) {
    return disableSelection || noMultiselect;
  }

  /**
   * Selects an item in the list using its display index. Note the item must be
   * visible in the rendered list.
   * @param index display index of item to select
   * @param single true if we are not being called in a bulk operation
   */
  _selectItemByDisplayIndex(index: number, single?: boolean) {
    const realIndex = this._displayIndexToRealIndex(index);
    this._selectItemByRealIndex(realIndex, single);
  }

  /**
   * Unselects an item in the list using its display index. Note the item must be
   * visible in the rendered list.
   * @param index display index of item to unselect
   * @param single true if we are not being called in a bulk operation
   */
  _unselectItemByDisplayIndex(index: number, single?: boolean) {
    const realIndex = this._displayIndexToRealIndex(index);
    this._unselectItemByRealIndex(realIndex, single);
  }

  /**
   * Selects an item in the list using its real index.
   */
  _selectItemByRealIndex(realIndex: number, single?: boolean) {
    if (this.rows[realIndex].selected && !single) {
      return;   // Avoid lots of useless work when no change.
    }
    this.set('rows.' + realIndex + '.selected', true);
    this._updateItemSelection(realIndex, true);
  }

  /**
   * Unselects an item in the list using its real index.
   */
  _unselectItemByRealIndex(realIndex: number, single?: boolean) {
    if (!this.rows[realIndex].selected && !single) {
      return;   // Avoid lots of useless work when no change.
    }
    this.set('rows.' + realIndex + '.selected', false);
    this._updateItemSelection(realIndex, false);
  }

  /**
   * Updates the show-details flag for a row after selection change.
   */
  _updateItemSelection(index: number, newValue: boolean) {
    const multipleSelected = this.selectedIndices.length > 1;
    this._updateInlineDetailsForRow(index, multipleSelected);
    const previousSelectedCount =
        this.selectedIndices.length + (newValue ? -1 : 1);
    const previousMultipleSelected = previousSelectedCount > 1;
    if (this.inlineDetailsMode === InlineDetailsDisplayMode.SINGLE_SELECT &&
        multipleSelected !== previousMultipleSelected) {
      /** If we are in SINGLE_SELECT mode and we have changed from having one
       * item selected to many or vice-versa, we need to update all the other
       * selected items.
       */
      for (let i = 0; i < this.rows.length; i++) {
        if (i !== index) {
          this._updateInlineDetailsForRow(i, multipleSelected);
        }
      }
    }
  }

  /**
   * Updates the inline details for one row after a selection change.
   */
  _updateInlineDetailsForRow(index: number, multipleSelected: boolean) {
    const rowDetailsElement = this._getRowDetailsContainer(index);
    this.rows[index]._updateInlineDetails(
        this.inlineDetailsMode, multipleSelected, rowDetailsElement);
    this.notifyPath('rows.' + index + '.showInlineDetails',
        this.rows[index].showInlineDetails);
  }

  /**
   * Selects all displayed items in the list.
   */
  _selectAllDisplayedItems() {
    const allElements = this.$.listContainer.querySelectorAll('paper-item') as NodeList;
    allElements.forEach((_, i) => this._selectItemByDisplayIndex(i));
  }

  /**
   * Unselects all displayed items in the list.
   */
  _unselectAllDisplayedItems() {
    const allElements = this.$.listContainer.querySelectorAll('paper-item') as NodeList;
    allElements.forEach((_, i) => this._unselectItemByDisplayIndex(i));
  }

  /**
   * Selects all items in the list.
   */
  _selectAll() {
    for (let i = 0; i < this.rows.length; ++i) {
      this._selectItemByRealIndex(i);
    }
  }

  /**
   * Unselects all items in the list.
   */
  _unselectAll() {
    for (let i = 0; i < this.rows.length; ++i) {
      this._unselectItemByRealIndex(i);
    }
  }

  /**
   * Called when the select/unselect all checkbox checked value is changed.
   */
  _selectAllChanged() {
    if (this.$.selectAllCheckbox.checked === true) {
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
  _rowClicked(e: MouseEvent) {
    if (this.disableSelection) {
      return;
    }
    const target = e.target as HTMLDivElement;
    const displayIndex = this.$.list.indexForElement(target);
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
  _rowDoubleClicked(e: MouseEvent) {
    const realIndex = this.$.list.modelForElement(e.target).itemsIndex;
    const ev = new ItemClickEvent('itemDoubleClick', { detail: {index: realIndex} });
    this.dispatchEvent(ev);
  }

  private _getRowDetailsContainer(index: number) {
    const nthDivSelector = 'div.row-details:nth-of-type(' + (index + 1) + ')';
    return this.$.listContainer.querySelector(nthDivSelector);
  }

  private _displayIndexToRealIndex(index: number) {
    const element = this.$.listContainer.querySelector(
        'paper-item:nth-of-type(' + (index + 1 ) + ')');
    return this.$.list.modelForElement(element).itemsIndex;
  }

}
