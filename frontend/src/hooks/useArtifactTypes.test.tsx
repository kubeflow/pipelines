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
import * as MlmdUtils from 'src/mlmd/MlmdUtils';
import { testBestPractices } from 'src/TestUtils';
import { queryKeys } from './queryKeys';
import { useArtifactTypes } from './useArtifactTypes';

testBestPractices();

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

function TestComponent({ enabled = true }: { enabled?: boolean }) {
  const { data, isSuccess } = useArtifactTypes({ enabled });
  return (
    <div>
      <span data-testid='isSuccess'>{String(isSuccess)}</span>
      <span data-testid='data'>{data ? JSON.stringify(data.length) : 'undefined'}</span>
    </div>
  );
}

describe('useArtifactTypes', () => {
  it('uses canonical query key artifactTypes', () => {
    expect(queryKeys.artifactTypes()).toEqual(['artifact_types']);
  });

  it('fetches artifact types and caches by canonical key', async () => {
    const mockArtifactTypes = [{ getId: () => 1, getName: () => 'test' }];
    vi.spyOn(MlmdUtils, 'getArtifactTypes').mockResolvedValue(mockArtifactTypes as never);

    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <TestComponent />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('isSuccess').textContent).toBe('true'));
    expect(MlmdUtils.getArtifactTypes).toHaveBeenCalledTimes(1);
  });

  it('respects enabled: false and does not fetch', async () => {
    const getArtifactTypesSpy = vi.spyOn(MlmdUtils, 'getArtifactTypes').mockResolvedValue([]);

    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <TestComponent enabled={false} />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('isSuccess').textContent).toBe('false'));
    expect(getArtifactTypesSpy).not.toHaveBeenCalled();
  });
});
