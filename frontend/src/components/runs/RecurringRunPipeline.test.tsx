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

import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { RecurringRunPipeline } from './RecurringRunPipeline';
import type { V2RecurringRun } from 'src/apisv2/recurringrun';
import { RoutePageFactory } from '../Router';
afterEach(cleanup);
const show = (run: V2RecurringRun) =>
  render(
    <MemoryRouter>
      <RecurringRunPipeline run={run} />
    </MemoryRouter>,
  );
it('links the registered pipeline and pinned version with encoded IDs', () => {
  show({ pipeline_version_reference: { pipeline_id: 'pipe/a', pipeline_version_id: 'v%1' } });
  expect(screen.getByRole('link', { name: 'pipe/a' })).toHaveAttribute(
    'href',
    RoutePageFactory.pipelineDetails('pipe/a'),
  );
  expect(screen.getByRole('link', { name: 'v%1' })).toHaveAttribute(
    'href',
    RoutePageFactory.pipelineVersionDetails('pipe/a', 'v%1'),
  );
});
it('explains a latest-version schedule without inventing a pinned version', () => {
  show({ pipeline_version_reference: { pipeline_id: 'pipe' } });
  expect(screen.getByText('Latest version at each scheduled run')).toBeInTheDocument();
  expect(screen.getAllByRole('link')).toHaveLength(1);
});
it('opens an inline definition using the existing recurring-run origin route', () => {
  show({ recurring_run_id: 'schedule/a', pipeline_spec: {} });
  expect(screen.getByRole('link')).toHaveAttribute(
    'href',
    `${RoutePageFactory.pipelineDetails('')}?fromRecurringRun=schedule%2Fa`,
  );
});
it('does not offer a broken link when a definition is unavailable', () => {
  show({ recurring_run_id: 'missing' });
  expect(screen.getByText('No pipeline definition available')).toBeInTheDocument();
  expect(screen.queryByRole('link')).not.toBeInTheDocument();
});
