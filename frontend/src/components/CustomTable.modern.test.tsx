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

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import CustomTable from './CustomTable';
import { RunsTable } from './modernization/RunsTable';
import type { ListRequest } from 'src/lib/Apis';

function Harness({ reload }: { reload: (request: ListRequest) => Promise<string> }) {
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  return (
    <CustomTable
      columns={[{ label: 'Run', sortKey: 'name' }]}
      rows={[
        { id: 'one', otherFields: ['Training'] },
        { id: 'two', otherFields: ['Evaluation'] },
      ]}
      reload={reload}
      initialSortColumn='name'
      selectedIds={selectedIds}
      updateSelection={setSelectedIds}
      renderTable={(table) => <RunsTable table={table} onOpenRun={vi.fn()} />}
    />
  );
}
afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
  localStorage.clear();
});

it('resets the request page token when filtering from a later page', async () => {
  const reload = vi.fn(async () => 'next-page');
  render(<Harness reload={reload} />);
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
  await waitFor(() =>
    expect(reload).toHaveBeenLastCalledWith(expect.objectContaining({ pageToken: 'next-page' })),
  );
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'Training' } });
  await waitFor(() =>
    expect(reload).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: '', filter: expect.stringContaining('Training') }),
    ),
  );
  expect(screen.getByRole('button', { name: 'Previous page' })).toBeDisabled();
});

it('keeps filter, selection and page size through an explicit controller reload', async () => {
  const reload = vi.fn(async () => '');
  render(<Harness reload={reload} />);
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  fireEvent.click(screen.getByRole('checkbox', { name: 'Select run Training' }));
  expect(
    screen.getByRole('checkbox', { name: 'Select all runs on this page' }),
  ).toBePartiallyChecked();
  fireEvent.change(screen.getByRole('combobox'), { target: { value: '20' } });
  await waitFor(() =>
    expect(reload).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageSize: 20, pageToken: '' }),
    ),
  );
  expect(screen.getByRole('checkbox', { name: 'Select run Training' })).toBeChecked();
});

it('keeps paging usable when preference storage cannot be read or written', async () => {
  vi.spyOn(localStorage, 'getItem').mockImplementation(() => {
    throw new Error('denied');
  });
  vi.spyOn(localStorage, 'setItem').mockImplementation(() => {
    throw new Error('denied');
  });
  const reload = vi.fn(async () => '');
  render(<Harness reload={reload} />);
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  expect(screen.getByRole('combobox')).toHaveValue('10');
  fireEvent.change(screen.getByRole('combobox'), { target: { value: '50' } });
  await waitFor(() =>
    expect(reload).toHaveBeenLastCalledWith(expect.objectContaining({ pageSize: 50 })),
  );
  expect(screen.getByRole('combobox')).toHaveValue('50');
});

it('does not let an older sort response reset newer pagination', async () => {
  let finishOld: (value: string) => void = () => {};
  const reload = vi.fn(async () => '');
  render(<Harness reload={reload} />);
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  reload.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finishOld = resolve;
      }),
  );
  fireEvent.click(screen.getByRole('button', { name: 'Run' }));
  fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'Training' } });
  await waitFor(() =>
    expect(reload).toHaveBeenLastCalledWith(
      expect.objectContaining({ filter: expect.stringContaining('Training') }),
    ),
  );
  await act(async () => {
    finishOld('stale-next-page');
  });
  expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled();
});
