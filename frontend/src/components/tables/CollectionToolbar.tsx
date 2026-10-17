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
import type { ReactNode } from 'react';
import { Search } from 'lucide-react';
import type { CustomTableRenderModel } from '../CustomTable';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { DensityMenu } from './DensityMenu';
import './RunsTable.css';

/** One control strip for list filters and resource-specific view controls. */
export function CollectionToolbar({
  table,
  filterLabel = table.filterLabel,
  children,
}: {
  table: CustomTableRenderModel;
  filterLabel?: string;
  children?: ReactNode;
}) {
  const id = useId();
  return (
    <div className='kfp-collection-toolbar'>
      {!table.noFilterBox && (
        <div className='kfp-collection-search'>
          <label htmlFor={id}>{filterLabel}</label>
          <Search size={16} aria-hidden='true' />
          <Input
            id={id}
            type='search'
            placeholder={filterLabel}
            value={table.filter}
            onChange={(event) => table.onFilterChange(event.target.value)}
          />
        </div>
      )}
      {table.filter && !table.noFilterBox && (
        <Button variant='ghost' size='sm' onClick={() => table.onFilterChange('')}>
          Clear filter
        </Button>
      )}
      {table.filterActions}
      <div className='kfp-collection-view-controls'>
        {children}
        <DensityMenu />
      </div>
    </div>
  );
}
