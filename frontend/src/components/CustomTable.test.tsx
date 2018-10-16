/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import * as React from 'react';
import CustomTable, { Column, Row, css } from './CustomTable';
import { shallow } from 'enzyme';

const props = {
  columns: [],
  orderAscending: true,
  pageSize: 10,
  reload: () => '' as any,
  rows: [],
  sortBy: 'asd',
};

const columns: Column[] = [
  {
    customRenderer: undefined,
    label: 'col1',
  },
  {
    customRenderer: undefined,
    label: 'col2',
  },
];

const rows: Row[] = [
  {
    id: 'row1',
    otherFields: ['cell1', 'cell2'],
  },
  {
    id: 'row2',
    otherFields: ['cell1', 'cell2'],
  },
];

describe('CustomTable', () => {
  it('renders without rows or columns', () => {
    const tree = shallow(<CustomTable {...props} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders empty message on no rows', () => {
    const tree = shallow(<CustomTable {...props} emptyMessage='test empty message' />);
    expect(tree).toMatchSnapshot();
  });

  it('renders some columns with equal widths without rows', () => {
    const tree = shallow(<CustomTable {...props} columns={[{ label: 'col1' }, { label: 'col2' }]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders without the checkboxes if disableSelection is true', () => {
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns}
      disableSelection={true} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders some columns with descending sort order on first column', () => {
    const tree = shallow(<CustomTable {...props} orderAscending={false}
      columns={[{ label: 'col1', sortKey: 'col1sortkey' }, { label: 'col2' }]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders columns with specified widths', () => {
    const testcolumns = [{
      flex: 3,
      label: 'col1',
    }, {
      flex: 1,
      label: 'col2',
    }];
    const tree = shallow(<CustomTable {...props} columns={testcolumns} />);
    expect(tree).toMatchSnapshot();
  });

  it('calls reload function with an empty page token to get rows', () => {
    const reload = jest.fn();
    shallow(<CustomTable {...props} reload={reload} />);
    expect(reload).toHaveBeenLastCalledWith({ pageToken: '' });
  });

  it('calls reload function with sort key of clicked column, while keeping same page', () => {
    const testcolumns = [{
      flex: 3,
      label: 'col1',
      sortKey: 'col1sortkey',
    }, {
      flex: 1,
      label: 'col2',
      sortKey: 'col2sortkey',
    }];
    const reload = jest.fn();
    const tree = shallow(<CustomTable {...props} reload={reload} columns={testcolumns} />);
    expect(reload).toHaveBeenLastCalledWith({ pageToken: '' });

    tree.find('WithStyles(TableSortLabel)').at(0).simulate('click');
    expect(reload).toHaveBeenLastCalledWith({ orderAscending: true, sortBy: 'col1sortkey' });
  });

  it('calls reload function with same sort key in reverse order if same column is clicked twice', () => {
    const testcolumns = [{
      flex: 3,
      label: 'col1',
      sortKey: 'col1sortkey',
    }, {
      flex: 1,
      label: 'col2',
      sortKey: 'col2sortkey',
    }];
    const reload = jest.fn();
    const tree = shallow(<CustomTable {...props} reload={reload} columns={testcolumns} />);
    expect(reload).toHaveBeenLastCalledWith({ pageToken: '' });

    tree.find('WithStyles(TableSortLabel)').at(0).simulate('click');
    expect(reload).toHaveBeenLastCalledWith({ orderAscending: true, sortBy: 'col1sortkey' });
    tree.setProps({ sortBy: 'col1sortkey' });
    tree.find('WithStyles(TableSortLabel)').at(0).simulate('click');
    expect(reload).toHaveBeenLastCalledWith({ orderAscending: false, sortBy: 'col1sortkey' });
  });

  it('does not call reload if clicked column has no sort key', () => {
    const testcolumns = [{
      flex: 3,
      label: 'col1',
    }, {
      flex: 1,
      label: 'col2',
    }];
    const reload = jest.fn();
    const tree = shallow(<CustomTable {...props} reload={reload} columns={testcolumns} />);
    expect(reload).toHaveBeenLastCalledWith({ pageToken: '' });

    tree.find('WithStyles(TableSortLabel)').at(0).simulate('click');
    expect(reload).toHaveBeenLastCalledWith({ pageToken: '' });
  });

  it('throws if number of cells in a row has too many cells', () => {
    expect(() => shallow(<CustomTable {...props} rows={rows} />)).toThrow();
  });

  it('throws if number of cells in a row has too few cells', () => {
    const testcolumns = [{ label: 'col1' }, { label: 'col2' }, { label: 'col3' }];
    expect(() => shallow(
      <CustomTable {...props} rows={rows} columns={testcolumns} />)
    ).toThrow();
  });

  it('renders some rows', () => {
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders some rows using a custom renderer', () => {
    columns[0].customRenderer = () => (<span>this is custom output</span>) as any;
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} />);
    expect(tree).toMatchSnapshot();
    columns[0].customRenderer = undefined;
  });

  it('displays warning icon with tooltip if row has error', () => {
    rows[0].error = 'dummy error';
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} />);
    expect(tree).toMatchSnapshot();
    rows[0].error = undefined;
  });

  it('starts out with no selected rows', () => {
    const spy = jest.fn();
    shallow(<CustomTable {...props} rows={rows} columns={columns} updateSelection={spy} />);
    expect(spy).not.toHaveBeenCalled();
  });

  it('calls update selection callback when items are selected', () => {
    const spy = jest.fn();
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} updateSelection={spy} />);
    tree.find('.row').at(0).simulate('click', { stopPropagation: () => null });
    expect(spy).toHaveBeenLastCalledWith(['row1']);
  });

  it('does not add items to selection when multiple clicked', () => {
    // Keeping track of selection is the parent's job.
    const spy = jest.fn();
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} updateSelection={spy} />);
    tree.find('.row').at(0).simulate('click', { stopPropagation: () => null });
    tree.find('.row').at(1).simulate('click', { stopPropagation: () => null });
    expect(spy).toHaveBeenLastCalledWith(['row2']);
  });

  it('does not call selectionCallback if disableSelection is true', () => {
    const spy = jest.fn();
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns}
      updateSelection={spy} disableSelection={true} />);
    tree.find('.row').at(0).simulate('click', { stopPropagation: () => null });
    tree.find('.row').at(1).simulate('click', { stopPropagation: () => null });
    expect(spy).not.toHaveBeenCalled();
  });

  it('handles no updateSelection method being passed', () => {
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} />);
    tree.find('.row').at(0).simulate('click', { stopPropagation: () => null });
    tree.find('.columnName WithStyles(Checkbox)').at(0).simulate('change', {
      target: { checked: true },
    });
  });

  it('selects all items when head checkbox is clicked', () => {
    const spy = jest.fn();
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} updateSelection={spy} />);
    tree.find('.columnName WithStyles(Checkbox)').at(0).simulate('change', {
      target: { checked: true },
    });
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
  });

  it('unselects all items when head checkbox is clicked and all items are selected', () => {
    const spy = jest.fn();
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} updateSelection={spy} />);
    tree.find('.columnName WithStyles(Checkbox)').at(0).simulate('change', {
      target: { checked: true },
    });
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
    tree.find('.columnName WithStyles(Checkbox)').at(0).simulate('change', {
      target: { checked: false },
    });
    expect(spy).toHaveBeenLastCalledWith([]);
  });

  it('selects all items if one item was checked then the head checkbox is clicked', () => {
    const spy = jest.fn();
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} updateSelection={spy} />);
    tree.find('.row').at(0).simulate('click', { stopPropagation: () => null });
    tree.find('.columnName WithStyles(Checkbox)').at(0).simulate('change', {
      target: { checked: true },
    });
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
  });

  it('disables previous and next page buttons if no next page token given', async () => {
    const reloadResult = Promise.resolve('');
    const spy = () => reloadResult;
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} reload={spy} />);
    await reloadResult;
    expect(tree.state()).toHaveProperty('maxPageNumber', 0);
    expect(tree.find('WithStyles(IconButton)').at(0).prop('disabled')).toBeTruthy();
    expect(tree.find('WithStyles(IconButton)').at(1).prop('disabled')).toBeTruthy();
  });

  it('enables next page button if next page token is given', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = () => reloadResult;
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} reload={spy} />);
    await reloadResult;
    expect(tree.state()).toHaveProperty('maxPageNumber', Number.MAX_SAFE_INTEGER);
    expect(tree.find('WithStyles(IconButton)').at(0).prop('disabled')).toBeTruthy();
    expect(tree.find('WithStyles(IconButton)').at(1).prop('disabled')).not.toBeTruthy();
  });

  it('calls reload with next page token when next page button is clicked', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(<CustomTable {...props} rows={rows} columns={columns} reload={spy} />);
    await reloadResult;

    tree.find('WithStyles(IconButton)').at(1).simulate('click');
    expect(spy).toHaveBeenLastCalledWith({ pageToken: 'some token' });
  });

  it('renders new rows after clicking next page, and enables previous page button', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(<CustomTable {...props} rows={[]} columns={columns} reload={spy} />);
    await reloadResult;

    tree.find('WithStyles(IconButton)').at(1).simulate('click');
    await reloadResult;
    expect(spy).toHaveBeenLastCalledWith({ pageToken: 'some token' });
    expect(tree.state()).toHaveProperty('currentPage', 1);
    tree.setProps({ rows: [rows[1]] });
    expect(tree).toMatchSnapshot();
    expect(tree.find('WithStyles(IconButton)').at(0).prop('disabled')).not.toBeTruthy();
  });

  it('renders new rows after clicking previous page, and enables next page button', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(<CustomTable {...props} rows={[]} columns={columns} reload={spy} />);
    await reloadResult;

    tree.find('WithStyles(IconButton)').at(1).simulate('click');
    await reloadResult;

    tree.find('WithStyles(IconButton)').at(0).simulate('click');
    await reloadResult;
    expect(spy).toHaveBeenLastCalledWith({ pageToken: '' });

    tree.setProps({ rows });
    expect(tree.find('WithStyles(IconButton)').at(0).prop('disabled')).toBeTruthy();
    expect(tree).toMatchSnapshot();
  });

  it('calls reload with a different page size, resets page token list when rows/page changes', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(<CustomTable {...props} rows={[]} columns={columns} reload={spy} />);

    tree.find('.' + css.rowsPerPage).simulate('change', { target: { value: 1234 } });
    await reloadResult;
    expect(spy).toHaveBeenLastCalledWith({ pageSize: 1234, pageToken: '' });
    expect(tree.state()).toHaveProperty('tokenList', ['', 'some token']);
  });

  it('calls reload with a different page size, resets page token list when rows/page changes', async () => {
    const reloadResult = Promise.resolve('');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(<CustomTable {...props} rows={[]} columns={columns} reload={spy} />);

    tree.find('.' + css.rowsPerPage).simulate('change', { target: { value: 1234 } });
    await reloadResult;
    expect(spy).toHaveBeenLastCalledWith({ pageSize: 1234, pageToken: '' });
    expect(tree.state()).toHaveProperty('tokenList', ['']);
  });
});
