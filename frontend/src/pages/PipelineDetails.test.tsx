/*
 * Copyright 2018 Google LLC
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

import * as React from 'react';
import * as StaticGraphParser from '../lib/StaticGraphParser';
import PipelineDetails from './PipelineDetails';
import TestUtils from '../TestUtils';
import { ApiExperiment } from '../apis/experiment';
import { ApiPipeline, ApiPipelineVersion } from '../apis/pipeline';
import { ApiRunDetail, ApiResourceType } from '../apis/run';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { RouteParams, RoutePage, QUERY_PARAMS } from '../components/Router';
import { graphlib } from 'dagre';
import { shallow, mount, ShallowWrapper, ReactWrapper } from 'enzyme';
import { ButtonKeys } from '../lib/Buttons';
import { TFunction } from 'i18next';

jest.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: ((key: string) => key) as any,
  }),
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: ((key: string) => key) as any };
    return Component;
  },
}));
describe('PipelineDetails', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipelineVersion');
  const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelineVersions');
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const deletePipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApi, 'deletePipelineVersion');
  const getPipelineVersionTemplateSpy = jest.spyOn(
    Apis.pipelineServiceApi,
    'getPipelineVersionTemplate',
  );
  const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');

  let tree: ShallowWrapper | ReactWrapper;
  let testPipeline: ApiPipeline = {};
  let testPipelineVersion: ApiPipelineVersion = {};
  let testRun: ApiRunDetail = {};
  let t: TFunction = (key: string) => key;

  function generateProps(fromRunSpec = false): PageProps {
    const match = {
      isExact: true,
      params: fromRunSpec
        ? {}
        : {
            [RouteParams.pipelineId]: testPipeline.id,
            [RouteParams.pipelineVersionId]:
              (testPipeline.default_version && testPipeline.default_version!.id) || '',
          },
      path: '',
      url: '',
    };
    const location = { search: fromRunSpec ? `?${QUERY_PARAMS.fromRunId}=test-run-id` : '' } as any;
    const pageProps = TestUtils.generatePageProps(
      PipelineDetails,
      location,
      match,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
      { t },
    );
    return pageProps;
  }

  beforeAll(() => jest.spyOn(console, 'error').mockImplementation());

  beforeEach(() => {
    jest.clearAllMocks();

    testPipeline = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test pipeline description',
      id: 'test-pipeline-id',
      name: 'test pipeline',
      parameters: [{ name: 'param1', value: 'value1' }],
      default_version: {
        id: 'test-pipeline-version-id',
        name: 'test-pipeline-version',
      },
    };

    testPipelineVersion = {
      id: 'test-pipeline-version-id',
      name: 'test-pipeline-version',
    };

    testRun = {
      run: {
        id: 'test-run-id',
        name: 'test run',
        pipeline_spec: {
          pipeline_id: 'run-pipeline-id',
        },
      },
    };

    getPipelineSpy.mockImplementation(() => Promise.resolve(testPipeline));
    getPipelineVersionSpy.mockImplementation(() => Promise.resolve(testPipelineVersion));
    listPipelineVersionsSpy.mockImplementation(() =>
      Promise.resolve({ versions: [testPipelineVersion] }),
    );
    getRunSpy.mockImplementation(() => Promise.resolve(testRun));
    getExperimentSpy.mockImplementation(() =>
      Promise.resolve({ id: 'test-experiment-id', name: 'test experiment' } as ApiExperiment),
    );
    // getTemplateSpy.mockImplementation(() => Promise.resolve({ template: 'test template' }));
    getPipelineVersionTemplateSpy.mockImplementation(() =>
      Promise.resolve({ template: 'test template' }),
    );
    createGraphSpy.mockImplementation(() => new graphlib.Graph());
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
    jest.resetAllMocks();
  });

  it('shows empty pipeline details with no graph', async () => {
    TestUtils.makeErrorResponseOnce(createGraphSpy, 'bad graph');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('shows pipeline name in page name, and breadcrumb to go back to pipelines', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'common:pipelines', href: RoutePage.PIPELINES }],
        pageTitle: testPipeline.name + ' (' + testPipelineVersion.name + ')',
      }),
    );
  });

  it(
    'shows all runs breadcrumbs, and "Pipeline details" as page title when the pipeline ' +
      'comes from a run spec that does not have an experiment',
    async () => {
      tree = shallow(<PipelineDetails {...generateProps(true)} />);
      await getRunSpy;
      await getPipelineVersionTemplateSpy;
      await TestUtils.flushPromises();
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'allRuns', href: '/runs' },
            {
              displayName: 'test run',
              href: '/runs/details/test-run-id',
            },
          ],
          pageTitle: 'pipelineDetails',
        }),
      );
    },
  );

  it(
    'shows all runs breadcrumbs, and "Pipeline details" as page title when the pipeline ' +
      'comes from a run spec that has an experiment',
    async () => {
      testRun.run!.resource_references = [
        { key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } },
      ];
      tree = shallow(<PipelineDetails {...generateProps(true)} />);
      await getRunSpy;
      await getExperimentSpy;
      await getPipelineVersionTemplateSpy;
      await TestUtils.flushPromises();
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'common:experiments', href: RoutePage.EXPERIMENTS },
            {
              displayName: 'test experiment',
              href: RoutePage.EXPERIMENT_DETAILS.replace(
                ':' + RouteParams.experimentId,
                'test-experiment-id',
              ),
            },
            {
              displayName: testRun.run!.name,
              href: RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, testRun.run!.id!),
            },
          ],
          pageTitle: 'pipelineDetails',
        }),
      );
    },
  );

  it('parses the workflow source in embedded pipeline spec as JSON and then converts it to YAML', async () => {
    testRun.run!.pipeline_spec = {
      pipeline_id: 'run-pipeline-id',
      workflow_manifest: '{"spec": {"arguments": {"parameters": [{"name": "output"}]}}}',
    };

    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await TestUtils.flushPromises();

    expect(tree.state('templateString')).toBe(
      'spec:\n  arguments:\n    parameters:\n      - name: output\n',
    );
  });

  it('shows load error banner when failing to parse the workflow source in embedded pipeline spec', async () => {
    testRun.run!.pipeline_spec = {
      pipeline_id: 'run-pipeline-id',
      workflow_manifest: 'not valid JSON',
    };

    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'Unexpected token o in JSON at position 1',
        message: 'parsePipelineSpecFailed: test-run-id. common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('shows load error banner when failing to get run details, when loading from run spec', async () => {
    TestUtils.makeErrorResponseOnce(getRunSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'cannotRetrieveRunDetails common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('shows load error banner when failing to get experiment details, when loading from run spec', async () => {
    testRun.run!.resource_references = [
      { key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } },
    ];
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'cannotRetrieveRunDetails common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('uses an empty string and does not show error when getTemplate response is empty', async () => {
    getPipelineVersionTemplateSpy.mockImplementationOnce(() => Promise.resolve({}));

    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(1); // Once to clear banner
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({}));

    expect(tree.state('templateString')).toBe('');
  });

  it('shows load error banner when failing to get pipeline', async () => {
    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'cannotRetrievePipelineDetails common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('shows load error banner when failing to get pipeline template', async () => {
    TestUtils.makeErrorResponseOnce(getPipelineVersionTemplateSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'cannotRetrievePipelineTemplate common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('shows no graph error banner when failing to parse graph', async () => {
    TestUtils.makeErrorResponseOnce(createGraphSpy, 'bad graph');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad graph',
        message: 'errorGenerateGraph common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('clears the error banner when refreshing the page', async () => {
    TestUtils.makeErrorResponseOnce(getPipelineVersionTemplateSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops',
        message: 'cannotRetrievePipelineTemplate common:clickDetails',
        mode: 'error',
      }),
    );

    (tree.instance() as PipelineDetails).refresh();

    expect(updateBannerSpy).toHaveBeenLastCalledWith({ t });
  });

  it('shows empty pipeline details with empty graph', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('sets summary shown state to false when clicking the Hide button', async () => {
    tree = mount(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    tree.update();
    expect(tree.state('summaryShown')).toBe(true);
    tree.find('Paper Button').simulate('click');
    expect(tree.state('summaryShown')).toBe(false);
  });

  it('collapses summary card when summary shown state is false', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    tree.setState({ summaryShown: false });
    expect(tree).toMatchSnapshot();
  });

  it('collapses summary card when summary shown state is false', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    tree.setState({ summaryShown: false });
    expect(tree).toMatchSnapshot();
  });

  it('has a new experiment button if it has a pipeline reference', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    expect(newExperimentBtn).toBeDefined();
  });

  it("has 'create run' toolbar button if viewing an embedded pipeline", async () => {
    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    /* create run and create pipeline version, so 2 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(2);
    const newRunBtn = instance.getInitialToolbarState().actions[
      (ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION, ButtonKeys.NEW_PIPELINE_VERSION)
    ];
    expect(newRunBtn).toBeDefined();
  });

  it('clicking new run button when viewing embedded pipeline navigates to the new run page with run ID', async () => {
    tree = shallow(<PipelineDetails {...generateProps(true)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newRunBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    newRunBtn!.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.fromRunId}=${testRun.run!.id}`,
    );
  });

  it("has 'create run' toolbar button if not viewing an embedded pipeline", async () => {
    tree = shallow(<PipelineDetails {...generateProps(false)} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    /* create run, create pipeline version, create experiment and delete run, so 4 */
    expect(Object.keys(instance.getInitialToolbarState().actions)).toHaveLength(4);
    const newRunBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    expect(newRunBtn).toBeDefined();
  });

  it('clicking new run button navigates to the new run page', async () => {
    tree = shallow(<PipelineDetails {...generateProps(false)} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newRunFromPipelineVersionBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    newRunFromPipelineVersionBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}&${
          QUERY_PARAMS.pipelineVersionId
        }=${testPipeline.default_version!.id!}`,
    );
  });

  it('clicking new run button when viewing half-loaded page navigates to the new run page with pipeline ID and version ID', async () => {
    tree = shallow(<PipelineDetails {...generateProps(false)} />);
    // Intentionally don't wait until all network requests finish.
    const instance = tree.instance() as PipelineDetails;
    const newRunFromPipelineVersionBtn = instance.getInitialToolbarState().actions[
      ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION
    ];
    newRunFromPipelineVersionBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}&${
          QUERY_PARAMS.pipelineVersionId
        }=${testPipeline.default_version!.id!}`,
    );
  });

  it('clicking new experiment button navigates to new experiment page', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    await newExperimentBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_EXPERIMENT + `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}`,
    );
  });

  it('clicking new experiment button when viewing half-loaded page navigates to the new experiment page with the pipeline ID', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    // Intentionally don't wait until all network requests finish.
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    await newExperimentBtn.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_EXPERIMENT + `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}`,
    );
  });

  it('has a delete button', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
  });

  it('shows delete confirmation dialog when delete button is clicked', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        title: 'common:delete common:this common:pipelineVersion?',
      }),
    );
  });

  it('does not call delete API for selected pipeline when delete dialog is canceled', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((b: any) => b.text === 'common:cancel');
    await cancelBtn.onClick();
    expect(deletePipelineVersionSpy).not.toHaveBeenCalled();
  });

  it('calls delete API when delete dialog is confirmed', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    expect(deletePipelineVersionSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineVersionSpy).toHaveBeenLastCalledWith(testPipeline.default_version!.id!);
  });

  it('calls delete API when delete dialog is confirmed and page is half-loaded', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    // Intentionally don't wait until all network requests finish.
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    expect(deletePipelineVersionSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineVersionSpy).toHaveBeenLastCalledWith(testPipeline.default_version!.id);
  });

  it('shows error dialog if deletion fails', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(deletePipelineVersionSpy, 'woops');
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    expect(updateDialogSpy).toHaveBeenCalledTimes(2); // Delete dialog + error dialog
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content:
          'common:failedTo common:delete common:pipelineVersion: test-pipeline-version-id common:withError: "woops"',
        title: 'common:failedTo common:delete common:pipelineVersion',
      }),
    );
  });

  it('shows success snackbar if deletion succeeds', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    expect(updateSnackbarSpy).toHaveBeenCalledTimes(1);
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        message: 'common:delete common:succeededFor common:this common:pipelineVersion',
        open: true,
      }),
    );
  });

  it('opens side panel on clicked node, shows message when node is not found in graph', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'some-node-id');
    expect(tree.state('selectedNodeId')).toBe('some-node-id');
    expect(tree).toMatchSnapshot();
  });

  it('shows clicked node info in the side panel if it is in the graph', async () => {
    const g = new graphlib.Graph();
    const info = new StaticGraphParser.SelectedNodeInfo();
    info.args = ['test arg', 'test arg2'];
    info.command = ['test command', 'test command 2'];
    info.condition = 'test condition';
    info.image = 'test image';
    info.inputs = [
      ['key1', 'val1'],
      ['key2', 'val2'],
    ];
    info.outputs = [
      ['key3', 'val3'],
      ['key4', 'val4'],
    ];
    info.nodeType = 'container';
    g.setNode('node1', { info, label: 'node1' });
    createGraphSpy.mockImplementation(() => g);

    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    clickGraphNode(tree, 'node1');
    expect(tree).toMatchSnapshot();
  });

  it('shows pipeline source code when config tab is clicked', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 1);
    expect(tree.state('selectedTab')).toBe(1);
    expect(tree).toMatchSnapshot();
  });

  it('closes side panel when close button is clicked', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    tree.setState({ selectedNodeId: 'some-node-id' });
    tree.find('SidePanel').simulate('close');
    expect(tree.state('selectedNodeId')).toBe('');
    expect(tree).toMatchSnapshot();
  });

  it('shows correct versions in version selector', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineVersionTemplateSpy;
    await TestUtils.flushPromises();
    expect(tree.state('versions')).toHaveLength(1);
    expect(tree).toMatchSnapshot();
  });
});

function clickGraphNode(wrapper: ShallowWrapper, nodeId: string) {
  // TODO: use dom events instead
  wrapper
    .find('EnhancedGraph')
    .dive()
    .dive()
    .simulate('click', nodeId);
}
