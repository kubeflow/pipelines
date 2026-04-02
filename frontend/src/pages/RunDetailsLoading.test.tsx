/*
 * Copyright 2024 The Kubeflow Authors
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

import { render, screen, waitFor, act } from '@testing-library/react';
import * as React from 'react';
import { QueryClientProvider } from '@tanstack/react-query';
import { RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { queryClientTest, testBestPractices } from 'src/TestUtils';
import RunDetailsRouter from './RunDetailsRouter';
import { vi } from 'vitest';

// Mock the APIs
vi.mock('src/lib/Apis', () => ({
  Apis: {
    runServiceApiV2: {
      getRun: vi.fn(),
    },
    pipelineServiceApiV2: {
      getPipelineVersion: vi.fn(),
    },
  },
}));

testBestPractices();

describe('RunDetails loading behavior', () => {
  let getRunSpy: any;
  let getPipelineVersionSpy: any;

  const testRunId = 'test-run-id';
  const testPipelineId = 'test-pipeline-id';
  const testPipelineVersionId = 'test-pipeline-version-id';

  const mockRunData = {
    run: {
      id: testRunId,
      name: 'Test Run',
      status: 'Succeeded',
      created_at: new Date(),
    },
    pipeline_version_reference: {
      pipeline_id: testPipelineId,
      pipeline_version_id: testPipelineVersionId,
    },
  };

  const mockPipelineVersionData = {
    pipeline_spec: {
      pipeline_info: { name: 'test-pipeline' },
    },
  };

  // Create a test wrapper with QueryClientProvider
  const wrapper = ({ children }: { children: React.ReactElement }) => (
    <QueryClientProvider client={queryClientTest}>{children}</QueryClientProvider>
  );

  const mockPageProps = {
    match: {
      params: {
        [RouteParams.runId]: testRunId,
      },
      isExact: true,
      path: '',
      url: '',
    },
    history: { push: vi.fn() } as any,
    location: {} as any,
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  };

  beforeEach(() => {
    // Setup spies
    getRunSpy = vi.mocked(Apis.runServiceApiV2.getRun);
    getPipelineVersionSpy = vi.mocked(Apis.pipelineServiceApiV2.getPipelineVersion);

    // Default mock implementations
    getRunSpy.mockResolvedValue(mockRunData);
    getPipelineVersionSpy.mockResolvedValue(mockPipelineVersionData);
  });

  afterEach(() => {
    vi.clearAllMocks();
    queryClientTest.clear();
  });

  it('shows loading spinner during initial data fetch', async () => {
    // Arrange: Set up a delay to simulate network latency
    getRunSpy.mockImplementation(
      () =>
        new Promise(resolve => {
          setTimeout(() => resolve(mockRunData), 100);
        }),
    );

    // Act: Render the component
    render(<RunDetailsRouter {...mockPageProps} />, { wrapper });

    // Assert: Loading spinner should be visible initially
    expect(screen.getByText('Currently loading run information.')).toBeInTheDocument();
    expect(screen.getByRole('progressbar')).toBeInTheDocument();

    // Wait for loading to complete
    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalled();
    });

    // After loading completes, spinner should be gone
    await waitFor(() => {
      expect(screen.queryByText('Currently loading run information.')).not.toBeInTheDocument();
    });
  });

  it('does NOT show loading spinner during refetch (isFetching state)', async () => {
    // Arrange: Initial data is already loaded
    getRunSpy.mockResolvedValue(mockRunData);

    // Render and wait for initial load
    render(<RunDetailsRouter {...mockPageProps} />, { wrapper });

    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalled();
    });

    // Verify spinner is not visible after initial load
    expect(screen.queryByText('Currently loading run information.')).not.toBeInTheDocument();

    // Act: Simulate a refetch by invalidating the query
    await act(async () => {
      queryClientTest.invalidateQueries(['v2_run_detail']);
    });

    // Set up a delay for the refetch
    getRunSpy.mockImplementation(
      () =>
        new Promise(resolve => {
          setTimeout(() => resolve(mockRunData), 100);
        }),
    );

    // Assert: During refetch, the loading spinner should NOT appear
    // This is the key behavior: isLoading is false during refetch, only isFetching is true
    expect(screen.queryByText('Currently loading run information.')).not.toBeInTheDocument();

    // Wait for refetch to complete
    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalledTimes(2);
    });

    // Spinner should still not be visible
    expect(screen.queryByText('Currently loading run information.')).not.toBeInTheDocument();
  });

  it('hides loading spinner when isLoading is false even if runMetadata is null', async () => {
    // This test ensures that when isLoading is false, we don't show the spinner
    // even if runMetadata hasn't been loaded yet (which shouldn't happen in normal flow)

    // Arrange: Mock successful immediate load
    getRunSpy.mockResolvedValue(mockRunData);

    // Act: Render and wait for load to complete
    render(<RunDetailsRouter {...mockPageProps} />, { wrapper });

    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalled();
    });

    // Assert: After loading completes, isLoading becomes false and spinner is hidden
    expect(screen.queryByText('Currently loading run information.')).not.toBeInTheDocument();
  });

  it('handles template string query loading state correctly', async () => {
    // Arrange: Make the pipeline version query slow
    getPipelineVersionSpy.mockImplementation(
      () =>
        new Promise(resolve => {
          setTimeout(() => resolve(mockPipelineVersionData), 150);
        }),
    );

    // Act: Render the component
    render(<RunDetailsRouter {...mockPageProps} />, { wrapper });

    // Assert: Should show loading because templateStrIsLoading is true
    expect(screen.getByText('Currently loading run information.')).toBeInTheDocument();

    // Wait for both queries to complete
    await waitFor(() => {
      expect(getPipelineVersionSpy).toHaveBeenCalled();
    });

    // After both queries complete, loading should be false
    await waitFor(() => {
      expect(screen.queryByText('Currently loading run information.')).not.toBeInTheDocument();
    });
  });
});


