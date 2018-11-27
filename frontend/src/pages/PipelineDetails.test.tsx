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
import PipelineDetails, { css } from './PipelineDetails';
import TestUtils from '../TestUtils';
import { ApiPipeline } from '../apis/pipeline';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { QUERY_PARAMS } from '../lib/URLParser';
import { RouteParams, RoutePage } from '../components/Router';
import { graphlib } from 'dagre';
import { shallow, mount, ShallowWrapper, ReactWrapper } from 'enzyme';

describe('PipelineDetails', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const deletePipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'deletePipeline');
  const getTemplateSpy = jest.spyOn(Apis.pipelineServiceApi, 'getTemplate');
  const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');

  let tree: ShallowWrapper | ReactWrapper;
  let testPipeline: ApiPipeline = {};

  function generateProps(): PageProps {
    // getInitialToolbarState relies on page props having been populated, so fill those first
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: { params: { [RouteParams.pipelineId]: testPipeline.id }, isExact: true, path: '', url: '' },
      toolbarProps: { actions: [], breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
    return Object.assign(pageProps, {
      toolbarProps: new PipelineDetails(pageProps).getInitialToolbarState(),
    });
  }

  beforeAll(() => jest.spyOn(console, 'error').mockImplementation());
  afterAll(() => jest.resetAllMocks());

  beforeEach(() => {
    testPipeline = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test job description',
      enabled: true,
      id: 'test-job-id',
      max_concurrency: '50',
      name: 'test job',
      pipeline_spec: {
        parameters: [{ name: 'param1', value: 'value1' }],
        pipeline_id: 'some-pipeline-id',
      },
      trigger: {
        periodic_schedule: {
          end_time: new Date(2018, 10, 9, 8, 7, 6),
          interval_second: '3600',
          start_time: new Date(2018, 9, 8, 7, 6),
        }
      },
    } as ApiPipeline;

    historyPushSpy.mockClear();
    updateBannerSpy.mockClear();
    updateDialogSpy.mockClear();
    updateSnackbarSpy.mockClear();
    updateToolbarSpy.mockClear();
    deletePipelineSpy.mockReset();
    getPipelineSpy.mockImplementation(() => Promise.resolve(testPipeline));
    getPipelineSpy.mockClear();
    getTemplateSpy.mockImplementation(() => Promise.resolve('{}'));
    getTemplateSpy.mockClear();
    createGraphSpy.mockImplementation(() => new graphlib.Graph());
    createGraphSpy.mockClear();
  });

  afterEach(() => tree.unmount());

  it('shows empty pipeline details with no graph', async () => {
    TestUtils.makeErrorResponseOnce(createGraphSpy, 'bad graph');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('shows load error banner when failing to get pipeline', async () => {
    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'woops',
      message: 'Cannot retrieve pipeline details. Click Details for more information.',
      mode: 'error',
    }));
  });

  it('shows load error banner when failing to get pipeline template', async () => {
    TestUtils.makeErrorResponseOnce(getTemplateSpy, 'woops');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'woops',
      message: 'Cannot retrieve pipeline template. Click Details for more information.',
      mode: 'error',
    }));
  });

  it('shows no graph error banner when failing to parse graph', async () => {
    TestUtils.makeErrorResponseOnce(createGraphSpy, 'bad graph');
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad graph',
      message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
      mode: 'error',
    }));
  });

  it('shows empty pipeline details with empty graph', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('sets summary shown state to false when clicking the Hide button', async () => {
    tree = mount(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    tree.update();
    expect(tree.state('summaryShown')).toBe(true);
    tree.find('Paper Button').simulate('click');
    expect(tree.state('summaryShown')).toBe(false);
  });

  it('collapses summary card when summary shown state is false', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    tree.setState({ summaryShown: false });
    expect(tree).toMatchSnapshot();
  });

  it('shows the summary card when clicking Show button', async () => {
    tree = mount(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    tree.setState({ summaryShown: false });
    tree.find(`.${css.footer} Button`).simulate('click');
    expect(tree.state('summaryShown')).toBe(true);
  });

  it('has a new experiment button', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Start an experiment');
    expect(newExperimentBtn).toBeDefined();
  });

  it('clicking new experiment button navigates to new experiment page', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const newExperimentBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Start an experiment');
    await newExperimentBtn!.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_EXPERIMENT + `?${QUERY_PARAMS.pipelineId}=${testPipeline.id}`);
  });

  it('has a delete button', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as PipelineDetails;
    const deleteBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Delete');
    expect(deleteBtn).toBeDefined();
  });

  it('shows delete confirmation dialog when delete buttin is clicked', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    const deleteBtn = (tree.instance() as PipelineDetails)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      title: 'Delete this pipeline?',
    }));
  });

  it('does not call delete API for selected pipeline when delete dialog is canceled', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    const deleteBtn = (tree.instance() as PipelineDetails)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((b: any) => b.text === 'Cancel');
    await cancelBtn.onClick();
    expect(deletePipelineSpy).not.toHaveBeenCalled();
  });

  it('calls delete API when delete dialog is confirmed', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineSpy).toHaveBeenCalledTimes(1);
    expect(deletePipelineSpy).toHaveBeenLastCalledWith(testPipeline.id);
  });

  it('shows error dialog if deletion fails', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(deletePipelineSpy, 'woops');
    await getTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(updateDialogSpy).toHaveBeenCalledTimes(2); // Delete dialog + error dialog
    expect(updateDialogSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      content: 'woops',
      title: 'Failed to delete pipeline',
    }));
  });

  it('shows success snackbar if deletion succeeds', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as PipelineDetails)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(updateSnackbarSpy).toHaveBeenCalledTimes(1);
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      message: 'Successfully deleted pipeline: ' + testPipeline.name,
      open: true,
    }));
  });

  it('opens side panel on clicked node, shows message when node is not found in graph', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'some-node-id');
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
    info.inputs = [['key1', 'val1'], ['key2', 'val2']];
    info.outputs = [['key3', 'val3'], ['key4', 'val4']];
    info.nodeType = 'container';
    g.setNode('node1', { info, label: 'node1' });
    createGraphSpy.mockImplementation(() => g);

    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    tree.find('Graph').simulate('click', 'node1');
    expect(tree).toMatchSnapshot();
  });

  it('shows pipeline source code when config tab is clicked', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    tree.find('MD2Tabs').simulate('switch', 1);
    expect(tree.state('selectedTab')).toBe(1);
    expect(tree).toMatchSnapshot();
  });

  it('closes side panel when close button is clicked', async () => {
    tree = shallow(<PipelineDetails {...generateProps()} />);
    await getTemplateSpy;
    await TestUtils.flushPromises();
    tree.setState({ selectedNodeId: 'some-node-id' });
    tree.find('SidePanel').simulate('close');
    expect(tree.state('selectedNodeId')).toBe('');
    expect(tree).toMatchSnapshot();
  });
});
