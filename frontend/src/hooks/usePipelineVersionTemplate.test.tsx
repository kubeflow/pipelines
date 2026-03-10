/*
 * Copyright 2025 The Kubeflow Authors
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

import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { Apis } from 'src/lib/Apis';
import { testBestPractices } from 'src/TestUtils';
import { queryKeys } from './queryKeys';
import { usePipelineVersionTemplate } from './usePipelineVersionTemplate';

testBestPractices();

function TestComponent({
  pipelineId,
  pipelineVersionId,
}: {
  pipelineId: string | undefined;
  pipelineVersionId: string | undefined;
}) {
  const { data, isSuccess, isFetching } = usePipelineVersionTemplate(pipelineId, pipelineVersionId);
  return (
    <div>
      <span data-testid='isSuccess'>{String(isSuccess)}</span>
      <span data-testid='isFetching'>{String(isFetching)}</span>
      <span data-testid='data'>{data ?? 'undefined'}</span>
    </div>
  );
}

describe('usePipelineVersionTemplate', () => {
  it('uses pipelineVersionTemplate query key with pipelineId and pipelineVersionId', () => {
    expect(queryKeys.pipelineVersionTemplate('pipeline-1', 'version-1')).toEqual([
      'PipelineVersionTemplate',
      { pipelineId: 'pipeline-1', pipelineVersionId: 'version-1' },
    ]);
  });

  it('fetches pipeline version template when enabled', async () => {
    const mockPipelineVersion = {
      pipeline_spec: { pipelineInfo: { name: 'test' } },
    };
    vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion').mockResolvedValue(
      mockPipelineVersion as never,
    );

    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <TestComponent pipelineId='pipeline-1' pipelineVersionId='version-1' />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('isSuccess').textContent).toBe('true'));
    expect(screen.getByTestId('data').textContent).toContain('pipelineInfo');
    expect(Apis.pipelineServiceApiV2.getPipelineVersion).toHaveBeenCalledWith(
      'pipeline-1',
      'version-1',
    );
  });

  it('does not fetch when pipelineId is undefined (enabled: false)', async () => {
    const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');

    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <TestComponent pipelineId={undefined} pipelineVersionId='version-1' />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('isFetching').textContent).toBe('false'));
    expect(screen.getByTestId('data').textContent).toBe('undefined');
    expect(getPipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('does not fetch when pipelineVersionId is undefined (enabled: false)', async () => {
    const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');

    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <TestComponent pipelineId='pipeline-1' pipelineVersionId={undefined} />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('isFetching').textContent).toBe('false'));
    expect(screen.getByTestId('data').textContent).toBe('undefined');
    expect(getPipelineVersionSpy).not.toHaveBeenCalled();
  });
});
