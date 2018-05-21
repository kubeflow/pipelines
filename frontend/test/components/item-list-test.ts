import * as assert from 'assert';

import {
  ColumnTypeName,
  ItemListElement,
  ItemListRow
} from '../../src/components/item-list/item-list';

let fixture: ItemListElement;
const TEST_TAG = 'item-list';

function resetFixture(): void {
  const old = document.querySelector(TEST_TAG);
  if (old) {
    document.body.removeChild(old);
  }
  document.body.appendChild(document.createElement(TEST_TAG));
  fixture = document.querySelector(TEST_TAG) as any;
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
    const listContainer = fixture.$.listContainer as HTMLElement;
    return listContainer.querySelectorAll('paper-item')[i] as HTMLElement;
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
    resetFixture();
    fixture.columns = [{
      name: 'col1',
      type: ColumnTypeName.STRING,
    }, {
      name: 'col2',
      type: ColumnTypeName.STRING,
    }];
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
    (Polymer.dom as any).flush();
  });

  it('displays a row per item in list', () => {
    const listContainer = fixture.$.listContainer as HTMLElement;
    const children = listContainer.querySelectorAll('paper-item');
    assert(children.length === 5, 'five rows should appear');
  });

  it('starts out with all items unselected', () => {
    assert(JSON.stringify(fixture.selectedIndices) === '[]', 'all items should be unselected');
  });

  it('displays column names in the header row', () => {
    // Column 0 is for the checkbox
    const header = fixture.$.header as HTMLElement;
    const col1 = header.children[1] as HTMLElement;
    const col2 = header.children[2] as HTMLElement;
    assert(col1.innerText.trim() === 'col1', 'header should have first column name');
    assert(col2.innerText.trim() === 'col2', 'header should have second column name');
  });

  it('displays column names in the item rows', () => {
    for (let i = 0; i < 5; ++i) {
      const row = getRow(i);
      const firstCol = row.children[1] as HTMLElement;
      const secondCol = row.children[2] as HTMLElement;
      const firstColText = 'first column ' + (i + 1);
      const secondColText = 'second column ' + (i + 1);
      assert(firstCol.innerText.trim() === firstColText, 'first column should show on item');
      assert(secondCol.innerText.trim() === secondColText, 'second column should show on item');
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
    assert(icon0.icon === 'folder');
    assert(icon1.icon === 'search');
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

    assert(JSON.stringify(fixture.selectedIndices) === '[0,2]',
        'first and third items should be selected');
  });

  it('can select/unselect all', () => {
    const header = fixture.$.header as HTMLElement;
    const c = header.querySelector('#selectAllCheckbox') as HTMLElement;
    c.click();

    assert(JSON.stringify(fixture.selectedIndices) === '[0,1,2,3,4]',
        'all items should be in the selected indices array');
    assert(isSelected(0) && isSelected(1) && isSelected(2) && isSelected(3) && isSelected(4),
        'all items should be selected');

    c.click();

    assert(JSON.stringify(fixture.selectedIndices) === '[]',
        'no items should be in the selected indices array');
    assert(!isSelected(0) && !isSelected(1) && !isSelected(2) && !isSelected(3) && !isSelected(4),
        'all items should be unselected');
  });

  it('selects all items if the Select All checkbox is clicked with one item selected', () => {
    const header = fixture.$.header as HTMLElement;
    fixture._selectItemByDisplayIndex(1);
    const c = header.querySelector('#selectAllCheckbox') as HTMLElement;
    c.click();

    assert(JSON.stringify(fixture.selectedIndices) === '[0,1,2,3,4]',
        'all items should be selected');
  });

  it('checks the Select All checkbox if all items are selected individually', () => {
    const header = fixture.$.header as HTMLElement;
    const c = header.querySelector('#selectAllCheckbox') as any;

    assert(!c.checked, 'Select All checkbox should start out unchecked');

    fixture._selectItemByDisplayIndex(0);
    fixture._selectItemByDisplayIndex(1);
    fixture._selectItemByDisplayIndex(2);
    fixture._selectItemByDisplayIndex(3);
    fixture._selectItemByDisplayIndex(4);

    assert(c.checked, 'Select All checkbox should become checked');
  });

  it('unchecks the Select All checkbox if one item becomes unchecked', () => {
    const header = fixture.$.header as HTMLElement;
    const c = header.querySelector('#selectAllCheckbox') as any;

    fixture._selectAll();
    assert(c.checked, 'Select All checkbox should be checked');

    fixture._unselectItemByDisplayIndex(1);
    assert(!c.checked, 'Select All checkbox should become unchecked after unselecting an item');
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

    assert(JSON.stringify(fixture.selectedIndices) === '[0,1,2,3,4]',
        'all rows 1 to 5 should be selected');
  });

  it('hides the header when no-header property is used', () => {
    const header = fixture.$.header as HTMLElement;
    assert(header.offsetHeight !== 0, 'the header row should be visible');

    fixture.hideHeader = true;
    assert(header.offsetHeight === 0, 'the header row should be hidden');
  });

  it('prevents item selection when disable-selection property is used', () => {
    fixture.disableSelection = true;

    const firstRow = getRow(0);
    firstRow.click();

    assert(!isSelected(0), 'first item should not be selected when selection is disabled');
  });

  describe('local filtering', () => {

    beforeEach(() => {
      fixture.filterLocally = true;
    });

    it('hides the filter box by default', () => {
      const filterBox = fixture.$.filterBox as HTMLElement;
      assert(filterBox.offsetHeight === 0, 'filter box should not show by default');
    });

    it('shows/hides filter box when toggle is clicked', () => {
      const filterBox = fixture.$.filterBox as HTMLElement;
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      filterToggle.click();
      assert(filterBox.offsetHeight > 0,
          'filter box should show when toggle is clicked');

      filterToggle.click();
      assert(filterBox.offsetHeight === 0,
          'filter box should hide when toggle is clicked again');
    });

    it('filters items when typing characters in the filter box', () => {
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      const listContainer = fixture.$.listContainer as HTMLElement;
      filterToggle.click();
      fixture.filterString = '3';
      Polymer.flush();
      const rows = listContainer.querySelectorAll('.row');
      assert(rows.length === 1, 'only one item has "3" in its name');
      assert(rows[0].children[1].textContent.trim() === 'first column 3',
          'filter should only return the third item');
    });

    it('shows all items when filter string is deleted', () => {
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      const listContainer = fixture.$.listContainer as HTMLElement;
      filterToggle.click();
      fixture.filterString = '3';
      Polymer.flush();
      fixture.filterString = '';
      Polymer.flush();
      const rows = listContainer.querySelectorAll('.row');
      assert(rows.length === 5, 'should show all rows after filter string is deleted');
    });

    it('filters items based on first column only', () => {
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      const listContainer = fixture.$.listContainer as HTMLElement;
      filterToggle.click();
      fixture.filterString = 'second';
      Polymer.flush();
      const rows = listContainer.querySelectorAll('.row');
      assert(rows.length === 0,
          'should not show any rows, since no row has "second" in its first column');
    });

    it('ignores case when filtering', () => {
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      const listContainer = fixture.$.listContainer as HTMLElement;
      filterToggle.click();
      fixture.filterString = 'COLUMN 4';
      Polymer.flush();
      const rows = listContainer.querySelectorAll('.row');
      assert(rows.length === 1,
          'should show one row containing "column 4", since filtering is case insensitive');
      assert(rows[0].children[1].textContent.trim() === 'first column 4',
          'filter should return the fourth item');
    });

    it('resets filter when filter box is closed', () => {
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      const listContainer = fixture.$.listContainer as HTMLElement;
      filterToggle.click();
      fixture.filterString = '3';
      Polymer.flush();
      filterToggle.click();
      Polymer.flush();
      assert(listContainer.querySelectorAll('.row').length === 5,
          'all rows should show after closing filter box');
    });

    it('selects only visible items when Select All checkbox is clicked', () => {
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      const selectAllCheckbox = fixture.$.selectAllCheckbox as HTMLElement;
      filterToggle.click();
      fixture.filterString = '3';
      Polymer.flush();
      selectAllCheckbox.click();

      fixture.filterString = '';
      Polymer.flush();

      assert(!isSelected(0), 'only third item should be selected');
      assert(!isSelected(1), 'only third item should be selected');
      assert(isSelected(2), 'only third item should be selected');
      assert(!isSelected(3), 'only third item should be selected');
      assert(!isSelected(4), 'only third item should be selected');
    });

    it('returns the correct selectedIndices result matching clicked items when filtering', () => {
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      filterToggle.click();
      fixture.filterString = '2';
      Polymer.flush();

      const firstRow = getRow(0);
      firstRow.click();
      const selectedIndices = fixture.selectedIndices;
      assert(selectedIndices.length === 1, 'only one item should be selected');
      assert(selectedIndices[0] === 1, 'the second index (only one shown) should be selected');
    });
  });

  describe('remote filtering', () => {

    const rows = [
      new ItemListRow({ columns: ['item c*', new Date('11/11/2017, 8:58:42 AM')] }),
      new ItemListRow({ columns: ['item a*', new Date('11/11/2017, 8:59:42 AM')] }),
      new ItemListRow({ columns: ['item b', new Date('11/11/2017, 8:57:42 AM')] })
    ];

    beforeEach(() => {
      resetFixture();
      fixture.filterLocally = false;
      fixture.columns = [{
        name: 'col1',
        type: ColumnTypeName.STRING,
      }, {
        name: 'col2',
        type: ColumnTypeName.STRING,
      }];
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

    it('should not change list when doing remote filtering', () => {
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      const listContainer = fixture.$.listContainer as HTMLElement;
      filterToggle.click();
      fixture.filterString = 'a';
      Polymer.flush();
      assert(listContainer.querySelectorAll('.row').length === 5, 'should show all items');
    });
  });

  describe('sorting', () => {
    const rows = [
      new ItemListRow({ columns: ['item c*', new Date('11/11/2017, 8:58:42 AM')] }),
      new ItemListRow({ columns: ['item a*', new Date('11/11/2017, 8:59:42 AM')] }),
      new ItemListRow({ columns: ['item b', new Date('11/11/2017, 8:57:42 AM')] })
    ];

    const col0SortedOrder = [1, 2, 0];
    const col1SortedOrder = [2, 0, 1];
    const col0ReverseOrder = col0SortedOrder.slice().reverse();

    beforeEach(async () => {
      resetFixture();
      const list = fixture.$.list as Polymer.DomRepeat;
      fixture.rows = rows;
      fixture.columns = [{
        name: 'col1',
        type: ColumnTypeName.STRING,
      }, {
        name: 'col2',
        type: ColumnTypeName.DATE,
      }];
      list.render();
    });

    it('sorts on first column by default', () => {
      const listContainer = fixture.$.listContainer as HTMLElement;
      const renderedRows = listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; ++i) {
        const columns = renderedRows[i].querySelectorAll('.column');
        const sortedColumns = fixture.rows[col0SortedOrder[i]].columns;
        assert(columns[0].textContent.trim() === sortedColumns[0]);
        assert(columns[1].textContent.trim() ===
            new Date(sortedColumns[1].toString()).toLocaleString());
      }
    });

    it('switches sort to descending order if first column is sorted on again', () => {
      const listContainer = fixture.$.listContainer as HTMLElement;
      const renderedRows = listContainer.querySelectorAll('.row');
      const list = fixture.$.list as Polymer.DomRepeat;
      fixture._sortBy(0);
      list.render();
      for (let i = 0; i < fixture.rows.length; ++i) {
        const columns = renderedRows[i].querySelectorAll('.column');
        const sortedColumns = fixture.rows[col0ReverseOrder[i]].columns;
        assert(columns[0].textContent.trim() === sortedColumns[0]);
        assert(columns[1].textContent.trim() ===
            new Date(sortedColumns[1].toString()).toLocaleString());
      }
    });

    it('when switching to sorting on second column, uses ascending order', () => {
      const listContainer = fixture.$.listContainer as HTMLElement;
      const list = fixture.$.list as Polymer.DomRepeat;
      fixture._sortBy(1);
      list.render();
      const renderedRows = listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; ++i) {
        const columns = renderedRows[i].querySelectorAll('.column');
        const sortedColumns = fixture.rows[col1SortedOrder[i]].columns;
        assert(columns[0].textContent.trim() === sortedColumns[0]);
        assert(columns[1].textContent.trim() ===
            new Date(sortedColumns[1].toString()).toLocaleString());
      }
    });

    it('shows arrow icon on the sorted column', () => {
      const list = fixture.$.list as Polymer.DomRepeat;
      const headerIcons = fixture.$.header.querySelectorAll('.sort-icon') as any;
      assert(!headerIcons[0].hidden, 'first column should show sort icon');
      assert(headerIcons[0].icon === 'arrow-upward',
          'first column should show ascending sort icon');
      assert(headerIcons[1].hidden, 'second column icon should be hidden');

      fixture._sortBy(1);
      list.render();
      assert(headerIcons[0].hidden, 'first column icon should be hidden');
      assert(!headerIcons[1].hidden, 'second column should show sort icon');
      assert(headerIcons[1].icon === 'arrow-upward',
          'second column should show ascending sort icon');

      fixture._sortBy(1);
      list.render();
      assert(headerIcons[0].hidden, 'first column icon should be hidden');
      assert(!headerIcons[1].hidden, 'second column should show sort icon');
      assert(headerIcons[1].icon === 'arrow-downward',
          'second column should show descending sort icon');
    });

    it('sorts the clicked column', () => {
      const list = fixture.$.list as Polymer.DomRepeat;
      const headerButtons = fixture.$.header.querySelectorAll('.column-button') as any;
      const headerIcons = fixture.$.header.querySelectorAll('.sort-icon') as any;
      headerButtons[0].click();
      list.render();
      assert(headerIcons[0].icon === 'arrow-downward',
          'first column should show descending sort icon');

      headerButtons[1].click();
      list.render();
      assert(headerIcons[1].icon === 'arrow-upward',
          'second column should show ascending sort icon');
    });

    it('sorts while filtering is active', () => {
      fixture.filterLocally = true;
      const list = fixture.$.list as Polymer.DomRepeat;
      const listContainer = fixture.$.listContainer as HTMLElement;
      const filterToggle = fixture.$.filterToggle as HTMLElement;
      filterToggle.click();
      fixture.filterString = '*';
      list.render();

      let renderedRows = listContainer.querySelectorAll('.row');
      // row 0
      let columns0 = renderedRows[0].querySelectorAll('.column');
      assert(columns0[0].textContent.trim() === fixture.rows[1].columns[0]);
      assert(columns0[1].textContent.trim() ===
        new Date(fixture.rows[1].columns[1].toString()).toLocaleString());
      // row 1
      let columns1 = renderedRows[1].querySelectorAll('.column');
      assert(columns1[0].textContent.trim() === fixture.rows[0].columns[0]);
      assert(columns1[1].textContent.trim() ===
        new Date(fixture.rows[0].columns[1].toString()).toLocaleString());

      fixture._sortBy(0);
      list.render();
      renderedRows = listContainer.querySelectorAll('.row');
      // row 0
      columns0 = renderedRows[0].querySelectorAll('.column');
      assert(columns0[0].textContent.trim() === fixture.rows[0].columns[0]);
      assert(columns0[1].textContent.trim() ===
        new Date(fixture.rows[0].columns[1].toString()).toLocaleString());
      // row 1
      columns1 = renderedRows[1].querySelectorAll('.column');
      assert(columns1[0].textContent.trim() === fixture.rows[1].columns[0]);
      assert(columns1[1].textContent.trim() ===
        new Date(fixture.rows[1].columns[1].toString()).toLocaleString());
    });

    it('sorts numbers correctly', () => {
      const listContainer = fixture.$.listContainer as HTMLElement;
      fixture.columns = [{
        name: 'col1',
        type: ColumnTypeName.NUMBER,
      }];
      fixture.rows = [
        new ItemListRow({ columns: [11] }),
        new ItemListRow({ columns: [1] }),
        new ItemListRow({ columns: [2] }),
      ];
      const sortedOrder = [1, 2, 0];

      const renderedRows = listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; ++i) {
        const columns = renderedRows[i].querySelectorAll('.column');
        assert(columns[0].textContent.trim() ===
            fixture.rows[sortedOrder[i]].columns[0].toString());
      }
    });

    it('sorts correctly when there are equal values', () => {
      const listContainer = fixture.$.listContainer as HTMLElement;
      fixture.columns = [{
        name: 'col1',
        type: ColumnTypeName.NUMBER,
      }];
      fixture.rows = [
        new ItemListRow({ columns: [2] }),
        new ItemListRow({ columns: [1] }),
        new ItemListRow({ columns: [2] }),
        new ItemListRow({ columns: [1] }),
        new ItemListRow({ columns: [11] }),
        new ItemListRow({ columns: [2] }),
      ];
      const sortedOrder = [1, 3, 0, 2, 5, 4];

      const renderedRows = listContainer.querySelectorAll('.row');
      for (let i = 0; i < fixture.rows.length; ++i) {
        const columns = renderedRows[i].querySelectorAll('.column');
        assert(columns[0].textContent.trim() ===
            fixture.rows[sortedOrder[i]].columns[0].toString());
      }
    });

    it('returns the correct selectedIndices result matching clicked items when sorted', () => {
      fixture.columns = [{
        name: 'col1',
        type: ColumnTypeName.NUMBER,
      }];
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
      assert(selectedIndices.length === 1, 'only one item should be selected');
      assert(selectedIndices[0] === 2, 'the third index (shown first) should be selected');
    });
  });

  after(() => {
    document.body.removeChild(fixture);
  });
});
