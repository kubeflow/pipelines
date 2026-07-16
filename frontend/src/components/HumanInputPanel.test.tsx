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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import { testBestPractices } from 'src/TestUtils';
import { HumanInputPanel } from 'src/components/HumanInputPanel';

testBestPractices();

describe('HumanInputPanel', () => {
  it('submits the declared default value', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        text: async () =>
          JSON.stringify({
            suspended: true,
            parameters: { decision: { default: 'NO', enum: ['YES', 'NO'] } },
          }),
      })
      .mockResolvedValueOnce({ ok: true, text: async () => '{}' })
      .mockResolvedValue({ ok: true, text: async () => JSON.stringify({ suspended: false }) });
    vi.stubGlobal('fetch', fetchMock);

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <HumanInputPanel runId='run-id' nodeDisplayName='approval' />
      </QueryClientProvider>,
    );

    await screen.findByRole('alert');
    fireEvent.click(screen.getByRole('button', { name: 'Submit and resume the pipeline' }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        'apis/v2beta1/runs/run-id/nodes/approval:setIntermediateInputs',
        expect.objectContaining({ body: JSON.stringify({ parameters: { decision: 'NO' } }) }),
      );
    });
  });
});
