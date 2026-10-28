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

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TablePagination } from './TablePagination';

function model() {
  return {
    pageSize: 10,
    isBusy: false,
    canPrevious: true,
    canNext: true,
    onPageSizeChange: vi.fn(),
    onPrevious: vi.fn(),
    onNext: vi.fn(),
  };
}

it('forwards page controls without owning pagination state', async () => {
  const table = model();
  render(<TablePagination table={table} id='size' className='pagination' />);
  await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Rows per page' }), '100');
  expect(table.onPageSizeChange).toHaveBeenCalledExactlyOnceWith(100);
  await userEvent.click(screen.getByRole('button', { name: 'Previous page' }));
  await userEvent.click(screen.getByRole('button', { name: 'Next page' }));
  expect(table.onPrevious).toHaveBeenCalledOnce();
  expect(table.onNext).toHaveBeenCalledOnce();
  expect(screen.getByRole('combobox')).toHaveValue('10');
});

it('blocks paging while busy and respects the available directions after refresh', async () => {
  const table = model();
  const view = render(
    <TablePagination table={{ ...table, isBusy: true }} id='size' className='pagination' />,
  );
  expect(screen.getByRole('combobox')).toBeDisabled();
  for (const button of screen.getAllByRole('button')) {
    expect(button).toBeDisabled();
    await userEvent.click(button);
  }
  expect(table.onPrevious).not.toHaveBeenCalled();
  expect(table.onNext).not.toHaveBeenCalled();
  view.rerender(
    <TablePagination table={{ ...table, canNext: false }} id='size' className='pagination' />,
  );
  expect(screen.getByRole('combobox')).toBeEnabled();
  expect(screen.getByRole('button', { name: 'Previous page' })).toBeEnabled();
  expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled();
});

it('omits pagination when the controller disables it', () => {
  render(
    <TablePagination
      table={{ ...model(), disablePaging: true }}
      id='size'
      className='pagination'
    />,
  );
  expect(screen.queryByRole('combobox')).toBeNull();
  expect(screen.queryByRole('button')).toBeNull();
});
