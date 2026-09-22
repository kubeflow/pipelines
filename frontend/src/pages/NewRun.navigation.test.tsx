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

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { HashRouter } from 'react-router';
import { NavigationErrorBoundary } from 'src/atoms/NavigationErrorBoundary';
import Router, { RoutePage } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { flushPromisesInAct } from 'src/TestUtils';
import { NewRun } from './NewRun';

it.each(['pipeline', 'pipeline version'])(
  'stays on the cancel destination when the pending %s request fails',
  async (resource) => {
    const originalUrl = window.location.href;
    window.history.replaceState(
      null,
      '',
      '/#/runs/new?pipelineId=pipeline-1&pipelineVersionId=version-1',
    );
    let rejectRequest!: (reason: Error) => void;
    const pending = new Promise<never>((_, reject) => {
      rejectRequest = reject;
    });
    const getPipeline = vi.spyOn(Apis.pipelineServiceApi, 'getPipeline').mockResolvedValue({
      id: 'pipeline-1',
      name: 'Pipeline',
      parameters: [],
    });
    const getVersion = vi.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
    const request = resource === 'pipeline' ? getPipeline : getVersion;
    request.mockReturnValue(pending);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = render(
      <QueryClientProvider client={queryClient}>
        <HashRouter>
          <NavigationErrorBoundary>
            <Router
              configs={[
                { path: RoutePage.NEW_RUN, Component: NewRun },
                { path: RoutePage.RUNS, Component: () => <div>Cancel destination</div> },
              ]}
            />
          </NavigationErrorBoundary>
        </HashRouter>
      </QueryClientProvider>,
    );
    try {
      await waitFor(() => expect(request).toHaveBeenCalled());
      fireEvent.click(screen.getByRole('button', { name: 'Cancel', exact: true }));
      await screen.findByText('Cancel destination');
      expect(window.location.hash).toBe('#/runs');

      await act(async () => rejectRequest(new Error('Late request failure')));
      await flushPromisesInAct();

      expect(window.location.hash).toBe('#/runs');
      expect(screen.getByText('Cancel destination')).toBeVisible();
      expect(screen.queryByRole('button', { name: 'Cancel', exact: true })).not.toBeInTheDocument();
    } finally {
      view.unmount();
      queryClient.clear();
      getPipeline.mockRestore();
      getVersion.mockRestore();
      window.history.replaceState(null, '', originalUrl);
    }
  },
);
