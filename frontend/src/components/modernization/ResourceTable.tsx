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

import { Fragment, useId } from 'react';
import type { CSSProperties } from 'react';
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  ChevronLeft,
  ChevronRight,
  TriangleAlert,
} from 'lucide-react';
import { ExpandState } from '../CustomTable';
import type { CustomTableRenderModel, Row } from '../CustomTable';
import { Button } from '../ui/button';
import { Checkbox } from '../ui/checkbox';
import { Input } from '../ui/input';
import './RunsTable.css';

interface ResourceTableProps {
  table: CustomTableRenderModel;
  onOpenRow?: (id: string) => void;
  label?: string;
  singular?: string;
  plural?: string;
  filterLabel?: string;
  minWidth?: CSSProperties['minWidth'];
  columnWidths?: CSSProperties['width'][];
  minScrollHeight?: CSSProperties['minHeight'];
  className?: string;
  getRowLabel?: (row: Row) => string;
}

function defaultRowLabel(row: Row): string {
  const value = row.otherFields[0];
  return typeof value === 'string' || typeof value === 'number' ? String(value) : row.id;
}

export function ResourceTable({
  table,
  onOpenRow,
  label = 'Resources',
  singular = 'resource',
  plural = 'resources',
  filterLabel = table.filterLabel,
  minWidth = 640,
  columnWidths,
  minScrollHeight,
  className = '',
  getRowLabel = defaultRowLabel,
}: ResourceTableProps) {
  const filterId = useId();
  const pageSizeId = useId();
  const visibleSelected = table.rows.filter((row) => table.selectedIds.includes(row.id)).length;
  const allSelected = table.rows.length > 0 && visibleSelected === table.rows.length;
  const filteredEmpty = !table.rows.length && !!table.filter;
  const cellCount =
    table.columns.length + (table.disableSelection ? 0 : 1) + (table.getExpandedContent ? 1 : 0);

  return (
    <div className={className ? `kfp-runs-table-view ${className}` : 'kfp-runs-table-view'}>
      {!table.noFilterBox && (
        <div className='kfp-runs-filter-row'>
          <label htmlFor={filterId}>{filterLabel}</label>
          <Input
            id={filterId}
            type='search'
            placeholder={filterLabel}
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
        <div
          className='kfp-runs-table-scroll'
          style={{ minHeight: minScrollHeight }}
          role='region'
          aria-label={`${label} table`}
          tabIndex={0}
        >
          <table
            className='kfp-runs-table'
            aria-label={label}
            aria-busy={table.isBusy}
            style={{ minWidth, tableLayout: columnWidths ? 'fixed' : undefined }}
          >
            {columnWidths && (
              <colgroup>
                {!table.disableSelection && <col style={{ width: 46 }} />}
                {table.getExpandedContent && <col style={{ width: 36 }} />}
                {table.columns.map((column, index) => (
                  <col key={column.label} style={{ width: columnWidths[index] }} />
                ))}
              </colgroup>
            )}
            <thead>
              <tr>
                {!table.disableSelection && (
                  <th scope='col' className='kfp-runs-selection'>
                    {!table.useRadioButtons && (
                      <Checkbox
                        aria-label={`Select all ${plural} on this page`}
                        checked={allSelected}
                        indeterminate={visibleSelected > 0 && !allSelected}
                        disabled={
                          !table.rows.length ||
                          table.isBusy ||
                          (table.disableAdditionalSelection && visibleSelected > 0 && !allSelected)
                        }
                        onCheckedChange={table.onSelectAll}
                      />
                    )}
                  </th>
                )}
                {table.getExpandedContent && (
                  <th scope='col' aria-label='Row details' className='kfp-resource-expansion' />
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
                      <span role='status'>Loading {plural}…</span>
                    ) : table.errorMessage ? (
                      <span>{table.errorMessage}</span>
                    ) : filteredEmpty ? (
                      <>
                        <strong>No {plural} match</strong>
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
              {table.rows.map((row, rowIndex) => (
                <Fragment key={row.id}>
                  <tr
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
                      if (onOpenRow) onOpenRow(row.id);
                      else if (
                        !table.isBusy &&
                        !table.disableSelection &&
                        (!table.disableAdditionalSelection || table.selectedIds.includes(row.id))
                      )
                        table.onSelect(row.id);
                    }}
                  >
                    {!table.disableSelection && (
                      <td className='kfp-runs-selection'>
                        {table.useRadioButtons ? (
                          <input
                            type='radio'
                            name={`${filterId}-selection`}
                            aria-label={`Select ${singular} ${getRowLabel(row)}`}
                            checked={table.selectedIds.includes(row.id)}
                            disabled={table.isBusy}
                            onChange={() => table.onSelect(row.id)}
                          />
                        ) : (
                          <Checkbox
                            aria-label={`Select ${singular} ${getRowLabel(row)}`}
                            checked={table.selectedIds.includes(row.id)}
                            disabled={
                              table.isBusy ||
                              (!table.selectedIds.includes(row.id) &&
                                table.disableAdditionalSelection)
                            }
                            onCheckedChange={() => table.onSelect(row.id)}
                          />
                        )}
                      </td>
                    )}
                    {table.getExpandedContent && (
                      <td className='kfp-resource-expansion'>
                        {row.expandState !== ExpandState.NONE && (
                          <Button
                            variant='ghost'
                            size='icon'
                            aria-label={`${row.expandState === ExpandState.EXPANDED ? 'Collapse' : 'Expand'} ${singular} ${getRowLabel(row)}`}
                            aria-expanded={row.expandState === ExpandState.EXPANDED}
                            aria-controls={`${filterId}-details-${rowIndex}`}
                            disabled={table.isBusy}
                            onClick={() => table.onToggleExpansion?.(rowIndex)}
                          >
                            <ChevronRight
                              size={16}
                              aria-hidden='true'
                              className={
                                row.expandState === ExpandState.EXPANDED
                                  ? 'kfp-resource-expanded'
                                  : undefined
                              }
                            />
                          </Button>
                        )}
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
                  {table.getExpandedContent && row.expandState === ExpandState.EXPANDED && (
                    <tr className='kfp-resource-details'>
                      <td colSpan={cellCount} id={`${filterId}-details-${rowIndex}`}>
                        {table.getExpandedContent(rowIndex)}
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
        <div className='kfp-runs-table-footer'>
          <span role='status' aria-live='polite'>
            {table.isBusy && table.rows.length ? `Refreshing ${plural}…` : ''}
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
