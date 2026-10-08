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
import { ResourceTable } from './tables/ResourceTable';

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
  renderTable: (table) => (
    <ResourceTable table={table} label='Test resources' getRowLabel={(row) => row.id} />
  ),
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

function getHeaderCheckbox(): HTMLElement {
  return screen.getByRole('checkbox', { name: 'Select all resources on this page' });
}

function getRowsPerPageCombobox(): HTMLElement {
  return screen.getByRole('combobox', { name: 'Rows per page' });
}

async function selectRowsPerPage(pageSize: number): Promise<void> {
  await waitFor(() => expect(getRowsPerPageCombobox()).toBeEnabled());
  await act(async () => {
    fireEvent.change(getRowsPerPageCombobox(), { target: { value: String(pageSize) } });
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
    expect(screen.getByLabelText('Filter')).toBeInTheDocument();
    wrapper.unmount();
  });

  it('renders with provided filter label', async () => {
    const wrapper = renderTable({ filterLabel: 'test filter label' });
    await flushPromisesInAct();
    expect(screen.getByLabelText('test filter label')).toBeInTheDocument();
    wrapper.unmount();
  });

  it('renders without filter box', async () => {
    const wrapper = renderTable({ noFilterBox: true });
    await flushPromisesInAct();
    expect(screen.queryByLabelText('Filter')).toBeNull();
    wrapper.unmount();
  });

  it('renders without rows or columns', async () => {
    const wrapper = renderTable();
    await flushPromisesInAct();
    expect(screen.queryAllByTestId('table-row')).toHaveLength(0);
    wrapper.unmount();
  });

  it('renders empty message on no rows', async () => {
    const wrapper = renderTable({ emptyMessage: 'test empty message' });
    await flushPromisesInAct();
    expect(screen.getByText('test empty message')).toBeInTheDocument();
    wrapper.unmount();
  });

  it('renders named column headers in the supplied order without rows', async () => {
    const wrapper = renderTable({ columns: [{ label: 'col1' }, { label: 'col2' }] });
    await flushPromisesInAct();
    expect(
      screen
        .getAllByRole('columnheader')
        .map((header) => header.textContent)
        .filter(Boolean),
    ).toEqual(['col1', 'col2']);
    wrapper.unmount();
  });

  it('renders without the checkboxes if disableSelection is true', async () => {
    const wrapper = renderTable({ rows, columns, disableSelection: true });
    await flushPromisesInAct();
    expect(
      screen.queryByRole('checkbox', { name: 'Select all resources on this page' }),
    ).toBeNull();
    wrapper.unmount();
  });

  it('renders descending sort order on the initial column', async () => {
    const wrapper = renderTable({
      columns: [{ label: 'col1', sortKey: 'col1sortkey' }, { label: 'col2' }],
      initialSortOrder: 'desc',
    });
    await flushPromisesInAct();
    expect(screen.getByRole('columnheader', { name: 'col1' })).toHaveAttribute(
      'aria-sort',
      'descending',
    );
    expect(screen.getByRole('columnheader', { name: 'col2' })).not.toHaveAttribute('aria-sort');
    wrapper.unmount();
  });

  it('forwards supplied column metadata to the explicit renderer', async () => {
    const suppliedColumns = [
      { flex: 3, label: 'col1' },
      { flex: 1, label: 'col2' },
    ];
    const adapter = vi.fn(baseProps.renderTable);
    const wrapper = renderTable({ columns: suppliedColumns, renderTable: adapter });
    await flushPromisesInAct();
    expect(adapter.mock.lastCall?.[0].columns).toBe(suppliedColumns);
    expect(screen.getByRole('columnheader', { name: 'col1' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'col2' })).toBeInTheDocument();
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

  it('restores next-page navigation when a direct refresh recovers a page token', async () => {
    let nextToken = '';
    const reload = vi.fn(async () => nextToken);
    const wrapper = renderTable({ rows, columns, reload });
    await waitFor(() => expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled());
    nextToken = 'after-retry';
    await act(async () => {
      await wrapper.instance().reload();
    });
    expect(screen.getByRole('button', { name: 'Next page' })).toBeEnabled();
    nextToken = '';
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    await waitFor(() =>
      expect(reload).toHaveBeenLastCalledWith(
        expect.objectContaining({ pageToken: 'after-retry' }),
      ),
    );
  });

  it('refreshes paging tokens without losing the current page or previous-page token', async () => {
    let nextToken = '';
    const reload = vi.fn(async (request: Parameters<CustomTableProps['reload']>[0]) =>
      request.pageToken ? nextToken : 'page-one',
    );
    const wrapper = renderTable({ rows, columns, reload });
    await waitFor(() => expect(wrapper.state('tokenList')).toEqual(['', 'page-one']));
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    await waitFor(() => expect(wrapper.state('currentPage')).toBe(1));
    expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled();
    nextToken = 'new-page-two';
    await act(async () => {
      await wrapper.instance().reload();
    });
    expect(wrapper.state('currentPage')).toBe(1);
    expect(wrapper.state('tokenList')).toEqual(['', 'page-one', 'new-page-two']);
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    await waitFor(() =>
      expect(reload).toHaveBeenLastCalledWith(
        expect.objectContaining({ pageToken: 'new-page-two' }),
      ),
    );
  });

  it('does not let an older direct refresh overwrite a newer page token', async () => {
    const reload = vi.fn<CustomTableProps['reload']>(async () => '');
    const wrapper = renderTable({ rows, columns, reload });
    await waitFor(() => expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled());
    let finishOlder!: (token: string) => void;
    const olderResult = new Promise<string>((resolve) => {
      finishOlder = resolve;
    });
    reload.mockImplementationOnce(() => olderResult).mockResolvedValueOnce('latest-token');
    let olderRefresh!: Promise<string>;
    act(() => {
      olderRefresh = wrapper.instance().reload();
    });
    await act(async () => {
      await wrapper.instance().reload();
    });
    await act(async () => {
      finishOlder('stale-token');
      await olderRefresh;
    });
    expect(wrapper.state('tokenList')).toEqual(['', 'latest-token']);
    expect(screen.getByRole('button', { name: 'Next page' })).toBeEnabled();
  });

  it('does not let an older page request overwrite a newer filter reset', async () => {
    const reload = vi.fn<CustomTableProps['reload']>(async () => 'page-one');
    const wrapper = renderTable({ rows, columns, reload });
    await waitFor(() => expect(wrapper.state('tokenList')).toEqual(['', 'page-one']));
    let finishPage!: (token: string) => void;
    reload.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishPage = resolve;
        }),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    reload.mockResolvedValueOnce('filtered-next');
    await act(async () => {
      await wrapper.instance()._requestFilter('new');
    });
    await act(async () => {
      finishPage('stale-next');
    });
    expect(wrapper.state('currentPage')).toBe(0);
    expect(wrapper.state('tokenList')).toEqual(['', 'filtered-next']);
  });

  it('does not let an older filter reset overwrite a newer direct refresh', async () => {
    const reload = vi.fn<CustomTableProps['reload']>(async () => 'page-one');
    const wrapper = renderTable({ rows, columns, reload });
    await waitFor(() => expect(wrapper.state('tokenList')).toEqual(['', 'page-one']));
    let finishFilter!: (token: string) => void;
    reload.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishFilter = resolve;
        }),
    );
    let filtering!: Promise<void>;
    act(() => {
      filtering = wrapper.instance()._requestFilter('old');
    });
    reload.mockResolvedValueOnce('latest-next');
    await act(async () => {
      await wrapper.instance().reload();
    });
    await act(async () => {
      finishFilter('stale-next');
      await filtering;
    });
    expect(wrapper.state('currentPage')).toBe(0);
    expect(wrapper.state('tokenList')).toEqual(['', 'latest-next']);
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

  it('offers sorting only for columns with a sort key', async () => {
    renderTable({
      columns: [{ label: 'sortable', sortKey: 'sortableKey' }, { label: 'unsortable' }],
      rows,
    });
    await flushPromisesInAct();
    expect(
      within(screen.getByRole('columnheader', { name: 'sortable' })).getByRole('button'),
    ).toBeInTheDocument();
    expect(
      within(screen.getByRole('columnheader', { name: 'unsortable' })).queryByRole('button'),
    ).toBeNull();
  });

  it('renders some rows', async () => {
    const wrapper = renderTable({ rows, columns });
    await flushPromisesInAct();
    expect(screen.getAllByTestId('table-row')).toHaveLength(2);
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
    const headerCheckbox = getHeaderCheckbox();
    fireEvent.click(headerCheckbox);
  });

  it('selects all items when head checkbox is clicked', async () => {
    const spy = vi.fn();
    const wrapper = renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    const headerCheckbox = getHeaderCheckbox();
    fireEvent.click(headerCheckbox);
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
    wrapper.unmount();
  });

  it('unselects all items when head checkbox is clicked and all items are selected', async () => {
    const spy = vi.fn();
    const wrapper = renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    const headerCheckbox = getHeaderCheckbox();
    fireEvent.click(headerCheckbox);
    expect(spy).toHaveBeenLastCalledWith(['row1', 'row2']);
    wrapper.rerender({
      ...baseProps,
      rows,
      columns,
      updateSelection: spy,
      selectedIds: ['row1', 'row2'],
    });
    const updatedHeaderCheckbox = getHeaderCheckbox();
    fireEvent.click(updatedHeaderCheckbox);
    expect(spy).toHaveBeenLastCalledWith([]);
    wrapper.unmount();
  });

  it('selects all items if one item was checked then the head checkbox is clicked', async () => {
    const spy = vi.fn();
    const wrapper = renderTable({ rows, columns, updateSelection: spy });
    await flushPromisesInAct();
    fireEvent.click(screen.getAllByTestId('table-row')[0]);
    const headerCheckbox = getHeaderCheckbox();
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
    const prevBtn = screen.getByRole('button', { name: 'Previous page' });
    const nextBtn = screen.getByRole('button', { name: 'Next page' });
    expect(prevBtn).toBeDisabled();
    expect(nextBtn).toBeDisabled();
    wrapper.unmount();
  });

  it('enables next page button if next page token is given', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows, columns, reload: spy });
    await flushPromisesInAct();
    const prevBtn = screen.getByRole('button', { name: 'Previous page' });
    const nextBtn = screen.getByRole('button', { name: 'Next page' });
    expect(wrapper.state()).toHaveProperty('maxPageIndex', Number.MAX_SAFE_INTEGER);
    expect(prevBtn).toBeDisabled();
    expect(nextBtn).not.toBeDisabled();
    wrapper.unmount();
  });

  it('calls reload with next page token when next page button is clicked', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows, columns, reload: spy });
    await flushPromisesInAct();
    const nextBtn = screen.getByRole('button', { name: 'Next page' });
    fireEvent.click(nextBtn);
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
    const nextBtn = screen.getByRole('button', { name: 'Next page' });
    fireEvent.click(nextBtn);
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
    const prevBtn = screen.getByRole('button', { name: 'Previous page' });
    expect(prevBtn).not.toBeDisabled();
    wrapper.unmount();
  });

  it('renders new rows after clicking previous page, and enables next page button', async () => {
    const reloadResult = Promise.resolve('some token');
    const spy = vi.fn(() => reloadResult);
    const wrapper = renderTable({ rows: [], columns, reload: spy });
    await flushPromisesInAct();
    const prevBtn = screen.getByRole('button', { name: 'Previous page' });
    const nextBtn = screen.getByRole('button', { name: 'Next page' });
    fireEvent.click(nextBtn);
    await flushPromisesInAct();
    fireEvent.click(prevBtn);
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
    const prevBtnAfterUpdate = screen.getByRole('button', { name: 'Previous page' });
    expect(prevBtnAfterUpdate).toBeDisabled();
    await flushPromisesInAct();
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
    expect(screen.getByRole('button', { name: 'Expand resource row1' })).toBeInTheDocument();
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
    expect(
      screen.queryByRole('checkbox', { name: 'Select all resources on this page' }),
    ).toBeNull();
    wrapper.unmount();
  });

  it('forwards expanded row state to the renderer', async () => {
    const row = { ...rows[0], expandState: ExpandState.EXPANDED };
    const wrapper = renderTable({ rows: [row], columns, getExpandComponent: () => null });
    await flushPromisesInAct();
    expect(screen.getByRole('button', { name: 'Collapse resource row1' })).toHaveAttribute(
      'aria-expanded',
      'true',
    );
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
    expect(screen.getByText('Hello World')).toBeInTheDocument();
    wrapper.unmount();
  });

  it('calls prop to toggle expansion', async () => {
    const row = { ...rows[0], expandState: ExpandState.EXPANDED };
    const toggleSpy = vi.fn();
    renderTable({
      rows: [row, { ...row, id: 'row2' }, { ...row, id: 'row3' }],
      columns,
      getExpandComponent: () => <span>Hello World</span>,
      toggleExpansion: toggleSpy,
    });
    await flushPromisesInAct();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse resource row2' }));
    expect(toggleSpy).toHaveBeenCalledWith(1);
  });

  it('renders a table with sorting disabled', async () => {
    const wrapper = renderTable({
      rows,
      columns: columns.map((column) => ({ ...column, sortKey: column.label })),
      disableSorting: true,
    });
    await flushPromisesInAct();
    expect(
      within(screen.getByRole('columnheader', { name: 'col1' })).queryByRole('button'),
    ).toBeNull();
    expect(
      within(screen.getByRole('columnheader', { name: 'col2' })).queryByRole('button'),
    ).toBeNull();
    expect(screen.getAllByTestId('table-row')).toHaveLength(2);
    wrapper.unmount();
  });

  it('updates the filter string in state when the filter box input changes', async () => {
    const wrapper = renderTable({ rows, columns });
    await flushPromisesInAct();
    fireEvent.change(screen.getByRole('searchbox', { name: 'Filter' }), {
      target: { value: 'test filter' },
    });
    expect(wrapper.state('filterString')).toEqual('test filter');
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

it('reloads changed predicates without remounting or losing the current filter and sort', async () => {
  const reload = vi.fn().mockResolvedValue('next-token');
  const props = {
    ...baseProps,
    reload,
    initialFilterString: 'saved',
    initialSortColumn: 'name',
    initialSortOrder: 'asc' as const,
  };
  const wrapper = renderTable(props);
  await flushPromisesInAct();
  const instance = wrapper.instance();
  reload.mockClear();
  const predicate = {
    key: 'type',
    operation: V2beta1PredicateOperation.IN,
    int_values: { values: [6] },
  };
  wrapper.rerender({ ...props, filterPredicates: [predicate] });
  await flushPromisesInAct();
  expect(wrapper.instance()).toBe(instance);
  expect(reload).toHaveBeenCalledTimes(1);
  expect(reload.mock.lastCall?.[0]).toMatchObject({
    pageToken: '',
    sortBy: 'name',
    orderAscending: true,
  });
  expect(JSON.parse(decodeURIComponent(reload.mock.lastCall?.[0].filter))).toEqual({
    predicates: [
      { key: 'name', operation: V2beta1PredicateOperation.IS_SUBSTRING, string_value: 'saved' },
      predicate,
    ],
  });
  wrapper.rerender({ ...props, filterPredicates: [{ ...predicate }] });
  await flushPromisesInAct();
  expect(reload).toHaveBeenCalledTimes(1);
  expect(wrapper.state('filterString')).toBe('saved');
});
