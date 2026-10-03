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

import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import type { CustomTableRenderModel } from '../CustomTable';
import { RunStatus } from './RunStatus';
import { RunsTable } from './RunsTable';
import { ThemeProvider } from './ThemeProvider';
import type { V2beta1RuntimeState } from 'src/apisv2beta1/run';

const fixtures: { id: string; name: string; state: V2beta1RuntimeState; duration: string }[] = [
  { id: 'run-7f3a2c', name: 'churn-model nightly', state: 'RUNNING', duration: '00:03:42' },
  { id: 'run-9b1e07', name: 'feature-backfill', state: 'SUCCEEDED', duration: '00:12:08' },
  { id: 'run-c41a88', name: 'ranker eval · candidate 21', state: 'FAILED', duration: '00:02:56' },
  { id: 'run-0be3b5', name: 'embeddings refresh · weekly', state: 'PENDING', duration: '—' },
];

function RunsPreview({ empty = false, loading = false }: { empty?: boolean; loading?: boolean }) {
  const [filter, setFilter] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [opened, setOpened] = useState('');
  const visible = empty
    ? []
    : fixtures.filter((run) => run.name.toLowerCase().includes(filter.toLowerCase()));
  const table: CustomTableRenderModel = {
    columns: [
      {
        label: 'Run',
        customRenderer: ({ id, value }) => (
          <div className='kfp-runs-name'>
            <a
              href={`#${id}`}
              onClick={(event) => {
                event.preventDefault();
                setOpened(id);
              }}
            >
              {value}
            </a>
            <span className='kfp-runs-id'>{id}</span>
          </div>
        ),
      },
      { label: 'Status', customRenderer: ({ value }) => <RunStatus state={value} /> },
      { label: 'Pipeline version' },
      { label: 'Experiment' },
      { label: 'Duration' },
      { label: 'Started' },
    ],
    rows: visible.map((run) => ({
      id: run.id,
      otherFields: [run.name, run.state, 'training · v14', 'Churn', run.duration, '2 hours ago'],
    })),
    selectedIds,
    filter,
    filterLabel: 'Filter runs by name',
    onFilterChange: setFilter,
    sortBy: '',
    sortOrder: 'desc',
    onSort: () => {},
    onSelect: (id) =>
      setSelectedIds((ids) =>
        ids.includes(id) ? ids.filter((selected) => selected !== id) : [...ids, id],
      ),
    onSelectAll: (checked) => setSelectedIds(checked ? visible.map((run) => run.id) : []),
    isBusy: loading,
    emptyMessage: 'No available runs found.',
    pageSize: 10,
    onPageSizeChange: () => {},
    canPrevious: false,
    canNext: false,
    onPrevious: () => {},
    onNext: () => {},
    disablePaging: true,
  };
  return (
    <div style={{ padding: 32 }}>
      <h1 style={{ marginBottom: 8, fontSize: 22, fontWeight: 600 }}>Runs</h1>
      <p style={{ color: 'var(--muted-foreground)' }}>Illustrative records for component review.</p>
      <RunsTable table={table} onOpenRun={setOpened} />
      <p role='status' style={{ marginTop: 16 }}>
        {opened ? `Selected run for inspection: ${opened}` : ''}
      </p>
    </div>
  );
}
const meta = {
  title: 'Modernization/Runs table',
  component: RunsPreview,
  parameters: { layout: 'fullscreen' },
  decorators: [
    (Story, context) => (
      <ThemeProvider
        defaultTheme={context.parameters.theme === 'dark' ? 'dark' : 'light'}
        storageKey={`kfp.storybook.runs.${context.id}`}
      >
        <Story />
      </ThemeProvider>
    ),
  ],
} satisfies Meta<typeof RunsPreview>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Light: Story = {};
export const Dark: Story = { parameters: { theme: 'dark' } };
export const Empty: Story = { args: { empty: true } };
export const Loading: Story = { args: { empty: true, loading: true } };
