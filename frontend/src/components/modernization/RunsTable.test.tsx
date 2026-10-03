/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Link, MemoryRouter } from 'react-router';
import type { CustomTableRenderModel } from '../CustomTable';
import { RunsTable } from './RunsTable';
import { RunStatus } from './RunStatus';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';

function model(overrides: Partial<CustomTableRenderModel> = {}): CustomTableRenderModel {
  return {
    columns: [
      {
        label: 'Run',
        sortKey: 'name',
        customRenderer: ({ id, value }) => <Link to={`/runs/details/${id}`}>{value}</Link>,
      },
      { label: 'Status' },
      { label: 'Started', sortKey: 'created_at' },
    ],
    rows: [
      { id: 'one', otherFields: ['Training', 'Running', 'Today'] },
      { id: 'two', otherFields: ['Evaluation', 'Succeeded', 'Yesterday'] },
    ],
    selectedIds: [],
    filter: '',
    filterLabel: 'Filter runs by name',
    onFilterChange: vi.fn(),
    sortBy: 'created_at',
    sortOrder: 'desc',
    onSort: vi.fn(),
    onSelect: vi.fn(),
    onSelectAll: vi.fn(),
    isBusy: false,
    emptyMessage: 'No available runs found.',
    pageSize: 10,
    onPageSizeChange: vi.fn(),
    canPrevious: false,
    canNext: true,
    onPrevious: vi.fn(),
    onNext: vi.fn(),
    ...overrides,
  };
}
function renderTable(table = model(), onOpenRun = vi.fn()) {
  return render(
    <MemoryRouter>
      <RunsTable table={table} onOpenRun={onOpenRun} />
    </MemoryRouter>,
  );
}

it('keeps row navigation, checkbox selection and resource links separate', async () => {
  const table = model();
  const open = vi.fn();
  renderTable(table, open);
  await userEvent.click(screen.getByRole('checkbox', { name: 'Select run Training' }));
  expect(table.onSelect).toHaveBeenCalledExactlyOnceWith('one');
  expect(open).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole('link', { name: 'Training' }));
  expect(open).not.toHaveBeenCalled();
  fireEvent.click(screen.getAllByTestId('table-row')[0]);
  expect(open).toHaveBeenCalledExactlyOnceWith('one');
});

it('derives header selection from visible IDs when other pages retain selections', () => {
  const table = model({ selectedIds: ['other-page-a', 'other-page-b'] });
  const view = renderTable(table);
  const header = () => screen.getByRole('checkbox', { name: 'Select all runs on this page' });
  expect(header()).not.toBeChecked();
  expect(header()).not.toBePartiallyChecked();
  view.rerender(
    <MemoryRouter>
      <RunsTable table={{ ...table, selectedIds: ['one', 'other-page-a'] }} onOpenRun={vi.fn()} />
    </MemoryRouter>,
  );
  expect(header()).toBePartiallyChecked();
  view.rerender(
    <MemoryRouter>
      <RunsTable
        table={{ ...table, selectedIds: ['one', 'two', 'other-page-a'] }}
        onOpenRun={vi.fn()}
      />
    </MemoryRouter>,
  );
  expect(header()).toBeChecked();
});

it('supports keyboard selection, sorting and named paging controls', async () => {
  const table = model();
  renderTable(table);
  const checkbox = screen.getByRole('checkbox', { name: 'Select run Training' });
  checkbox.focus();
  await userEvent.keyboard(' ');
  expect(table.onSelect).toHaveBeenCalledWith('one');
  expect(screen.getByRole('columnheader', { name: 'Started' })).toHaveAttribute(
    'aria-sort',
    'descending',
  );
  await userEvent.click(screen.getByRole('button', { name: 'Run' }));
  expect(table.onSort).toHaveBeenCalledWith('name');
  expect(
    within(screen.getByRole('columnheader', { name: 'Status' })).queryByRole('button'),
  ).toBeNull();
  expect(screen.getByRole('button', { name: 'Previous page' })).toBeDisabled();
  await userEvent.click(screen.getByRole('button', { name: 'Next page' }));
  expect(table.onNext).toHaveBeenCalledOnce();
  await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Rows per page' }), '50');
  expect(table.onPageSizeChange).toHaveBeenCalledWith(50);
});

it('distinguishes initial loading, first-use empty and filtered empty with clear action', async () => {
  const table = model({ rows: [], isBusy: true });
  const view = renderTable(table);
  expect(screen.getByText('Loading runs…')).toBeInTheDocument();
  expect(screen.queryByText('No available runs found.')).toBeNull();
  view.rerender(
    <MemoryRouter>
      <RunsTable table={{ ...table, isBusy: false }} onOpenRun={vi.fn()} />
    </MemoryRouter>,
  );
  expect(screen.getByText('No available runs found.')).toBeInTheDocument();
  view.rerender(
    <MemoryRouter>
      <RunsTable table={{ ...table, isBusy: false, filter: 'missing' }} onOpenRun={vi.fn()} />
    </MemoryRouter>,
  );
  expect(screen.getByText('No runs match')).toBeInTheDocument();
  await userEvent.click(screen.getAllByRole('button', { name: 'Clear filter' })[0]);
  expect(table.onFilterChange).toHaveBeenCalledWith('');
});

it('retains visible rows and error details during refresh while disabling selection and paging', () => {
  renderTable(
    model({
      isBusy: true,
      rows: [
        {
          id: 'one',
          otherFields: ['Training', 'Running', 'Today'],
          error: 'Access to the pipeline version was denied.',
        },
      ],
    }),
  );
  expect(screen.getByRole('table', { name: 'Runs' })).toHaveAttribute('aria-busy', 'true');
  expect(screen.getByText('Access to the pipeline version was denied.')).toBeVisible();
  expect(screen.getByText('Refreshing runs…')).toBeVisible();
  expect(screen.getByRole('checkbox', { name: 'Select run Training' })).toHaveAttribute(
    'aria-disabled',
    'true',
  );
  expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled();
});

it.each([
  ['SUCCEEDED', 'Succeeded'],
  ['RUNNING', 'Running'],
  ['FAILED', 'Failed'],
  ['PENDING', 'Pending'],
  ['CANCELING', 'Canceling'],
  ['CANCELED', 'Canceled'],
  ['PAUSED', 'Paused'],
  ['SKIPPED', 'Skipped'],
  ['RUNTIME_STATE_UNSPECIFIED', 'Unknown'],
] as const)('names the %s state without relying on color', (state, label) => {
  render(<RunStatus state={state} />);
  expect(screen.getByText(label)).toBeVisible();
});

it('renders missing and unrecognized runtime states as Unknown', () => {
  const view = render(<RunStatus />);
  expect(screen.getByText('Unknown')).toBeVisible();
  view.rerender(<RunStatus state={'FUTURE_STATE' as V2beta1RuntimeState} />);
  expect(screen.getByText('Unknown')).toBeVisible();
});
