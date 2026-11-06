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

import { Fragment, useId, useMemo, useRef, useState } from 'react';
import type { CSSProperties, ReactNode } from 'react';
import { ArrowDown, ArrowUp, ArrowUpDown, ChevronRight, TriangleAlert } from 'lucide-react';
import { LocalStorage } from '../../lib/LocalStorage';
import { ExpandState } from '../CustomTable';
import type { CustomTableRenderModel, Row } from '../CustomTable';
import { Button } from '../ui/button';
import { Checkbox } from '../ui/checkbox';
import { CollectionToolbar } from './CollectionToolbar';
import { TableFooter } from './TableFooter';
import './RunsTable.css';

interface ResourceTableProps {
  table: CustomTableRenderModel;
  headerActions?: Record<string, ReactNode>;
  activeFilters?: ReactNode;
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
  headerActions,
  activeFilters,
  onOpenRow,
  label = 'Resources',
  singular = 'resource',
  plural = 'resources',
  filterLabel = table.filterLabel,
  minWidth = 640,
  columnWidths: explicitColumnWidths,
  minScrollHeight,
  className = '',
  getRowLabel = defaultRowLabel,
}: ResourceTableProps) {
  const columnWidths =
    explicitColumnWidths ??
    (table.columns.some((column) => column.width !== undefined)
      ? table.columns.map((column) => column.width)
      : undefined);
  const tableId = JSON.stringify([
    label,
    table.columns.map((column) => column.sortKey || column.label),
  ]);
  const storedWidths = useMemo(() => LocalStorage.getTableColumnWidths(tableId), [tableId]);
  const [savedWidths, setSavedWidths] = useState<Record<string, number[] | undefined>>({});
  const candidateWidths = Object.prototype.hasOwnProperty.call(savedWidths, tableId)
    ? savedWidths[tableId]
    : storedWidths;
  const resizedWidths =
    candidateWidths?.length === table.columns.length ? candidateWidths : undefined;
  const storedFlexibleColumn = useMemo(
    () => LocalStorage.getTableFlexibleColumn(tableId),
    [tableId],
  );
  const [flexibleColumns, setFlexibleColumns] = useState<Record<string, number | undefined>>({});
  const preferredFlexibleColumn = flexibleColumns[tableId] ?? storedFlexibleColumn;
  const flexibleColumn =
    preferredFlexibleColumn !== undefined && preferredFlexibleColumn < table.columns.length
      ? preferredFlexibleColumn
      : table.columns.length - 1;
  const tableElement = useRef<HTMLTableElement>(null);
  const drag = useRef<{ start: number; widths: number[]; index: number } | null>(null);
  const readWidths = () =>
    Array.from(
      tableElement.current?.querySelectorAll<HTMLTableCellElement>('th[data-resizable]') || [],
      (cell) => Math.max(80, Math.min(2000, Math.round(cell.getBoundingClientRect().width))),
    );
  const saveWidths = (widths?: number[], flexIndex?: number) => {
    setSavedWidths((previous) => ({ ...previous, [tableId]: widths }));
    setFlexibleColumns((previous) => ({ ...previous, [tableId]: flexIndex }));
    LocalStorage.saveTableColumnWidths(tableId, widths, flexIndex);
  };
  const resizeColumn = (widths: number[], index: number, width: number) => {
    // Keep the user's target exact; another column absorbs unused container space.
    const autoIndex = table.columns.findIndex(
      (_, position) => position !== index && columnWidths?.[position] === undefined,
    );
    const flexIndex =
      autoIndex >= 0
        ? autoIndex
        : index === table.columns.length - 1
          ? 0
          : table.columns.length - 1;
    saveWidths(
      widths.map((value, position) =>
        position === index ? Math.max(80, Math.min(2000, Math.round(width))) : value,
      ),
      flexIndex,
    );
  };
  const controlWidth = (table.disableSelection ? 0 : 40) + (table.getExpandedContent ? 36 : 0);
  const filterId = useId();
  const visibleSelected = table.rows.filter((row) => table.selectedIds.includes(row.id)).length;
  const allSelected = table.rows.length > 0 && visibleSelected === table.rows.length;
  const filteredEmpty = !table.rows.length && !!table.filter;
  const cellCount =
    table.columns.length + (table.disableSelection ? 0 : 1) + (table.getExpandedContent ? 1 : 0);

  return (
    <div className={className ? `kfp-runs-table-view ${className}` : 'kfp-runs-table-view'}>
      {(!table.noFilterBox || table.filterActions) && (
        <CollectionToolbar table={table} filterLabel={filterLabel} />
      )}
      {activeFilters && <div className='kfp-active-filters'>{activeFilters}</div>}
      <div className='kfp-runs-table-card'>
        <div
          className='kfp-runs-table-scroll'
          style={{ minHeight: minScrollHeight }}
          role='region'
          aria-label={`${label} table`}
          tabIndex={0}
        >
          <table
            ref={tableElement}
            className='kfp-runs-table'
            aria-label={label}
            aria-busy={table.isBusy}
            style={{
              minWidth: resizedWidths
                ? resizedWidths.reduce(
                    (total, width, index) =>
                      total + (index === flexibleColumn && table.columns.length > 1 ? 80 : width),
                    controlWidth,
                  )
                : minWidth,
              tableLayout: 'fixed',
              width: resizedWidths ? '100%' : undefined,
            }}
          >
            <colgroup>
              {!table.disableSelection && <col style={{ width: 40 }} />}
              {table.getExpandedContent && <col style={{ width: 36 }} />}
              {table.columns.map((column, index) => (
                <col
                  key={column.label}
                  style={{
                    width: resizedWidths
                      ? index === flexibleColumn && table.columns.length > 1
                        ? undefined
                        : resizedWidths[index]
                      : columnWidths?.[index],
                  }}
                />
              ))}
            </colgroup>
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
                {table.columns.map((column, index) => {
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
                      data-resizable
                      aria-label={column.label}
                      aria-sort={
                        sorted
                          ? table.sortOrder === 'asc'
                            ? 'ascending'
                            : 'descending'
                          : undefined
                      }
                    >
                      <span className='kfp-column-heading'>
                        {sortable ? (
                          <button type='button' onClick={() => table.onSort(column.sortKey!)}>
                            {column.label}
                            <SortIcon size={13} aria-hidden='true' />
                          </button>
                        ) : (
                          column.label
                        )}
                        {headerActions?.[column.sortKey || column.label]}
                      </span>
                      <span
                        className='kfp-column-resize'
                        role='separator'
                        aria-orientation='vertical'
                        aria-label={`Resize ${column.label} column`}
                        aria-valuemin={80}
                        aria-valuemax={2000}
                        aria-valuenow={
                          resizedWidths && (index !== flexibleColumn || table.columns.length === 1)
                            ? resizedWidths[index]
                            : 80
                        }
                        aria-valuetext={
                          resizedWidths && (index !== flexibleColumn || table.columns.length === 1)
                            ? `${resizedWidths[index]} pixels`
                            : 'Automatic width'
                        }
                        tabIndex={0}
                        title='Drag to resize. Use left/right arrows to adjust; Home resets all widths.'
                        onKeyDown={(event) => {
                          if (event.key === 'Home') {
                            event.preventDefault();
                            saveWidths();
                          } else if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
                            event.preventDefault();
                            const widths = readWidths();
                            resizeColumn(
                              widths,
                              index,
                              widths[index] + (event.key === 'ArrowRight' ? 16 : -16),
                            );
                          }
                        }}
                        onDoubleClick={() => saveWidths()}
                        onPointerDown={(event) => {
                          if (event.button !== 0) return;
                          event.preventDefault();
                          event.currentTarget.focus();
                          event.currentTarget.setPointerCapture(event.pointerId);
                          drag.current = { start: event.clientX, widths: readWidths(), index };
                        }}
                        onPointerMove={(event) => {
                          if (!drag.current) return;
                          const { start, widths, index: draggedIndex } = drag.current;
                          resizeColumn(
                            widths,
                            draggedIndex,
                            widths[draggedIndex] + event.clientX - start,
                          );
                        }}
                        onPointerUp={() => {
                          drag.current = null;
                        }}
                        onPointerCancel={() => {
                          drag.current = null;
                        }}
                        onLostPointerCapture={() => {
                          drag.current = null;
                        }}
                      />
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
                    data-selected={!table.disableSelection && table.selectedIds.includes(row.id)}
                    data-interactive={!table.disableSelection || !!onOpenRow}
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
                      if (table.isBusy) return;
                      if (table.disableSelection) onOpenRow?.(row.id);
                      else if (
                        !table.disableAdditionalSelection ||
                        table.selectedIds.includes(row.id)
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
                      <td
                        key={column.label}
                        data-column={index === 0 ? 'primary' : undefined}
                        title={
                          typeof row.otherFields[index] === 'string'
                            ? row.otherFields[index]
                            : undefined
                        }
                      >
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
        {resizedWidths && (
          <div className='kfp-column-reset'>
            <Button variant='ghost' size='sm' onClick={() => saveWidths()}>
              Reset column widths
            </Button>
          </div>
        )}
        <TableFooter table={table} plural={plural} />
      </div>
    </div>
  );
}
