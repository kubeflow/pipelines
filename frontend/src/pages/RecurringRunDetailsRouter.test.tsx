/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { act, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { RouteParams } from 'src/components/Router';
import * as features from 'src/features';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { PageProps } from './Page';
import RecurringRunDetailsRouter from './RecurringRunDetailsRouter';

vi.mock('src/pages/RecurringRunDetailsV2', () => ({
  default: () => <div data-testid='recurring-run-details-v2' />,
}));

vi.mock('src/pages/functional_components/RecurringRunDetailsV2FC', () => ({
  RecurringRunDetailsV2FC: () => <div data-testid='recurring-run-details-v2-fc' />,
}));

const recurringRunId = 'test-recurring-run-id';
let client: QueryClient;
let props: PageProps;

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  props = {
    navigate: vi.fn(),
    location: { pathname: `/recurringrun/details/${recurringRunId}` } as any,
    params: { [RouteParams.recurringRunId]: recurringRunId },
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  };
});

afterEach(() => {
  client.clear();
  vi.restoreAllMocks();
});

function page() {
  return (
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <RecurringRunDetailsRouter {...props} />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

it.each([
  { pipeline_version_reference: { pipeline_id: 'pipeline' } },
  { pipeline_version_reference: { pipeline_id: 'pipeline', pipeline_version_id: 'version' } },
  { pipeline_spec: { pipelineInfo: { name: 'inline-pipeline' } } },
  {},
])('renders native metadata without resolving the pipeline source: %j', async (source) => {
  vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun').mockResolvedValue({
    recurring_run_id: recurringRunId,
    ...source,
  });
  const getPipelineVersion = vi
    .spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion')
    .mockRejectedValue(new Error('Version not found'));

  render(page());

  expect(await screen.findByTestId('recurring-run-details-v2')).toBeInTheDocument();
  expect(getPipelineVersion).not.toHaveBeenCalled();
  expect(props.updateBanner).not.toHaveBeenCalled();
});

it('renders the functional native page when enabled', async () => {
  vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
    (key) => key === features.FeatureKey.FUNCTIONAL_COMPONENT,
  );
  vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun').mockResolvedValue({
    recurring_run_id: recurringRunId,
    pipeline_version_reference: { pipeline_id: 'pipeline' },
  });

  render(page());

  expect(await screen.findByTestId('recurring-run-details-v2-fc')).toBeInTheDocument();
});

it('shows loading while the recurring-run request is pending', () => {
  vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun').mockReturnValue(new Promise(() => {}));

  render(page());

  expect(screen.getByRole('progressbar')).toBeInTheDocument();
});

it('keeps cached details visible during refetch and after a failed refresh', async () => {
  const run: V2beta1RecurringRun = {
    recurring_run_id: recurringRunId,
    pipeline_version_reference: { pipeline_id: 'pipeline' },
  };
  const getRecurringRun = vi
    .spyOn(Apis.recurringRunServiceApi, 'getRecurringRun')
    .mockResolvedValue(run);
  render(page());
  await screen.findByTestId('recurring-run-details-v2');

  let rejectRefresh!: (error: Error) => void;
  getRecurringRun.mockReturnValueOnce(
    new Promise((_resolve, reject) => {
      rejectRefresh = reject;
    }),
  );
  act(() => {
    void client.invalidateQueries({ queryKey: queryKeys.v2RecurringRunDetail(recurringRunId) });
  });
  expect(screen.getByTestId('recurring-run-details-v2')).toBeInTheDocument();
  expect(screen.queryByRole('progressbar')).toBeNull();

  await act(async () => rejectRefresh(new Error('Refresh unavailable')));
  await waitFor(() =>
    expect(props.updateBanner).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'error', additionalInfo: 'Refresh unavailable' }),
    ),
  );
  expect(screen.getByTestId('recurring-run-details-v2')).toBeInTheDocument();
  expect(screen.queryByRole('alert')).toBeNull();
});

it('shows an initial request failure and recovers on retry', async () => {
  const getRecurringRun = vi
    .spyOn(Apis.recurringRunServiceApi, 'getRecurringRun')
    .mockRejectedValue(new Error('Run unavailable'));
  render(page());
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Unable to load recurring run details',
  );
  await waitFor(() =>
    expect(props.updateBanner).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'error', additionalInfo: 'Run unavailable' }),
    ),
  );

  getRecurringRun.mockResolvedValue({ recurring_run_id: recurringRunId });
  await act(async () => {
    await client.invalidateQueries({ queryKey: queryKeys.v2RecurringRunDetail(recurringRunId) });
  });
  expect(await screen.findByTestId('recurring-run-details-v2')).toBeInTheDocument();
  expect(screen.queryByRole('alert')).toBeNull();
});
