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
import type { CustomTableRenderModel } from '../CustomTable';
import { TablePagination } from './TablePagination';

export function TableFooter({ table, plural }: { table: CustomTableRenderModel; plural: string }) {
  const id = useId();
  return (
    <div className='kfp-runs-table-footer'>
      <span role='status' aria-live='polite'>
        {table.isBusy
          ? table.rows.length
            ? `Refreshing ${plural}…`
            : ''
          : table.selectedIds.length
            ? `${table.selectedIds.length} selected`
            : `${table.rows.length} on this page`}
      </span>
      <TablePagination table={table} id={id} className='kfp-runs-pagination' />
    </div>
  );
}
