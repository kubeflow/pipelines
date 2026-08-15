/*
 * Copyright 2021 The Kubeflow Authors
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

import { act, fireEvent, queryByText, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';

import {
  ArtifactArtifactType,
  PipelineTaskTaskState,
  PipelineTaskTaskPodType,
  PipelineTaskTaskType,
  V2beta1PipelineTask,
  V2beta1Run,
  V2beta1RuntimeState,
} from 'src/apisv2beta1/run';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { RoutePage, RouteParams } from 'src/components/Router';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { Apis } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { mockResizeObserver, testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { queryKeys } from 'src/hooks/queryKeys';
import * as DynamicFlow from 'src/lib/v2/DynamicFlow';
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { PageProps } from './Page';
import { RunDetailsInternalProps } from './RunDetails';
import { RunDetailsV2 } from './RunDetailsV2';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';

vi.mock('src/components/Editor', () => ({
  default: ({ value }: { value?: string }) => <pre data-testid='Editor'>{value}</pre>,
}));

testBestPractices();
describe('RunDetailsV2', () => {
  const RUN_ID = '1';
  const TEST_PIPELINE_SPEC = convertYamlToV2PipelineSpec(v2YamlTemplateString);

  let updateBannerSpy: any;
  let updateDialogSpy: any;
  let updateSnackbarSpy: any;
  let updateToolbarSpy: any;
  let historyPushSpy: any;
  let historyReplaceSpy: any;

  function deferred<T>() {
    let resolve!: (value: T) => void;
    const promise = new Promise<T>((resolvePromise) => {
      resolve = resolvePromise;
    });
    return { promise, resolve };
  }

  function generateProps(): RunDetailsInternalProps &
    PageProps & { parsedPipelineSpec: PipelineSpec } {
    const pageProps: PageProps = {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: '' as any,
      match: {
        params: {
          [RouteParams.runId]: RUN_ID,
        },
        isExact: true,
        path: '',
        url: '',
      },
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
    return Object.assign(pageProps, {
      gkeMetadata: {},
      parsedPipelineSpec: TEST_PIPELINE_SPEC,
    });
  }
  const TEST_RUN: V2beta1Run = {
    created_at: new Date(2018, 8, 5, 4, 3, 2),
    scheduled_at: new Date(2018, 8, 6, 4, 3, 2),
    finished_at: new Date(2018, 8, 7, 4, 3, 2),
    description: 'test run description',
    experiment_id: 'some-experiment-id',
    run_id: 'test-run-id',
    display_name: 'test run',
    pipeline_spec: {
      pipeline_id: 'some-pipeline-id',
      pipeline_manifest: '{some-template-string}',
    },
    runtime_config: { parameters: { param1: 'value1' } },
    state: V2beta1RuntimeState.SUCCEEDED,
  };
  const TEST_EXPERIMENT: V2beta1Experiment = {
    created_at: '2021-01-24T18:03:08Z',
    description: 'All runs will be grouped here.',
    experiment_id: 'some-experiment-id',
    display_name: 'Default',
    storage_state: V2beta1ExperimentStorageState.AVAILABLE,
  };
  const TEST_TASKS: V2beta1PipelineTask[] = [
    {
      task_id: 'root-task',
      run_id: RUN_ID,
      name: 'root',
      type: PipelineTaskTaskType.ROOT,
      state: PipelineTaskTaskState.SUCCEEDED,
    },
    {
      task_id: 'preprocess-task',
      parent_task_id: 'root-task',
      run_id: RUN_ID,
      name: 'preprocess',
      display_name: 'preprocess',
      type: PipelineTaskTaskType.RUNTIME,
      state: PipelineTaskTaskState.SUCCEEDED,
    },
    {
      task_id: 'train-task',
      parent_task_id: 'root-task',
      run_id: RUN_ID,
      name: 'train',
      display_name: 'train',
      type: PipelineTaskTaskType.RUNTIME,
      state: PipelineTaskTaskState.SUCCEEDED,
      outputs: {
        artifacts: [
          {
            artifact_key: 'model',
            artifacts: [
              {
                artifact_id: 'model-artifact',
                name: 'model',
                type: ArtifactArtifactType.Model,
                uri: 's3://pipeline-root/model',
              },
            ],
          },
        ],
      },
    },
  ];

  function renderRunDetailsWithSearch(search: string) {
    const props = generateProps();
    const renderPage = (nextSearch: string) => (
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={TEST_RUN}
          {...props}
          location={{ pathname: `/runs/details/${RUN_ID}`, search: nextSearch } as any}
        />
      </CommonTestWrapper>
    );
    const view = render(renderPage(search));
    return {
      rerenderWithSearch: (nextSearch: string) => view.rerender(renderPage(nextSearch)),
    };
  }

  beforeEach(() => {
    mockResizeObserver();

    updateBannerSpy = vi.fn();
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    updateToolbarSpy = vi.fn();
    historyPushSpy = vi.fn();
    historyReplaceSpy = vi.fn();

    vi.spyOn(Apis.runServiceApiV2, 'tasks').mockResolvedValue({ tasks: TEST_TASKS });
    vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment').mockResolvedValue(TEST_EXPERIMENT);
  });

  afterEach(() => vi.useRealTimers());

  it('Render detail page with reactflow', async () => {
    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('DagCanvas')).not.toBeNull();
  });

  it('opens the native task targeted by a related-task link', async () => {
    const props = generateProps();
    props.location = {
      hash: '#logs',
      pathname: `/runs/details/${RUN_ID}`,
      search: '?task=preprocess-task&view=graph',
      state: undefined,
    } as any;

    render(
      <CommonTestWrapper>
        <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...props} />
      </CommonTestWrapper>,
    );

    fireEvent.click(await screen.findByText('Task Details'));
    await screen.findByText('preprocess-task');
    await waitFor(() =>
      expect(document.querySelector('[data-id="task.preprocess"]')).toHaveClass('selected'),
    );
    fireEvent.click(screen.getByRole('button', { name: 'close' }));
    expect(historyReplaceSpy).toHaveBeenCalledWith({
      ...props.location,
      search: '?view=graph',
    });
  });

  it('selects a new task when query-only navigation changes the deep-link target', async () => {
    const { rerenderWithSearch } = renderRunDetailsWithSearch('?task=preprocess-task');
    await waitFor(() =>
      expect(document.querySelector('[data-id="task.preprocess"]')).toHaveClass('selected'),
    );

    rerenderWithSearch('?task=train-task');

    await waitFor(() => {
      expect(document.querySelector('[data-id="task.train"]')).toHaveClass('selected');
      expect(document.querySelector('[data-id="task.preprocess"]')).not.toHaveClass('selected');
    });
  });

  it('clears a deep-linked selection when query-only navigation removes the task target', async () => {
    const { rerenderWithSearch } = renderRunDetailsWithSearch('?task=preprocess-task&view=graph');
    await waitFor(() =>
      expect(document.querySelector('[data-id="task.preprocess"]')).toHaveClass('selected'),
    );

    rerenderWithSearch('?view=graph');

    await waitFor(() => {
      expect(document.querySelector('[data-id="task.preprocess"]')).not.toHaveClass('selected');
    });
  });

  it('preserves a canvas selection when selecting a node clears the task query', async () => {
    const { rerenderWithSearch } = renderRunDetailsWithSearch('?task=preprocess-task');
    await waitFor(() =>
      expect(document.querySelector('[data-id="task.preprocess"]')).toHaveClass('selected'),
    );

    fireEvent.click(document.querySelector('[data-id="task.train"]')!);
    await waitFor(() =>
      expect(document.querySelector('[data-id="task.train"]')).toHaveClass('selected'),
    );
    expect(historyReplaceSpy).toHaveBeenCalledWith({
      pathname: `/runs/details/${RUN_ID}`,
      search: '',
    });

    rerenderWithSearch('');
    await act(async () => {});

    expect(document.querySelector('[data-id="task.train"]')).toHaveClass('selected');
    expect(document.querySelector('[data-id="task.preprocess"]')).not.toHaveClass('selected');
  });

  it('clears stale task details when query-only navigation targets an unknown task', async () => {
    const { rerenderWithSearch } = renderRunDetailsWithSearch('?task=preprocess-task');
    await waitFor(() =>
      expect(document.querySelector('[data-id="task.preprocess"]')).toHaveClass('selected'),
    );

    rerenderWithSearch('?task=missing-task');

    await waitFor(() => {
      expect(document.querySelector('[data-id="task.preprocess"]')).not.toHaveClass('selected');
    });
  });

  it('does not expose sub-DAG navigation for a linked root task', async () => {
    const props = generateProps();
    props.location = {
      pathname: `/runs/details/${RUN_ID}`,
      search: '?task=root-task',
    } as any;

    render(
      <CommonTestWrapper>
        <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...props} />
      </CommonTestWrapper>,
    );

    expect(await screen.findByText('Task Details')).toBeVisible();
    expect(screen.queryByText('Open Sub-DAG')).toBeNull();
  });

  it('keeps runtime flow elements stable across same-props rerenders', async () => {
    const reconcileRuntimeFlowElementsSpy = vi.spyOn(DynamicFlow, 'reconcileRuntimeFlowElements');
    const props = generateProps();

    const view = render(
      <CommonTestWrapper>
        <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...props}></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() => expect(reconcileRuntimeFlowElementsSpy).toHaveBeenCalled());
    const callCountAfterLoad = reconcileRuntimeFlowElementsSpy.mock.calls.length;

    view.rerender(
      <CommonTestWrapper>
        <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...props}></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await act(async () => {});
    expect(reconcileRuntimeFlowElementsSpy).toHaveBeenCalledTimes(callCountAfterLoad);
  });

  it('preserves task identity when a poll returns byte-equivalent timestamps', async () => {
    vi.useFakeTimers();
    const reconcileRuntimeFlowElementsSpy = vi.spyOn(DynamicFlow, 'reconcileRuntimeFlowElements');
    const tasksSpy = vi.spyOn(Apis.runServiceApiV2, 'tasks').mockImplementation(async () => ({
      tasks: TEST_TASKS.map((task) => ({
        ...task,
        create_time: new Date('2026-08-14T12:00:00Z'),
      })),
    }));
    const runningRun = {
      ...TEST_RUN,
      finished_at: undefined,
      state: V2beta1RuntimeState.RUNNING,
    };

    render(
      <CommonTestWrapper>
        <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={runningRun} {...generateProps()} />
      </CommonTestWrapper>,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(reconcileRuntimeFlowElementsSpy).toHaveBeenCalled();
    const reconciliationsAfterLoad = reconcileRuntimeFlowElementsSpy.mock.calls.length;

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(tasksSpy).toHaveBeenCalledTimes(2);
    expect(reconcileRuntimeFlowElementsSpy).toHaveBeenCalledTimes(reconciliationsAfterLoad);
    reconcileRuntimeFlowElementsSpy.mockRestore();
    vi.useRealTimers();
  });

  it('Shows error banner when tasks cannot be retrieved', async () => {
    vi.spyOn(Apis.runServiceApiV2, 'tasks').mockRejectedValue(
      new Error('Task service unavailable'),
    );

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'Task service unavailable',
          message: 'Cannot get tasks for this run. Refresh the page to try again.',
          mode: 'error',
        }),
      ),
    );
  });

  it('Shows experiment warning banner when experiment fetch fails and task fetch succeeds', async () => {
    vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment').mockRejectedValue(
      new Error('Experiment not found'),
    );

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          additionalInfo: 'Experiment not found',
          message: 'Error: failed to retrieve experiment details.',
          mode: 'warning',
        }),
      ),
    );
  });

  it('uses the selected namespace for pod logs when experiment lookup fails', async () => {
    vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment').mockRejectedValue(
      new Error('Experiment not found'),
    );
    vi.spyOn(Apis.runServiceApiV2, 'tasks').mockResolvedValue({
      tasks: TEST_TASKS.map((task) =>
        task.name === 'preprocess'
          ? {
              ...task,
              create_time: new Date('2026-08-12T12:00:00Z'),
              pods: [{ name: 'preprocess-pod', type: PipelineTaskTaskPodType.EXECUTOR }],
            }
          : task,
      ),
    });
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('live logs');

    render(
      <NamespaceContext.Provider value='team-a'>
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>
      </NamespaceContext.Provider>,
    );

    fireEvent.click(await screen.findByText('preprocess'));
    fireEvent.click(await screen.findByText('Logs'));

    await waitFor(() =>
      expect(getPodLogsSpy).toHaveBeenCalledWith(RUN_ID, 'preprocess-pod', 'team-a', '2026-08-12'),
    );
  });

  it('Shows task error banner even when experiment also fails (task error takes precedence)', async () => {
    vi.spyOn(Apis.runServiceApiV2, 'tasks').mockRejectedValue(
      new Error('Task service unavailable'),
    );
    vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment').mockRejectedValue(
      new Error('Experiment not found'),
    );

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          message: 'Cannot get tasks for this run. Refresh the page to try again.',
          mode: 'error',
        }),
      ),
    );
  });

  it('Does not clear experiment warning when task fetch succeeds after experiment fails', async () => {
    vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment').mockRejectedValue(
      new Error('Experiment not found'),
    );

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    // Wait for both queries to settle — the last banner call should be the experiment warning,
    // not a clear ({}) from the task success path.
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          message: 'Error: failed to retrieve experiment details.',
          mode: 'warning',
        }),
      ),
    );
  });

  it('Shows no banner when tasks and experiment load', async () => {
    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() => expect(updateBannerSpy).toHaveBeenLastCalledWith({}));
  });

  it('shows cached run data with an inline warning when a run refresh fails', async () => {
    const props = generateProps();
    const view = render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={TEST_RUN}
          runRefreshError={new Error('Run service unavailable')}
          {...props}
        />
      </CommonTestWrapper>,
    );

    await screen.findByText(
      'Unable to refresh this run. The last known run state is still shown. Refresh the page to try again.',
    );
    fireEvent.click(screen.getByRole('button', { name: 'Details' }));
    await screen.findByText('Run service unavailable');

    view.rerender(
      <CommonTestWrapper>
        <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...props} />
      </CommonTestWrapper>,
    );

    expect(
      screen.queryByText(
        'Unable to refresh this run. The last known run state is still shown. Refresh the page to try again.',
      ),
    ).not.toBeInTheDocument();
  });

  it("shows run title and experiments' links", async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    getRunSpy.mockResolvedValue(TEST_RUN);
    const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
    getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

    await act(async () => {
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );
    });

    await waitFor(() =>
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          pageTitleTooltip: 'test run',
        }),
      ),
    );
    await waitFor(() =>
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
            {
              displayName: 'Default',
              href: `/experiments/details/some-experiment-id`,
            },
          ],
        }),
      ),
    );
  });

  it('shows top bar buttons', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    getRunSpy.mockResolvedValue(TEST_RUN);
    const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
    getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

    await act(async () => {
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );
    });

    await waitFor(() =>
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          actions: expect.objectContaining({
            archive: expect.objectContaining({ disabled: false, title: 'Archive' }),
            retry: expect.objectContaining({ disabled: true, title: 'Retry' }),
            terminateRun: expect.objectContaining({ disabled: true, title: 'Terminate' }),
            cloneRun: expect.objectContaining({ disabled: false, title: 'Clone run' }),
          }),
        }),
      ),
    );
  });

  it('derives the terminate action from the current run state', async () => {
    const props = generateProps();
    const runningRun = { ...TEST_RUN, state: V2beta1RuntimeState.RUNNING };
    const succeededRun = { ...TEST_RUN, state: V2beta1RuntimeState.SUCCEEDED };
    const view = render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={runningRun}
          {...props}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );
    const getLatestTerminateDisabled = () => {
      const actionUpdates = updateToolbarSpy.mock.calls.filter(([update]: any[]) => update.actions);
      return actionUpdates[actionUpdates.length - 1]?.[0].actions.terminateRun.disabled;
    };

    await waitFor(() => expect(getLatestTerminateDisabled()).toBe(false));

    view.rerender(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={succeededRun}
          {...props}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );
    await waitFor(() => expect(getLatestTerminateDisabled()).toBe(true));

    view.rerender(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={runningRun}
          {...props}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );
    await waitFor(() => expect(getLatestTerminateDisabled()).toBe(false));
  });

  it('reconciles tasks only after the polling owner discovers the retried run', async () => {
    const onRetryStarted = vi.fn();
    const tasksSpy = vi.mocked(Apis.runServiceApiV2.tasks);
    vi.spyOn(Apis.runServiceApiV2, 'retryRun').mockResolvedValue({});
    const props = generateProps();
    const view = render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          onRetryStarted={onRetryStarted}
          run={{ ...TEST_RUN, state: V2beta1RuntimeState.FAILED }}
          {...props}
        />
      </CommonTestWrapper>,
    );
    await waitFor(() => expect(tasksSpy).toHaveBeenCalledTimes(1));

    const getRetryAction = () => {
      const actionUpdates = updateToolbarSpy.mock.calls.filter(([update]: any[]) => update.actions);
      return actionUpdates.at(-1)?.[0].actions.retry.action;
    };
    await waitFor(() => expect(getRetryAction()).toBeDefined());
    getRetryAction()();
    const confirmButton = updateDialogSpy.mock.calls
      .at(-1)?.[0]
      .buttons.find((button: { text: string }) => button.text === 'Retry');
    await confirmButton.onClick();

    expect(onRetryStarted).toHaveBeenCalledTimes(1);
    expect(tasksSpy).toHaveBeenCalledTimes(1);

    view.rerender(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          onRetryStarted={onRetryStarted}
          retryRefreshVersion={1}
          run={{ ...TEST_RUN, state: V2beta1RuntimeState.FAILED }}
          {...props}
        />
      </CommonTestWrapper>,
    );
    await waitFor(() => expect(tasksSpy).toHaveBeenCalledTimes(2));
  });

  it('recovers retry task reconciliation and bounds terminal intermediate snapshots', async () => {
    vi.useFakeTimers();
    const tasksSpy = vi.mocked(Apis.runServiceApiV2.tasks);
    const unfinishedTasks = TEST_TASKS.map((task) =>
      task.task_id === 'preprocess-task' ? { ...task, state: PipelineTaskTaskState.RUNNING } : task,
    );
    tasksSpy
      .mockRejectedValueOnce(new Error('Task service unavailable'))
      .mockResolvedValueOnce({ tasks: unfinishedTasks })
      .mockResolvedValue({ tasks: TEST_TASKS });

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          retryRefreshVersion={1}
          run={{ ...TEST_RUN, state: V2beta1RuntimeState.FAILED }}
          {...generateProps()}
        />
      </CommonTestWrapper>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(tasksSpy).toHaveBeenCalledTimes(1);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(2);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);

    await act(async () => vi.advanceTimersByTimeAsync(20_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);
  });

  it('bounds terminal task reconciliation when every task request fails', async () => {
    vi.useFakeTimers();
    const tasksSpy = vi
      .mocked(Apis.runServiceApiV2.tasks)
      .mockRejectedValue(new Error('Task service unavailable'));

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={{ ...TEST_RUN, state: V2beta1RuntimeState.FAILED }}
          {...generateProps()}
        />
      </CommonTestWrapper>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(tasksSpy).toHaveBeenCalledTimes(1);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(2);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);

    await act(async () => vi.advanceTimersByTimeAsync(20_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);
  });

  it('allows two unfinished retry snapshots before accepting the finished tasks', async () => {
    vi.useFakeTimers();
    const tasksSpy = vi.mocked(Apis.runServiceApiV2.tasks);
    const unfinishedTasks = TEST_TASKS.map((task) =>
      task.task_id === 'preprocess-task' ? { ...task, state: PipelineTaskTaskState.RUNNING } : task,
    );
    tasksSpy
      .mockResolvedValueOnce({ tasks: unfinishedTasks })
      .mockResolvedValueOnce({ tasks: unfinishedTasks })
      .mockResolvedValue({ tasks: TEST_TASKS });

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          retryRefreshVersion={1}
          run={{ ...TEST_RUN, state: V2beta1RuntimeState.FAILED }}
          {...generateProps()}
        />
      </CommonTestWrapper>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(tasksSpy).toHaveBeenCalledTimes(1);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(2);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);

    await act(async () => vi.advanceTimersByTimeAsync(20_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);
  });

  it('does not count a cancelled task request as an accepted reconciliation snapshot', async () => {
    vi.useFakeTimers();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const firstRequest = deferred<{ tasks: V2beta1PipelineTask[] }>();
    const replacementRequest = deferred<{ tasks: V2beta1PipelineTask[] }>();
    const unfinishedTasks = TEST_TASKS.map((task) =>
      task.task_id === 'preprocess-task' ? { ...task, state: PipelineTaskTaskState.RUNNING } : task,
    );
    const tasksSpy = vi.mocked(Apis.runServiceApiV2.tasks);
    tasksSpy
      .mockReturnValueOnce(firstRequest.promise)
      .mockReturnValueOnce(replacementRequest.promise)
      .mockResolvedValue({ tasks: TEST_TASKS });
    const taskQueryKey = queryKeys.runTasks(RUN_ID, 1);

    render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            retryRefreshVersion={1}
            run={{ ...TEST_RUN, state: V2beta1RuntimeState.FAILED }}
            {...generateProps()}
          />
        </QueryClientProvider>
      </MemoryRouter>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(tasksSpy).toHaveBeenCalledTimes(1);

    await act(async () => queryClient.cancelQueries({ queryKey: taskQueryKey }));
    let replacementRefetch!: Promise<void>;
    act(() => {
      replacementRefetch = queryClient.refetchQueries({ queryKey: taskQueryKey });
    });
    expect(tasksSpy).toHaveBeenCalledTimes(2);

    firstRequest.resolve({ tasks: unfinishedTasks });
    replacementRequest.resolve({ tasks: unfinishedTasks });
    await act(async () => replacementRefetch);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);
  });

  it('refetches a fresh cached task snapshot when mounting a terminal run', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    queryClient.setQueryData(queryKeys.runTasks(RUN_ID), TEST_TASKS);
    const tasksSpy = vi.mocked(Apis.runServiceApiV2.tasks);

    render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...generateProps()} />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    await waitFor(() => expect(tasksSpy).toHaveBeenCalled());
  });

  it('continues terminal task reconciliation after remounting with unfinished cached tasks', async () => {
    vi.useFakeTimers();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const unfinishedTasks = TEST_TASKS.map((task) =>
      task.task_id === 'preprocess-task' ? { ...task, state: PipelineTaskTaskState.RUNNING } : task,
    );
    queryClient.setQueryData(queryKeys.runTasks(RUN_ID), unfinishedTasks);
    const tasksSpy = vi
      .mocked(Apis.runServiceApiV2.tasks)
      .mockResolvedValueOnce({ tasks: unfinishedTasks })
      .mockResolvedValueOnce({ tasks: unfinishedTasks })
      .mockResolvedValue({ tasks: TEST_TASKS });
    const props = generateProps();

    const firstView = render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...props} />
        </QueryClientProvider>
      </MemoryRouter>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(tasksSpy).toHaveBeenCalledTimes(1);
    firstView.unmount();

    render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...props} />
        </QueryClientProvider>
      </MemoryRouter>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(tasksSpy).toHaveBeenCalledTimes(2);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);
  });

  it('keeps cached task data in the graph when a background refresh fails', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData(queryKeys.runTasks(RUN_ID), TEST_TASKS);
    vi.mocked(Apis.runServiceApiV2.tasks).mockRejectedValue(new Error('temporary task outage'));
    const reconcileSpy = vi.spyOn(DynamicFlow, 'reconcileRuntimeFlowElements');

    render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={TEST_RUN} {...generateProps()} />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({ additionalInfo: 'temporary task outage', mode: 'error' }),
      ),
    );
    expect(reconcileSpy).toHaveBeenCalledWith(
      expect.anything(),
      expect.anything(),
      TEST_TASKS,
      expect.anything(),
    );
  });

  it('refetches tasks once when an active run becomes terminal', async () => {
    const tasksSpy = vi.spyOn(Apis.runServiceApiV2, 'tasks');
    const props = generateProps();
    const runningRun = { ...TEST_RUN, state: V2beta1RuntimeState.RUNNING };
    const succeededRun = { ...TEST_RUN, state: V2beta1RuntimeState.SUCCEEDED };
    const view = render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={runningRun}
          {...props}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() => expect(tasksSpy).toHaveBeenCalledTimes(1));

    view.rerender(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={succeededRun}
          {...props}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );
    await waitFor(() => expect(tasksSpy).toHaveBeenCalledTimes(2));

    view.rerender(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={v2YamlTemplateString}
          run={succeededRun}
          {...props}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );
    await act(async () => {});
    expect(tasksSpy).toHaveBeenCalledTimes(2);
  });

  it('bounds unfinished task reconciliation after an active run becomes terminal', async () => {
    vi.useFakeTimers();
    const tasksSpy = vi.mocked(Apis.runServiceApiV2.tasks);
    const unfinishedTasks = TEST_TASKS.map((task) =>
      task.task_id === 'preprocess-task' ? { ...task, state: PipelineTaskTaskState.RUNNING } : task,
    );
    tasksSpy
      .mockResolvedValueOnce({ tasks: unfinishedTasks })
      .mockResolvedValueOnce({ tasks: unfinishedTasks })
      .mockResolvedValueOnce({ tasks: unfinishedTasks })
      .mockResolvedValue({ tasks: TEST_TASKS });
    const props = generateProps();
    const runningRun = { ...TEST_RUN, state: V2beta1RuntimeState.RUNNING };
    const failedRun = { ...TEST_RUN, state: V2beta1RuntimeState.FAILED };
    const view = render(
      <CommonTestWrapper>
        <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={runningRun} {...props} />
      </CommonTestWrapper>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(tasksSpy).toHaveBeenCalledTimes(1);

    view.rerender(
      <CommonTestWrapper>
        <RunDetailsV2 pipeline_job={v2YamlTemplateString} run={failedRun} {...props} />
      </CommonTestWrapper>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(tasksSpy).toHaveBeenCalledTimes(2);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(3);

    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tasksSpy).toHaveBeenCalledTimes(4);

    await act(async () => vi.advanceTimersByTimeAsync(20_000));
    expect(tasksSpy).toHaveBeenCalledTimes(4);
  });

  describe('topbar tabs', () => {
    it('switches to Detail tab', async () => {
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Detail'));

      screen.getByText('Run details');
      screen.getByText('Run ID');
      screen.getByText('Workflow name');
      screen.getByText('Status');
      screen.getByText('Description');
      screen.getByText('Created at');
      screen.getByText('Started at');
      screen.getByText('Finished at');
      screen.getByText('Duration');
    });

    it('shows content in Detail tab', async () => {
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Detail'));

      screen.getByText('test-run-id'); // 'Run ID'
      screen.getByText('test run'); // 'Workflow name'
      screen.getByText('test run description'); // 'Description'
      screen.getByText('9/5/2018, 4:03:02 AM'); //'Created at'
      screen.getByText('9/6/2018, 4:03:02 AM'); // 'Started at'
      screen.getByText('9/7/2018, 4:03:02 AM'); // 'Finished at'
      screen.getByText('48:00:00'); // 'Duration'
    });

    it('handles no creation time', async () => {
      const noCreateTimeRun: V2beta1Run = {
        // created_at: new Date(2018, 8, 5, 4, 3, 2),
        scheduled_at: new Date(2018, 8, 6, 4, 3, 2),
        finished_at: new Date(2018, 8, 7, 4, 3, 2),
        experiment_id: 'some-experiment-id',
        run_id: 'test-run-id',
        display_name: 'test run',
        description: 'test run description',
        state: V2beta1RuntimeState.SUCCEEDED,
      };
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={noCreateTimeRun}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Detail'));

      expect(screen.getAllByText('-').length).toEqual(2); // create time and duration are empty.
    });

    it('handles no finish time', async () => {
      const noFinsihTimeRun: V2beta1Run = {
        created_at: new Date(2018, 8, 5, 4, 3, 2),
        scheduled_at: new Date(2018, 8, 6, 4, 3, 2),
        // finished_at: new Date(2018, 8, 7, 4, 3, 2),
        experiment_id: 'some-experiment-id',
        run_id: 'test-run-id',
        display_name: 'test run',
        description: 'test run description',
        state: V2beta1RuntimeState.SUCCEEDED,
      };
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={noFinsihTimeRun}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Detail'));

      expect(screen.getAllByText('-').length).toEqual(2); // finish time and duration are empty.
    });

    it('shows actual retry start time from state_history when RUNNING entry has update_time', async () => {
      const retryTime = new Date(2018, 8, 8, 4, 3, 2);
      const runWithHistory: V2beta1Run = {
        ...TEST_RUN,
        scheduled_at: new Date(2018, 8, 6, 4, 3, 2),
        state_history: [
          { state: V2beta1RuntimeState.RUNNING, update_time: new Date(2018, 8, 6, 4, 3, 2) },
          { state: V2beta1RuntimeState.FAILED, update_time: new Date(2018, 8, 6, 5, 0, 0) },
          { state: V2beta1RuntimeState.RUNNING, update_time: retryTime },
        ],
      };
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={runWithHistory}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Detail'));

      screen.getByText(retryTime.toLocaleString());
      screen.getByText('Scheduled at');
    });

    it('falls back to scheduled_at when RUNNING entry has no update_time', async () => {
      const scheduledTime = new Date(2018, 8, 6, 4, 3, 2);
      const runWithNoUpdateTime: V2beta1Run = {
        ...TEST_RUN,
        scheduled_at: scheduledTime,
        state_history: [{ state: V2beta1RuntimeState.RUNNING, update_time: undefined }],
      };
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={runWithNoUpdateTime}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Detail'));

      screen.getByText(scheduledTime.toLocaleString());
      expect(screen.queryByText('Scheduled at')).toBeNull();
    });

    it('does not show Scheduled at row when actual start equals scheduled_at', async () => {
      const sameTime = new Date(2018, 8, 6, 4, 3, 2);
      const runSameTime: V2beta1Run = {
        ...TEST_RUN,
        scheduled_at: sameTime,
        state_history: [{ state: V2beta1RuntimeState.RUNNING, update_time: sameTime }],
      };
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={runSameTime}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Detail'));

      expect(screen.queryByText('Scheduled at')).toBeNull();
    });

    it('shows run parameters', async () => {
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Detail'));

      screen.getByText('param1'); // 'Parameter name'
      screen.getByText('value1'); // 'Parameter value'
    });

    it('switches to Pipeline Spec tab', async () => {
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      await userEvent.click(screen.getByText('Pipeline Spec'));
      await screen.findByTestId('spec-ir');
    });

    it('shows Execution Sidepanel', async () => {
      const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
      getRunSpy.mockResolvedValue(TEST_RUN);
      const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
      getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      // Default view has no side panel.
      expect(screen.queryByText('Input/Output')).toBeNull();
      expect(screen.queryByText('Task Details')).toBeNull();

      // Select execution to open side panel.
      // Use fireEvent: user-event v14 creates events with non-configurable view, which breaks
      // d3-drag (@xyflow/react) when event.view is null in jsdom.
      fireEvent.click(screen.getByText('preprocess'));
      screen.getByText('Input/Output');
      screen.getByText('Task Details');

      // Close side panel.
      fireEvent.click(screen.getByLabelText('close'));
      expect(screen.queryByText('Input/Output')).toBeNull();
      expect(screen.queryByText('Task Details')).toBeNull();
    });

    it('shows Artifact Sidepanel', async () => {
      const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
      getRunSpy.mockResolvedValue(TEST_RUN);
      const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
      getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={v2YamlTemplateString}
            run={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );

      // Default view has no side panel.
      expect(screen.queryByText('Artifact Info')).toBeNull();
      expect(screen.queryByText('Visualization')).toBeNull();

      // Select artifact to open side panel.
      // Use fireEvent: user-event v14 creates events with non-configurable view, which breaks
      // d3-drag (@xyflow/react) when event.view is null in jsdom.
      await waitFor(() => expect(updateBannerSpy).toHaveBeenLastCalledWith({}));
      fireEvent.click(screen.getByText('model'));
      expect(screen.getAllByText('Artifact Info')).toHaveLength(2);
      screen.getByText('Visualization');

      // Close side panel.
      fireEvent.click(screen.getByLabelText('close'));
      expect(screen.queryByText('Artifact Info')).toBeNull();
      expect(screen.queryByText('Visualization')).toBeNull();
    });
  });
});
