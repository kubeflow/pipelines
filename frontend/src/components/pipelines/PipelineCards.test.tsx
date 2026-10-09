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
import { ExpandState } from '../CustomTable';
import type { CustomTableRenderModel } from '../CustomTable';
import { PipelineCards } from './PipelineCards';

function table(overrides: Partial<CustomTableRenderModel> = {}): CustomTableRenderModel {
  return {
    columns: [
      { label: 'Pipeline name', sortKey: 'display_name', flex: 1 },
      { label: 'Description', flex: 1 },
      { label: 'Uploaded on', sortKey: 'created_at', flex: 1 },
    ],
    rows: ['First', 'Second'].map((name, index) => ({
      id: String(index),
      expandState: ExpandState.COLLAPSED,
      otherFields: [{ display_name: name }, `${name} description`, '9/26/2026'],
    })),
    selectedIds: [],
    filter: '',
    filterLabel: 'Filter pipelines',
    sortBy: 'created_at',
    sortOrder: 'desc',
    pageSize: 10,
    isBusy: false,
    canPrevious: false,
    canNext: true,
    onFilterChange: vi.fn(),
    onSort: vi.fn(),
    onSelect: vi.fn(),
    onSelectAll: vi.fn(),
    onPageSizeChange: vi.fn(),
    onPrevious: vi.fn(),
    onNext: vi.fn(),
    onToggleExpansion: vi.fn(),
    getExpandedContent: vi.fn(() => <div>Versions loaded on expansion</div>),
    emptyMessage: 'No pipelines found.',
    ...overrides,
  };
}

it('keeps versions lazy and expansion separate from selection', async () => {
  const model = table();
  const view = render(<PipelineCards table={model} />);
  expect(model.getExpandedContent).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole('button', { name: 'Expand pipeline First' }));
  expect(model.onToggleExpansion).toHaveBeenCalledWith(0);
  expect(model.onSelect).not.toHaveBeenCalled();
  view.rerender(
    <PipelineCards
      table={{
        ...model,
        rows: [{ ...model.rows[0], expandState: ExpandState.EXPANDED }, model.rows[1]],
      }}
    />,
  );
  expect(screen.getByText('Versions loaded on expansion')).toBeVisible();
  expect(model.getExpandedContent).toHaveBeenCalledWith(0);
  expect(screen.getByRole('button', { name: 'Collapse pipeline First' })).toHaveAttribute(
    'aria-expanded',
    'true',
  );
});

it('supports keyboard selection and communicates partial page selection', async () => {
  const model = table({ selectedIds: ['0'] });
  render(<PipelineCards table={model} />);
  expect(
    screen.getByRole('checkbox', { name: 'Select all pipelines on this page' }),
  ).toHaveAttribute('aria-checked', 'mixed');
  const second = screen.getByRole('checkbox', { name: 'Select pipeline Second' });
  second.focus();
  await userEvent.keyboard(' ');
  expect(model.onSelect).toHaveBeenCalledWith('1');
  await userEvent.click(
    screen.getByRole('checkbox', { name: 'Select all pipelines on this page' }),
  );
  expect(vi.mocked(model.onSelectAll).mock.calls[0][0]).toBe(true);
});

it('connects filtering, sorting and paging to the existing controller', async () => {
  const model = table();
  render(<PipelineCards table={model} />);
  fireEvent.change(screen.getByRole('searchbox', { name: 'Filter pipelines' }), {
    target: { value: 'First' },
  });
  expect(model.onFilterChange).toHaveBeenCalledWith('First');
  await userEvent.selectOptions(
    screen.getByRole('combobox', { name: 'Sort pipelines' }),
    'display_name',
  );
  expect(model.onSort).toHaveBeenLastCalledWith('display_name');
  await userEvent.click(screen.getByRole('button', { name: 'Sort ascending' }));
  expect(model.onSort).toHaveBeenLastCalledWith('created_at');
  await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Rows per page' }), '20');
  expect(model.onPageSizeChange).toHaveBeenCalledWith(20);
  await userEvent.click(screen.getByRole('button', { name: 'Next page' }));
  expect(model.onNext).toHaveBeenCalledOnce();
  expect(screen.getByRole('button', { name: 'Previous page' })).toBeDisabled();
});

it('distinguishes initial loading, failure, and an empty filtered result', () => {
  const model = table({ rows: [], isBusy: true });
  const view = render(<PipelineCards table={model} />);
  expect(screen.getByText('Loading pipelines…')).toHaveAttribute('role', 'status');
  view.rerender(
    <PipelineCards
      table={{ ...model, isBusy: false, errorMessage: 'Unable to load pipelines.' }}
    />,
  );
  expect(screen.getByText('Unable to load pipelines.')).toBeVisible();
  expect(screen.queryByText('No pipelines found.')).toBeNull();
  view.rerender(<PipelineCards table={{ ...model, isBusy: false, filter: 'unmatched' }} />);
  expect(screen.getByText('No pipelines match')).toBeVisible();
  expect(
    within(screen.getByRole('list', { name: 'Pipelines' })).queryAllByRole('listitem'),
  ).toHaveLength(0);
});
