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
import { QueryClientProvider } from '@tanstack/react-query';
import * as JsYaml from 'js-yaml';
import * as features from 'src/features';
import { CommonTestWrapper } from 'src/TestWrapper';
import { RouteParams } from 'src/components/Router';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { queryClientTest } from 'src/TestUtils';
import { V2beta1Run, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import RunDetailsRouter, { RUN_DETAILS_REFETCH_INTERVAL } from './RunDetailsRouter';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';
import { vi } from 'vitest';

const observedRetryCallbacks = vi.hoisted(() => [] as Array<() => void>);

vi.mock('src/pages/RunDetailsV2', () => ({
  RunDetailsV2: (props: any) => {
    observedRetryCallbacks.push(props.onRetryStarted);
    return (
      <>
        <div
          data-testid='run-details-v2'
          data-pipeline-job={props.pipeline_job}
          data-run-refresh-error={props.runRefreshError?.message}
          data-run-state={props.run.state}
        />
        <input data-testid='run-details-mount' defaultValue={props.run.run_id} />
        <button onClick={props.onRetryStarted}>Retry started</button>
      </>
    );
  },
}));

vi.mock('src/pages/RunDetails', () => ({
  __esModule: true,
  default: (props: any) => (
    <div data-testid='enhanced-run-details' data-is-loading={String(props.isLoading)} />
  ),
}));

const TEST_RUN_ID = 'test-run-id';
const TEST_PIPELINE_ID = 'test-pipeline-id';
const TEST_PIPELINE_VERSION_ID = 'test-pipeline-version-id';

const v2PipelineSpec = JsYaml.load(v2YamlTemplateString);

function generateProps(runId = TEST_RUN_ID) {
  return {
    history: { push: vi.fn(), replace: vi.fn() } as any,
    location: { pathname: `/runs/details/${runId}` } as any,
    match: {
      isExact: true,
      params: { [RouteParams.runId]: runId },
      path: '',
      url: '',
    },
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  } as any;
}

describe('RunDetailsRouter', () => {
  let getRunSpy: ReturnType<typeof vi.spyOn>;
  let getPipelineVersionSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.clearAllMocks();
    observedRetryCallbacks.length = 0;
    getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
    vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
      (featureKey) => featureKey === features.FeatureKey.V2_ALPHA,
    );
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('renders EnhancedRunDetails with isLoading=true while run is fetching', () => {
    getRunSpy.mockReturnValue(new Promise(() => {}));

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    const element = screen.getByTestId('enhanced-run-details');
    expect(element).toBeInTheDocument();
    expect(element.dataset.isLoading).toBe('true');
  });

  it('renders RunDetailsV2 when template is a v2 pipeline spec', async () => {
    const v2Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
    };
    getRunSpy.mockResolvedValue(v2Run);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('run-details-v2')).toBeInTheDocument();
    });
  });

  it('keeps the retry callback stable across parent rerenders', async () => {
    const v2Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
      state: V2beta1RuntimeState.FAILED,
    };
    getRunSpy.mockResolvedValue(v2Run);
    const { rerender } = render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await screen.findByTestId('run-details-v2');
    const firstCallback = observedRetryCallbacks.at(-1);

    rerender(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => expect(observedRetryCallbacks.length).toBeGreaterThan(1));

    expect(observedRetryCallbacks.at(-1)).toBe(firstCallback);
  });

  it('polls an active v2 run and stops after observing its terminal state', async () => {
    vi.useFakeTimers();
    const runningRun: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
      state: V2beta1RuntimeState.RUNNING,
    };
    const succeededRun: V2beta1Run = {
      ...runningRun,
      state: V2beta1RuntimeState.SUCCEEDED,
    };
    getRunSpy.mockResolvedValueOnce(runningRun).mockResolvedValue(succeededRun);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('run-details-v2')).toHaveAttribute(
      'data-run-state',
      V2beta1RuntimeState.RUNNING,
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(RUN_DETAILS_REFETCH_INTERVAL);
      await Promise.resolve();
      await Promise.resolve();
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(getRunSpy).toHaveBeenCalledTimes(2);
    expect(screen.getByTestId('run-details-v2')).toHaveAttribute(
      'data-run-state',
      V2beta1RuntimeState.SUCCEEDED,
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(RUN_DETAILS_REFETCH_INTERVAL * 2);
    });
    expect(getRunSpy).toHaveBeenCalledTimes(2);
  });

  it('keeps cached v2 run details visible when a background refresh fails', async () => {
    vi.useFakeTimers();
    const runningRun: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
      state: V2beta1RuntimeState.RUNNING,
    };
    getRunSpy
      .mockResolvedValueOnce(runningRun)
      .mockRejectedValueOnce(new Error('Run service unavailable'))
      .mockResolvedValue(runningRun);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByTestId('run-details-v2')).toHaveAttribute(
      'data-run-state',
      V2beta1RuntimeState.RUNNING,
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(RUN_DETAILS_REFETCH_INTERVAL);
      await Promise.resolve();
      await Promise.resolve();
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(getRunSpy).toHaveBeenCalledTimes(2);
    expect(screen.getByTestId('run-details-v2')).toHaveAttribute(
      'data-run-refresh-error',
      'Run service unavailable',
    );
    expect(screen.queryByTestId('enhanced-run-details')).not.toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(RUN_DETAILS_REFETCH_INTERVAL);
      await Promise.resolve();
      await Promise.resolve();
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(getRunSpy).toHaveBeenCalledTimes(3);
    expect(screen.getByTestId('run-details-v2')).not.toHaveAttribute('data-run-refresh-error');
  });

  it('keeps polling after retry until a fresh active run state is observed', async () => {
    vi.useFakeTimers();
    const failedRun: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
      state: V2beta1RuntimeState.FAILED,
    };
    const runningRun = { ...failedRun, state: V2beta1RuntimeState.RUNNING };
    getRunSpy.mockResolvedValueOnce(failedRun).mockRejectedValueOnce(new Error('temporary outage'));

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    await act(async () => screen.getByRole('button', { name: 'Retry started' }).click());
    expect(getRunSpy).toHaveBeenCalledTimes(2);

    getRunSpy.mockResolvedValue(runningRun);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(RUN_DETAILS_REFETCH_INTERVAL);
      await Promise.resolve();
      await Promise.resolve();
      await vi.advanceTimersByTimeAsync(1);
    });

    expect(getRunSpy).toHaveBeenCalledTimes(3);
    expect(screen.getByTestId('run-details-v2')).toHaveAttribute(
      'data-run-state',
      V2beta1RuntimeState.RUNNING,
    );
  });

  it('remounts the v2 detail subtree when the run ID changes', async () => {
    const runOne: V2beta1Run = { run_id: 'run-1', pipeline_spec: v2PipelineSpec };
    const runTwo: V2beta1Run = { run_id: 'run-2', pipeline_spec: v2PipelineSpec };
    getRunSpy.mockImplementation(async (runId) => (runId === 'run-1' ? runOne : runTwo));
    const { rerender } = render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps('run-1')} />
      </CommonTestWrapper>,
    );
    await screen.findByTestId('run-details-v2');

    rerender(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps('run-2')} />
      </CommonTestWrapper>,
    );

    await waitFor(() => expect(screen.getByTestId('run-details-mount')).toHaveValue('run-2'));
  });

  it('renders EnhancedRunDetails (V1) when template is not a v2 pipeline spec', async () => {
    const argoWorkflow = {
      apiVersion: 'argoproj.io/v1alpha1',
      kind: 'Workflow',
      metadata: { name: 'test' },
      spec: { arguments: { parameters: [{ name: 'output' }] } },
    };
    const v1Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: argoWorkflow,
    };
    getRunSpy.mockResolvedValue(v1Run);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalledWith(TEST_RUN_ID);
    });

    const element = screen.getByTestId('enhanced-run-details');
    expect(element).toBeInTheDocument();
  });

  it('does not add a router poller for an active v1 run', async () => {
    vi.useFakeTimers();
    const argoWorkflow = {
      apiVersion: 'argoproj.io/v1alpha1',
      kind: 'Workflow',
      metadata: { name: 'test' },
      spec: { arguments: { parameters: [{ name: 'output' }] } },
    };
    getRunSpy.mockResolvedValue({
      run_id: TEST_RUN_ID,
      pipeline_spec: argoWorkflow,
      state: V2beta1RuntimeState.RUNNING,
    });

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByTestId('enhanced-run-details')).toBeInTheDocument();
    expect(getRunSpy).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(RUN_DETAILS_REFETCH_INTERVAL * 2);
    });
    expect(getRunSpy).toHaveBeenCalledTimes(1);
  });

  it('renders EnhancedRunDetails with isLoading=true while pipeline version template is fetching', async () => {
    const runWithVersionRef: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    getRunSpy.mockResolvedValue(runWithVersionRef);
    getPipelineVersionSpy.mockReturnValue(new Promise(() => {}));

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalledWith(TEST_RUN_ID);
    });

    const element = screen.getByTestId('enhanced-run-details');
    expect(element).toBeInTheDocument();
    expect(element.dataset.isLoading).toBe('true');
  });

  describe('template refetch regression', () => {
    afterEach(() => {
      queryClientTest.clear();
    });

    it('keeps EnhancedRunDetails out of loading state during template refetch after the template is cached', async () => {
      const argoWorkflow = {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
        metadata: { name: 'from-version' },
        spec: { arguments: { parameters: [{ name: 'output' }] } },
      };
      const runWithVersionRef: V2beta1Run = {
        run_id: TEST_RUN_ID,
        pipeline_version_reference: {
          pipeline_id: TEST_PIPELINE_ID,
          pipeline_version_id: TEST_PIPELINE_VERSION_ID,
        },
      };
      const pipelineVersion: V2beta1PipelineVersion = {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
        pipeline_spec: argoWorkflow,
      };
      // Use a dedicated query client so the test can invalidate the cached template query directly.
      const wrapper = (props: { children: React.ReactElement }) => (
        <QueryClientProvider client={queryClientTest}>{props.children}</QueryClientProvider>
      );

      getRunSpy.mockResolvedValue(runWithVersionRef);
      getPipelineVersionSpy.mockResolvedValue(pipelineVersion);

      render(<RunDetailsRouter {...generateProps()} />, { wrapper });

      await waitFor(() => {
        expect(getPipelineVersionSpy).toHaveBeenCalledWith(
          TEST_PIPELINE_ID,
          TEST_PIPELINE_VERSION_ID,
        );
      });
      expect(screen.getByTestId('enhanced-run-details').dataset.isLoading).toBe('false');

      getPipelineVersionSpy.mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            setTimeout(() => resolve(pipelineVersion), 100);
          }),
      );

      act(() => {
        queryClientTest.invalidateQueries({
          queryKey: queryKeys.pipelineVersionTemplate(TEST_PIPELINE_ID, TEST_PIPELINE_VERSION_ID),
        });
      });

      expect(screen.getByTestId('enhanced-run-details').dataset.isLoading).toBe('false');

      await waitFor(() => {
        expect(getPipelineVersionSpy).toHaveBeenCalledTimes(2);
      });
      expect(screen.getByTestId('enhanced-run-details').dataset.isLoading).toBe('false');
    });
  });

  it('does not fetch pipeline version when run has an inline pipeline_spec', async () => {
    // usePipelineVersionTemplate is only enabled when pipelineId and pipelineVersionId
    // are both present. A run with only pipeline_spec has no version reference, so the
    // hook stays disabled and getPipelineVersion is never called.
    const v2Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
    };
    getRunSpy.mockResolvedValue(v2Run);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('run-details-v2')).toBeInTheDocument();
    });
    expect(getPipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('prefers inline pipeline_spec over pipeline version template when both are present', async () => {
    const v2Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    getRunSpy.mockResolvedValue(v2Run);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(screen.getByTestId('run-details-v2')).toBeInTheDocument();
    });
    expect(getPipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('renders EnhancedRunDetails when getRun fails', async () => {
    getRunSpy.mockRejectedValue(new Error('Not found'));

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getRunSpy).toHaveBeenCalledWith(TEST_RUN_ID);
    });

    await waitFor(() => {
      const element = screen.getByTestId('enhanced-run-details');
      expect(element).toBeInTheDocument();
      expect(element.dataset.isLoading).toBe('false');
    });
  });

  it('shows error banner when pipeline version template fetch fails', async () => {
    const runWithVersionRef: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    getRunSpy.mockResolvedValue(runWithVersionRef);
    getPipelineVersionSpy.mockRejectedValue(new Error('Version not found'));

    const props = generateProps();
    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...props} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(props.updateBanner).toHaveBeenCalledWith(
        expect.objectContaining({
          message: expect.stringContaining('failed to retrieve pipeline version template'),
          mode: 'error',
        }),
      );
    });
  });

  it('does not show error banner when inline pipeline_spec is present and getPipelineVersion rejects', async () => {
    const v2Run: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_spec: v2PipelineSpec,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    getRunSpy.mockResolvedValue(v2Run);
    getPipelineVersionSpy.mockRejectedValue(new Error('Version not found'));

    const props = generateProps();
    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...props} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('run-details-v2')).toBeInTheDocument();
    });
    expect(props.updateBanner).not.toHaveBeenCalled();
  });

  it('fetches template from pipeline version when run has no inline spec', async () => {
    const runWithVersionRef: V2beta1Run = {
      run_id: TEST_RUN_ID,
      pipeline_version_reference: {
        pipeline_id: TEST_PIPELINE_ID,
        pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      },
    };
    const pipelineVersion: V2beta1PipelineVersion = {
      pipeline_id: TEST_PIPELINE_ID,
      pipeline_version_id: TEST_PIPELINE_VERSION_ID,
      pipeline_spec: v2PipelineSpec,
    };
    getRunSpy.mockResolvedValue(runWithVersionRef);
    getPipelineVersionSpy.mockResolvedValue(pipelineVersion);

    render(
      <CommonTestWrapper>
        <RunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getPipelineVersionSpy).toHaveBeenCalledWith(
        TEST_PIPELINE_ID,
        TEST_PIPELINE_VERSION_ID,
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('run-details-v2')).toBeInTheDocument();
    });
  });
});
