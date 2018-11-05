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
import PipelineList from './PipelineList';
import TestUtils from '../TestUtils';
import { ApiPipeline } from '../apis/pipeline';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { shallow } from 'enzyme';
import { range } from 'lodash';

describe('PipelineList', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const listPipelinesSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');

  function generateProps(): PageProps {
    return {
      history: {} as any,
      location: '' as any,
      match: '' as any,
      toolbarProps: PipelineList.prototype.getInitialToolbarState(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  async function mountWithNPipelines(n: number) {
    listPipelinesSpy.mockImplementationOnce(() => ({
      pipelines: range(n).map(i => ({ id: 'test-pipeline-id' + i, name: 'test pipeline name' + i })),
    }));
    const tree = TestUtils.mountWithRouter(PipelineList, generateProps());
    await listPipelinesSpy;
    await TestUtils.flushPromises();
    tree.update(); // Make sure the tree is updated before searching it
    return tree;
  }

  beforeEach(() => {
    // Reset mocks
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();
    listPipelinesSpy.mockReset();
  });

  it('renders an empty list with empty state message', () => {
    expect(shallow(<PipelineList {...generateProps()} />)).toMatchSnapshot();
  });

  it('renders a list of one pipeline', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({
      pipelines: [{
        created_at: new Date(2018, 8, 22, 11, 5, 48),
        description: 'test pipeline description',
        name: 'pipeline1',
        parameters: [],
      } as ApiPipeline]
    });
    await listPipelinesSpy;
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one pipeline with no description or created date', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({
      pipelines: [{
        created_at: new Date(2018, 8, 22, 11, 5, 48),
        description: 'test pipeline description',
        name: 'pipeline1',
        parameters: [],
      } as ApiPipeline]
    });
    await listPipelinesSpy;
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one pipeline with error', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({
      pipelines: [{
        created_at: new Date(2018, 8, 22, 11, 5, 48),
        description: 'test pipeline description',
        error: 'oops! could not load pipeline',
        name: 'pipeline1',
        parameters: [],
      } as ApiPipeline]
    });
    await listPipelinesSpy;
    expect(tree).toMatchSnapshot();
  });

  it('calls Apis to list pipelines, sorted by creation time in descending order', async () => {
    listPipelinesSpy.mockImplementationOnce(() => ({ pipelines: [{ name: 'pipeline1' }] }));
    const tree = TestUtils.mountWithRouter(PipelineList, generateProps());
    await listPipelinesSpy;
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(tree.state()).toHaveProperty('pipelines', [{ name: 'pipeline1' }]);
  });

  it('has a Refresh button, clicking it refreshes the pipeline list', async () => {
    listPipelinesSpy.mockImplementation(() => ({ pipelines: [{ name: 'pipeline1' }] }));
    const instance = TestUtils.mountWithRouter(PipelineList, generateProps())
      .instance() as PipelineList;
    await listPipelinesSpy;
    expect(listPipelinesSpy.mock.calls.length).toBe(1);
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    await refreshBtn!.action();
    expect(listPipelinesSpy.mock.calls.length).toBe(2);
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('shows error banner when listing pipelines fails', async () => {
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'bad stuff happened');
    TestUtils.mountWithRouter(PipelineList, generateProps());
    await listPipelinesSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
      mode: 'error',
    }));
  });

  it('shows error banner when listing pipelines fails after refresh', async () => {
    const instance = TestUtils.mountWithRouter(PipelineList, generateProps())
      .instance() as PipelineList;
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'bad stuff happened');
    await refreshBtn!.action();
    expect(listPipelinesSpy.mock.calls.length).toBe(2);
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
      mode: 'error',
    }));
  });

  it('hides error banner when listing pipelines fails then succeeds', async () => {
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'bad stuff happened');
    const instance = TestUtils.mountWithRouter(PipelineList, generateProps())
      .instance() as PipelineList;
    await listPipelinesSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
      mode: 'error',
    }));
    updateBannerSpy.mockReset();

    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    listPipelinesSpy.mockImplementationOnce(() => ({ pipelines: [{ name: 'pipeline1' }] }));
    await refreshBtn!.action();
    expect(listPipelinesSpy.mock.calls.length).toBe(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('renders pipeline names as links to their details pages', async () => {
    const tree = await mountWithNPipelines(1);
    const link = tree.find('a[children="test pipeline name0"]');
    expect(link).toHaveLength(1);
    expect(link.prop('href')).toBe(RoutePage.PIPELINE_DETAILS.replace(
      ':' + RouteParams.pipelineId, 'test-pipeline-id0'
    ));
  });

  it('enables delete button when one pipeline is selected', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').simulate('click');
    expect(updateToolbarSpy.mock.calls).toHaveLength(2); // Initial call, then selection update
    const calls = updateToolbarSpy.mock.calls[1];
    expect(calls[0].actions.find((b: any) => b.title === 'Delete')).toHaveProperty('disabled', false);
  });

  it('enables delete button when two pipelines are selected', async () => {
    const tree = await mountWithNPipelines(2);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(1).simulate('click');
    expect(updateToolbarSpy.mock.calls).toHaveLength(3); // Initial call, then selection updates
    const calls = updateToolbarSpy.mock.calls[2];
    expect(calls[0].actions.find((b: any) => b.title === 'Delete')).toHaveProperty('disabled', false);
  });

  it('re-disables delete button pipelines are unselected', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(0).simulate('click');
    expect(updateToolbarSpy.mock.calls).toHaveLength(3); // Initial call, then selection updates
    const calls = updateToolbarSpy.mock.calls[2];
    expect(calls[0].actions.find((b: any) => b.title === 'Delete')).toHaveProperty('disabled', true);
  });

  it('shows delete dialog when delete button is clicked', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').at(0).simulate('click');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    expect(call).toHaveProperty('title', 'Delete 1 Pipeline?');
  });

  it('shows delete dialog when delete button is clicked, indicating several pipelines to delete', async () => {
    const tree = await mountWithNPipelines(5);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(2).simulate('click');
    tree.find('.tableRow').at(3).simulate('click');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    expect(call).toHaveProperty('title', 'Delete 3 Pipelines?');
  });

});
