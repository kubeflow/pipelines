import { assert } from 'chai';
import { TableViewer } from '../../src/components/table-viewer/table-viewer';
import { resetFixture } from './test-utils';

let fixture: TableViewer;

async function _resetFixture(): Promise<void> {
  return resetFixture('table-viewer', undefined, (f: TableViewer) => {
    fixture = f;
  });
}

describe('table-viewer', () => {

  beforeEach(async () => {
    await _resetFixture();
  });

  const rows = [
    ['r1c1', 'r1c2', 'r1c3'],
    ['r2c1', 'r2c2', 'r2c3'],
    ['r3c1', 'r3c2', 'r3c3'],
    ['r4c1', 'r4c2', 'r4c3'],
    ['r5c1', 'r5c2', 'r5c3'],
  ];

  it('shows header row', () => {
    const header = ['col1', 'col2', 'col2'];
    fixture.header = header;
    Polymer.flush();
    assert.strictEqual(fixture.table.rows[0].cells[0].tagName, 'TH');
    assert.strictEqual(fixture.table.rows[0].innerText, header.join('\t'));
  });

  it('shows table rows without header', () => {
    fixture.rows = rows;
    Polymer.flush();

    assert.strictEqual(fixture.table.rows.length, rows.length + 1);
    assert.strictEqual(fixture.table.rows[1].cells[0].tagName, 'TD');
    Array.from(fixture.table.rows).slice(1).forEach((r, i) => {
      assert.strictEqual(r.innerText, rows[i].join('\t'));
    });
  });

  it('shows page 1 out of 1 if pagesize is greater than number of rows', () => {
    fixture.rows = rows;
    fixture.pageSize = rows.length + 5;
    Polymer.flush();

    assert.strictEqual(+fixture.pageIndexElement.innerText.trim(), 1);
    assert.strictEqual(+fixture.numPagesElement.innerText.trim(), 1);
  });

  it('shows page 1 out of 1 if pagesize is equal to number of rows', () => {
    fixture.rows = rows;
    fixture.pageSize = rows.length;
    Polymer.flush();

    assert.strictEqual(+fixture.pageIndexElement.innerText.trim(), 1);
    assert.strictEqual(+fixture.numPagesElement.innerText.trim(), 1);
  });

  it('shows page 1 out of 2 if pagesize is less than number of rows', () => {
    fixture.rows = rows;
    // two pages
    fixture.pageSize = rows.length - 1;
    Polymer.flush();

    assert.strictEqual(+fixture.pageIndexElement.innerText.trim(), 1);
    assert.strictEqual(+fixture.numPagesElement.innerText.trim(), 2);
  });

  it('divides correctly for larger numbers', () => {
    fixture.rows = new Array(43);
    fixture.pageSize = 10;
    Polymer.flush();

    assert.strictEqual(+fixture.pageIndexElement.innerText.trim(), 1);
    assert.strictEqual(+fixture.numPagesElement.innerText.trim(), 5);

    fixture.pageSize = 1;
    assert.strictEqual(+fixture.numPagesElement.innerText.trim(), 43);
  });

  it('starts with first page', () => {
    fixture.rows = rows;
    fixture.pageSize = 1;
    Polymer.flush();

    assert.strictEqual(fixture.pageIndex, 1);
    assert.strictEqual(fixture.table.rows[1].cells[0].innerText, rows[0][0]);
  });

  it('disables previous page button, enables next page button if there are two pages', () => {
    fixture.rows = rows;
    // two pages
    fixture.pageSize = rows.length - 1;
    Polymer.flush();

    assert.strictEqual(fixture.pageIndex, 1);
    assert.strictEqual(fixture.table.rows[1].cells[0].innerText, rows[0][0]);
    assert(fixture.prevPageButton.disabled);
    assert(!fixture.nextPageButton.disabled);
  });

  it('switches to next page when there are two pages, disables next page button', () => {
    fixture.rows = rows;
    // two pages
    fixture.pageSize = rows.length - 1;
    Polymer.flush();

    fixture.nextPageButton.click();
    Polymer.flush();

    assert.strictEqual(+fixture.numPagesElement.innerText.trim(), 2);
    assert.strictEqual(+fixture.pageIndexElement.innerText.trim(), 2);
    assert.strictEqual(fixture.table.rows[1].cells[0].innerText, rows[rows.length - 1][0]);
    assert(!fixture.prevPageButton.disabled);
    assert(fixture.nextPageButton.disabled);
  });

  it('disables previous page button again after switching away then back to first page', () => {
    fixture.rows = rows;
    // two pages
    fixture.pageSize = rows.length - 1;
    Polymer.flush();

    fixture.nextPageButton.click();
    Polymer.flush();

    fixture.prevPageButton.click();
    Polymer.flush();

    assert(fixture.prevPageButton.disabled);
    assert(!fixture.nextPageButton.disabled);
  });

  after(() => {
    document.body.removeChild(fixture);
  });

});
