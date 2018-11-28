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
import RunDetails from './RunDetails';
import TestUtils from '../TestUtils';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiRunDetail, ApiResourceType } from '../apis/run';
import { Apis } from '../lib/Apis';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { PageProps } from './Page';
import { PlotType } from '../components/viewers/Viewer';
import { QUERY_PARAMS } from '../lib/URLParser';
import { RouteParams, RoutePage } from '../components/Router';
import { Workflow } from 'third_party/argo-ui/argo_template';
import { shallow } from 'enzyme';

describe('RunDetails', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const getPodLogsSpy = jest.spyOn(Apis, 'getPodLogs');

  let testRun: ApiRunDetail = {};

  function generateProps(): PageProps {
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: { params: { [RouteParams.runId]: testRun.run!.id }, isExact: true, path: '', url: '' },
      toolbarProps: { actions: [], breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
    return Object.assign(pageProps, {
      toolbarProps: new RunDetails(pageProps).getInitialToolbarState(),
    });
  }

  beforeAll(() => jest.spyOn(console, 'error').mockImplementation());
  afterAll(() => jest.resetAllMocks());

  beforeEach(() => {
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
    historyPushSpy.mockClear();
    updateBannerSpy.mockClear();
    updateDialogSpy.mockClear();
    updateSnackbarSpy.mockClear();
    updateToolbarSpy.mockClear();
    getRunSpy.mockReset();
    getRunSpy.mockImplementation(() => Promise.resolve(testRun));
    getExperimentSpy.mockClear();
    getExperimentSpy.mockImplementation(() => Promise.resolve('{}'));
    getPodLogsSpy.mockClear();
    getPodLogsSpy.mockImplementation(() => 'test logs');
  });

  it('shows success run status in page title', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const lastCall = updateToolbarSpy.mock.calls[2][0];
    expect(lastCall.pageTitle).toMatchSnapshot();
    tree.unmount();
  });

  it('shows failure run status in page title', async () => {
    testRun.run!.status = 'Failed';
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const lastCall = updateToolbarSpy.mock.calls[2][0];
    expect(lastCall.pageTitle).toMatchSnapshot();
    tree.unmount();
  });

  it('has a refresh button, clicking it calls get run API again', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    await refreshBtn!.action();
    expect(getRunSpy).toHaveBeenCalledTimes(2);
    tree.unmount();
  });

  it('has a clone button, clicking it navigates to new run page', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const cloneBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Clone');
    expect(cloneBtn).toBeDefined();
    await cloneBtn!.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.cloneFromRun}=${testRun.run!.id}`);
    tree.unmount();
  });

  it('renders an empty run', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('calls the get run API once to load it', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    expect(getRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
    tree.unmount();
  });

  it('shows an error banner if get run API fails', async () => {
    TestUtils.makeErrorResponseOnce(getRunSpy, 'woops');
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once initially to clear
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'woops',
      message: 'Error: failed to retrieve run: ' + testRun.run!.id + '. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('shows an error banner if get experiment API fails', async () => {
    testRun.run!.resource_references = [{ key: { id: 'experiment1', type: ApiResourceType.EXPERIMENT } }];
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'woops');
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once initially to clear
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'woops',
      message: 'Error: failed to retrieve run: ' + testRun.run!.id + '. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('calls the get experiment API once to load it if the run has its reference', async () => {
    testRun.run!.resource_references = [{ key: { id: 'experiment1', type: ApiResourceType.EXPERIMENT } }];
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    expect(getRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
    expect(getExperimentSpy).toHaveBeenCalledTimes(1);
    expect(getExperimentSpy).toHaveBeenLastCalledWith('experiment1');
    tree.unmount();
  });

  it('shows workflow errors as page error', async () => {
    jest.spyOn(WorkflowParser, 'getWorkflowError').mockImplementationOnce(() => 'some error message');
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear on init, once for error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'some error message',
      message: 'Error: found errors when executing run: ' + testRun.run!.id + '. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('switches to config tab', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 1);
    expect(tree.state('selectedTab')).toBe(1);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows run config fields', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      metadata: {
        creationTimestamp: new Date(2018, 6, 5, 4, 3, 2).toISOString(),
      },
      spec: {
        arguments: {
          parameters: [{
            name: 'param1',
            value: 'value1',
          }, {
            name: 'param2',
            value: 'value2',
          }],
        },
      },
      status: {
        finishedAt: new Date(2018, 6, 6, 5, 4, 3).toISOString(),
        phase: 'Skipped',
        startedAt: new Date(2018, 6, 5, 4, 3, 2).toISOString(),
      },
    } as Workflow);
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 1);
    expect(tree.state('selectedTab')).toBe(1);
    expect(tree).toMatchSnapshot();
    tree.unmount();
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
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 1);
    expect(tree.state('selectedTab')).toBe(1);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows run config fields - handles no metadata', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        finishedAt: new Date(2018, 6, 6, 5, 4, 3).toISOString(),
        phase: 'Skipped',
        startedAt: new Date(2018, 6, 5, 4, 3, 2).toISOString(),
      },
    } as Workflow);
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 1);
    expect(tree.state('selectedTab')).toBe(1);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows a one-node graph', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', }, }, },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('opens side panel when graph node is clicked', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', }, }, },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree).toMatchSnapshot();
    tree.unmount();
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
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    expect(tree.state('selectedNodeDetails')).toHaveProperty('phaseMessage',
      'This step is in ' + testRun.run!.status + ' state with this message: some test message');
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows clicked node output in side pane', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', }, }, },
    });
    const parserSpy = jest.spyOn(WorkflowParser, 'loadNodeOutputPaths').mockImplementationOnce(() =>
      [{ source: 'gcs', bucket: 'somebucket', key: 'somekey' }]);
    const loaderSpy = jest.spyOn(OutputArtifactLoader, 'load').mockImplementationOnce(() =>
      [{ type: PlotType.TENSORBOARD, url: 'some url' }]);
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    await parserSpy;
    await loaderSpy;
    expect(parserSpy).toHaveBeenCalledTimes(1);
    expect(parserSpy).toHaveBeenLastCalledWith({ id: 'node1' });
    expect(loaderSpy).toHaveBeenCalledTimes(1);
    expect(loaderSpy).toHaveBeenLastCalledWith({ bucket: 'somebucket', key: 'somekey', source: 'gcs' });
    expect(tree.state('selectedNodeDetails')).toMatchObject({ id: 'node1' });
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('switches to inputs/outputs tab in side pane', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        nodes: {
          node1: {
            id: 'node1',
            inputs: {
              parameters: [
                { name: 'input1', value: 'val1' },
              ],
            },
            name: 'node1',
            outputs: {
              parameters: [
                { name: 'output1', value: 'val1' },
                { name: 'output2', value: 'value2' },
              ]
            },
            phase: 'Succeeded',
          },
        },
      },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 1);
    expect(tree.state('sidepanelSelectedTab')).toEqual(1);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('switches to logs tab in side pane', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', }, }, },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    expect(tree.state('sidepanelSelectedTab')).toEqual(2);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('loads and shows logs in side pane', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', }, }, },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    await getPodLogsSpy;
    expect(getPodLogsSpy).toHaveBeenCalledTimes(1);
    expect(getPodLogsSpy).toHaveBeenLastCalledWith('node1');
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows error banner atop logs area if fetching logs failed', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', }, }, },
    });
    TestUtils.makeErrorResponseOnce(getPodLogsSpy, 'getting logs failed');
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    await getPodLogsSpy;
    await TestUtils.flushPromises();
    expect(tree.state()).toMatchObject({
      logsBannerAdditionalInfo: 'getting logs failed',
      logsBannerMessage: 'Error: failed to retrieve logs. Click Details for more information.',
      logsBannerMode: 'error',
    });
    expect(tree).toMatchSnapshot();
    tree.unmount();
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
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    await getPodLogsSpy;
    await TestUtils.flushPromises();
    expect(getPodLogsSpy).not.toHaveBeenCalled();
    expect(tree.state()).toMatchObject({
      logsBannerAdditionalInfo: '',
      logsBannerMessage: '',
    });
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('closes side panel when close button is clicked', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', }, }, },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    tree.find('SidePanel').simulate('close');
    expect(tree.state('selectedNodeDetails')).toBeNull();
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('keeps side pane open and on same tab when refresh is clicked', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', }, }, },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(2);

    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    await refreshBtn!.action();
    expect(getRunSpy).toHaveBeenCalledTimes(2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(2);
  });

  it('keeps side pane open and on same tab when more nodes are added after refresh', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: {
        nodes: {
          node1: { id: 'node1', },
          node2: { id: 'node2', },
        },
      },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(2);

    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    await refreshBtn!.action();
    expect(getRunSpy).toHaveBeenCalledTimes(2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(2);
  });

  it('keeps side pane open and on same tab when run status changes, shows new status', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(2);
    expect(updateToolbarSpy).toHaveBeenCalledTimes(3);

    const thirdCall = updateToolbarSpy.mock.calls[2][0];
    expect(thirdCall.pageTitle).toMatchSnapshot();

    testRun.run!.status = 'Failed';
    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    await refreshBtn!.action();
    const fourthCall = updateToolbarSpy.mock.calls[3][0];
    expect(fourthCall.pageTitle).toMatchSnapshot();
  });

  it('keeps side pane open and on same tab when logs change after refresh', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('id', 'node1');
    expect(tree.state('sidepanelSelectedTab')).toEqual(2);

    getPodLogsSpy.mockImplementationOnce(() => 'new test logs');
    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    await refreshBtn!.action();
    expect(tree).toMatchSnapshot();
  });

  it('dismisses log failure error banner when logs can be fetched after refresh', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    TestUtils.makeErrorResponseOnce(getPodLogsSpy, 'getting logs failed');
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    await getPodLogsSpy;
    await TestUtils.flushPromises();
    expect(tree.state()).toMatchObject({
      logsBannerAdditionalInfo: 'getting logs failed',
      logsBannerMessage: 'Error: failed to retrieve logs. Click Details for more information.',
      logsBannerMode: 'error',
    });

    testRun.run!.status = 'Failed';
    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    await refreshBtn!.action();
    expect(tree.state()).toMatchObject({
      logsBannerAdditionalInfo: '',
      logsBannerMessage: '',
    });
  });

  it('shows node message banner if node receives message after refresh', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', phase: 'Succeeded', message: '' } } },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('phaseMessage', undefined);

    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', phase: 'Succeeded', message: 'some node message' } } },
    });
    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    await refreshBtn!.action();
    expect(tree.state('selectedNodeDetails')).toHaveProperty('phaseMessage',
      'This step is in Succeeded state with this message: some node message');
  });

  it('dismisses node message banner if node loses message after refresh', async () => {
    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1', phase: 'Succeeded', message: 'some node message' } } },
    });
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    tree.find('MD2Tabs').at(1).simulate('switch', 2);
    expect(tree.state('selectedNodeDetails')).toHaveProperty('phaseMessage',
      'This step is in Succeeded state with this message: some node message');

    testRun.pipeline_runtime!.workflow_manifest = JSON.stringify({
      status: { nodes: { node1: { id: 'node1' } } },
    });
    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    await refreshBtn!.action();
    expect(tree.state('selectedNodeDetails')).toHaveProperty('phaseMessage', undefined);
  });
});
