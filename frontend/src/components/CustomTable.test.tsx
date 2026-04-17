/*
 * Copyright 2018 The Kubeflow Authors
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
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { vi } from 'vitest';
import CustomTable, { Column, ExpandState, Row } from './CustomTable';
import TestUtils, { flushPromisesInAct, invokeAndFlush } from '../TestUtils';
import { V2beta1PredicateOperation } from '../apisv2beta1/filter';
import { logger } from 'src/lib/Utils';

type CustomTableProps = React.ComponentProps<typeof CustomTable>;

type CustomTableState = CustomTable['state'];

class CustomTableTest extends CustomTable {
  public _requestFilter(filterString?: string): Promise<void> {
    return super._requestFilter(filterString);
  }
}

class CustomTableWrapper {
  private readonly _instanceHolder: { current: CustomTableTest | null };
  private _renderResult: ReturnType<typeof render>;

  public constructor(
    instanceHolder: { current: CustomTableTest | null },
    renderResult: ReturnType<typeof render>,
  ) {
    this._instanceHolder = instanceHolder;
    this._renderResult = renderResult;
  }

  public instance(): CustomTableTest {
    const instance = this._instanceHolder.current;
    if (!instance) {
      throw new Error('CustomTable instance not available');
    }
    return instance;
  }

  public state<K extends keyof CustomTableState>(key?: K): CustomTableState | CustomTableState[K] {
    const state = this.instance().state;
    return key ? state[key] : state;
  }

  public rerender(props: CustomTableProps): void {
    const setTableRef = (instance: CustomTableTest | null): void => {
      this._instanceHolder.current = instance;
    };
    this._renderResult.rerender(<CustomTableTest ref={setTableRef} {...props} />);
  }

  public unmount(): void {
    this._renderResult.unmount();
  }

  public renderResult(): ReturnType<typeof render> {
    return this._renderResult;
  }
}

const baseProps: CustomTableProps = {
  columns: [],
  reload: async () => '',
  rows: [],
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

function renderTable(overrides: Partial<CustomTableProps> = {}): CustomTableWrapper {
  const props = { ...baseProps, ...overrides } as CustomTableProps;
  const instanceHolder: { current: CustomTableTest | null } = { current: null };
  const setTableRef = (instance: CustomTableTest | null): void => {
    instanceHolder.current = instance;
  };
  const renderResult = render(<CustomTableTest ref={setTableRef} {...props} />);
  if (!instanceHolder.current) {
    throw new Error('CustomTable instance not available');
  }
  return new CustomTableWrapper(instanceHolder, renderResult);
}

function getHeaderCheckbox(container: HTMLElement): HTMLInputElement {
  const checkbox = container.querySelector(
    '[class*="header"] input[type="checkbox"]',
  ) as HTMLInputElement | null;
  if (!checkbox) {
    throw new Error('Header checkbox not found.');
  }
  return checkbox;
}

function getRowsPerPageCombobox(): HTMLElement {
  const footer = screen.getByText('Rows per page:').closest('div');
  if (!footer) {
    throw new Error('Unable to locate table footer containing the rows-per-page selector');
  }
  return within(footer).getByRole('combobox');
}

async function selectRowsPerPage(pageSize: number): Promise<void> {
  await act(async () => {
    fireEvent.mouseDown(getRowsPerPageCombobox());
  });
  const option = await screen.findByRole('option', { name: String(pageSize) });
  await act(async () => {
    fireEvent.click(option);
    await TestUtils.flushPromises();
  });
}

describe('CustomTable', () => {
  beforeEach(() => {
    vi.useRealTimers();
    localStorage.clear();
  });

  it('renders with default filter label', async () => {
    const wrapper = renderTable();
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders with provided filter label', async () => {
    const wrapper = renderTable({ filterLabel: 'test filter label' });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders without filter box', async () => {
    const wrapper = renderTable({ noFilterBox: true });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders without rows or columns', async () => {
    const wrapper = renderTable();
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders empty message on no rows', async () => {
    const wrapper = renderTable({ emptyMessage: 'test empty message' });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders some columns with equal widths without rows', async () => {
    const wrapper = renderTable({ columns: [{ label: 'col1' }, { label: 'col2' }] });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders without the checkboxes if disableSelection is true', async () => {
    const wrapper = renderTable({ rows, columns, disableSelection: true });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders some columns with descending sort order on first column', async () => {
    const wrapper = renderTable({
      columns: [{ label: 'col1', sortKey: 'col1sortkey' }, { label: 'col2' }],
      initialSortOrder: 'desc',
    });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders columns with specified widths', async () => {
    const wrapper = renderTable({
      columns: [
        {
          flex: 3,
          label: 'col1',
        },
        {
          flex: 1,
          label: 'col2',
        },
      ],
    });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('calls reload function with an empty page token to get rows', async () => {
    const reload = vi.fn(async () => '');
    renderTable({ reload });
    await waitFor(() => expect(reload).toHaveBeenCalled());
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: '',
    });
  });

  it('calls reload function with sort key of clicked column, while keeping same page', async () => {
    const testColumns = [
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
    const reload = vi.fn(async () => '');
    renderTable({ columns: testColumns, reload });
    await waitFor(() => expect(reload).toHaveBeenCalled());
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: 'col1sortkey desc',
    });

    fireEvent.click(screen.getByText('col2'));
    await waitFor(() =>
      expect(reload).toHaveBeenLastCalledWith({
        filter: '',
        orderAscending: true,
        pageSize: 10,
        pageToken: '',
        sortBy: 'col2sortkey',
      }),
    );
  });

  it('calls reload function with same sort key in reverse order if same column is clicked twice', async () => {
    const testColumns = [
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
    const reload = vi.fn(async () => '');
    const wrapper = renderTable({ columns: testColumns, reload });
    await waitFor(() => expect(reload).toHaveBeenCalled());
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: 'col1sortkey desc',
    });

    fireEvent.click(screen.getByText('col2'));
    await waitFor(() =>
      expect(reload).toHaveBeenLastCalledWith({
        filter: '',
        orderAscending: true,
        pageSize: 10,
        pageToken: '',
        sortBy: 'col2sortkey',
      }),
    );

    fireEvent.click(screen.getByText('col2'));
    await waitFor(() =>
      expect(reload).toHaveBeenLastCalledWith({
        filter: '',
        orderAscending: false,
        pageSize: 10,
        pageToken: '',
        sortBy: 'col2sortkey desc',
      }),
    );
    wrapper.unmount();
  });

  it('does not call reload if clicked column has no sort key', async () => {
    const testColumns = [
      {
        flex: 3,
        label: 'col1',
      },
      {
        flex: 1,
        label: 'col2',
      },
    ];
    const reload = vi.fn(async () => '');
    renderTable({ columns: testColumns, reload });
    await waitFor(() => expect(reload).toHaveBeenCalled());
    const previousCallCount = reload.mock.calls.length;
    fireEvent.click(screen.getByText('col1'));
    expect(reload).toHaveBeenCalledTimes(previousCallCount);
  });

  it('does not render sort icon for columns without sort key', async () => {
    renderTable({
      columns: [{ label: 'sortable', sortKey: 'sortableKey' }, { label: 'unsortable' }],
      rows,
    });
    await flushPromisesInAct();

    const sortableHeader = screen.getByText('sortable').closest('.MuiTableSortLabel-root');
    const unsortableHeader = screen.getByText('unsortable').closest('.MuiTableSortLabel-root');

    expect(sortableHeader?.querySelector('.MuiTableSortLabel-icon')).toBeTruthy();
    expect(unsortableHeader?.querySelector('.MuiTableSortLabel-icon')).toBeNull();
  });

  it('logs error if row has more cells than columns', () => {
    const loggerSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);
    const wrapper = renderTable({ rows });
    expect(loggerSpy).toHaveBeenCalledWith(
      'Rows must have the same number of cells defined in columns',
    );
    wrapper.unmount();
    loggerSpy.mockRestore();
  });

  it('logs error if row has fewer cells than columns', () => {
    const loggerSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);
    const testColumns = [{ label: 'col1' }, { label: 'col2' }, { label: 'col3' }];
    const wrapper = renderTable({ rows, columns: testColumns });
    expect(loggerSpy).toHaveBeenCalledWith(
      'Rows must have the same number of cells defined in columns',
    );
    wrapper.unmount();
    loggerSpy.mockRestore();
  });

  it('renders some rows', async () => {
    const wrapper = renderTable({ rows, columns });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  }, 20000);

  it('starts out with no selected rows', async () => {
    const spy = vi.fn();
    renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    expect(spy).not.toHaveBeenCalled();
  });

  it('calls update selection callback when items are selected', async () => {
    const spy = vi.fn();
    renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    fireEvent.click(screen.getAllByTestId('table-row')[0]);
    expect(spy).toHaveBeenLastCalledWith(['row1']);
  });

  it('does not add items to selection when multiple rows are clicked', async () => {
    const spy = vi.fn();
    renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    fireEvent.click(screen.getAllByTestId('table-row')[0]);
    fireEvent.click(screen.getAllByTestId('table-row')[1]);
    expect(spy).toHaveBeenLastCalledWith(['row2']);
  });

  it('passes both selectedIds and the newly selected row to updateSelection when a row is clicked', async () => {
    const selectedIds = ['previouslySelectedRow'];
    const spy = vi.fn();
    renderTable({ rows, columns, selectedIds, updateSelection: spy });
    await flushPromisesInAct();
    fireEvent.click(screen.getAllByTestId('table-row')[0]);
    expect(spy).toHaveBeenLastCalledWith(['previouslySelectedRow', 'row1']);
  });

  it('does not call selectionCallback if disableSelection is true', async () => {
    const spy = vi.fn();
    renderTable({ rows, columns, updateSelection: spy, disableSelection: true });
    await flushPromisesInAct();
    fireEvent.click(screen.getAllByTestId('table-row')[0]);
    fireEvent.click(screen.getAllByTestId('table-row')[1]);
    expect(spy).not.toHaveBeenCalled();
  });

  it('handles no updateSelection method being passed', async () => {
    renderTable({ rows, columns });
    await flushPromisesInAct();
    fireEvent.click(screen.getAllByTestId('table-row')[0]);
    const headerCheckbox = getHeaderCheckbox(document.body);
    fireEvent.click(headerCheckbox);
  });

  it('selects all items when head checkbox is clicked', async () => {
    const spy = vi.fn();
    const wrapper = renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    const headerCheckbox = getHeaderCheckbox(wrapper.renderResult().container);
    fireEvent.click(headerCheckbox);
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
    wrapper.unmount();
  });

  it('unselects all items when head checkbox is clicked and all items are selected', async () => {
    const spy = vi.fn();
    const wrapper = renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    const headerCheckbox = getHeaderCheckbox(wrapper.renderResult().container);
    fireEvent.click(headerCheckbox);
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
    wrapper.rerender({
      ...baseProps,
      rows,
      columns,
      updateSelection: spy,
      selectedIds: ['row1', 'row2'],
    });
    const updatedHeaderCheckbox = getHeaderCheckbox(wrapper.renderResult().container);
    fireEvent.click(updatedHeaderCheckbox);
    expect(spy).toHaveBeenLastCalledWith([]);
    wrapper.unmount();
  });

  it('selects all items if one item was checked then the head checkbox is clicked', async () => {
    const spy = vi.fn();
    const wrapper = renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    fireEvent.click(screen.getAllByTestId('table-row')[0]);
    const headerCheckbox = getHeaderCheckbox(wrapper.renderResult().container);
    fireEvent.click(headerCheckbox);
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
    wrapper.unmount();
  });

  it('deselects all other items if one item is selected in radio button mode', async () => {
    const selectedIds = ['previouslySelectedRow'];
    const spy = vi.fn();
    renderTable({ rows, columns, useRadioButtons: true, selectedIds, updateSelection: spy });
    await flushPromisesInAct();
    fireEvent.click(screen.getAllByTestId('table-row')[0]);
    expect(spy).toHaveBeenLastCalledWith(['row1']);
  });

  it('disables previous and next page buttons if no next page token given', async () => {
    const reloadResult = Promise.resolve('');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows, columns, reload: spy });
    await flushPromisesInAct();
    expect(wrapper.state()).toHaveProperty('maxPageIndex', 0);
    const buttons = wrapper.renderResult().container.querySelectorAll('button');
    expect(buttons[0].hasAttribute('disabled')).toBe(true);
    expect(buttons[1].hasAttribute('disabled')).toBe(true);
    wrapper.unmount();
  });

  it('enables next page button if next page token is given', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows, columns, reload: spy });
    await flushPromisesInAct();
    const buttons = wrapper.renderResult().container.querySelectorAll('button');
    expect(wrapper.state()).toHaveProperty('maxPageIndex', Number.MAX_SAFE_INTEGER);
    expect(buttons[0].hasAttribute('disabled')).toBe(true);
    expect(buttons[1].hasAttribute('disabled')).toBe(false);
    wrapper.unmount();
  });

  it('calls reload with next page token when next page button is clicked', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows, columns, reload: spy });
    await flushPromisesInAct();
    const buttons = wrapper.renderResult().container.querySelectorAll('button');
    fireEvent.click(buttons[1]);
    await waitFor(() =>
      expect(spy).toHaveBeenLastCalledWith({
        filter: '',
        orderAscending: false,
        pageSize: 10,
        pageToken: 'some token',
        sortBy: '',
      }),
    );
    wrapper.unmount();
  });

  it('renders new rows after clicking next page, and enables previous page button', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows: [], columns, reload: spy });
    await flushPromisesInAct();
    const buttons = wrapper.renderResult().container.querySelectorAll('button');
    fireEvent.click(buttons[1]);
    await flushPromisesInAct();
    expect(spy).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 10,
      pageToken: 'some token',
      sortBy: '',
    });
    expect(wrapper.state()).toHaveProperty('currentPage', 1);
    wrapper.rerender({ ...baseProps, rows: [rows[1]], columns, reload: spy });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    const updatedButtons = wrapper.renderResult().container.querySelectorAll('button');
    expect(updatedButtons[0].hasAttribute('disabled')).toBe(false);
    wrapper.unmount();
  });

  it('renders new rows after clicking previous page, and enables next page button', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows: [], columns, reload: spy });
    await flushPromisesInAct();
    const buttons = wrapper.renderResult().container.querySelectorAll('button');
    fireEvent.click(buttons[1]);
    await flushPromisesInAct();
    fireEvent.click(buttons[0]);
    await flushPromisesInAct();
    await waitFor(() =>
      expect(spy).toHaveBeenLastCalledWith({
        filter: '',
        orderAscending: false,
        pageSize: 10,
        pageToken: '',
        sortBy: '',
      }),
    );
    wrapper.rerender({ ...baseProps, rows, columns, reload: spy });
    const updatedButtons = wrapper.renderResult().container.querySelectorAll('button');
    expect(updatedButtons[0].hasAttribute('disabled')).toBe(true);
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('reloads with new page size and appends next-page token when rows/page changes', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows: [], columns, reload: spy });
    await selectRowsPerPage(20);
    expect(spy).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 20,
      pageToken: '',
      sortBy: '',
    });
    expect(wrapper.state()).toHaveProperty('tokenList', ['', 'some token']);
    wrapper.unmount();
  });

  it('reloads with new page size and resets token list when no next page exists', async () => {
    const reloadResult = Promise.resolve('');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows: [], columns, reload: spy });
    await selectRowsPerPage(20);
    await reloadResult;
    expect(spy).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 20,
      pageToken: '',
      sortBy: '',
    });
    expect(wrapper.state()).toHaveProperty('tokenList', ['']);
    wrapper.unmount();
  });

  it('renders a collapsed row', async () => {
    const row = { ...rows[0], expandState: ExpandState.COLLAPSED };
    const wrapper = renderTable({ rows: [row], columns, getExpandComponent: () => null });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders a collapsed row when selection is disabled', async () => {
    const row = { ...rows[0], expandState: ExpandState.COLLAPSED };
    const wrapper = renderTable({
      rows: [row],
      columns,
      getExpandComponent: () => null,
      disableSelection: true,
    });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders an expanded row', async () => {
    const row = { ...rows[0], expandState: ExpandState.EXPANDED };
    const wrapper = renderTable({ rows: [row], columns });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders an expanded row with expanded component below it', async () => {
    const row = { ...rows[0], expandState: ExpandState.EXPANDED };
    const wrapper = renderTable({
      rows: [row],
      columns,
      getExpandComponent: () => <span>Hello World</span>,
    });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('calls prop to toggle expansion', async () => {
    const row = { ...rows[0], expandState: ExpandState.EXPANDED };
    const toggleSpy = vi.fn();
    renderTable({
      rows: [row, row, row],
      columns,
      getExpandComponent: () => <span>Hello World</span>,
      toggleExpansion: toggleSpy,
    });
    await flushPromisesInAct();
    const expandButtons = screen.getAllByLabelText('Expand');
    fireEvent.click(expandButtons[1]);
    expect(toggleSpy).toHaveBeenCalledWith(1);
  });

  it('renders a table with sorting disabled', async () => {
    const wrapper = renderTable({ rows, columns, disableSorting: true });
    await flushPromisesInAct();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('updates the filter string in state when the filter box input changes', async () => {
    const wrapper = renderTable({ rows, columns });
    await invokeAndFlush(() =>
      wrapper.instance().handleFilterChange({ target: { value: 'test filter' } }),
    );
    expect(wrapper.state('filterString')).toEqual('test filter');
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('reloads the table with the encoded filter object', async () => {
    const reload = vi.fn(async () => '');
    const wrapper = renderTable({ rows, columns, reload });
    await flushPromisesInAct();
    await act(async () => {
      await wrapper.instance()._requestFilter('test filter');
    });
    const expectedEncodedFilter = encodeURIComponent(
      JSON.stringify({
        predicates: [
          {
            key: 'name',
            operation: V2beta1PredicateOperation.IS_SUBSTRING,
            string_value: 'test filter',
          },
        ],
      }),
    );
    expect(wrapper.state('filterStringEncoded')).toEqual(expectedEncodedFilter);
    expect(reload).toHaveBeenLastCalledWith({
      filter: expectedEncodedFilter,
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      sortBy: '',
    });
    wrapper.unmount();
  });

  it('uses an empty filter if requestFilter is called with no filter', async () => {
    const wrapper = renderTable({ rows, columns });
    await invokeAndFlush(() => wrapper.instance()._requestFilter());
    expect(wrapper.state('filterStringEncoded')).toEqual('');
    wrapper.unmount();
  });

  it('The initial filter string is called during first reload', async () => {
    const reload = vi.fn(async () => '');
    renderTable({
      rows,
      columns,
      reload,
      initialFilterString: 'test filter',
    });
    const expectedEncodedFilter = encodeURIComponent(
      JSON.stringify({
        predicates: [
          {
            key: 'name',
            operation: V2beta1PredicateOperation.IS_SUBSTRING,
            string_value: 'test filter',
          },
        ],
      }),
    );
    await waitFor(() =>
      expect(reload).toHaveBeenLastCalledWith({
        filter: expectedEncodedFilter,
        orderAscending: false,
        pageSize: 10,
        pageToken: '',
        sortBy: '',
      }),
    );
  });

  it('The setFilterString method is called when the filter text is changed', async () => {
    const setFilterString = vi.fn();
    renderTable({ rows, columns, setFilterString });
    await flushPromisesInAct();
    fireEvent.change(screen.getByLabelText('Filter'), { target: { value: 'test filter' } });
    expect(setFilterString).toHaveBeenLastCalledWith('test filter');
  });

  it('reads page size from localStorage on mount', async () => {
    localStorage.setItem('tablePageSize', '50');
    const reload = vi.fn(async () => '');
    const wrapper = renderTable({ rows, columns, reload });
    await waitFor(() => expect(reload).toHaveBeenCalled());
    expect(reload).toHaveBeenLastCalledWith({
      filter: '',
      orderAscending: false,
      pageSize: 50,
      pageToken: '',
      sortBy: '',
    });
    wrapper.unmount();
  });

  it('persists page size across resources on the same details page', async () => {
    window.location.hash = '#/runs/details/run-abc123';
    const firstWrapper = renderTable({ rows: [], columns, reload: vi.fn(async () => '') });
    await selectRowsPerPage(20);
    firstWrapper.unmount();
    const reload = vi.fn(async () => '');
    window.location.hash = '#/runs/details/run-def456';
    const secondWrapper = renderTable({ rows, columns, reload });
    await waitFor(() =>
      expect(reload).toHaveBeenLastCalledWith({
        filter: '',
        orderAscending: false,
        pageSize: 20,
        pageToken: '',
        sortBy: '',
      }),
    );
    secondWrapper.unmount();
    window.location.hash = '';
  });
});
