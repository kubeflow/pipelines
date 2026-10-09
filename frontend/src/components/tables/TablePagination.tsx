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

import { ChevronLeft, ChevronRight } from 'lucide-react';
import type { CustomTableRenderModel } from '../CustomTable';
import { Button } from '../ui/button';

type PaginationModel = Pick<
  CustomTableRenderModel,
  | 'pageSize'
  | 'isBusy'
  | 'canPrevious'
  | 'canNext'
  | 'disablePaging'
  | 'onPageSizeChange'
  | 'onPrevious'
  | 'onNext'
>;

export function TablePagination({
  table,
  id,
  className,
}: {
  table: PaginationModel;
  id: string;
  className: string;
}) {
  if (table.disablePaging) return null;
  return (
    <div className={className}>
      <label htmlFor={id}>Rows per page</label>
      <select
        id={id}
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
  );
}
