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
import * as MlmdUtils from 'src/mlmd/MlmdUtils';
import { ArtifactType } from 'src/third_party/mlmd';
import { useArtifactTypes } from './useArtifactTypes';

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe('useArtifactTypes', () => {
  let getArtifactTypesSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    getArtifactTypesSpy = vi.spyOn(MlmdUtils, 'getArtifactTypes');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns artifact types on success', async () => {
    const type1 = new ArtifactType();
    type1.setId(1);
    type1.setName('system.Dataset');
    const type2 = new ArtifactType();
    type2.setId(2);
    type2.setName('system.Model');
    getArtifactTypesSpy.mockResolvedValue([type1, type2]);

    const { result } = renderHook(() => useArtifactTypes(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(2);
    expect(result.current.data![0].getName()).toBe('system.Dataset');
    expect(result.current.data![1].getName()).toBe('system.Model');
  });

  it('returns empty array when no artifact types exist', async () => {
    getArtifactTypesSpy.mockResolvedValue([]);

    const { result } = renderHook(() => useArtifactTypes(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual([]);
  });

  it('reports error when getArtifactTypes rejects', async () => {
    getArtifactTypesSpy.mockRejectedValue(new Error('MLMD unreachable'));

    const { result } = renderHook(() => useArtifactTypes(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error?.message).toBe('MLMD unreachable');
  });

  it('does not fetch when enabled is false', () => {
    const { result } = renderHook(() => useArtifactTypes({ enabled: false }), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(getArtifactTypesSpy).not.toHaveBeenCalled();
  });
});
