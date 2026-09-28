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
import { ArrowDown, ArrowUp, ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';
import { ExpandState } from '../CustomTable';
import type { CustomTableRenderModel, Row } from '../CustomTable';
import { Button } from '../ui/button';
import { Checkbox } from '../ui/checkbox';
import { Input } from '../ui/input';
import './Pipelines.css';

function rowName(row: Row): string {
  const value = row.otherFields[0] as { display_name?: string; name?: string };
  return value.display_name || value.name || row.id;
}

export function PipelineCards({ table }: { table: CustomTableRenderModel }) {
  const id = useId();
  const NameRenderer = table.columns[0].customRenderer;
  const DescriptionRenderer = table.columns[1].customRenderer;
  const visibleSelected = table.rows.filter((row) => table.selectedIds.includes(row.id)).length;
  const allSelected = table.rows.length > 0 && visibleSelected === table.rows.length;
  const SortIcon = table.sortOrder === 'asc' ? ArrowUp : ArrowDown;
  return (
    <div className='kfp-pipeline-cards-view'>
      <div className='kfp-pipeline-controls'>
        <label className='kfp-pipeline-filter' htmlFor={`${id}-filter`}>
          <span>Filter pipelines</span>
          <Input
            id={`${id}-filter`}
            type='search'
            placeholder='Filter pipelines'
            value={table.filter}
            onChange={(event) => table.onFilterChange(event.target.value)}
          />
        </label>
        {table.filter && (
          <Button variant='ghost' onClick={() => table.onFilterChange('')}>
            Clear filter
          </Button>
        )}
        <label className='kfp-pipeline-sort' htmlFor={`${id}-sort`}>
          Sort pipelines
          <select
            id={`${id}-sort`}
            value={table.sortBy}
            onChange={(event) => table.onSort(event.target.value)}
          >
            {table.columns
              .filter((column) => column.sortKey)
              .map((column) => (
                <option key={column.sortKey} value={column.sortKey}>
                  {column.label}
                </option>
              ))}
          </select>
        </label>
        <Button
          variant='secondary'
          size='icon'
          aria-label={`Sort ${table.sortOrder === 'asc' ? 'descending' : 'ascending'}`}
          onClick={() => table.onSort(table.sortBy)}
        >
          <SortIcon aria-hidden='true' />
        </Button>
      </div>
      <div className='kfp-pipeline-selection'>
        <Checkbox
          aria-label='Select all pipelines on this page'
          checked={allSelected}
          indeterminate={visibleSelected > 0 && !allSelected}
          disabled={!table.rows.length || table.isBusy}
          onCheckedChange={table.onSelectAll}
        />
        <span>Select this page</span>
        <span role='status'>
          {table.isBusy && table.rows.length ? 'Refreshing pipelines…' : ''}
        </span>
      </div>
      {!table.rows.length && (
        <div className='kfp-pipeline-empty'>
          {table.isBusy ? (
            <p role='status'>Loading pipelines…</p>
          ) : table.errorMessage ? (
            <p>{table.errorMessage}</p>
          ) : table.filter ? (
            <>
              <strong>No pipelines match</strong>
              <p>Try another name or clear the filter.</p>
            </>
          ) : (
            <p>{table.emptyMessage}</p>
          )}
        </div>
      )}
      <ul role='list' className='kfp-pipeline-grid' aria-label='Pipelines' aria-busy={table.isBusy}>
        {table.rows.map((row, index) => {
          const expanded = row.expandState === ExpandState.EXPANDED;
          const name = rowName(row);
          return (
            <li
              key={row.id}
              className='kfp-pipeline-card'
              data-testid='pipeline-card'
              data-row-id={row.id}
              data-selected={table.selectedIds.includes(row.id)}
              data-expanded={expanded}
            >
              <div className='kfp-pipeline-card-main'>
                <header>
                  <h2>
                    {NameRenderer ? <NameRenderer value={row.otherFields[0]} id={row.id} /> : name}
                  </h2>
                  <Checkbox
                    aria-label={`Select pipeline ${name}`}
                    checked={table.selectedIds.includes(row.id)}
                    disabled={table.isBusy}
                    onCheckedChange={() => table.onSelect(row.id)}
                  />
                </header>
                <div className='kfp-pipeline-description'>
                  {DescriptionRenderer ? (
                    <DescriptionRenderer value={row.otherFields[1]} id={row.id} />
                  ) : (
                    row.otherFields[1] || 'No description provided.'
                  )}
                </div>
                {row.error && (
                  <p role='note' className='kfp-pipeline-error'>
                    {row.error}
                  </p>
                )}
                <footer>
                  <span>Uploaded {row.otherFields[2] || '—'}</span>
                  <Button
                    variant='ghost'
                    size='sm'
                    aria-label={`${expanded ? 'Collapse' : 'Expand'} pipeline ${name}`}
                    aria-expanded={expanded}
                    aria-controls={`${id}-versions-${index}`}
                    disabled={table.isBusy}
                    onClick={() => table.onToggleExpansion?.(index)}
                  >
                    Versions{' '}
                    {expanded ? (
                      <ChevronDown size={14} aria-hidden='true' />
                    ) : (
                      <ChevronRight size={14} aria-hidden='true' />
                    )}
                  </Button>
                </footer>
              </div>
              {expanded && (
                <div className='kfp-pipeline-versions' id={`${id}-versions-${index}`}>
                  {table.getExpandedContent?.(index)}
                </div>
              )}
            </li>
          );
        })}
      </ul>
      <div className='kfp-pipeline-pagination'>
        <label htmlFor={`${id}-page-size`}>Rows per page</label>
        <select
          id={`${id}-page-size`}
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
    </div>
  );
}
