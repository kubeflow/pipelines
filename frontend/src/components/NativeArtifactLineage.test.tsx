// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as React from 'react';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import userEvent from '@testing-library/user-event';
import { V2beta1IOType } from 'src/apisv2beta1/artifact';
import { Apis } from 'src/lib/Apis';
import { mockResizeObserver } from 'src/TestUtils';
import NativeArtifactLineage from './NativeArtifactLineage';

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <NativeArtifactLineage artifactId='target' namespace='team' />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const produced = {
  id: 'link',
  artifact_id: 'target',
  task_id: 'producer',
  run_id: 'run',
  type: V2beta1IOType.OUTPUT,
  key: 'model',
};

beforeEach(() => {
  mockResizeObserver();
  vi.spyOn(Apis.artifactServiceApiV2, 'artifact_1').mockImplementation(async (id) => ({
    artifact_id: id,
    name: `Artifact ${id}`,
  }));
  vi.spyOn(Apis.runServiceApiV2, 'task_1').mockImplementation(async (run, id) => ({
    task_id: id,
    display_name: `Task ${id}`,
  }));
});
afterEach(() => vi.restoreAllMocks());

it('keeps the selected artifact keyboard-focusable without recentering', async () => {
  const user = userEvent.setup();
  vi.mocked(Apis.artifactServiceApiV2.artifact_1).mockResolvedValue({
    artifact_id: 'target',
    name: 'Model',
    uri: 's3://models/model',
  });
  vi.spyOn(Apis.artifactServiceApiV2, 'artifactTasks').mockResolvedValue({
    artifact_tasks: [],
  });
  mount();
  const selected = await within(screen.getByRole('region', { name: 'Lineage graph' })).findByRole(
    'button',
    { name: 'Model' },
  );
  for (let steps = 0; steps < 20 && document.activeElement !== selected; steps++) {
    await user.tab();
  }
  expect(selected).toHaveFocus();
  expect(selected).not.toBeDisabled();
  expect(selected).toHaveAttribute('aria-current', 'location');
  expect(selected).toHaveAttribute('aria-disabled', 'true');
  expect(selected).toHaveAccessibleDescription('Model\nArtifact target\ns3://models/model');
  await user.keyboard('{Enter} ');
  await user.click(selected);
  expect(screen.getByText('Neighborhood 1')).toBeInTheDocument();
});

it('loads a bounded neighborhood, toggles inputs, recenters and goes back', async () => {
  const list = vi
    .spyOn(Apis.artifactServiceApiV2, 'artifactTasks')
    .mockImplementation(async (tasks, runs, artifacts) => {
      if (tasks)
        return {
          artifact_tasks: [
            {
              id: 'input',
              task_id: 'producer',
              artifact_id: 'dataset',
              type: V2beta1IOType.COMPONENT_INPUT,
              key: 'data',
            },
            { id: 'other-output', artifact_id: 'not-an-input', type: V2beta1IOType.OUTPUT },
          ],
        };
      return { artifact_tasks: artifacts?.[0] === 'target' ? [produced] : [] };
    });
  mount();
  expect(await screen.findByRole('link', { name: 'Task producer' })).toHaveAttribute(
    'href',
    '/runs/details/run?task=producer',
  );
  await screen.findByRole('button', { name: 'Artifact dataset' });
  expect(list.mock.calls.filter((call) => !call[8]?.signal?.aborted)).toHaveLength(2);
  expect(list.mock.calls[0][5]).toBe(5);
  expect(list.mock.calls[0][8]?.signal).toBeInstanceOf(AbortSignal);
  fireEvent.click(screen.getByRole('button', { name: 'Hide input artifacts' }));
  expect(screen.queryByRole('button', { name: 'Artifact dataset' })).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Show input artifacts' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Artifact dataset' }));
  expect(await screen.findByText('Neighborhood 2')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Refresh lineage' }));
  expect(screen.getByText('Neighborhood 2')).toBeInTheDocument();
  await waitFor(() =>
    expect(list.mock.calls.some((call) => call[2]?.[0] === 'dataset')).toBe(true),
  );
  expect(screen.queryByText('Artifact not-an-input')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Back' }));
  expect(await screen.findByRole('link', { name: 'Task producer' })).toBeInTheDocument();
  expect(screen.getByText('Neighborhood 1')).toBeInTheDocument();
});

it('does not auto-drain pagination and stops repeated cursors visibly', async () => {
  const list = vi
    .spyOn(Apis.artifactServiceApiV2, 'artifactTasks')
    .mockImplementation(async (tasks) =>
      tasks ? { artifact_tasks: [] } : { artifact_tasks: [produced], next_page_token: 'next' },
    );
  mount();
  await screen.findByRole('button', { name: 'Load more relationships' });
  expect(list.mock.calls.every((call) => !call[4])).toBe(true);
  fireEvent.click(await screen.findByRole('button', { name: 'Load more relationships' }));
  expect(await screen.findByText(/service repeated a page token/)).toBeInTheDocument();
  expect(list.mock.calls.filter((call) => !call[0] && !call[8]?.signal?.aborted)).toHaveLength(2);
  expect(screen.queryByRole('button', { name: 'Load more relationships' })).not.toBeInTheDocument();
  expect(screen.getAllByRole('link', { name: 'Task producer' })).toHaveLength(1);
});

it('keeps loaded relationships on a failed next page and can recover', async () => {
  let failed = false;
  vi.spyOn(Apis.artifactServiceApiV2, 'artifactTasks').mockImplementation(
    async (tasks, _runs, _artifacts, _type, token) => {
      if (tasks) return { artifact_tasks: [] };
      if (token && !failed) {
        failed = true;
        throw new Error('temporary');
      }
      return { artifact_tasks: [produced], next_page_token: failed ? undefined : 'next' };
    },
  );
  mount();
  fireEvent.click(await screen.findByRole('button', { name: 'Load more relationships' }));
  expect(await screen.findByText(/Some relationships could not/)).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Task producer' })).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Retry relationships' }));
  await waitFor(() =>
    expect(screen.queryByText(/Some relationships could not/)).not.toBeInTheDocument(),
  );
  expect(failed).toBe(true);
});

it('loads consumer outputs, keeps failed artifact identities usable, and resets', async () => {
  vi.mocked(Apis.artifactServiceApiV2.artifact_1).mockImplementation(async (id) => {
    if (id === 'missing') throw new Error('forbidden');
    return { artifact_id: id, name: `Artifact ${id}` };
  });
  vi.spyOn(Apis.artifactServiceApiV2, 'artifactTasks').mockImplementation(
    async (tasks, runs, artifacts) => {
      if (tasks)
        return {
          artifact_tasks: [{ id: 'out', artifact_id: 'missing', type: V2beta1IOType.OUTPUT }],
        };
      return {
        artifact_tasks:
          artifacts?.[0] === 'target' ? [{ ...produced, type: V2beta1IOType.COMPONENT_INPUT }] : [],
      };
    },
  );
  mount();
  expect(await screen.findByText(/Details unavailable/)).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'missing' }));
  expect(await screen.findByText('Neighborhood 2')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Reset' }));
  expect(
    await within(screen.getByRole('region', { name: 'Lineage graph' })).findByRole('button', {
      name: 'Artifact target',
    }),
  ).toHaveAttribute('aria-disabled', 'true');
});

it('draws separate directed branches and maps adjacent artifacts to their own tasks', async () => {
  const roots = ['p1', 'p2', 'c1', 'c2'].map((task, index) => ({
    id: task,
    artifact_id: 'target',
    task_id: task,
    run_id: 'run',
    type: index < 2 ? V2beta1IOType.OUTPUT : V2beta1IOType.COMPONENT_INPUT,
  }));
  const list = vi
    .spyOn(Apis.artifactServiceApiV2, 'artifactTasks')
    .mockImplementation(async (tasks) => {
      if (!tasks) return { artifact_tasks: roots };
      const task = tasks[0];
      const producer = task.startsWith('p');
      return {
        artifact_tasks: [
          {
            id: `leaf-${task}`,
            artifact_id: `artifact-${task}`,
            task_id: task,
            type: producer ? V2beta1IOType.COMPONENT_INPUT : V2beta1IOType.OUTPUT,
          },
          {
            id: `wrong-${task}`,
            artifact_id: `wrong-${task}`,
            task_id: task,
            type: producer ? V2beta1IOType.OUTPUT : V2beta1IOType.COMPONENT_INPUT,
          },
        ],
        next_page_token: `more-${task}`,
      };
    });
  mount();
  const graph = screen.getByRole('region', { name: 'Lineage graph' });
  for (const task of ['p1', 'p2', 'c1', 'c2']) {
    const branch = await within(graph).findByRole('region', {
      name: `${task.startsWith('p') ? 'Producer' : 'Consumer'} Task ${task}`,
    });
    expect(
      await within(branch).findByRole('button', { name: `Artifact artifact-${task}` }),
    ).toBeInTheDocument();
    expect(within(branch).getByRole('link', { name: `Task ${task}` })).toHaveAttribute(
      'href',
      `/runs/details/run?task=${task}`,
    );
  }
  const edges = Array.from(graph.querySelectorAll('span[data-lineage-edge]')).map((edge) => [
    edge.getAttribute('data-from'),
    edge.getAttribute('data-to'),
  ]);
  expect(edges).toHaveLength(8);
  expect(edges).toEqual(
    expect.arrayContaining([
      ['task:p1', 'target:target'],
      ['task:p2', 'target:target'],
      ['target:target', 'task:c1'],
      ['target:target', 'task:c2'],
      ['task:p1:artifact:leaf-p1', 'task:p1'],
      ['task:p2:artifact:leaf-p2', 'task:p2'],
      ['task:c1', 'task:c1:artifact:leaf-c1'],
      ['task:c2', 'task:c2:artifact:leaf-c2'],
    ]),
  );
  expect(within(graph).queryByRole('button', { name: /Artifact wrong-/ })).not.toBeInTheDocument();
  expect(list.mock.calls.filter((call) => !call[8]?.signal?.aborted)).toHaveLength(5);
  expect(list.mock.calls.every((call) => !call[4] && call[5] === 5)).toBe(true);
});

it('uses named breadcrumbs to jump back and discard forward history', async () => {
  const user = userEvent.setup();
  vi.spyOn(Apis.artifactServiceApiV2, 'artifactTasks').mockImplementation(
    async (tasks, _runs, artifacts) => {
      if (tasks) {
        const next = tasks[0] === 'producer-target' ? 'dataset' : 'source';
        return {
          artifact_tasks: [{ id: next, artifact_id: next, type: V2beta1IOType.COMPONENT_INPUT }],
        };
      }
      const target = artifacts?.[0];
      return {
        artifact_tasks:
          target === 'source' ? [] : [{ ...produced, id: target, task_id: `producer-${target}` }],
      };
    },
  );
  mount();
  const graph = within(screen.getByRole('region', { name: 'Lineage graph' }));
  const history = within(screen.getByRole('navigation', { name: 'Lineage history' }));
  fireEvent.click(await graph.findByRole('button', { name: 'Artifact dataset' }));
  fireEvent.click(await graph.findByRole('button', { name: 'Artifact source' }));
  expect(await history.findByRole('button', { name: 'Artifact source' })).toHaveAttribute(
    'aria-current',
    'location',
  );
  expect(history.getAllByRole('button').map((button) => button.textContent)).toEqual([
    '',
    'Artifact target',
    'Artifact dataset',
    'Artifact source',
  ]);
  history.getByRole('button', { name: 'Artifact dataset' }).focus();
  await user.tab();
  const current = history.getByRole('button', { name: 'Artifact source' });
  expect(current).toHaveFocus();
  expect(current).toHaveAttribute('aria-current', 'location');
  expect(current).toHaveAttribute('aria-disabled', 'true');
  expect(current).not.toBeDisabled();
  await user.keyboard('{Enter}');
  expect(screen.getByText('Neighborhood 3')).toBeInTheDocument();
  fireEvent.click(history.getByRole('button', { name: 'Artifact dataset' }));
  expect(history.queryByRole('button', { name: 'Artifact source' })).not.toBeInTheDocument();
  expect(graph.getByRole('button', { name: 'Artifact dataset' })).toHaveAttribute(
    'aria-disabled',
    'true',
  );
  fireEvent.click(screen.getByRole('button', { name: 'Refresh lineage' }));
  expect(history.getByRole('button', { name: 'Artifact dataset' })).toHaveAttribute(
    'aria-current',
    'location',
  );
  fireEvent.click(history.getByRole('button', { name: 'Artifact target' }));
  expect(history.queryByRole('button', { name: 'Artifact dataset' })).not.toBeInTheDocument();
  expect(history.getByRole('button', { name: 'Back' })).toBeDisabled();
});
