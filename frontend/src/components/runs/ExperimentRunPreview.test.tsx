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

import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { vi } from 'vitest';
import { Apis } from 'src/lib/Apis';
import { V2RuntimeState } from 'src/apisv2/run';
import { ExperimentRunPreview } from './ExperimentRunPreview';

it('shows at most five linked runs without table or selection controls', () => {
  const listRuns = vi.spyOn(Apis.runServiceApiV2, 'listRuns');
  render(
    <MemoryRouter>
      <ExperimentRunPreview
        experimentId='exp/1'
        runs={Array.from({ length: 7 }, (_, i) => ({
          run_id: `run/${i}`,
          display_name: `Run ${i}`,
          state: V2RuntimeState.SUCCEEDED,
        }))}
      />
    </MemoryRouter>,
  );
  expect(screen.getAllByRole('listitem')).toHaveLength(5);
  expect(screen.getByRole('link', { name: 'Run 0' })).toHaveAttribute(
    'href',
    '/runs/details/run%2F0',
  );
  expect(screen.getByRole('link', { name: 'View all runs' })).toHaveAttribute(
    'href',
    '/experiments/details/exp%2F1',
  );
  expect(screen.queryByRole('table')).not.toBeInTheDocument();
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
  expect(listRuns).not.toHaveBeenCalled();
  listRuns.mockRestore();
});

it('keeps the detail link available when a preview request fails', async () => {
  const listRuns = vi
    .spyOn(Apis.runServiceApiV2, 'listRuns')
    .mockRejectedValue(new Error('unavailable'));
  render(
    <MemoryRouter>
      <ExperimentRunPreview experimentId='exp' archived />
    </MemoryRouter>,
  );
  expect(await screen.findByRole('alert')).toHaveTextContent('Could not load recent runs');
  expect(screen.getByRole('link', { name: 'View all runs' })).toHaveAttribute(
    'href',
    '/experiments/details/exp',
  );
  listRuns.mockRestore();
});

it('explains an empty archived preview', () => {
  render(
    <MemoryRouter>
      <ExperimentRunPreview experimentId='exp' archived runs={[]} />
    </MemoryRouter>,
  );
  expect(screen.getByText('No archived runs yet.')).toBeInTheDocument();
});
