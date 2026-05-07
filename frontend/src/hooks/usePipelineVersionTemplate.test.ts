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

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { Apis } from 'src/lib/Apis';
import { usePipelineVersionTemplate } from './usePipelineVersionTemplate';

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe('usePipelineVersionTemplate', () => {
  let getPipelineVersionSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns YAML string when pipeline version has a pipeline_spec', async () => {
    getPipelineVersionSpy.mockResolvedValue({
      pipeline_spec: { components: { root: {} } },
    });

    const { result } = renderHook(() => usePipelineVersionTemplate('pipeline-1', 'version-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toContain('components');
    expect(getPipelineVersionSpy).toHaveBeenCalledWith('pipeline-1', 'version-1');
  });

  it('returns empty string when pipeline_spec is undefined', async () => {
    getPipelineVersionSpy.mockResolvedValue({});

    const { result } = renderHook(() => usePipelineVersionTemplate('pipeline-1', 'version-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBe('');
  });

  it('does not fetch when pipelineId is undefined', async () => {
    const { result } = renderHook(() => usePipelineVersionTemplate(undefined, 'version-1'), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(getPipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('does not fetch when pipelineVersionId is undefined', async () => {
    const { result } = renderHook(() => usePipelineVersionTemplate('pipeline-1', undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(getPipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('does not fetch when both IDs are undefined', async () => {
    const { result } = renderHook(() => usePipelineVersionTemplate(undefined, undefined), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(getPipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('reports error when the API call rejects', async () => {
    getPipelineVersionSpy.mockRejectedValue(new Error('Network failure'));

    const { result } = renderHook(() => usePipelineVersionTemplate('pipeline-1', 'version-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error?.message).toBe('Network failure');
  });
});
