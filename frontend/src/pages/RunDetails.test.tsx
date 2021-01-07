/*
 * Copyright 2018-2019 Google LLC
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
import { Api, GetArtifactTypesResponse } from '@kubeflow/frontend';
import { render } from '@testing-library/react';
import * as dagre from 'dagre';
import { mount, ReactWrapper, shallow, ShallowWrapper } from 'enzyme';
import { createMemoryHistory } from 'history';
import * as React from 'react';
import { Router } from 'react-router-dom';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { Workflow } from 'third_party/argo-ui/argo_template';
import { ApiResourceType, ApiRunDetail, RunStorageState } from '../apis/run';
import { QUERY_PARAMS, RoutePage, RouteParams } from '../components/Router';
import { PlotType } from '../components/viewers/Viewer';
import { Apis, JSONObject } from '../lib/Apis';
import { ButtonKeys } from '../lib/Buttons';
import * as MlmdUtils from '../lib/MlmdUtils';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { NodePhase } from '../lib/StatusUtils';
import * as Utils from '../lib/Utils';
import WorkflowParser from '../lib/WorkflowParser';
import TestUtils from '../TestUtils';
import { PageProps } from './Page';
import EnhancedRunDetails, { RunDetailsInternalProps, TEST_ONLY } from './RunDetails';
import { TFunction } from 'i18next';

const RunDetails = TEST_ONLY.RunDetails;

jest.mock('../components/Graph', () => {
  return function GraphMock({ graph }: { graph: dagre.graphlib.Graph }) {
    return (
      <pre data-testid='graph'>
        {graph
          .nodes()
          .map(v => 'Node ' + v)
          .join('\n  ')}
        {graph
          .edges()
          .map(e => `Edge ${e.v} to ${e.w}`)
          .join('\n  ')}
      </pre>
    );
  };
});

const STEP_TABS = {
  INPUT_OUTPUT: 0,
  VISUALIZATIONS: 1,
  ML_METADATA: 2,
  VOLUMES: 3,
  LOGS: 4,
  POD: 5,
  EVENTS: 6,
  MANIFEST: 7,
};

const WORKFLOW_TEMPLATE = {
  metadata: {
    name: 'workflow1',
  },
};

const NODE_DETAILS_SELECTOR = '[data-testid="run-details-node-details"]';

describe('RunDetails', () => {
  let updateBannerSpy: any;
  let updateDialogSpy: any;
  let updateSnackbarSpy: any;
  let updateToolbarSpy: any;
  let historyPushSpy: any;
  let getRunSpy: any;
  let getExperimentSpy: any;
  let isCustomVisualizationsAllowedSpy: any;
  let getPodLogsSpy: any;
  let getPodInfoSpy: any;
  let pathsParser: any;
  let pathsWithStepsParser: any;
  let loaderSpy: any;
  let retryRunSpy: any;
  let terminateRunSpy: any;
  let artifactTypesSpy: any;
  let formatDateStringSpy: any;
  let getTfxRunContextSpy: any;
  let getKfpRunContextSpy: any;
  let warnSpy: any;

  let testRun: ApiRunDetail = {};
  let tree: ShallowWrapper | ReactWrapper;
  let identiT: TFunction = (key: string) => key;
  function generateProps(): RunDetailsInternalProps & PageProps {
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: { params: { [RouteParams.runId]: testRun.run!.id }, isExact: true, path: '', url: '' },
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
      t: identiT,
    };
    return Object.assign(pageProps, {
      toolbarProps: new RunDetails(pageProps).getInitialToolbarState(),
      gkeMetadata: {},
    });
  }

  beforeEach(() => {
    // The RunDetails page uses timers to periodically refresh
    jest.useFakeTimers();
    // TODO: mute error only for tests that are expected to have error
    jest.spyOn(console, 'error').mockImplementation(() => null);

    testRun = {
      pipeline_runtime: {
        workflow_manifest: '{}',
      },
      run: {
        created_at: new Date(2018, 8, 5, 4, 3, 2),
        description: 'test run description',
        id: 'test-run-id',
        name: 'test run',
        pipeline_spec: {
          parameters: [{ name: 'param1', value: 'value1' }],
          pipeline_id: 'some-pipeline-id',
        },
        status: 'Succeeded',
      },
    };
    updateBannerSpy = jest.fn();
    updateDialogSpy = jest.fn();
    updateSnackbarSpy = jest.fn();
    updateToolbarSpy = jest.fn();
    historyPushSpy = jest.fn();
    getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
    isCustomVisualizationsAllowedSpy = jest.spyOn(Apis, 'areCustomVisualizationsAllowed');
    getPodLogsSpy = jest.spyOn(Apis, 'getPodLogs');
    getPodInfoSpy = jest.spyOn(Apis, 'getPodInfo');
    pathsParser = jest.spyOn(WorkflowParser, 'loadNodeOutputPaths');
    pathsWithStepsParser = jest.spyOn(WorkflowParser, 'loadAllOutputPathsWithStepNames');
    loaderSpy = jest.spyOn(OutputArtifactLoader, 'load');
    retryRunSpy = jest.spyOn(Apis.runServiceApi, 'retryRun');
    terminateRunSpy = jest.spyOn(Apis.runServiceApi, 'terminateRun');
    artifactTypesSpy = jest.spyOn(Api.getInstance().metadataStoreService, 'getArtifactTypes');
    // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
    // test environments
    formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');
    getTfxRunContextSpy = jest.spyOn(MlmdUtils, 'getTfxRunContext').mockImplementation(() => {
      throw new Error('cannot find tfx run context');
    });
    getKfpRunContextSpy = jest.spyOn(MlmdUtils, 'getKfpRunContext').mockImplementation(() => {
      throw new Error('cannot find kfp run context');
    });
    // Hide expected warning messages
    warnSpy = jest.spyOn(Utils.logger, 'warn').mockImplementation();

    getRunSpy.mockImplementation(() => Promise.resolve(testRun));
    getExperimentSpy.mockImplementation(() =>
      Promise.resolve({ id: 'some-experiment-id', name: 'some experiment' }),
    );
    isCustomVisualizationsAllowedSpy.mockImplementation(() => Promise.resolve(false));
    getPodLogsSpy.mockImplementation(() => 'test logs');
    getPodInfoSpy.mockImplementation(() => ({ data: 'some data' } as JSONObject));
    pathsParser.mockImplementation(() => []);
    pathsWithStepsParser.mockImplementation(() => []);
    loaderSpy.mockImplementation(() => Promise.resolve([]));
    formatDateStringSpy.mockImplementation(() => '1/2/2019, 12:34:56 PM');
    artifactTypesSpy.mockImplementation(() => {
      // TODO: This is temporary workaround to let tfx artifact resolving logic fail early.
      // We should add proper testing for those cases later too.
      const response = new GetArtifactTypesResponse();
      response.setArtifactTypesList([]);
      return response;
    });
  });

  afterEach(async () => {
    if (tree && tree.exists()) {
      // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
      // depends on mocks/spies
      await tree.unmount();
    }
    jest.resetAllMocks();
    jest.restoreAllMocks();
  });

  it('shows success run status in page title', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const lastCall = updateToolbarSpy.mock.calls[2][0];
    expect(lastCall.pageTitle).toMatchSnapshot();
  });

  it('shows failure run status in page title', async () => {
    testRun.run!.status = 'Failed';
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const lastCall = updateToolbarSpy.mock.calls[2][0];
    expect(lastCall.pageTitle).toMatchSnapshot();
  });

  it('has a clone button, clicking it navigates to new run page', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const cloneBtn = instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RUN];
    expect(cloneBtn).toBeDefined();
    await cloneBtn!.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.cloneFromRun}=${testRun.run!.id}`,
    );
  });

  it('clicking the clone button when the page is half-loaded navigates to new run page with run id', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    // Intentionally don't wait until all network requests finish.
    const instance = tree.instance() as RunDetails;
    const cloneBtn = instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RUN];
    expect(cloneBtn).toBeDefined();
    await cloneBtn!.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.cloneFromRun}=${testRun.run!.id}`,
    );
  });

  it('has a retry button', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const retryBtn = instance.getInitialToolbarState().actions[ButtonKeys.RETRY];
    expect(retryBtn).toBeDefined();
  });

  it('shows retry confirmation dialog when retry button is clicked', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    const instance = tree.instance() as RunDetails;
    const retryBtn = instance.getInitialToolbarState().actions[ButtonKeys.RETRY];
    await retryBtn!.action();
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        title: 'common:retry common:this common:run?',
      }),
    );
  });

  it('does not call retry API for selected run when retry dialog is canceled', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    const instance = tree.instance() as RunDetails;
    const retryBtn = instance.getInitialToolbarState().actions[ButtonKeys.RETRY];
    await retryBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((b: any) => b.text === 'common:cancel');
    await cancelBtn.onClick();
    expect(retryRunSpy).not.toHaveBeenCalled();
  });

  it('calls retry API when retry dialog is confirmed', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const retryBtn = instance.getInitialToolbarState().actions[ButtonKeys.RETRY];
    await retryBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:retry');
    await confirmBtn.onClick();
    expect(retryRunSpy).toHaveBeenCalledTimes(1);
    expect(retryRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
  });

  it('calls retry API when retry dialog is confirmed and page is half-loaded', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    // Intentionally don't wait until all network requests finish.
    const instance = tree.instance() as RunDetails;
    const retryBtn = instance.getInitialToolbarState().actions[ButtonKeys.RETRY];
    await retryBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:retry');
    await confirmBtn.onClick();
    expect(retryRunSpy).toHaveBeenCalledTimes(1);
    expect(retryRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
  });

  it('shows an error dialog when retry API fails', async () => {
    retryRunSpy.mockImplementation(() => Promise.reject('mocked error'));

    tree = mount(<RunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const retryBtn = instance.getInitialToolbarState().actions[ButtonKeys.RETRY];
    await retryBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:retry');
    await confirmBtn.onClick();
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content:
          'common:failedTo common:retry common:run: test-run-id common:withError: ""mocked error""',
      }),
    );
    // There shouldn't be a snackbar on error.
    expect(updateSnackbarSpy).not.toHaveBeenCalled();
  });

  it('has a terminate button', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const terminateBtn = instance.getInitialToolbarState().actions[ButtonKeys.TERMINATE_RUN];
    expect(terminateBtn).toBeDefined();
  });

  it('shows terminate confirmation dialog when terminate button is clicked', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    const instance = tree.instance() as RunDetails;
    const terminateBtn = instance.getInitialToolbarState().actions[ButtonKeys.TERMINATE_RUN];
    await terminateBtn!.action();
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        title: 'common:terminate common:this common:run?',
      }),
    );
  });

  it('does not call terminate API for selected run when terminate dialog is canceled', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    const instance = tree.instance() as RunDetails;
    const terminateBtn = instance.getInitialToolbarState().actions[ButtonKeys.TERMINATE_RUN];
    await terminateBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((b: any) => b.text === 'common:cancel');
    await cancelBtn.onClick();
    expect(terminateRunSpy).not.toHaveBeenCalled();
  });

  it('calls terminate API when terminate dialog is confirmed', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const terminateBtn = instance.getInitialToolbarState().actions[ButtonKeys.TERMINATE_RUN];
    await terminateBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:terminate');
    await confirmBtn.onClick();
    expect(terminateRunSpy).toHaveBeenCalledTimes(1);
    expect(terminateRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
  });

  it('calls terminate API when terminate dialog is confirmed and page is half-loaded', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    // Intentionally don't wait until all network requests finish.
    const instance = tree.instance() as RunDetails;
    const terminateBtn = instance.getInitialToolbarState().actions[ButtonKeys.TERMINATE_RUN];
    await terminateBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:terminate');
    await confirmBtn.onClick();
    expect(terminateRunSpy).toHaveBeenCalledTimes(1);
    expect(terminateRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
  });

  it('has an Archive button if the run is not archived', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ARCHIVE)).toBeDefined();
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE)).toBeUndefined();
  });

  it('shows "All runs" in breadcrumbs if the run is not archived', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'allRuns', href: RoutePage.RUNS }],
      }),
    );
  });

  it('shows experiment name in breadcrumbs if the run is not archived', async () => {
    testRun.run!.resource_references = [
      { key: { id: 'some-experiment-id', type: ApiResourceType.EXPERIMENT } },
    ];
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [
          { displayName: 'common:experiments', href: '/experiments' },
          {
            displayName: 'some experiment',
            href: RoutePage.EXPERIMENT_DETAILS.replace(
              ':' + RouteParams.experimentId,
              'some-experiment-id',
            ),
          },
        ],
      }),
    );
  });

  it('has a Restore button if the run is archived', async () => {
    testRun.run!.storage_state = RunStorageState.ARCHIVED;
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE)).toBeDefined();
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ARCHIVE)).toBeUndefined();
  });

  it('shows Archive in breadcrumbs if the run is archived', async () => {
    testRun.run!.storage_state = RunStorageState.ARCHIVED;
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'common:archive', href: RoutePage.ARCHIVED_RUNS }],
      }),
    );
  });

  it('renders an empty run', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('calls the get run API once to load it', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    expect(getRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
  });

  it('shows an error banner if get run API fails', async () => {
    TestUtils.makeErrorResponseOnce(getRunSpy, 'woops');
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once initially to clear
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'errorRetrieveRun: test-run-id. common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('shows an error banner if get experiment API fails', async () => {
    testRun.run!.resource_references = [
      { key: { id: 'experiment1', type: ApiResourceType.EXPERIMENT } },
    ];
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'woops');
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once initially to clear
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'errorRetrieveRun: test-run-id. common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('calls the get experiment API once to load it if the run has its reference', async () => {
    testRun.run!.resource_references = [
      { key: { id: 'experiment1', type: ApiResourceType.EXPERIMENT } },
    ];
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    expect(getRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
    expect(getExperimentSpy).toHaveBeenCalledTimes(1);
    expect(getExperimentSpy).toHaveBeenLastCalledWith('experiment1');
  });

  it('shows workflow errors as page error', async () => {
    jest
      .spyOn(WorkflowParser, 'getWorkflowError')
      .mockImplementationOnce(() => 'some error message');
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear on init, once for error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'some error message',
        message: 'errorErrorsFoundRun: test-run-id. common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('switches to run output tab, shows empty message', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 1);
    expect(tree.state('selectedTab')).toBe(1);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it("loads the run's outputs in the output tab", async () => {
    pathsWithStepsParser.mockImplementation(() => [
      { stepName: 'step1', path: { source: 'gcs', bucket: 'somebucket', key: 'somekey' } },
    ]);
    pathsParser.mockImplementation(() => [{ source: 'gcs', bucket: 'somebucket', key: 'somekey' }]);
    loaderSpy.mockImplementation(() =>
      Promise.resolve([{ type: PlotType.TENSORBOARD, url: 'some url' }]),
    );
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 1);
    expect(tree.state('selectedTab')).toBe(1);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('switches to config tab', async () => {
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 2);
    expect(tree.state('selectedTab')).toBe(2);
    expect(tree).toMatchSnapshot();
  });

  it('shows run config fields', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      metadata: {
        creationTimestamp: new Date(2018, 6, 5, 4, 3, 2).toISOString(),
      },
      spec: {
        arguments: {
          parameters: [
            {
              name: 'param1',
              value: 'value1',
            },
            {
              name: 'param2',
              value: 'value2',
            },
          ],
        },
      },
      status: {
        finishedAt: new Date(2018, 6, 6, 5, 4, 3).toISOString(),
        phase: 'Skipped',
        startedAt: new Date(2018, 6, 5, 4, 3, 2).toISOString(),
      },
    } as Workflow);
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 2);
    expect(tree.state('selectedTab')).toBe(2);
    expect(tree).toMatchSnapshot();
  });

  it('shows run config fields - handles no description', async () => {
    delete testRun.run!.description;
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      metadata: {
        creationTimestamp: new Date(2018, 6, 5, 4, 3, 2).toISOString(),
      },
      status: {
        finishedAt: new Date(2018, 6, 6, 5, 4, 3).toISOString(),
        phase: 'Skipped',
        startedAt: new Date(2018, 6, 5, 4, 3, 2).toISOString(),
      },
    } as Workflow);
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 2);
    expect(tree.state('selectedTab')).toBe(2);
    expect(tree).toMatchSnapshot();
  });

  it('shows run config fields - handles no metadata', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        finishedAt: new Date(2018, 6, 6, 5, 4, 3).toISOString(),
        phase: 'Skipped',
        startedAt: new Date(2018, 6, 5, 4, 3, 2).toISOString(),
      },
    } as Workflow);
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 2);
    expect(tree.state('selectedTab')).toBe(2);
    expect(tree).toMatchSnapshot();
  });

  it('shows a one-node graph', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      ...WORKFLOW_TEMPLATE,
      status: { nodes: { node1: { id: 'node1' } } },
    });
    const { getByTestId } = render(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(getByTestId('graph')).toMatchInlineSnapshot(`
      <pre
        data-testid="graph"
      >
        Node node1
        Node node1-running-placeholder
        Edge node1 to node1-running-placeholder
      </pre>
    `);
  });

  it('shows a one-node compressed workflow graph', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      ...WORKFLOW_TEMPLATE,
      status: { compressedNodes: 'H4sIAAAAAAACE6tWystPSTVUslKoVspMAVJQfm0tAEBEv1kaAAAA' },
    });

    const { getByTestId } = render(<RunDetails {...generateProps()} />);

    await getRunSpy;
    await TestUtils.flushPromises();

    jest.useRealTimers();
    await new Promise(resolve => setTimeout(resolve, 500));
    jest.useFakeTimers();

    expect(getByTestId('graph')).toMatchInlineSnapshot(`
      <pre
        data-testid="graph"
      >
        Node node1
        Node node1-running-placeholder
        Edge node1 to node1-running-placeholder
      </pre>
    `);
  });

  it('shows a empty workflow graph if compressedNodes corrupt', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      ...WORKFLOW_TEMPLATE,
      status: { compressedNodes: 'Y29ycnVwdF9kYXRh' },
    });

    const { queryAllByTestId } = render(<RunDetails {...generateProps()} />);

    await getRunSpy;
    await TestUtils.flushPromises();

    jest.useRealTimers();
    await new Promise(resolve => setTimeout(resolve, 500));
    jest.useFakeTimers();

    expect(queryAllByTestId('graph')).toEqual([]);
  });

  it('opens side panel when graph node is clicked', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree).toMatchSnapshot();
  });

  it('shows clicked node message in side panel', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        nodes: {
          node1: {
            id: 'node1',
            message: 'some test message',
            phase: 'Succeeded',
          },
        },
      },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    expect(tree.state('selectedNodeDetails')).toHaveProperty(
      'phaseMessage',
      'stepStateMessage1 Succeeded stepStateMessage2: some test message',
    );
    expect(tree.find('Banner')).toMatchInlineSnapshot(`null`);
  });

  it('shows clicked node output in side pane', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    pathsWithStepsParser.mockImplementation(() => [
      { stepName: 'step1', path: { source: 'gcs', bucket: 'somebucket', key: 'somekey' } },
    ]);
    pathsParser.mockImplementation(() => [{ source: 'gcs', bucket: 'somebucket', key: 'somekey' }]);
    loaderSpy.mockImplementation(() =>
      Promise.resolve([{ type: PlotType.TENSORBOARD, url: 'some url' }]),
    );
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    await pathsParser;
    await pathsWithStepsParser;
    await loaderSpy;
    await artifactTypesSpy;
    await TestUtils.flushPromises();

    // TODO: fix this test and write additional tests for the ArtifactTabContent component.
    // expect(pathsWithStepsParser).toHaveBeenCalledTimes(1); // Loading output list
    // expect(pathsParser).toHaveBeenCalledTimes(1);
    // expect(pathsParser).toHaveBeenLastCalledWith({ id: 'node1' });
    // expect(loaderSpy).toHaveBeenCalledTimes(2);
    // expect(loaderSpy).toHaveBeenLastCalledWith({
    //   bucket: 'somebucket',
    //   key: 'somekey',
    //   source: 'gcs',
    // });
    // expect(tree.state('selectedNodeDetails')).toMatchObject({ id: 'node1' });
    // expect(tree).toMatchSnapshot();
  });

  it('switches to inputs/outputs tab in side pane', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        nodes: {
          node1: {
            id: 'node1',
            inputs: {
              parameters: [{ name: 'input1', value: 'val1' }],
            },
            name: 'node1',
            outputs: {
              parameters: [
                { name: 'output1', value: 'val1' },
                { name: 'output2', value: 'value2' },
              ],
            },
            phase: 'Succeeded',
          },
        },
      },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    tree
      .find('MD2Tabs')
      .at(1)
      .simulate('switch', STEP_TABS.INPUT_OUTPUT);
    await TestUtils.flushPromises();
    expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.INPUT_OUTPUT);
    expect(tree).toMatchSnapshot();
  });

  it('switches to volumes tab in side pane', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    tree
      .find('MD2Tabs')
      .at(1)
      .simulate('switch', STEP_TABS.VOLUMES);
    expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.VOLUMES);
    expect(tree).toMatchSnapshot();
  });

  it('switches to manifest tab in side pane', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    tree
      .find('MD2Tabs')
      .at(1)
      .simulate('switch', STEP_TABS.MANIFEST);
    expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.MANIFEST);
    expect(tree).toMatchSnapshot();
  });

  it('closes side panel when close button is clicked', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    await TestUtils.flushPromises();
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    tree.find('SidePanel').simulate('close');
    expect(tree.state('selectedNodeDetails')).toBeNull();
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('keeps side pane open and on same tab when page is refreshed', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    tree
      .find('MD2Tabs')
      .at(1)
      .simulate('switch', STEP_TABS.LOGS);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.LOGS);

    await (tree.instance() as RunDetails).refresh();
    expect(getRunSpy).toHaveBeenCalledTimes(2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.LOGS);
  });

  it('keeps side pane open and on same tab when more nodes are added after refresh', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        nodes: {
          node1: { id: 'node1' },
          node2: { id: 'node2' },
        },
      },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    tree
      .find('MD2Tabs')
      .at(1)
      .simulate('switch', STEP_TABS.LOGS);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.LOGS);

    await (tree.instance() as RunDetails).refresh();
    expect(getRunSpy).toHaveBeenCalledTimes(2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.LOGS);
  });

  it('keeps side pane open and on same tab when run status changes, shows new status', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    tree
      .find('MD2Tabs')
      .at(1)
      .simulate('switch', STEP_TABS.LOGS);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.LOGS);
    expect(updateToolbarSpy).toHaveBeenCalledTimes(3);

    const thirdCall = updateToolbarSpy.mock.calls[2][0];
    expect(thirdCall.pageTitle).toMatchSnapshot();

    testRun.run!.status = 'Failed';
    await (tree.instance() as RunDetails).refresh();
    const fourthCall = updateToolbarSpy.mock.calls[3][0];
    expect(fourthCall.pageTitle).toMatchSnapshot();
  });

  it('shows node message banner if node receives message after refresh', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', phase: 'Succeeded', message: '' } } },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    tree
      .find('MD2Tabs')
      .at(1)
      .simulate('switch', STEP_TABS.LOGS);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('phaseMessage', undefined);

    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        nodes: { node1: { id: 'node1', phase: 'Succeeded', message: 'some node message' } },
      },
    });
    await (tree.instance() as RunDetails).refresh();
    expect(tree.state('selectedNodeDetails')).toHaveProperty(
      'phaseMessage',
      'stepStateMessage1 Succeeded stepStateMessage2: some node message',
    );
  });

  it('dismisses node message banner if node loses message after refresh', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        nodes: { node1: { id: 'node1', phase: 'Succeeded', message: 'some node message' } },
      },
    });
    tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    tree
      .find('MD2Tabs')
      .at(1)
      .simulate('switch', STEP_TABS.LOGS);
    expect(tree.state('selectedNodeDetails')).toHaveProperty(
      'phaseMessage',
      'stepStateMessage1 Succeeded stepStateMessage2: some node message',
    );

    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    await (tree.instance() as RunDetails).refresh();
    expect(tree.state('selectedNodeDetails')).toHaveProperty('phaseMessage', undefined);
  });

  [NodePhase.RUNNING, NodePhase.PENDING, NodePhase.UNKNOWN].forEach(unfinishedStatus => {
    it(`displays a spinner if graph is not defined and run has status: ${unfinishedStatus}`, async () => {
      const unfinishedRun = {
        pipeline_runtime: {
          // No graph
          workflow_manifest: '{}',
        },
        run: {
          id: 'test-run-id',
          name: 'test run',
          status: unfinishedStatus,
        },
      };
      getRunSpy.mockImplementationOnce(() => Promise.resolve(unfinishedRun));

      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();

      expect(tree).toMatchSnapshot();
    });
  });

  [NodePhase.ERROR, NodePhase.FAILED, NodePhase.SUCCEEDED, NodePhase.SKIPPED].forEach(
    finishedStatus => {
      it(`displays a message indicating there is no graph if graph is not defined and run has status: ${finishedStatus}`, async () => {
        const unfinishedRun = {
          pipeline_runtime: {
            // No graph
            workflow_manifest: '{}',
          },
          run: {
            id: 'test-run-id',
            name: 'test run',
            status: finishedStatus,
          },
        };
        getRunSpy.mockImplementationOnce(() => Promise.resolve(unfinishedRun));

        tree = shallow(<RunDetails {...generateProps()} />);
        await getRunSpy;
        await TestUtils.flushPromises();

        expect(tree).toMatchSnapshot();
      });
    },
  );

  it('shows an error banner if the custom visualizations state API fails', async () => {
    TestUtils.makeErrorResponseOnce(isCustomVisualizationsAllowedSpy, 'woops');
    tree = shallow(<RunDetails {...generateProps()} />);
    await isCustomVisualizationsAllowedSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once initially to clear
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'errorEnableCustomVis common:clickDetails',
        mode: 'error',
      }),
    );
  });

  describe('logs tab', () => {
    it('switches to logs tab in side pane', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1' } } },
        metadata: { namespace: 'ns' },
      });
      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.LOGS);
      expect(tree).toMatchSnapshot();
    });

    it('loads and shows logs in side pane', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: {
          nodes: {
            node1: {
              id: 'node1',
              phase: 'Running',
            },
          },
        },
        metadata: { namespace: 'ns' },
      });

      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      await getPodLogsSpy;
      expect(getPodLogsSpy).toHaveBeenCalledTimes(1);
      expect(getPodLogsSpy).toHaveBeenLastCalledWith('test-run-id', 'node1', 'ns');
      expect(tree).toMatchSnapshot();
    });

    it('shows stackdriver link next to logs in GKE', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1', phase: 'Succeeded' } } },
        metadata: { namespace: 'ns' },
      });
      tree = shallow(
        <RunDetails
          {...generateProps()}
          gkeMetadata={{ projectId: 'test-project-id', clusterName: 'test-cluster-name' }}
        />,
      );
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      await getPodLogsSpy;
      await TestUtils.flushPromises();
      expect(tree.find(NODE_DETAILS_SELECTOR)).toMatchInlineSnapshot(`
        <div
          className="page"
          data-testid="run-details-node-details"
        >
          <div
            className="page"
          >
            <div
              className=""
            >
              logsCanBeViewed
               
              <a
                className="link unstyled"
                href="https://console.cloud.google.com/logs/viewer?project=test-project-id&interval=NO_LIMIT&advancedFilter=resource.type%3D\\"k8s_container\\"%0Aresource.labels.cluster_name:\\"test-cluster-name\\"%0Aresource.labels.pod_name:\\"node1\\""
                rel="noopener noreferrer"
                target="_blank"
              >
                stackdriverKubernetesMonitoring
              </a>
              .
            </div>
            <div
              className="pageOverflowHidden"
            >
              <LogViewer
                logLines={
                  Array [
                    "test logs",
                  ]
                }
              />
            </div>
          </div>
        </div>
      `);
    });

    it("loads logs in run's namespace", async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        metadata: { namespace: 'username' },
        status: { nodes: { node1: { id: 'node1', phase: 'Succeeded' } } },
      });
      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      await getPodLogsSpy;
      expect(getPodLogsSpy).toHaveBeenCalledTimes(1);
      expect(getPodLogsSpy).toHaveBeenLastCalledWith('test-run-id', 'node1', 'username');
    });

    it('shows warning banner and link to Stackdriver in logs area if fetching logs failed and cluster is in GKE', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1', phase: 'Failed' } } },
        metadata: { namespace: 'ns' },
      });
      TestUtils.makeErrorResponseOnce(getPodLogsSpy, 'pod not found');
      tree = shallow(
        <RunDetails
          {...generateProps()}
          gkeMetadata={{ projectId: 'test-project-id', clusterName: 'test-cluster-name' }}
        />,
      );
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      await getPodLogsSpy;
      await TestUtils.flushPromises();
      expect(tree.find(NODE_DETAILS_SELECTOR)).toMatchInlineSnapshot(`
        <div
          className="page"
          data-testid="run-details-node-details"
        >
          <div
            className="page"
          >
            <withI18nextTranslation(Banner)
              additionalInfo="errorPodLogsReasons common:errorResponse: pod not found"
              message="errorPodLogs stackdriverKubernetesMonitoringView"
              mode="info"
              refresh={[Function]}
              showTroubleshootingGuideLink={false}
            />
            <div
              className=""
            >
              logsCanBeViewed
               
              <a
                className="link unstyled"
                href="https://console.cloud.google.com/logs/viewer?project=test-project-id&interval=NO_LIMIT&advancedFilter=resource.type%3D\\"k8s_container\\"%0Aresource.labels.cluster_name:\\"test-cluster-name\\"%0Aresource.labels.pod_name:\\"node1\\""
                rel="noopener noreferrer"
                target="_blank"
              >
                stackdriverKubernetesMonitoring
              </a>
              .
            </div>
          </div>
        </div>
      `);
    });

    it('shows warning banner without stackdriver link in logs area if fetching logs failed and cluster is not in GKE', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1', phase: 'Failed' } } },
        metadata: { namespace: 'ns' },
      });
      TestUtils.makeErrorResponseOnce(getPodLogsSpy, 'pod not found');
      tree = shallow(<RunDetails {...generateProps()} gkeMetadata={{}} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      await getPodLogsSpy;
      await TestUtils.flushPromises();
      expect(tree.find('[data-testid="run-details-node-details"]')).toMatchInlineSnapshot(`
        <div
          className="page"
          data-testid="run-details-node-details"
        >
          <div
            className="page"
          >
            <withI18nextTranslation(Banner)
              additionalInfo="errorPodLogsReasons common:errorResponse: pod not found"
              message="errorPodLogs"
              mode="info"
              refresh={[Function]}
              showTroubleshootingGuideLink={false}
            />
          </div>
        </div>
      `);
    });

    it('does not load logs if clicked node status is skipped', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: {
          nodes: {
            node1: {
              id: 'node1',
              phase: 'Skipped',
            },
          },
        },
      });
      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      await getPodLogsSpy;
      await TestUtils.flushPromises();
      expect(getPodLogsSpy).not.toHaveBeenCalled();
      expect(tree.state()).toMatchObject({
        logsBannerAdditionalInfo: '',
        logsBannerMessage: '',
      });
      expect(tree).toMatchSnapshot();
    });

    it('keeps side pane open and on same tab when logs change after refresh', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1', phase: 'Succeeded' } } },
        metadata: { namespace: 'ns' },
      });
      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
      expect(tree.state('sidepanelSelectedTab')).toEqual(STEP_TABS.LOGS);

      getPodLogsSpy.mockImplementationOnce(() => 'new test logs');
      await (tree.instance() as RunDetails).refresh();
      expect(tree).toMatchSnapshot();
    });

    it('shows error banner if fetching logs failed not because pod has gone away', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1', phase: 'Succeeded' } } },
        metadata: { namespace: 'ns' },
      });
      TestUtils.makeErrorResponseOnce(getPodLogsSpy, 'getting logs failed');
      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      await getPodLogsSpy;
      await TestUtils.flushPromises();
      expect(tree.state()).toMatchObject({
        logsBannerAdditionalInfo: 'common:errorResponse: getting logs failed',
        logsBannerMessage: 'errorPodLogs',
        logsBannerMode: 'error',
      });
    });

    it('dismisses log failure warning banner when logs can be fetched after refresh', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1', phase: 'Failed' } } },
        metadata: { namespace: 'ns' },
      });
      TestUtils.makeErrorResponseOnce(getPodLogsSpy, 'getting logs failed');
      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.LOGS);
      await getPodLogsSpy;
      await TestUtils.flushPromises();
      expect(tree.state()).toMatchObject({
        logsBannerAdditionalInfo: 'common:errorResponse: getting logs failed',
        logsBannerMessage: 'errorPodLogs',
        logsBannerMode: 'error',
      });

      testRun.run!.status = 'Failed';
      await (tree.instance() as RunDetails).refresh();
      expect(tree.state()).toMatchObject({
        logsBannerAdditionalInfo: '',
        logsBannerMessage: '',
      });
    });
  });

  describe('pod tab', () => {
    it('shows pod info', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1', phase: 'Failed' } } },
        metadata: { namespace: 'ns' },
      });
      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.POD);
      await getPodInfoSpy;
      await TestUtils.flushPromises();

      expect(tree.find(NODE_DETAILS_SELECTOR)).toMatchInlineSnapshot(`
        <div
          className="page"
          data-testid="run-details-node-details"
        >
          <div
            className="page"
          >
            <PodInfo
              name="node1"
              namespace="ns"
              t={[Function]}
            />
          </div>
        </div>
      `);
    });

    it('does not show pod pane if selected node skipped', async () => {
      testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
        status: { nodes: { node1: { id: 'node1', phase: 'Skipped' } } },
        metadata: { namespace: 'ns' },
      });
      tree = shallow(<RunDetails {...generateProps()} />);
      await getRunSpy;
      await TestUtils.flushPromises();
      clickGraphNode(tree, 'node1');
      tree
        .find('MD2Tabs')
        .at(1)
        .simulate('switch', STEP_TABS.POD);
      await TestUtils.flushPromises();

      expect(tree.find(NODE_DETAILS_SELECTOR)).toMatchInlineSnapshot(`
        <div
          className="page"
          data-testid="run-details-node-details"
        />
      `);
    });
  });

  describe('auto refresh', () => {
    beforeEach(() => {
      testRun.run!.status = NodePhase.PENDING;
    });

    it('starts an interval of 5 seconds to auto refresh the page', async () => {
      tree = shallow(<RunDetails {...generateProps()} />);
      await TestUtils.flushPromises();

      expect(setInterval).toHaveBeenCalledTimes(1);
      expect(setInterval).toHaveBeenCalledWith(expect.any(Function), 5000);
    });

    it('refreshes after each interval', async () => {
      tree = shallow(<RunDetails {...generateProps()} />);
      await TestUtils.flushPromises();

      const refreshSpy = jest.spyOn(tree.instance() as RunDetails, 'refresh');

      expect(refreshSpy).toHaveBeenCalledTimes(0);

      jest.runOnlyPendingTimers();
      expect(refreshSpy).toHaveBeenCalledTimes(1);
      await TestUtils.flushPromises();
    }, 10000);

    [NodePhase.ERROR, NodePhase.FAILED, NodePhase.SUCCEEDED, NodePhase.SKIPPED].forEach(status => {
      it(`sets 'runFinished' to true if run has status: ${status}`, async () => {
        testRun.run!.status = status;
        tree = shallow(<RunDetails {...generateProps()} />);
        await TestUtils.flushPromises();

        expect(tree.state('runFinished')).toBe(true);
      });
    });

    [NodePhase.PENDING, NodePhase.RUNNING, NodePhase.UNKNOWN].forEach(status => {
      it(`leaves 'runFinished' false if run has status: ${status}`, async () => {
        testRun.run!.status = status;
        tree = shallow(<RunDetails {...generateProps()} />);
        await TestUtils.flushPromises();

        expect(tree.state('runFinished')).toBe(false);
      });
    });

    it('pauses auto refreshing if window loses focus', async () => {
      tree = shallow(<RunDetails {...generateProps()} />);
      await TestUtils.flushPromises();

      expect(setInterval).toHaveBeenCalledTimes(1);
      expect(clearInterval).toHaveBeenCalledTimes(0);

      window.dispatchEvent(new Event('blur'));
      await TestUtils.flushPromises();

      expect(clearInterval).toHaveBeenCalledTimes(1);
    });

    it('resumes auto refreshing if window loses focus and then regains it', async () => {
      // Declare that the run has not finished
      testRun.run!.status = NodePhase.PENDING;
      tree = shallow(<RunDetails {...generateProps()} />);
      await TestUtils.flushPromises();

      expect(tree.state('runFinished')).toBe(false);
      expect(setInterval).toHaveBeenCalledTimes(1);
      expect(clearInterval).toHaveBeenCalledTimes(0);

      window.dispatchEvent(new Event('blur'));
      await TestUtils.flushPromises();

      expect(clearInterval).toHaveBeenCalledTimes(1);

      window.dispatchEvent(new Event('focus'));
      await TestUtils.flushPromises();

      expect(setInterval).toHaveBeenCalledTimes(2);
    });

    it('does not resume auto refreshing if window loses focus and then regains it but run is finished', async () => {
      // Declare that the run has finished
      testRun.run!.status = NodePhase.SUCCEEDED;
      tree = shallow(<RunDetails {...generateProps()} />);
      await TestUtils.flushPromises();

      expect(tree.state('runFinished')).toBe(true);
      expect(setInterval).toHaveBeenCalledTimes(0);
      expect(clearInterval).toHaveBeenCalledTimes(0);

      window.dispatchEvent(new Event('blur'));
      await TestUtils.flushPromises();

      // We expect 0 calls because the interval was never set, so it doesn't need to be cleared
      expect(clearInterval).toHaveBeenCalledTimes(0);

      window.dispatchEvent(new Event('focus'));
      await TestUtils.flushPromises();

      expect(setInterval).toHaveBeenCalledTimes(0);
    });
  });

  describe('EnhancedRunDetails', () => {
    it('redirects to experiments page when namespace changes', () => {
      const history = createMemoryHistory({
        initialEntries: ['/does-not-matter'],
      });
      const { rerender } = render(
        <Router history={history}>
          <NamespaceContext.Provider value='ns1'>
            <EnhancedRunDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).not.toEqual('/experiments');
      rerender(
        <Router history={history}>
          <NamespaceContext.Provider value='ns2'>
            <EnhancedRunDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/experiments');
    });

    it('does not redirect when namespace stays the same', () => {
      const history = createMemoryHistory({
        initialEntries: ['/initial-path'],
      });
      const { rerender } = render(
        <Router history={history}>
          <NamespaceContext.Provider value='ns1'>
            <EnhancedRunDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/initial-path');
      rerender(
        <Router history={history}>
          <NamespaceContext.Provider value='ns1'>
            <EnhancedRunDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/initial-path');
    });

    it('does not redirect when namespace initializes', () => {
      const history = createMemoryHistory({
        initialEntries: ['/initial-path'],
      });
      const { rerender } = render(
        <Router history={history}>
          <NamespaceContext.Provider value={undefined}>
            <EnhancedRunDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/initial-path');
      rerender(
        <Router history={history}>
          <NamespaceContext.Provider value='ns1'>
            <EnhancedRunDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/initial-path');
    });
  });
});

function clickGraphNode(wrapper: ShallowWrapper, nodeId: string) {
  // TODO: use dom events instead
  wrapper.find('GraphMock').simulate('click', nodeId);
}
