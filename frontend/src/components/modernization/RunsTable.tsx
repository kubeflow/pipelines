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

import { useId } from 'react';
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  ChevronLeft,
  ChevronRight,
  TriangleAlert,
} from 'lucide-react';
import type { CustomTableRenderModel } from '../CustomTable';
import { Button } from '../ui/button';
import { Checkbox } from '../ui/checkbox';
import { Input } from '../ui/input';
import './RunsTable.css';

interface RunsTableProps {
  table: CustomTableRenderModel;
  onOpenRun: (id: string) => void;
}

export function RunsTable({ table, onOpenRun }: RunsTableProps) {
  const filterId = useId();
  const pageSizeId = useId();
  const visibleSelected = table.rows.filter((row) => table.selectedIds.includes(row.id)).length;
  const allSelected = table.rows.length > 0 && visibleSelected === table.rows.length;
  const filteredEmpty = !table.rows.length && !!table.filter;
  const cellCount = table.columns.length + (table.disableSelection ? 0 : 1);

  return (
    <div className='kfp-runs-table-view'>
      {!table.noFilterBox && (
        <div className='kfp-runs-filter-row'>
          <label htmlFor={filterId}>Filter runs by name</label>
          <Input
            id={filterId}
            type='search'
            placeholder='Filter runs by name'
            value={table.filter}
            onChange={(event) => table.onFilterChange(event.target.value)}
          />
          {table.filter && (
            <Button variant='ghost' onClick={() => table.onFilterChange('')}>
              Clear filter
            </Button>
          )}
        </div>
      )}
      <div className='kfp-runs-table-card'>
        <div className='kfp-runs-table-scroll' role='region' aria-label='Runs table' tabIndex={0}>
          <table className='kfp-runs-table' aria-label='Runs' aria-busy={table.isBusy}>
            <thead>
              <tr>
                {!table.disableSelection && (
                  <th scope='col' className='kfp-runs-selection'>
                    <Checkbox
                      aria-label='Select all runs on this page'
                      checked={allSelected}
                      indeterminate={visibleSelected > 0 && !allSelected}
                      disabled={!table.rows.length || table.isBusy}
                      onCheckedChange={table.onSelectAll}
                    />
                  </th>
                )}
                {table.columns.map((column) => {
                  const sortable = !table.disableSorting && !!column.sortKey;
                  const sorted = sortable && table.sortBy === column.sortKey;
                  const SortIcon = sorted
                    ? table.sortOrder === 'asc'
                      ? ArrowUp
                      : ArrowDown
                    : ArrowUpDown;
                  return (
                    <th
                      key={column.label}
                      scope='col'
                      aria-sort={
                        sorted
                          ? table.sortOrder === 'asc'
                            ? 'ascending'
                            : 'descending'
                          : undefined
                      }
                    >
                      {sortable ? (
                        <button type='button' onClick={() => table.onSort(column.sortKey!)}>
                          {column.label}
                          <SortIcon size={13} aria-hidden='true' />
                        </button>
                      ) : (
                        column.label
                      )}
                    </th>
                  );
                })}
              </tr>
            </thead>
            <tbody>
              {!table.rows.length && (
                <tr>
                  <td colSpan={cellCount} className='kfp-runs-empty'>
                    {table.isBusy ? (
                      <span role='status'>Loading runs…</span>
                    ) : table.errorMessage ? (
                      <span>{table.errorMessage}</span>
                    ) : filteredEmpty ? (
                      <>
                        <strong>No runs match</strong>
                        <p>Try another name or clear the filter.</p>
                        <Button variant='secondary' onClick={() => table.onFilterChange('')}>
                          Clear filter
                        </Button>
                      </>
                    ) : (
                      table.emptyMessage
                    )}
                  </td>
                </tr>
              )}
              {table.rows.map((row) => (
                <tr
                  key={row.id}
                  data-testid='table-row'
                  data-row-id={row.id}
                  data-selected={table.selectedIds.includes(row.id)}
                  onClick={(event) => {
                    if (
                      event.button !== 0 ||
                      event.metaKey ||
                      event.ctrlKey ||
                      event.shiftKey ||
                      event.altKey
                    )
                      return;
                    if (
                      event.target instanceof Element &&
                      event.target.closest('a, button, input, select, [role="checkbox"]')
                    )
                      return;
                    onOpenRun(row.id);
                  }}
                >
                  {!table.disableSelection && (
                    <td className='kfp-runs-selection'>
                      <Checkbox
                        aria-label={`Select run ${row.otherFields[0] || row.id}`}
                        checked={table.selectedIds.includes(row.id)}
                        disabled={table.isBusy}
                        onCheckedChange={() => table.onSelect(row.id)}
                      />
                    </td>
                  )}
                  {table.columns.map((column, index) => (
                    <td key={column.label}>
                      {column.customRenderer
                        ? column.customRenderer({ value: row.otherFields[index], id: row.id })
                        : row.otherFields[index]}
                      {index === 0 && row.error && (
                        <span className='kfp-runs-warning' role='note' aria-label={row.error}>
                          <TriangleAlert size={14} aria-hidden='true' />
                          {row.error}
                        </span>
                      )}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className='kfp-runs-table-footer'>
          <span role='status' aria-live='polite'>
            {table.isBusy && table.rows.length ? 'Refreshing runs…' : ''}
          </span>
          {!table.disablePaging && (
            <div className='kfp-runs-pagination'>
              <label htmlFor={pageSizeId}>Rows per page</label>
              <select
                id={pageSizeId}
                value={table.pageSize}
                disabled={table.isBusy}
                onChange={(event) => table.onPageSizeChange(Number(event.target.value))}
              >
                {[10, 20, 50, 100].map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
              <Button
                variant='ghost'
                size='icon'
                aria-label='Previous page'
                data-testid='prev-page-btn'
                disabled={table.isBusy || !table.canPrevious}
                onClick={table.onPrevious}
              >
                <ChevronLeft aria-hidden='true' />
              </Button>
              <Button
                variant='ghost'
                size='icon'
                aria-label='Next page'
                data-testid='next-page-btn'
                disabled={table.isBusy || !table.canNext}
                onClick={table.onNext}
              >
                <ChevronRight aria-hidden='true' />
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
