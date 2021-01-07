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
import CustomTable, { Column, ExpandState, Row, css } from './CustomTable';
import TestUtils from '../TestUtils';
import { PredicateOp } from '../apis/filter';
import { shallow } from 'enzyme';

jest.mock('react-i18next', () => ({
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: ((key: string) => key) as any };
    return Component;
  },
}));

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

// tslint:disable-next-line:no-console
const consoleErrorBackup = console.error;
let consoleSpy: jest.Mock;

class CustomTableTest extends CustomTable {
  public _requestFilter(filterString?: string): Promise<void> {
    return super._requestFilter(filterString);
  }
}

describe('CustomTable', () => {
  beforeAll(() => {
    consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => null);
  });

  afterAll(() => {
    // tslint:disable-next-line:no-console
    console.error = consoleErrorBackup;
  });

  it('renders with default filter label', async () => {
    const tree = shallow(<CustomTable t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders with provided filter label', async () => {
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} filterLabel='test filter label' />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders without filter box', async () => {
    const tree = shallow(<CustomTable t={(key: any) => key} {...props} noFilterBox={true} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders without rows or columns', async () => {
    const tree = shallow(<CustomTable t={(key: any) => key} {...props} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders empty message on no rows', async () => {
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} emptyMessage='test empty message' />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders some columns with equal widths without rows', async () => {
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        columns={[{ label: 'col1' }, { label: 'col2' }]}
      />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders without the checkboxes if disableSelection is true', async () => {
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        disableSelection={true}
      />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders some columns with descending sort order on first column', async () => {
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        initialSortOrder='desc'
        columns={[{ label: 'col1', sortKey: 'col1sortkey' }, { label: 'col2' }]}
      />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders columns with specified widths', async () => {
    const testcolumns = [
      {
        flex: 3,
        label: 'col1',
      },
      {
        flex: 1,
        label: 'col2',
      },
    ];
    const tree = shallow(<CustomTable t={(key: any) => key} {...props} columns={testcolumns} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('calls reload function with an empty page token to get rows', () => {
    const reload = jest.fn();
    shallow(<CustomTable t={(key: any) => key} {...props} reload={reload} />);
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: '',
    });
  });

  it('calls reload function with sort key of clicked column, while keeping same page', () => {
    const testcolumns = [
      {
        flex: 3,
        label: 'col1',
        sortKey: 'col1sortkey',
      },
      {
        flex: 1,
        label: 'col2',
        sortKey: 'col2sortkey',
      },
    ];
    const reload = jest.fn();
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} reload={reload} columns={testcolumns} />,
    );
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: 'col1sortkey desc',
    });

    tree
      .find('WithStyles(TableSortLabel)')
      .at(1)
      .simulate('click');
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: true,
      pageSize: 10,
      pageToken: '',
      sortBy: 'col2sortkey',
    });
  });

  it('calls reload function with same sort key in reverse order if same column is clicked twice', () => {
    const testcolumns = [
      {
        flex: 3,
        label: 'col1',
        sortKey: 'col1sortkey',
      },
      {
        flex: 1,
        label: 'col2',
        sortKey: 'col2sortkey',
      },
    ];
    const reload = jest.fn();
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} reload={reload} columns={testcolumns} />,
    );
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: 'col1sortkey desc',
    });

    tree
      .find('WithStyles(TableSortLabel)')
      .at(1)
      .simulate('click');
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: true,
      pageSize: 10,
      pageToken: '',
      sortBy: 'col2sortkey',
    });
    tree.setProps({ sortBy: 'col1sortkey' });
    tree
      .find('WithStyles(TableSortLabel)')
      .at(1)
      .simulate('click');
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: 'col2sortkey desc',
    });
  });

  it('does not call reload if clicked column has no sort key', () => {
    const testcolumns = [
      {
        flex: 3,
        label: 'col1',
      },
      {
        flex: 1,
        label: 'col2',
      },
    ];
    const reload = jest.fn();
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} reload={reload} columns={testcolumns} />,
    );
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: '',
    });

    tree
      .find('WithStyles(TableSortLabel)')
      .at(0)
      .simulate('click');
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: '',
    });
  });

  it('logs error if row has more cells than columns', () => {
    shallow(<CustomTable t={(key: any) => key} {...props} rows={rows} />);
    expect(consoleSpy).toHaveBeenLastCalledWith(
      'Rows must have the same number of cells defined in columns',
    );
  });

  it('logs error if row has fewer cells than columns', () => {
    const testcolumns = [{ label: 'col1' }, { label: 'col2' }, { label: 'col3' }];
    shallow(<CustomTable t={(key: any) => key} {...props} rows={rows} columns={testcolumns} />);
    expect(consoleSpy).toHaveBeenLastCalledWith(
      'Rows must have the same number of cells defined in columns',
    );
  });

  it('renders some rows', async () => {
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={rows} columns={columns} />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('starts out with no selected rows', () => {
    const spy = jest.fn();
    shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        updateSelection={spy}
      />,
    );
    expect(spy).not.toHaveBeenCalled();
  });

  it('calls update selection callback when items are selected', () => {
    const spy = jest.fn();
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        updateSelection={spy}
      />,
    );
    tree
      .find('.row')
      .at(0)
      .simulate('click', { stopPropagation: () => null });
    expect(spy).toHaveBeenLastCalledWith(['row1']);
  });

  it('does not add items to selection when multiple rows are clicked', () => {
    // Keeping track of selection is the parent's job.
    const spy = jest.fn();
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        updateSelection={spy}
      />,
    );
    tree
      .find('.row')
      .at(0)
      .simulate('click', { stopPropagation: () => null });
    tree
      .find('.row')
      .at(1)
      .simulate('click', { stopPropagation: () => null });
    expect(spy).toHaveBeenLastCalledWith(['row2']);
  });

  it('passes both selectedIds and the newly selected row to updateSelection when a row is clicked', () => {
    // Keeping track of selection is the parent's job.
    const selectedIds = ['previouslySelectedRow'];
    const spy = jest.fn();
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        selectedIds={selectedIds}
        rows={rows}
        columns={columns}
        updateSelection={spy}
      />,
    );
    tree
      .find('.row')
      .at(0)
      .simulate('click', { stopPropagation: () => null });
    expect(spy).toHaveBeenLastCalledWith(['previouslySelectedRow', 'row1']);
  });

  it('does not call selectionCallback if disableSelection is true', () => {
    const spy = jest.fn();
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        updateSelection={spy}
        disableSelection={true}
      />,
    );
    tree
      .find('.row')
      .at(0)
      .simulate('click', { stopPropagation: () => null });
    tree
      .find('.row')
      .at(1)
      .simulate('click', { stopPropagation: () => null });
    expect(spy).not.toHaveBeenCalled();
  });

  it('handles no updateSelection method being passed', () => {
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={rows} columns={columns} />,
    );
    tree
      .find('.row')
      .at(0)
      .simulate('click', { stopPropagation: () => null });
    tree
      .find('.columnName WithStyles(Checkbox)')
      .at(0)
      .simulate('change', {
        target: { checked: true },
      });
  });

  it('selects all items when head checkbox is clicked', () => {
    const spy = jest.fn();
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        updateSelection={spy}
      />,
    );
    tree
      .find('.columnName WithStyles(Checkbox)')
      .at(0)
      .simulate('change', {
        target: { checked: true },
      });
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
  });

  it('unselects all items when head checkbox is clicked and all items are selected', () => {
    const spy = jest.fn();
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        updateSelection={spy}
      />,
    );
    tree
      .find('.columnName WithStyles(Checkbox)')
      .at(0)
      .simulate('change', {
        target: { checked: true },
      });
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
    tree
      .find('.columnName WithStyles(Checkbox)')
      .at(0)
      .simulate('change', {
        target: { checked: false },
      });
    expect(spy).toHaveBeenLastCalledWith([]);
  });

  it('selects all items if one item was checked then the head checkbox is clicked', () => {
    const spy = jest.fn();
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        updateSelection={spy}
      />,
    );
    tree
      .find('.row')
      .at(0)
      .simulate('click', { stopPropagation: () => null });
    tree
      .find('.columnName WithStyles(Checkbox)')
      .at(0)
      .simulate('change', {
        target: { checked: true },
      });
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
  });

  it('deselects all other items if one item is selected in radio button mode', () => {
    // Simulate that another row has already been selected. Just clicking another row first won't
    // work here because the parent is where the selectedIds state is kept
    const selectedIds = ['previouslySelectedRow'];
    const spy = jest.fn();
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        useRadioButtons={true}
        selectedIds={selectedIds}
        rows={rows}
        columns={columns}
        updateSelection={spy}
      />,
    );
    tree
      .find('.row')
      .at(0)
      .simulate('click', { stopPropagation: () => null });
    expect(spy).toHaveBeenLastCalledWith(['row1']);
  });

  it('disables previous and next page buttons if no next page token given', async () => {
    const reloadResult = Promise.resolve('');
    const spy = () => reloadResult;
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={rows} columns={columns} reload={spy} />,
    );
    await TestUtils.flushPromises();
    expect(tree.state()).toHaveProperty('maxPageIndex', 0);
    expect(
      tree
        .find('WithStyles(IconButton)')
        .at(0)
        .prop('disabled'),
    ).toBeTruthy();
    expect(
      tree
        .find('WithStyles(IconButton)')
        .at(1)
        .prop('disabled'),
    ).toBeTruthy();
  });

  it('enables next page button if next page token is given', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = () => reloadResult;
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={rows} columns={columns} reload={spy} />,
    );
    await reloadResult;
    expect(tree.state()).toHaveProperty('maxPageIndex', Number.MAX_SAFE_INTEGER);
    expect(
      tree
        .find('WithStyles(IconButton)')
        .at(0)
        .prop('disabled'),
    ).toBeTruthy();
    expect(
      tree
        .find('WithStyles(IconButton)')
        .at(1)
        .prop('disabled'),
    ).not.toBeTruthy();
  });

  it('calls reload with next page token when next page button is clicked', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={rows} columns={columns} reload={spy} />,
    );
    await TestUtils.flushPromises();

    tree
      .find('WithStyles(IconButton)')
      .at(1)
      .simulate('click');
    expect(spy).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: 'some token',
      sortBy: '',
    });
  });

  it('renders new rows after clicking next page, and enables previous page button', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={[]} columns={columns} reload={spy} />,
    );
    await TestUtils.flushPromises();

    tree
      .find('WithStyles(IconButton)')
      .at(1)
      .simulate('click');
    await TestUtils.flushPromises();
    expect(spy).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: 'some token',
      sortBy: '',
    });
    expect(tree.state()).toHaveProperty('currentPage', 1);
    tree.setProps({ rows: [rows[1]] });
    expect(tree).toMatchSnapshot();
    expect(
      tree
        .find('WithStyles(IconButton)')
        .at(0)
        .prop('disabled'),
    ).not.toBeTruthy();
  });

  it('renders new rows after clicking previous page, and enables next page button', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={[]} columns={columns} reload={spy} />,
    );
    await reloadResult;

    tree
      .find('WithStyles(IconButton)')
      .at(1)
      .simulate('click');
    await reloadResult;

    tree
      .find('WithStyles(IconButton)')
      .at(0)
      .simulate('click');
    await TestUtils.flushPromises();
    expect(spy).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: '',
    });

    tree.setProps({ rows });
    expect(
      tree
        .find('WithStyles(IconButton)')
        .at(0)
        .prop('disabled'),
    ).toBeTruthy();
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('calls reload with a different page size, resets page token list when rows/page changes', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={[]} columns={columns} reload={spy} />,
    );

    tree.find('.' + css.rowsPerPage).simulate('change', { target: { value: 1234 } });
    await TestUtils.flushPromises();
    expect(spy).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 1234,
      pageToken: '',
      sortBy: '',
    });
    expect(tree.state()).toHaveProperty('tokenList', ['', 'some token']);
  });

  it('calls reload with a different page size, resets page token list when rows/page changes', async () => {
    const reloadResult = Promise.resolve('');
    const spy = jest.fn(() => reloadResult);
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={[]} columns={columns} reload={spy} />,
    );

    tree.find('.' + css.rowsPerPage).simulate('change', { target: { value: 1234 } });
    await reloadResult;
    expect(spy).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 1234,
      pageToken: '',
      sortBy: '',
    });
    expect(tree.state()).toHaveProperty('tokenList', ['']);
  });

  it('renders a collapsed row', async () => {
    const row = { ...rows[0] };
    row.expandState = ExpandState.COLLAPSED;
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={[row]}
        columns={columns}
        getExpandComponent={() => null}
      />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders a collapsed row when selection is disabled', async () => {
    const row = { ...rows[0] };
    row.expandState = ExpandState.COLLAPSED;
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={[row]}
        columns={columns}
        getExpandComponent={() => null}
        disableSelection={true}
      />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders an expanded row', async () => {
    const row = { ...rows[0] };
    row.expandState = ExpandState.EXPANDED;
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={[row]} columns={columns} />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders an expanded row with expanded component below it', async () => {
    const row = { ...rows[0] };
    row.expandState = ExpandState.EXPANDED;
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={[row]}
        columns={columns}
        getExpandComponent={() => <span>Hello World</span>}
      />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('calls prop to toggle expansion', () => {
    const row = { ...rows[0] };
    const toggleSpy = jest.fn();
    const stopPropagationSpy = jest.fn();
    row.expandState = ExpandState.EXPANDED;
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={[row, row, row]}
        columns={columns}
        getExpandComponent={() => <span>Hello World</span>}
        toggleExpansion={toggleSpy}
      />,
    );
    tree
      .find('.' + css.expandButton)
      .at(1)
      .simulate('click', { stopPropagation: stopPropagationSpy });
    expect(toggleSpy).toHaveBeenCalledWith(1);
    expect(stopPropagationSpy).toHaveBeenCalledWith();
  });

  it('renders a table with sorting disabled', async () => {
    const tree = shallow(
      <CustomTable
        t={(key: any) => key}
        {...props}
        rows={rows}
        columns={columns}
        disableSorting={true}
      />,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('updates the filter string in state when the filter box input changes', async () => {
    const tree = shallow(
      <CustomTable t={(key: any) => key} {...props} rows={rows} columns={columns} />,
    );
    (tree.instance() as CustomTable).handleFilterChange({ target: { value: 'test filter' } });
    await TestUtils.flushPromises();
    expect(tree.state('filterString')).toEqual('test filter');
    expect(tree).toMatchSnapshot();
  });

  it('reloads the table with the encoded filter object', async () => {
    const reload = jest.fn();
    const tree = shallow(
      <CustomTableTest
        t={(key: any) => key}
        {...props}
        reload={reload}
        rows={rows}
        columns={columns}
      />,
    );
    // lodash's debounce function doesn't play nice with Jest, so we skip the handleChange function
    // and call _requestFilter directly.
    (tree.instance() as CustomTableTest)._requestFilter('test filter');
    const expectedEncodedFilter = encodeURIComponent(
      JSON.stringify({
        predicates: [
          {
            key: 'name',
            op: PredicateOp.ISSUBSTRING,
            string_value: 'test filter',
          },
        ],
      }),
    );
    expect(tree.state('filterStringEncoded')).toEqual(expectedEncodedFilter);
    expect(reload).toHaveBeenLastCalledWith({
      filter: expectedEncodedFilter,
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: '',
    });
  });

  it('uses an empty filter if requestFilter is called with no filter', async () => {
    const tree = shallow(
      <CustomTableTest t={(key: any) => key} {...props} rows={rows} columns={columns} />,
    );
    (tree.instance() as CustomTableTest)._requestFilter();
    expect(tree.state('filterStringEncoded')).toEqual('');
  });
});
