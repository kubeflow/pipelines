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

import {
  act,
  fireEvent,
  queryByText,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';

import { V2beta1Run, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { RoutePage, RouteParams } from 'src/components/Router';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { Api } from 'src/mlmd/Api';
import { KFP_V2_RUN_CONTEXT_TYPE } from 'src/mlmd/MlmdUtils';
import { mockResizeObserver, testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import * as DynamicFlow from 'src/lib/v2/DynamicFlow';
import {
  Context,
  Execution,
  GetContextByTypeAndNameRequest,
  GetContextByTypeAndNameResponse,
  GetExecutionsByContextResponse,
  Value,
} from 'src/third_party/mlmd';
import * as metadataStoreServicePb from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
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

  let updateBannerSpy: any;
  let updateDialogSpy: any;
  let updateSnackbarSpy: any;
  let updateToolbarSpy: any;
  let historyPushSpy: any;

  function generateProps(): RunDetailsInternalProps & PageProps {
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
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

  const LOOP_PIPELINE_JOB = JSON.stringify({
    pipelineInfo: { name: 'late-loop-metadata' },
    deploymentSpec: { executors: { 'exec-train': { container: { image: 'test' } } } },
    root: {
      dag: {
        tasks: { loop: { taskInfo: { name: 'loop' }, componentRef: { name: 'comp-loop' } } },
      },
    },
    components: {
      'comp-loop': {
        dag: {
          tasks: { train: { taskInfo: { name: 'train' }, componentRef: { name: 'comp-train' } } },
        },
      },
      'comp-train': { executorLabel: 'exec-train' },
    },
  });

  function createLoopMetadata() {
    const root = new Execution().setId(1).setLastKnownState(Execution.State.RUNNING);
    root.getCustomPropertiesMap().set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue(''));
    const loop = new Execution().setId(2).setLastKnownState(Execution.State.RUNNING);
    loop
      .getCustomPropertiesMap()
      .set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue('loop'))
      .set(DynamicFlow.PARENT_DAG_ID_KEY, new Value().setIntValue(1));
    const iterations = [0, 1].map((index) => {
      const iteration = new Execution()
        .setId(3 + index)
        .setLastKnownState(index === 0 ? Execution.State.COMPLETE : Execution.State.FAILED);
      iteration
        .getCustomPropertiesMap()
        .set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue('loop'))
        .set(DynamicFlow.PARENT_DAG_ID_KEY, new Value().setIntValue(2))
        .set(DynamicFlow.ITERATION_INDEX_KEY, new Value().setIntValue(index));
      return iteration;
    });
    const leaves = iterations.map((iteration, index) => {
      const leaf = new Execution()
        .setId(5 + index)
        .setLastKnownState(iteration.getLastKnownState());
      leaf
        .getCustomPropertiesMap()
        .set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue('train'))
        .set(DynamicFlow.PARENT_DAG_ID_KEY, new Value().setIntValue(iteration.getId()))
        .set(
          'display_name',
          new Value().setStringValue(index === 0 ? 'Selected train' : 'Sibling train'),
        );
      return leaf;
    });
    return { root, loop, iterations, leaves };
  }

  beforeEach(() => {
    mockResizeObserver();

    updateBannerSpy = vi.fn();
    updateToolbarSpy = vi.fn();

    const contextResponse = new GetContextByTypeAndNameResponse();
    contextResponse.setContext(new Context());
    vi.spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName').mockResolvedValue(
      contextResponse,
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByContext').mockResolvedValue(
      new GetExecutionsByContextResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByContext').mockResolvedValue(
      new metadataStoreServicePb.GetArtifactsByContextResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs').mockResolvedValue(
      new metadataStoreServicePb.GetEventsByExecutionIDsResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactTypes').mockResolvedValue(
      new metadataStoreServicePb.GetArtifactTypesResponse(),
    );
  });

  it('opens a sub-DAG with missing task metadata and updates it after a refetch', async () => {
    const pipelineJob = JSON.stringify({
      pipelineInfo: { name: 'sibling-dags' },
      deploymentSpec: { executors: { 'exec-train': { container: { image: 'test' } } } },
      root: {
        dag: {
          tasks: {
            'dag-a': { taskInfo: { name: 'dag-a' }, componentRef: { name: 'comp-dag' } },
            'dag-b': { taskInfo: { name: 'dag-b' }, componentRef: { name: 'comp-dag' } },
          },
        },
      },
      components: {
        'comp-dag': {
          dag: {
            tasks: { train: { taskInfo: { name: 'train' }, componentRef: { name: 'comp-train' } } },
          },
        },
        'comp-train': { executorLabel: 'exec-train' },
      },
    });
    const rootExecution = new Execution().setId(1);
    rootExecution
      .getCustomPropertiesMap()
      .set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue(''));
    const dagExecution = new Execution().setId(2).setLastKnownState(Execution.State.COMPLETE);
    dagExecution
      .getCustomPropertiesMap()
      .set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue('dag-a'))
      .set(DynamicFlow.PARENT_DAG_ID_KEY, new Value().setIntValue(1));
    const siblingDagExecution = new Execution()
      .setId(3)
      .setLastKnownState(Execution.State.COMPLETE);
    siblingDagExecution
      .getCustomPropertiesMap()
      .set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue('dag-b'))
      .set(DynamicFlow.PARENT_DAG_ID_KEY, new Value().setIntValue(1));
    const siblingTaskExecution = new Execution().setId(4).setLastKnownState(Execution.State.FAILED);
    siblingTaskExecution
      .getCustomPropertiesMap()
      .set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue('train'))
      .set(DynamicFlow.PARENT_DAG_ID_KEY, new Value().setIntValue(3));
    const getExecutionsSpy = vi.spyOn(
      Api.getInstance().metadataStoreService,
      'getExecutionsByContext',
    );
    getExecutionsSpy.mockResolvedValue(
      new GetExecutionsByContextResponse().setExecutionsList([
        rootExecution,
        dagExecution,
        siblingDagExecution,
        siblingTaskExecution,
      ]),
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <RunDetailsV2 pipeline_job={pipelineJob} run={TEST_RUN} {...generateProps()} />
        </QueryClientProvider>
      </BrowserRouter>,
    );

    const dagNode = screen.getByTitle('dag-a');
    await within(dagNode).findByTestId('CheckCircleIcon');
    fireEvent.click(within(dagNode).getByTestId('expand-button'));

    expect(screen.getByText('train')).toBeInTheDocument();
    expect(screen.queryByTestId('ErrorIcon')).not.toBeInTheDocument();
    expect(screen.queryByTestId('CheckCircleIcon')).not.toBeInTheDocument();
    expect(screen.getByTestId('DagCanvas')).toBeInTheDocument();

    const taskExecution = new Execution().setId(5).setLastKnownState(Execution.State.COMPLETE);
    taskExecution
      .getCustomPropertiesMap()
      .set(DynamicFlow.TASK_NAME_KEY, new Value().setStringValue('train'))
      .set(DynamicFlow.PARENT_DAG_ID_KEY, new Value().setIntValue(2))
      .set('display_name', new Value().setStringValue('Train model'));
    getExecutionsSpy.mockResolvedValue(
      new GetExecutionsByContextResponse().setExecutionsList([
        rootExecution,
        dagExecution,
        siblingDagExecution,
        siblingTaskExecution,
        taskExecution,
      ]),
    );
    await act(async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.mlmdPackage(RUN_ID) });
    });

    expect(await screen.findByText('Train model')).toBeInTheDocument();
    expect(screen.getByTestId('CheckCircleIcon')).toBeInTheDocument();
    expect(screen.queryByTestId('ErrorIcon')).not.toBeInTheDocument();
    expect(screen.queryByTitle('dag-a')).not.toBeInTheDocument();
  });

  it.each(['missing loop execution', 'missing iteration count'])(
    'recovers the open loop after a refetch with %s',
    async (missingMetadata) => {
      const { root, loop, iterations, leaves } = createLoopMetadata();
      const initialExecutions =
        missingMetadata === 'missing loop execution' ? [root] : [root, loop];
      const executionsSpy = vi.mocked(
        Api.getInstance().metadataStoreService.getExecutionsByContext,
      );
      executionsSpy.mockResolvedValue(
        new GetExecutionsByContextResponse().setExecutionsList(initialExecutions),
      );
      const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      render(
        <BrowserRouter>
          <QueryClientProvider client={queryClient}>
            <RunDetailsV2 pipeline_job={LOOP_PIPELINE_JOB} run={TEST_RUN} {...generateProps()} />
          </QueryClientProvider>
        </BrowserRouter>,
      );
      await waitFor(() =>
        expect(queryClient.getQueryData(queryKeys.mlmdPackage(RUN_ID))).toMatchObject({
          executions: initialExecutions,
        }),
      );
      fireEvent.click(within(screen.getByTitle('loop')).getByTestId('expand-button'));
      expect(screen.getByText('train')).toBeInTheDocument();
      const canvas = screen.getByTestId('DagCanvas');

      const hydratedLoop = loop.clone();
      hydratedLoop
        .getCustomPropertiesMap()
        .set(DynamicFlow.ITERATION_COUNT_KEY, new Value().setIntValue(2));
      const recoveredExecutions = [root, hydratedLoop, ...iterations, ...leaves];
      executionsSpy.mockResolvedValue(
        new GetExecutionsByContextResponse().setExecutionsList(recoveredExecutions),
      );
      await act(async () =>
        queryClient.invalidateQueries({ queryKey: queryKeys.mlmdPackage(RUN_ID), exact: true }),
      );
      expect(executionsSpy).toHaveBeenCalledTimes(2);
      expect(queryClient.getQueryData(queryKeys.mlmdPackage(RUN_ID))).toMatchObject({
        executions: recoveredExecutions,
      });
      const selectedIteration = await screen.findByTitle('loop.0');
      expect(within(selectedIteration).getByTestId('CheckCircleIcon')).toBeInTheDocument();
      expect(within(screen.getByTitle('loop.1')).getByTestId('ErrorIcon')).toBeInTheDocument();
      expect(screen.queryByText('train')).not.toBeInTheDocument();
      expect(screen.queryByTitle('loop')).not.toBeInTheDocument();
      expect(screen.getByTestId('DagCanvas')).toBe(canvas);

      fireEvent.click(within(selectedIteration).getByTestId('expand-button'));
      expect(await screen.findByText('Selected train')).toBeInTheDocument();
      expect(screen.getByTestId('CheckCircleIcon')).toBeInTheDocument();
      expect(screen.queryByTestId('ErrorIcon')).not.toBeInTheDocument();
    },
  );

  it('recovers selected iteration ancestry without borrowing the sibling leaf state', async () => {
    const { root, loop, iterations, leaves } = createLoopMetadata();
    loop.getCustomPropertiesMap().set(DynamicFlow.ITERATION_COUNT_KEY, new Value().setIntValue(2));
    const initialExecutions = [root, loop, iterations[1], ...leaves];
    const executionsSpy = vi.mocked(Api.getInstance().metadataStoreService.getExecutionsByContext);
    executionsSpy.mockResolvedValue(
      new GetExecutionsByContextResponse().setExecutionsList(initialExecutions),
    );
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <RunDetailsV2 pipeline_job={LOOP_PIPELINE_JOB} run={TEST_RUN} {...generateProps()} />
        </QueryClientProvider>
      </BrowserRouter>,
    );
    await within(screen.getByTitle('loop')).findByTestId('RefreshIcon');
    fireEvent.click(within(screen.getByTitle('loop')).getByTestId('expand-button'));
    const selectedIteration = screen.getByTitle('loop.0');
    expect(within(selectedIteration).queryByTestId('CheckCircleIcon')).not.toBeInTheDocument();
    expect(within(selectedIteration).queryByTestId('ErrorIcon')).not.toBeInTheDocument();
    fireEvent.click(within(selectedIteration).getByTestId('expand-button'));
    expect(screen.getByText('train')).toBeInTheDocument();
    expect(screen.queryByTestId('CheckCircleIcon')).not.toBeInTheDocument();
    expect(screen.queryByTestId('ErrorIcon')).not.toBeInTheDocument();
    const canvas = screen.getByTestId('DagCanvas');

    const recoveredExecutions = [root, loop, ...iterations, ...leaves];
    executionsSpy.mockResolvedValue(
      new GetExecutionsByContextResponse().setExecutionsList(recoveredExecutions),
    );
    await act(async () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.mlmdPackage(RUN_ID), exact: true }),
    );
    expect(executionsSpy).toHaveBeenCalledTimes(2);
    expect(await screen.findByText('Selected train')).toBeInTheDocument();
    expect(screen.getByTestId('CheckCircleIcon')).toBeInTheDocument();
    expect(screen.queryByTestId('ErrorIcon')).not.toBeInTheDocument();
    expect(screen.queryByText('Sibling train')).not.toBeInTheDocument();
    expect(screen.queryByTitle('loop.0')).not.toBeInTheDocument();
    expect(screen.getByTestId('DagCanvas')).toBe(canvas);
  });

  it.each(
    [Execution.State.COMPLETE, Execution.State.FAILED].flatMap((state) =>
      [
        'leaf execution',
        'leaf ancestor',
        'iteration execution',
        'loop execution',
        'iteration count',
      ].map((omission) => ({ state, omission })),
    ),
  )(
    'restores the open graph after omitting $omission with state $state',
    async ({ state, omission }) => {
      const { root, loop, iterations, leaves } = createLoopMetadata();
      const selectedIndex = state === Execution.State.COMPLETE ? 0 : 1;
      loop
        .getCustomPropertiesMap()
        .set(DynamicFlow.ITERATION_COUNT_KEY, new Value().setIntValue(2));
      const completeExecutions = [root, loop, ...iterations, ...leaves];
      const executionsSpy = vi.mocked(
        Api.getInstance().metadataStoreService.getExecutionsByContext,
      );
      executionsSpy.mockResolvedValue(
        new GetExecutionsByContextResponse().setExecutionsList(completeExecutions),
      );
      const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      render(
        <BrowserRouter>
          <QueryClientProvider client={queryClient}>
            <RunDetailsV2 pipeline_job={LOOP_PIPELINE_JOB} run={TEST_RUN} {...generateProps()} />
          </QueryClientProvider>
        </BrowserRouter>,
      );
      await within(screen.getByTitle('loop')).findByTestId('RefreshIcon');
      fireEvent.click(within(screen.getByTitle('loop')).getByTestId('expand-button'));
      const iterationTitle = `loop.${selectedIndex}`;
      const expectedIcon = state === Execution.State.COMPLETE ? 'CheckCircleIcon' : 'ErrorIcon';
      const otherIcon = state === Execution.State.COMPLETE ? 'ErrorIcon' : 'CheckCircleIcon';
      const leafTitle = selectedIndex === 0 ? 'Selected train' : 'Sibling train';
      await within(screen.getByTitle(iterationTitle)).findByTestId(expectedIcon);
      const isLeafLayer = omission.startsWith('leaf');
      if (isLeafLayer) {
        fireEvent.click(within(screen.getByTitle(iterationTitle)).getByTestId('expand-button'));
        await within(screen.getByTitle(leafTitle)).findByTestId(expectedIcon);
      }
      const nodeTitle = isLeafLayer ? 'train' : iterationTitle;
      const layerTitle = isLeafLayer ? iterationTitle : 'loop';
      const canvas = screen.getByTestId('DagCanvas');
      const loopWithoutCount = loop.clone();
      loopWithoutCount.getCustomPropertiesMap().del(DynamicFlow.ITERATION_COUNT_KEY);
      const omittedId =
        omission === 'leaf execution'
          ? leaves[selectedIndex].getId()
          : omission === 'leaf ancestor' || omission === 'iteration execution'
            ? iterations[selectedIndex].getId()
            : omission === 'loop execution'
              ? loop.getId()
              : undefined;
      const partialExecutions = completeExecutions
        .filter((execution) => execution.getId() !== omittedId)
        .map((execution) =>
          omission === 'iteration count' && execution.getId() === loop.getId()
            ? loopWithoutCount
            : execution,
        );
      executionsSpy.mockResolvedValue(
        new GetExecutionsByContextResponse().setExecutionsList(partialExecutions),
      );
      await act(async () =>
        queryClient.invalidateQueries({ queryKey: queryKeys.mlmdPackage(RUN_ID), exact: true }),
      );
      expect(executionsSpy).toHaveBeenCalledTimes(2);
      expect(queryClient.getQueryData(queryKeys.mlmdPackage(RUN_ID))).toMatchObject({
        executions: partialExecutions,
      });
      await waitFor(() => {
        const unavailableNode = screen.getByTitle(nodeTitle);
        expect(within(unavailableNode).queryByTestId(expectedIcon)).not.toBeInTheDocument();
        expect(within(unavailableNode).queryByTestId(otherIcon)).not.toBeInTheDocument();
      });
      expect(screen.getByRole('button', { name: layerTitle })).toBeDisabled();
      expect(screen.getByTestId('DagCanvas')).toBe(canvas);
      expect(updateBannerSpy).not.toHaveBeenCalledWith(expect.objectContaining({ mode: 'error' }));

      executionsSpy.mockResolvedValue(
        new GetExecutionsByContextResponse().setExecutionsList(completeExecutions),
      );
      await act(async () =>
        queryClient.invalidateQueries({ queryKey: queryKeys.mlmdPackage(RUN_ID), exact: true }),
      );
      expect(executionsSpy).toHaveBeenCalledTimes(3);
      await waitFor(() => {
        const restoredNode = screen.getByTitle(isLeafLayer ? leafTitle : iterationTitle);
        expect(within(restoredNode).getByTestId(expectedIcon)).toBeInTheDocument();
        expect(within(restoredNode).queryByTestId(otherIcon)).not.toBeInTheDocument();
      });
      expect(screen.getByRole('button', { name: layerTitle })).toBeDisabled();
      expect(screen.getByTestId('DagCanvas')).toBe(canvas);
      expect(canvas.querySelectorAll('.react-flow__node')).toHaveLength(isLeafLayer ? 1 : 2);
      if (isLeafLayer) {
        expect(
          screen.getByText(selectedIndex === 0 ? 'Selected train' : 'Sibling train'),
        ).toBeInTheDocument();
        expect(
          screen.queryByText(selectedIndex === 0 ? 'Sibling train' : 'Selected train'),
        ).not.toBeInTheDocument();
      }
    },
  );

  it('reconciles an open loop to an empty graph for a valid zero iteration count', async () => {
    const { root, loop } = createLoopMetadata();
    const executionsSpy = vi.mocked(Api.getInstance().metadataStoreService.getExecutionsByContext);
    executionsSpy.mockResolvedValue(
      new GetExecutionsByContextResponse().setExecutionsList([root, loop]),
    );
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <RunDetailsV2 pipeline_job={LOOP_PIPELINE_JOB} run={TEST_RUN} {...generateProps()} />
        </QueryClientProvider>
      </BrowserRouter>,
    );
    await within(screen.getByTitle('loop')).findByTestId('RefreshIcon');
    fireEvent.click(within(screen.getByTitle('loop')).getByTestId('expand-button'));
    expect(screen.getByText('train')).toBeInTheDocument();
    const canvas = screen.getByTestId('DagCanvas');
    const zeroLoop = loop.clone();
    zeroLoop
      .getCustomPropertiesMap()
      .set(DynamicFlow.ITERATION_COUNT_KEY, new Value().setIntValue(0));
    executionsSpy.mockResolvedValue(
      new GetExecutionsByContextResponse().setExecutionsList([root, zeroLoop]),
    );
    await act(async () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.mlmdPackage(RUN_ID), exact: true }),
    );
    await waitFor(() => expect(screen.queryByText('train')).not.toBeInTheDocument());
    expect(canvas.querySelectorAll('.react-flow__node')).toHaveLength(0);
    expect(screen.getByTestId('DagCanvas')).toBe(canvas);
    expect(screen.getByRole('button', { name: 'loop' })).toBeDisabled();
  });

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

  it('Shows error banner when disconnected from MLMD', async () => {
    vi.spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName').mockRejectedValue(
      new Error('Not connected to MLMD'),
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
          additionalInfo:
            'Cannot find context with {"typeName":"system.PipelineRun","contextName":"1"}: Not connected to MLMD',
          message: 'Cannot get MLMD objects from Metadata store.',
          mode: 'error',
        }),
      ),
    );
  });

  it('Shows experiment warning banner when experiment fetch fails and MLMD succeeds', async () => {
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

  it('Shows MLMD error banner even when experiment also fails (MLMD takes precedence)', async () => {
    vi.spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName').mockRejectedValue(
      new Error('Not connected to MLMD'),
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
          message: 'Cannot get MLMD objects from Metadata store.',
          mode: 'error',
        }),
      ),
    );
  });

  it('Does not clear experiment warning when MLMD succeeds after experiment fails', async () => {
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
    // NOT a clear ({}) from the MLMD success path.
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          message: 'Error: failed to retrieve experiment details.',
          mode: 'warning',
        }),
      ),
    );
  });

  it('Shows no banner when connected from MLMD', async () => {
    vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment').mockResolvedValue(TEST_EXPERIMENT);
    vi.spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName').mockImplementation(
      (request: GetContextByTypeAndNameRequest) => {
        const response = new GetContextByTypeAndNameResponse();
        if (
          request.getTypeName() === KFP_V2_RUN_CONTEXT_TYPE &&
          request.getContextName() === RUN_ID
        ) {
          response.setContext(new Context());
        }
        return response;
      },
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByContext').mockResolvedValue(
      new GetExecutionsByContextResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByContext').mockResolvedValue(
      new metadataStoreServicePb.GetArtifactsByContextResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs').mockResolvedValue(
      new metadataStoreServicePb.GetEventsByExecutionIDsResponse(),
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

    await waitFor(() => expect(updateBannerSpy).toHaveBeenLastCalledWith({}));
  });

  it("shows run title and experiments' links", async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    getRunSpy.mockResolvedValue(TEST_RUN);
    const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
    getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

    vi.spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName').mockImplementation(
      (request: GetContextByTypeAndNameRequest) => {
        const response = new GetContextByTypeAndNameResponse();
        response.setContext(new Context());
        return response;
      },
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByContext').mockResolvedValue(
      new GetExecutionsByContextResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByContext').mockResolvedValue(
      new metadataStoreServicePb.GetArtifactsByContextResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs').mockResolvedValue(
      new metadataStoreServicePb.GetEventsByExecutionIDsResponse(),
    );

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

    vi.spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName').mockImplementation(
      () => {
        const response = new GetContextByTypeAndNameResponse();
        response.setContext(new Context());
        return response;
      },
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByContext').mockResolvedValue(
      new GetExecutionsByContextResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByContext').mockResolvedValue(
      new metadataStoreServicePb.GetArtifactsByContextResponse(),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs').mockResolvedValue(
      new metadataStoreServicePb.GetEventsByExecutionIDsResponse(),
    );

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
      fireEvent.click(screen.getByText('model'));
      screen.getByText('Artifact Info');
      screen.getByText('Visualization');

      // Close side panel.
      fireEvent.click(screen.getByLabelText('close'));
      expect(screen.queryByText('Artifact Info')).toBeNull();
      expect(screen.queryByText('Visualization')).toBeNull();
    });
  });
});
