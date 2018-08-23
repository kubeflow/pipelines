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

import '../../src/components/item-list/item-list';

import { assert } from 'chai';
import {
  ColumnTypeName,
  ItemListColumn,
  ItemListElement,
  ItemListRow,
} from '../../src/components/item-list/item-list';
import { NewListPageEvent } from '../../src/model/events';
import { resetFixture } from './test-utils';

let fixture: ItemListElement;

async function _resetFixture(filterLocally = false, sortLocally = false): Promise<void> {
  return resetFixture('item-list', (f: ItemListElement) => {
    f.filterLocally = filterLocally;
    f.sortLocally = sortLocally;
    fixture = f;
  });
}

describe('item-list', () => {
  /**
   * Returns true iff the item at the specified index is checked
   * @param i index of item to check
   */
  function isSelected(i: number): boolean {
    const row = getRow(i);
    const box = getCheckbox(i);
    // Check three things: the index is in the selectedIndices array, the row
    // has the selected attribute, and its checkbox is checked. If any of these
    // is missing then all of them should be false, otherwise we're in an
    // inconsistent state.
    if (fixture.selectedIndices.indexOf(i) > -1
      && row.hasAttribute('selected')
      && box.checked) {
      return true;
    } else if (fixture.selectedIndices.indexOf(i) === -1
      && !row.hasAttribute('selected')
      && !box.checked) {
      return false;
    } else {
      throw new Error('inconsistent state for row ' + i);
    }
  }

  /**
   * Returns the row element at the specified index
   * @param i index of row whose element to return
   */
  function getRow(i: number): HTMLElement {
    const listContainer = fixture.listContainer;
    return listContainer.querySelectorAll('paper-item')[i];
  }

  /**
   * Returns the checkbox element at the specified index
   * @param i index of row whose checkbox element to return
   */
  function getCheckbox(i: number): any {
    const row = getRow(i);
    return row.querySelector('paper-checkbox');
  }

  /**
   * Rows must be recreated on each test with the fixture, to avoid state leakage.
   */
  beforeEach(() => {
    _resetFixture();
    fixture.columns = [
      new ItemListColumn('col1', ColumnTypeName.STRING),
      new ItemListColumn('col2', ColumnTypeName.STRING),
    ];
    const rows = [
      new ItemListRow({
        columns: ['first column 1', 'second column 1'],
      }),
      new ItemListRow({
        columns: ['first column 2', 'second column 2'],
      }),
      new ItemListRow({
        columns: ['first column 3', 'second column 3'],
      }),
      new ItemListRow({
        columns: ['first column 4', 'second column 4'],
      }),
      new ItemListRow({
        columns: ['first column 5', 'second column 5'],
      }),
    ];
    fixture.rows = rows;
    Polymer.flush();
  });

  it('shows empty message iff there are no items', () => {
    const emptyMessage = 'test empty message';
    fixture.emptyMessage = emptyMessage;
    assert.strictEqual(fixture.emptyMessageSpan.innerText, '', 'No empty message should show');
    fixture.rows = [];
    Polymer.flush();
    assert.strictEqual(fixture.emptyMessageSpan.innerText, emptyMessage,
        'Empty message should show after emptying items');
  });

  it('displays a row per item in list', () => {
    const rows = fixture.listContainer.querySelectorAll('.row');
    assert.strictEqual(rows.length, 5, 'five rows should appear');
  });

  it('returns the right cell at row and column indexes', () => {
    for (let i = 0; i < 4; ++i) {
      const [row, col] = [Math.round(i / 2), i % 2];
      const el = fixture.getCellElement(row + 1, col + 1);
      assert.strictEqual(el.innerText, fixture.rows[row].columns[col]);
    }
  });

  it('throws an error if bad row/column index specified to get cell element', () => {
    assert.throws(() => fixture.getCellElement(0, 0), 'Could not find row 0');
    assert.throws(() => fixture.getCellElement(1, 0), 'Could not find column 0');
    assert.throws(() => fixture.getCellElement(-1, 0), 'Could not find row -1');
  });

  it('uses custom render method if provided', () => {
    fixture.renderColumn = (value: any, colIndex: number, rowIndex: number) => {
      if (!rowIndex && !colIndex) {
        return 'custom text';
      } else if (rowIndex && colIndex) {
        return '<p class="strong">bold!</p>';
      } else {
        return value;
      }
    };
    fixture.rows = [
      new ItemListRow({
        columns: ['col1', 'col2'],
      }),
      new ItemListRow({
        columns: ['col1', 'col2'],
      }),
    ];
    Polymer.flush();

    const [row1, row2] = [getRow(0), getRow(1)];
    assert.strictEqual(row1.children[1].innerHTML.trim(), 'custom text');
    assert.strictEqual(row1.children[2].innerHTML.trim(), 'col2');
    assert.strictEqual(row2.children[1].innerHTML.trim(), 'col1');
    assert.strictEqual(row2.children[2].innerHTML.trim(), '<p class="strong">bold!</p>');
  });

  it('starts out with all items unselected', () => {
    assert.strictEqual(JSON.stringify(fixture.selectedIndices), '[]',
        'all items should be unselected');
  });

  it('displays column names in the header row', () => {
    // Column 0 is for the checkbox
    const col1 = fixture.header.children[1] as HTMLElement;
    const col2 = fixture.header.children[2] as HTMLElement;
    assert.strictEqual(col1.innerText.trim(), 'col1', 'header should have first column name');
    assert.strictEqual(col2.innerText.trim(), 'col2', 'header should have second column name');
  });

  it('displays column names in the item rows', () => {
    for (let i = 0; i < 5; ++i) {
      const row = getRow(i);
      const firstCol = row.children[1] as HTMLElement;
      const secondCol = row.children[2] as HTMLElement;
      const firstColText = 'first column ' + (i + 1);
      const secondColText = 'second column ' + (i + 1);
      assert.strictEqual(firstCol.innerText.trim(), firstColText,
          'first column should show on item');
      assert.strictEqual(secondCol.innerText.trim(), secondColText,
          'second column should show on item');
    }
  });

  it('displays icons correctly', () => {
    fixture.rows = [
      new ItemListRow({ columns: [''], icon: 'folder' }),
      new ItemListRow({ columns: [''], icon: 'search' }),
    ];
    const row0 = getRow(0);
    const row1 = getRow(1);
    const icon0 = row0.children[0].children[1] as any;
    const icon1 = row1.children[0].children[1] as any;
    assert.strictEqual(icon0.icon, 'folder');
    assert.strictEqual(icon1.icon, 'search');
  });

  it('selects items', () => {
    fixture._selectItemByDisplayIndex(1);

    assert(!isSelected(0), 'first item should not be selected');
    assert(isSelected(1), 'second item should be selected');
    assert(!isSelected(2), 'third item should not be selected');
    assert(!isSelected(3), 'fourth item should not be selected');
    assert(!isSelected(4), 'fifth item should not be selected');
  });

  it('returns selected items', () => {
    fixture._selectItemByDisplayIndex(0);
    fixture._selectItemByDisplayIndex(2);

    assert.strictEqual(JSON.stringify(fixture.selectedIndices), '[0,2]',
        'first and third items should be selected');
  });

  it('can select/unselect all', () => {
    fixture.selectAllCheckbox.click();

    assert.strictEqual(JSON.stringify(fixture.selectedIndices), '[0,1,2,3,4]',
        'all items should be in the selected indices array');
    assert(isSelected(0) && isSelected(1) && isSelected(2) && isSelected(3) && isSelected(4),
        'all items should be selected');

    fixture.selectAllCheckbox.click();

    assert.strictEqual(JSON.stringify(fixture.selectedIndices), '[]',
        'no items should be in the selected indices array');
    assert(!isSelected(0) && !isSelected(1) && !isSelected(2) && !isSelected(3) && !isSelected(4),
        'all items should be unselected');
  });

  it('selects all items if the Select All checkbox is clicked with one item selected', () => {
    fixture._selectItemByDisplayIndex(1);
    fixture.selectAllCheckbox.click();

    assert.strictEqual(JSON.stringify(fixture.selectedIndices), '[0,1,2,3,4]',
        'all items should be selected');
  });

  it('checks the Select All checkbox if all items are selected individually', () => {
    assert(!fixture.selectAllCheckbox.checked, 'Select All checkbox should start out unchecked');

    fixture._selectItemByDisplayIndex(0);
    fixture._selectItemByDisplayIndex(1);
    fixture._selectItemByDisplayIndex(2);
    fixture._selectItemByDisplayIndex(3);
    fixture._selectItemByDisplayIndex(4);

    assert(fixture.selectAllCheckbox.checked, 'Select All checkbox should become checked');
  });

  it('unchecks the Select All checkbox if one item becomes unchecked', () => {
    fixture._selectAll();
    assert(fixture.selectAllCheckbox.checked, 'Select All checkbox should be checked');

    fixture._unselectItemByDisplayIndex(1);
    assert(!fixture.selectAllCheckbox.checked,
        'Select All checkbox should be unchecked after unselecting an item');
  });

  it('selects the clicked item', () => {
    const firstRow = getRow(0);
    firstRow.click();

    assert(isSelected(0), 'first item should be selected');

    firstRow.click();

    assert(isSelected(0), 'first item should still be selected');
  });

  it('selects only the clicked item if the checkbox was not clicked', () => {
    const firstRow = getRow(0);
    const secondRow = getRow(1);

    firstRow.click();
    secondRow.click();

    assert(!isSelected(0) && isSelected(1), 'only the second item should still be selected');
  });

  it('can do multi-selection using the checkboxes', () => {
    const firstCheckbox = getCheckbox(0);
    const thirdCheckbox = getCheckbox(2);

    firstCheckbox.click();
    assert(isSelected(0), 'first item should be selected');

    thirdCheckbox.click();
    assert(isSelected(0) && isSelected(2), 'both first and third items should be selected');
    assert(!isSelected(1) && !isSelected(3) && !isSelected(4), 'the rest should not be selected');

    firstCheckbox.click();
    assert(!isSelected(0) && !isSelected(1) && isSelected(2) && !isSelected(3) && !isSelected(4),
        'first item should be unselected after the second click');
  });

  it('can do multi-selection using the ctrl key', () => {
    const firstRow = getRow(0);
    const thirdRow = getRow(2);

    firstRow.click();
    thirdRow.dispatchEvent(new MouseEvent('click', {
      ctrlKey: true,
    }));

    assert(isSelected(0) && isSelected(2), 'both first and third items should be selected');
    assert(!isSelected(1) && !isSelected(3) && !isSelected(4), 'the rest should not be selected');

    firstRow.dispatchEvent(new MouseEvent('click', {
      ctrlKey: true,
    }));

    assert(!isSelected(0), 'first row should be unselected on ctrl click');
    assert(isSelected(2), 'third item should still be selected');
  });

  it('can do multi-selection using the shift key', () => {
    const firstRow = getRow(0);
    const thirdRow = getRow(2);

    thirdRow.click();
    firstRow.dispatchEvent(new MouseEvent('click', {
      shiftKey: true,
    }));

    assert(isSelected(0) && isSelected(1) && isSelected(2), 'items 1 to 3 should be selected');
    assert(!isSelected(3) && !isSelected(4), 'the rest should be unselected');
  });

  it('can do multi-selection with ctrl and shift key combinations', () => {
    const firstRow = getRow(0);
    const thirdRow = getRow(2);
    const fifthRow = getRow(4);

    thirdRow.click();
    firstRow.dispatchEvent(new MouseEvent('click', {
      ctrlKey: true,
    }));
    fifthRow.dispatchEvent(new MouseEvent('click', {
      shiftKey: true,
    }));

    assert.strictEqual(JSON.stringify(fixture.selectedIndices), '[0,1,2,3,4]',
        'all rows 1 to 5 should be selected');
  });

  it('hides the header when no-header property is used', () => {
    assert.notStrictEqual(fixture.header.offsetHeight, 0, 'the header row should be visible');

    fixture.hideHeader = true;
    assert.strictEqual(fixture.header.offsetHeight, 0, 'the header row should be hidden');
  });

  it('prevents item selection when disable-selection property is used', () => {
    fixture.disableSelection = true;

    const firstRow = getRow(0);
    firstRow.click();

    assert(!isSelected(0), 'first item should not be selected when selection is disabled');
  });

  describe('pagination', () => {
    const pageResponses = [
      {
        nextPageToken: 'page-2',
        rows: [
          new ItemListRow({ columns: ['item a', new Date('11/11/2017, 8:56:42 AM')] }),
          new ItemListRow({ columns: ['item b', new Date('11/11/2017, 8:57:42 AM')] }),
        ]
      },
      {
        nextPageToken: 'page-3',
        rows: [
          new ItemListRow({ columns: ['item c', new Date('11/11/2017, 8:58:42 AM')] }),
          new ItemListRow({ columns: ['item d', new Date('11/11/2017, 8:59:42 AM')] }),
        ]
      },
      {
        nextPageToken: '',
        rows: [
          new ItemListRow({ columns: ['item e', new Date('11/11/2017, 9:00:42 AM')] }),
        ]
      }
    ];

    const loadNewPage = (ev: Event) => {
      const detail = (ev as NewListPageEvent).detail;
      fixture.updateNextPageToken(pageResponses[detail.pageNumber].nextPageToken);
      fixture.rows = pageResponses[detail.pageNumber].rows;
      fixture.list.render();
    };

    beforeEach(async () => {
      _resetFixture();
      fixture.addEventListener(NewListPageEvent.name, loadNewPage);
      fixture.columns = [
        new ItemListColumn('col1', ColumnTypeName.STRING),
        new ItemListColumn('col2', ColumnTypeName.DATE),
      ];
      loadNewPage(new NewListPageEvent('', 0, 20, '', ''));
    });

    it('disables previous page button on first page', () => {
      assert.strictEqual(
          fixture.previousPageButton.disabled,
          true,
          'Previous page button should be disabled');

    });

    it('enables next page button on first page', () => {
      assert.strictEqual(
          fixture.nextPageButton.disabled,
          false,
          'Next page button should not be disabled');
    });

    it('can navigate to next page', () => {
      // Verify first page before navigating to second.
      assert.strictEqual(fixture.rows.length, 2, 'First page should have only 2 elements');
      assert(
          fixture.rows[0].columns[0] === 'item a' &&
          fixture.rows[1].columns[0] === 'item b',
          'First page should have correct rows');

      fixture.nextPageButton.click();

      assert.strictEqual(fixture.rows.length, 2, 'Second page should have only 2 elements');
      assert(
          fixture.rows[0].columns[0] === 'item c' &&
          fixture.rows[1].columns[0] === 'item d',
          'Second page should have correct rows');
    });

    it('can navigate to previous page after navigating to next page', () => {
      // Verify first page before navigating to second.
      assert.strictEqual(fixture.rows.length, 2, 'First page should have only 2 elements');
      assert(
          fixture.rows[0].columns[0] === 'item a' &&
          fixture.rows[1].columns[0] === 'item b',
          'First page should have correct rows');

      fixture.nextPageButton.click();

        // Verify second page before navigating to first.
      assert.strictEqual(
          fixture.previousPageButton.disabled,
          false,
          'Previous page button should be disabled after navigating to second page');
      assert.strictEqual(
          fixture.nextPageButton.disabled,
          false,
          'Next page button should be enabled after navigating to second page');
      assert.strictEqual(fixture.rows.length, 2, 'Second page should have only 2 elements');
      assert(
          fixture.rows[0].columns[0] === 'item c' &&
          fixture.rows[1].columns[0] === 'item d',
          'Second page should have correct rows');

      fixture.previousPageButton.click();

      assert.strictEqual(
          fixture.previousPageButton.disabled,
          true,
          'Previous page button should be disabled after returning to first page');
      assert.strictEqual(
          fixture.nextPageButton.disabled,
          false,
          'Next page button should be enabled after returning to first page');
      assert.strictEqual(fixture.rows.length, 2, 'First page should have only 2 elements');
      assert(
          fixture.rows[0].columns[0] === 'item a' &&
          fixture.rows[1].columns[0] === 'item b',
          'First page should have correct rows');
    });

    it('disables next page button after reaching final page', () => {
      // The test sets up 3 pages
      fixture.nextPageButton.click();
      assert.strictEqual(
          fixture.nextPageButton.disabled,
          false,
          'Next page button should not be disabled');

      fixture.nextPageButton.click();

      assert.strictEqual(
          fixture.nextPageButton.disabled,
          true,
          'Next page button should be disabled');
    });

    it('starts with next page button disabled if nextPageToken is empty', () => {
      // Simulating only one page of results by initially loading the final page.
      loadNewPage(new NewListPageEvent('', pageResponses.length - 1, 20, '' , ''));

      assert.strictEqual(
          fixture.nextPageButton.disabled,
          true,
          'Next page button should be disabled if nextPageToken is empty');
    });
  });

  describe('filtering', () => {

    beforeEach(() => {
      _resetFixture(true);
      fixture.columns = [
        new ItemListColumn('col1', ColumnTypeName.STRING),
        new ItemListColumn('col2', ColumnTypeName.STRING),
      ];
      fixture.rows = [
        new ItemListRow({
          columns: ['first column 1', 'second column 1'],
        }),
        new ItemListRow({
          columns: ['first column 2', 'second column 2'],
        }),
        new ItemListRow({
          columns: ['first column 3', 'second column 3'],
        }),
        new ItemListRow({
          columns: ['first column 4', 'second column 4'],
        }),
        new ItemListRow({
          columns: ['first column 5', 'second column 5'],
        }),
      ];
    });

    it('hides the filter box by default', () => {
      assert.strictEqual(fixture.filterBox.offsetHeight, 0,
          'filter box should not show by default');
    });

    it('shows/hides filter box when toggle is clicked', () => {
      fixture.filterToggleButton.click();
      assert(fixture.filterBox.offsetHeight > 0,
          'filter box should show when toggle is clicked');

      fixture.filterToggleButton.click();
      assert.strictEqual(fixture.filterBox.offsetHeight, 0,
          'filter box should hide when toggle is clicked again');
    });

    it('filters items when typing characters in the filter box', () => {
      fixture.filterToggleButton.click();
      fixture.filterString = '3';
      Polymer.flush();
      const rows = fixture.listContainer.querySelectorAll('.row');
      assert.strictEqual(rows.length, 1, 'only one item has "3" in its name');
      assert.strictEqual(rows[0].children[1].textContent!.trim(), 'first column 3',
          'filter should only return the third item');
    });

    it('shows all items when filter string is deleted', () => {
      fixture.filterToggleButton.click();
      fixture.filterString = '3';
      Polymer.flush();
      fixture.filterString = '';
      Polymer.flush();
      const rows = fixture.listContainer.querySelectorAll('.row');
      assert.strictEqual(rows.length, 5, 'should show all rows after filter string is deleted');
    });

    it('filters items based on first column only', () => {
      fixture.filterToggleButton.click();
      fixture.filterString = 'second';
      Polymer.flush();
      const rows = fixture.listContainer.querySelectorAll('.row');
      assert.strictEqual(rows.length, 0,
          'should not show any rows, since no row has "second" in its first column');
    });

    it('ignores case when filtering', () => {
      fixture.filterToggleButton.click();
      fixture.filterString = 'COLUMN 4';
      Polymer.flush();
      const rows = fixture.listContainer.querySelectorAll('.row');
      assert.strictEqual(rows.length, 1,
          'should show one row containing "column 4", since filtering is case insensitive');
      assert.strictEqual(rows[0].children[1].textContent!.trim(), 'first column 4',
          'filter should return the fourth item');
    });

    it('resets filter when filter box is closed', () => {
      fixture.filterToggleButton.click();
      fixture.filterString = '3';
      Polymer.flush();
      fixture.filterToggleButton.click();
      Polymer.flush();
      assert.strictEqual(fixture.listContainer.querySelectorAll('.row').length, 5,
          'all rows should show after closing filter box');
    });

    it('selects only visible items when Select All checkbox is clicked', () => {
      fixture.filterToggleButton.click();
      fixture.filterString = '3';
      Polymer.flush();
      fixture.selectAllCheckbox.click();

      fixture.filterString = '';
      Polymer.flush();

      assert(!isSelected(0), 'only third item should be selected');
      assert(!isSelected(1), 'only third item should be selected');
      assert(isSelected(2), 'only third item should be selected');
      assert(!isSelected(3), 'only third item should be selected');
      assert(!isSelected(4), 'only third item should be selected');
    });

    it('returns the correct selectedIndices result matching clicked items when filtering', () => {
      fixture.filterToggleButton.click();
      fixture.filterString = '2';
      Polymer.flush();

      const firstRow = getRow(0);
      firstRow.click();
      const selectedIndices = fixture.selectedIndices;
      assert.strictEqual(selectedIndices.length, 1, 'only one item should be selected');
      assert.strictEqual(selectedIndices[0], 1,
          'the second index (only one shown) should be selected');
    });

    // Remote filtering is handled by the backend, not the item-list component.
    it('should not change list when doing remote filtering', () => {
      _resetFixture(false);
      fixture.columns = [
        new ItemListColumn('col1', ColumnTypeName.STRING),
        new ItemListColumn('col2', ColumnTypeName.STRING),
      ];
      fixture.rows = [
        new ItemListRow({
          columns: ['first column 1', 'second column 1'],
        }),
        new ItemListRow({
          columns: ['first column 2', 'second column 2'],
        }),
        new ItemListRow({
          columns: ['first column 3', 'second column 3'],
        }),
        new ItemListRow({
          columns: ['first column 4', 'second column 4'],
        }),
        new ItemListRow({
          columns: ['first column 5', 'second column 5'],
        }),
      ];
      fixture.filterToggleButton.click();
      fixture.filterString = 'a';
      Polymer.flush();
      assert.strictEqual(fixture.listContainer.querySelectorAll('.row').length, 5,
          'should show all items');
    });
  });

  describe('sorting', () => {
    const rows = [
      new ItemListRow({ columns: ['item cx', new Date('11/11/2017, 8:58:42 AM')] }),
      new ItemListRow({ columns: ['item ax', new Date('11/11/2017, 8:59:42 AM')] }),
      new ItemListRow({ columns: ['item b', new Date('11/11/2017, 8:57:42 AM')] })
    ];

    const col0SortedOrder = [1, 2, 0];
    const col1SortedOrder = [2, 0, 1];
    const col0ReverseOrder = col0SortedOrder.slice().reverse();

    beforeEach(async () => {
      _resetFixture(false, true);
      fixture.rows = rows;
      fixture.columns = [
        new ItemListColumn('col1', ColumnTypeName.STRING, 'col1SortKey'),
        new ItemListColumn('col2', ColumnTypeName.DATE, 'col2SortKey'),
      ];
      fixture.list.render();
    });

    it('sorts on first column by default', () => {
      const renderedRows = fixture.listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; i++) {
        const columns = renderedRows[i].querySelectorAll('.column');
        const sortedColumns = fixture.rows[col0SortedOrder[i]].columns;
        assert.strictEqual(columns[0].textContent!.trim(), sortedColumns[0]);
        assert.strictEqual(
            columns[1].textContent!.trim(),
            new Date(sortedColumns[1]!.toString()).toLocaleString());
      }
    });

    it('switches sort to descending order if first column is sorted on again', () => {
      const renderedRows = fixture.listContainer.querySelectorAll('.row');
      fixture._sortBy(0);
      fixture.list.render();
      for (let i = 0; i < fixture.rows.length; i++) {
        const columns = renderedRows[i].querySelectorAll('.column');
        const sortedColumns = fixture.rows[col0ReverseOrder[i]].columns;
        assert.strictEqual(columns[0].textContent!.trim(), sortedColumns[0]);
        assert.strictEqual(
            columns[1].textContent!.trim(),
            new Date(sortedColumns[1]!.toString()).toLocaleString());
      }
    });

    it('when switching to sorting on second column, uses ascending order', () => {
      fixture._sortBy(1);
      fixture.list.render();
      const renderedRows = fixture.listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; i++) {
        const columns = renderedRows[i].querySelectorAll('.column');
        const sortedColumns = fixture.rows[col1SortedOrder[i]].columns;
        assert.strictEqual(columns[0].textContent!.trim(), sortedColumns[0]);
        assert.strictEqual(
            columns[1].textContent!.trim(),
            new Date(sortedColumns[1]!.toString()).toLocaleString());
      }
    });

    it('shows arrow icon on the sorted column', () => {
      const headerIcons = fixture.header.querySelectorAll('.sort-icon') as any;
      assert(!headerIcons[0].hidden, 'first column should show sort icon');
      assert.strictEqual(headerIcons[0].icon, 'arrow-upward',
          'first column should show ascending sort icon');
      assert(headerIcons[1].hidden, 'second column icon should be hidden');

      fixture._sortBy(1);
      fixture.list.render();
      assert(headerIcons[0].hidden, 'first column icon should be hidden');
      assert(!headerIcons[1].hidden, 'second column should show sort icon');
      assert.strictEqual(headerIcons[1].icon, 'arrow-upward',
          'second column should show ascending sort icon');

      fixture._sortBy(1);
      fixture.list.render();
      assert(headerIcons[0].hidden, 'first column icon should be hidden');
      assert(!headerIcons[1].hidden, 'second column should show sort icon');
      assert.strictEqual(headerIcons[1].icon, 'arrow-downward',
          'second column should show descending sort icon');
    });

    it('sorts the clicked column', () => {
      const headerButtons = fixture.header.querySelectorAll('.column-button') as any;
      const headerIcons = fixture.header.querySelectorAll('.sort-icon') as any;
      headerButtons[0].click();
      fixture.list.render();
      assert.strictEqual(headerIcons[0].icon, 'arrow-downward',
          'first column should show descending sort icon');

      headerButtons[1].click();
      fixture.list.render();
      assert.strictEqual(headerIcons[1].icon, 'arrow-upward',
          'second column should show ascending sort icon');
    });

    it('sorts numbers correctly', () => {
      fixture.columns = [ new ItemListColumn('col1', ColumnTypeName.NUMBER) ];
      fixture.rows = [
        new ItemListRow({ columns: [11] }),
        new ItemListRow({ columns: [1] }),
        new ItemListRow({ columns: [2] }),
      ];
      const sortedOrder = [1, 2, 0];

      const renderedRows = fixture.listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; i++) {
        const columns = renderedRows[i].querySelectorAll('.column');
        assert.strictEqual(
            columns[0].textContent!.trim(),
            fixture.rows[sortedOrder[i]].columns[0]!.toString());
      }
    });

    it('sorts correctly when there are equal values', () => {
      fixture.columns = [ new ItemListColumn('col1', ColumnTypeName.NUMBER) ];
      fixture.rows = [
        new ItemListRow({ columns: [2] }),
        new ItemListRow({ columns: [1] }),
        new ItemListRow({ columns: [2] }),
        new ItemListRow({ columns: [1] }),
        new ItemListRow({ columns: [11] }),
        new ItemListRow({ columns: [2] }),
      ];
      const sortedOrder = [1, 3, 0, 2, 5, 4];

      const renderedRows = fixture.listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; i++) {
        const columns = renderedRows[i].querySelectorAll('.column');
        assert.strictEqual(
            columns[0].textContent!.trim(),
            fixture.rows[sortedOrder[i]].columns[0]!.toString());
      }
    });

    it('returns the correct selectedIndices result matching clicked items when sorted', () => {
      fixture.columns = [ new ItemListColumn('col1', ColumnTypeName.NUMBER) ];
      fixture.rows = [
        new ItemListRow({ columns: [1] }),
        new ItemListRow({ columns: [2] }),
        new ItemListRow({ columns: [3] }),
      ];
      // Reverse the sorting
      fixture._sortBy(0);

      const firstRow = getRow(0);
      firstRow.click();
      const selectedIndices = fixture.selectedIndices;
      assert.strictEqual(selectedIndices.length, 1, 'only one item should be selected');
      assert.strictEqual(selectedIndices[0], 2, 'the third index (shown first) should be selected');
    });

    // Remote sorting is handled by the backend, not the item-list component.
    it('should not change row order when doing remote sorting', () => {
      _resetFixture(false, false);
      fixture.columns = [ new ItemListColumn('col1', ColumnTypeName.NUMBER) ];
      fixture.rows = [
        new ItemListRow({ columns: [3] }),
        new ItemListRow({ columns: [1] }),
        new ItemListRow({ columns: [2] }),
        new ItemListRow({ columns: [4] }),
      ];
      const unsortedValues = [3, 1, 2, 4];

      Polymer.flush();
      const renderedRows = fixture.listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; i++) {
        const columns = renderedRows[i].querySelectorAll('.column');
        assert.strictEqual(columns[0].textContent!.trim(), unsortedValues[i].toString());
      }
    });

    // Remote sorting is handled by the backend, not the item-list component.
    it('should allow simulaneous, local sorting and filtering', () => {
      _resetFixture(true, true);
      fixture.columns = [
        new ItemListColumn('col1', ColumnTypeName.STRING),
        new ItemListColumn('col2', ColumnTypeName.NUMBER),
      ];
      fixture.rows = [
        new ItemListRow({ columns: ['a', 3] }),
        new ItemListRow({ columns: ['a', 1] }),
        new ItemListRow({ columns: ['a', 2] }),
        new ItemListRow({ columns: ['b', 3] }),
        new ItemListRow({ columns: ['b', 1] }),
        new ItemListRow({ columns: ['b', 2] }),
      ];
      fixture.filterToggleButton.click();
      fixture.filterString = 'b';
      // Sort by second column
      fixture._sortBy(1);

      fixture.list.render();

      const expectedColumnValues = ['1', '2', '3'];
      const renderedRows = fixture.listContainer.querySelectorAll('.row');
      for (let i = 0; i < renderedRows.length; i++) {
        const columns = renderedRows[i].querySelectorAll('.column');
        assert.strictEqual(columns[1].textContent!.trim(), expectedColumnValues[i]);
      }
    });
  });

  after(() => {
    document.body.removeChild(fixture);
  });
});
