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

import type { ReactNode } from 'react';
import type { CustomTableRenderModel } from '../CustomTable';
import { ResourceTable } from './ResourceTable';

export function RunsTable({
  table,
  headerActions,
  activeFilters,
  onOpenRun,
  reservedRowCount,
}: {
  table: CustomTableRenderModel;
  headerActions?: Record<string, ReactNode>;
  activeFilters?: ReactNode;
  reservedRowCount?: number;
  onOpenRun: (id: string) => void;
}) {
  return (
    <ResourceTable
      table={table}
      headerActions={headerActions}
      activeFilters={activeFilters}
      onOpenRow={onOpenRun}
      label='Runs'
      singular='run'
      plural='runs'
      filterLabel='Search runs'
      minWidth={
        280 +
        table.columns.reduce(
          (total, column) => total + (typeof column.width === 'number' ? column.width : 0),
          0,
        )
      }
      className='kfp-runs-data-table'
      minScrollHeight={
        reservedRowCount === undefined ? undefined : 40 + Math.max(0, reservedRowCount) * 48
      }
    />
  );
}
