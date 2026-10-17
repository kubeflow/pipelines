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

import { useState } from 'react';
import { LayoutGrid, List } from 'lucide-react';
import { LocalStorage } from '../../lib/LocalStorage';
import type { CustomTableRenderModel, Row } from '../CustomTable';
import { Button } from '../ui/button';
import { ResourceTable } from '../tables/ResourceTable';
import { PipelineCards, PipelineControls } from './PipelineCards';

function pipelineName(row: Row): string {
  const value = row.otherFields[0] as { display_name?: string; name?: string };
  return value.display_name || value.name || row.id;
}

// CustomTable remains the owner of filtering, sorting, paging, expansion, and selection.
// Changing the presentation never remounts that controller or reloads its data.
export function PipelineCollection({ table }: { table: CustomTableRenderModel }) {
  const [view, setView] = useState(LocalStorage.getPipelineView);
  function changeView(next: 'table' | 'cards') {
    setView(next);
    LocalStorage.savePipelineView(next);
  }
  return (
    <div className='kfp-pipeline-collection'>
      <PipelineControls table={table} showSort={view === 'cards'}>
        <div className='kfp-pipeline-view-toggle' role='group' aria-label='Pipeline view'>
          <Button
            variant={view === 'table' ? 'secondary' : 'ghost'}
            size='sm'
            aria-pressed={view === 'table'}
            onClick={() => changeView('table')}
          >
            <List aria-hidden='true' /> Table
          </Button>
          <Button
            variant={view === 'cards' ? 'secondary' : 'ghost'}
            size='sm'
            aria-pressed={view === 'cards'}
            onClick={() => changeView('cards')}
          >
            <LayoutGrid aria-hidden='true' /> Cards
          </Button>
        </div>
      </PipelineControls>
      {view === 'cards' ? (
        <PipelineCards table={table} hideControls />
      ) : (
        <ResourceTable
          table={{ ...table, noFilterBox: true }}
          label='Pipelines'
          singular='pipeline'
          plural='pipelines'
          getRowLabel={pipelineName}
        />
      )}
    </div>
  );
}
