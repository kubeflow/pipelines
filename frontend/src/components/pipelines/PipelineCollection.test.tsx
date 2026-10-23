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

import { render as rtlRender, screen } from '@testing-library/react';
import type { CustomTableRenderModel } from '../CustomTable';
import { PipelineCollection } from './PipelineCollection';
import { ThemeProvider } from '../shell/ThemeProvider';
import type { ReactElement } from 'react';
import { LocalStorageKey } from '../../lib/LocalStorage';

const render = (ui: ReactElement) => rtlRender(ui, { wrapper: ThemeProvider });

beforeEach(() => {
  vi.stubGlobal('matchMedia', () => ({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});
afterEach(() => vi.unstubAllGlobals());

function table(overrides: Partial<CustomTableRenderModel> = {}): CustomTableRenderModel {
  return {
    columns: [
      {
        label: 'Pipeline name',
        sortKey: 'display_name',
        flex: 1,
        customRenderer: ({ value }) => <span>{value.display_name}</span>,
      },
      { label: 'Description', flex: 1 },
      { label: 'Uploaded on', sortKey: 'created_at', flex: 1 },
    ],
    rows: ['First', 'Second'].map((name, index) => ({
      id: String(index),
      otherFields: [{ display_name: name }, `${name} description`, '9/26/2026'],
    })),
    selectedIds: [],
    filter: '',
    filterLabel: 'Search pipelines',
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
    emptyMessage: 'No pipelines found.',
    ...overrides,
  };
}

it('uses a table regardless of previously saved card preferences', () => {
  localStorage.setItem(LocalStorageKey.pipelineView, 'cards');
  const model = table({ selectedIds: ['0'], filter: 'First', pageSize: 20 });
  render(<PipelineCollection table={model} />);
  expect(screen.getByRole('table', { name: 'Pipelines' })).toBeVisible();
  expect(screen.queryByRole('button', { name: 'Cards', exact: true })).not.toBeInTheDocument();
  expect(screen.getByRole('searchbox')).toHaveValue('First');
  expect(screen.getByRole('checkbox', { name: 'Select pipeline First' })).toBeChecked();
  expect(screen.getByRole('combobox', { name: 'Rows per page' })).toHaveValue('20');
});
